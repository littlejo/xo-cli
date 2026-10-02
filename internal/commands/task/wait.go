package task

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/config"
	"github.com/littlejo/xo-gocli/internal/output"
	"github.com/littlejo/xo-gocli/internal/taskwait"
)

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
			// (Ctrl+C propagates from the CLI down to the HTTP layer). The
			// polling and rendering are shared with the --wait flag of the
			// asynchronous actions (internal/taskwait).
			return taskwait.Wait(cmd.Context(), httpClient, taskwait.Options{
				Out:      cmd.OutOrStdout(),
				Stderr:   cmd.ErrOrStderr(),
				ID:       id,
				Format:   format,
				Query:    query,
				Deadline: waitTimeout,
				NotFound: func(id string, err error) error {
					return taskNotFound(id, err, cfg.Insecure)
				},
			})
		},
	}
	cmd.Flags().StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the completed task, e.g. 'status'")
	// Local --timeout: the wait deadline. It shadows the global HTTP --timeout
	// on this command (see the Long help); the HTTP timeout is set with
	// $XOA_TIMEOUT when needed.
	cmd.Flags().DurationVar(&waitTimeout, cli.FlagTimeout, 0, "wait at most this long for the task to complete, e.g. 2m (default: wait until it completes)")

	return cmd
}
