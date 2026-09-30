package task

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/config"
	"github.com/littlejo/xo-gocli/internal/output"
)

func newGetCommand() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a task",
		Long: `Get a task from Xen Orchestra.

The task is referenced by its id, as returned by 'xo task list' or by an
asynchronous operation (for example 'xo vm start').

Examples:
  xo task get a1b2c3d4e5f6
  xo task get <id> --output json
  xo task get <id> --query 'result'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if id == "" || strings.Contains(id, "/") {
				return fmt.Errorf("invalid task id %q", id)
			}
			if err := output.ValidateQuery(query); err != nil {
				return err
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

			var task map[string]any
			endpoint := tasksEndpoint + "/" + id
			if err := client.TypedGet(cmd.Context(), httpClient, endpoint, listParams{Fields: "*"}, &task); err != nil {
				return taskNotFound(id, err, cfg.Insecure)
			}

			return renderTask(cmd.OutOrStdout(), format, task, query)
		},
	}

	cmd.Flags().StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. 'result'")
	return cmd
}

// taskNotFound turns a lookup failure into the concise "task not found" form
// when the API returned a 404, and keeps the original error otherwise.
func taskNotFound(id string, err error, insecure bool) error {
	if err != nil && strings.Contains(err.Error(), "404") {
		return fmt.Errorf("task %q not found", id)
	}
	return cli.InsecureHint(fmt.Sprintf("cannot get task %q: %v", id, err), insecure)
}

// renderTask renders a single task in the requested format. The human format
// uses a single-row table with the 'xo task list' columns plus a message
// column (the useful part of the result for failed tasks); the structured
// formats emit the normalized object (or its --query projection).
func renderTask(w io.Writer, format output.Format, t map[string]any, query string) error {
	if format == output.FormatTable && query == "" {
		props, _ := t["properties"].(map[string]any)
		table := output.Table{
			Headers: []string{"ID", "STATUS", "TYPE", "NAME", "STARTED", "ENDED", "MESSAGE"},
			Rows: [][]string{{
				strField(t, "id"),
				strField(t, "status"),
				strField(props, "type"),
				strField(props, "name"),
				timeText(t["start"]),
				timeText(t["end"]),
				messageText(t),
			}},
		}
		return output.Render(w, format, table, t, nil)
	}

	if query != "" {
		queryResult, err := output.Query(query, t)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, t, queryResult)
	}

	normalized, err := output.Normalize(t)
	if err != nil {
		return err
	}
	return output.Render(w, format, output.Table{}, normalized, nil)
}

// messageText extracts a human readable message from the task result, which
// may be an object with a "message" field or a plain string.
func messageText(t map[string]any) string {
	if s, ok := t["result"].(string); ok {
		return s
	}
	result, ok := t["result"].(map[string]any)
	if !ok {
		return ""
	}
	return strField(result, "message")
}
