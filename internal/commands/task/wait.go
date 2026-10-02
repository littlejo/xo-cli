package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/config"
	"github.com/littlejo/xo-gocli/internal/output"
)

// pollInterval is how often a still-pending task is re-fetched. It matches the
// SDK's own task.Wait polling cadence (2 s). It is a variable (not a const) so
// tests can shorten it.
var pollInterval = 2 * time.Second

// terminalStatuses are the task statuses that will not change any further; the
// wait stops as soon as a task reaches one of them. interrupted is included
// even though the SDK's Task().Wait does not treat it as terminal: an
// interrupted task will not progress, so waiting for it is pointless.
var terminalStatuses = map[string]bool{
	"success":     true,
	"failure":     true,
	"interrupted": true,
}

func newWaitCommand() *cobra.Command {
	var (
		query       string
		waitTimeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "wait <id>",
		Short: "Wait for a task to complete",
		Long: `Wait for a task to reach a terminal state (success, failure or
interrupted), polling every 2 seconds, and print it like 'xo task get'.

The command blocks until the task completes. Ctrl+C cancels the wait. Use
--timeout to bound how long to wait; without it the wait is unbounded (it ends
only when the task completes or is interrupted).

Exit status reflects the task's outcome, so it is safe to use as a gate in
scripts:
  - 0 when the task completes successfully
  - non-zero when the task fails, is interrupted, the --timeout deadline is
    reached, or the task does not exist
The completed task is always printed (in the chosen --output format); a
non-zero exit is reported on stderr.

Note: this command's --timeout is the *wait* deadline, not the global HTTP
client --timeout. It shadows the global flag on this command, so the HTTP
timeout (applied to each poll) is still controlled by $XOA_TIMEOUT when needed.

The task is referenced by its id, as returned by 'xo task list' or by an
asynchronous operation (for example 'xo vm start').

Examples:
  xo task wait a1b2c3d4e5f6
  xo task wait <id> --timeout 5m
  xo task wait <id> --output json
  xo task wait <id> --query 'status'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if id == "" || strings.Contains(id, "/") {
				return fmt.Errorf("invalid task id %q", id)
			}
			if err := output.ValidateQuery(query); err != nil {
				return err
			}
			if waitTimeout < 0 {
				return fmt.Errorf("--timeout must be greater than or equal to 0")
			}
			format, err := output.ParseFormat(cli.OutputFormat(cmd))
			if err != nil {
				return err
			}

			cfg, err := config.Load(cli.ProfileName(cmd))
			if err != nil {
				return err
			}

			httpClient, err := cli.NewHTTPClient(cmd, cfg)
			if err != nil {
				return err
			}

			// The wait is bounded by --timeout when given; otherwise it runs
			// until the task completes or the command context is cancelled
			// (Ctrl+C propagates from the CLI down to the HTTP layer).
			ctx := cmd.Context()
			if waitTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, waitTimeout)
				defer cancel()
			}

			return runWait(cmd, ctx, httpClient, cfg, id, format, query, waitTimeout)
		},
	}

	cmd.Flags().StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the completed task, e.g. 'status'")
	// Local --timeout: the wait deadline. It shadows the global HTTP --timeout
	// on this command (see the Long help); the HTTP timeout is set with
	// $XOA_TIMEOUT when needed.
	cmd.Flags().DurationVar(&waitTimeout, cli.FlagTimeout, 0, "wait at most this long for the task to complete, e.g. 2m (default: wait until it completes)")

	return cmd
}

// runWait polls the task until it reaches a terminal state, then renders it
// like 'xo task get' and (unless it succeeded) reports the outcome as an error
// so the exit status reflects it. It honors ctx for cancellation (Ctrl+C) and
// the optional wait deadline set by --timeout.
//
// The poll loop uses the SDK's own REST client (client.TypedGet), the same
// single API boundary as 'xo task get'. The SDK's typed Task().Wait is
// deliberately not used because it unmarshals into payloads.Task, which cannot
// represent a task whose "result" is a plain string (a documented XO quirk —
// see TestTaskGetStringResult) and would loop to the deadline on such tasks;
// it also does not treat "interrupted" as terminal. If the SDK gains a
// result type that tolerates both shapes, this command should switch to
// Task().WaitWithTimeout.
func runWait(cmd *cobra.Command, ctx context.Context, httpClient *client.Client, cfg *config.ClientConfig, id string, format output.Format, query string, timeout time.Duration) error {
	w := cmd.OutOrStdout()
	noted := false

	for {
		var task map[string]any
		err := client.TypedGet(ctx, httpClient, tasksEndpoint+"/"+id, listParams{Fields: "*"}, &task)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				// The poll itself was cut off because the wait ended
				// (--timeout hit or Ctrl+C).
				return waitEnded(id, timeout, ctxErr)
			}
			return taskNotFound(id, err, cfg.Insecure)
		}

		status := strField(task, "status")
		if terminalStatuses[status] {
			// The task has finished: show it, then let the exit status reflect
			// the outcome (non-zero on failure / interruption).
			if err := renderTask(w, format, task, query); err != nil {
				return err
			}
			if status != "success" {
				return taskOutcomeError(id, status, task)
			}
			return nil
		}

		// Still pending: announce the wait once, then wait for the next poll
		// while honoring the context.
		if !noted {
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Waiting for task %s to complete...\n", id); err != nil {
				return err
			}
			noted = true
		}
		select {
		case <-ctx.Done():
			return waitEnded(id, timeout, ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// waitEnded turns an ended wait into a concise error. A --timeout deadline is
// reported as a timeout; a cancellation (Ctrl+C) as an interruption.
func waitEnded(id string, timeout time.Duration, ctxErr error) error {
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		return fmt.Errorf("task %s did not complete within %s", id, timeout)
	}
	return fmt.Errorf("waiting for task %s was interrupted", id)
}

// taskOutcomeError reports a completed task that did not succeed, carrying the
// task's own message when available (the useful part of a failed result).
func taskOutcomeError(id, status string, task map[string]any) error {
	if status == "interrupted" {
		return fmt.Errorf("task %s was interrupted", id)
	}
	if msg := messageText(task); msg != "" {
		return fmt.Errorf("task %s failed: %s", id, msg)
	}
	return fmt.Errorf("task %s failed", id)
}
