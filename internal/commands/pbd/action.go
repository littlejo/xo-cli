package pbd

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/output"
)

// newPlugCommand builds 'xo pbd plug': connect the PBD, attaching the SR to
// its host (SDK PBD().Plug, asynchronous: returns a task id).
func newPlugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plug <id>",
		Short: "Plug a PBD (connect the SR to its host)",
		Long: `Plug a PBD, connecting its storage repository (SR) to the host it belongs
to. Use 'xo pbd unplug <id>' to do the opposite.

The PBD is referenced by its UUID, as returned by 'xo pbd list'. The action
is asynchronous: the command prints the task id and returns. Track it with
'xo task get <task-id>' or 'xo task wait <task-id>'.

Examples:
  xo pbd plug 550e8400-e29b-41d4-a716-446655440001
  xo pbd plug <id> --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAction(cmd, args[0], "plug", func(ctx context.Context, xo library.Library, id uuid.UUID) (string, error) {
				return xo.PBD().Plug(ctx, id)
			})
		},
	}
	return cmd
}

// newUnplugCommand builds 'xo pbd unplug': disconnect the PBD, detaching the
// SR from its host (SDK PBD().Unplug, asynchronous: returns a task id).
func newUnplugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unplug <id>",
		Short: "Unplug a PBD (disconnect the SR from its host)",
		Long: `Unplug a PBD, disconnecting its storage repository (SR) from the host it
belongs to. Use 'xo pbd plug <id>' to re-connect it.

The PBD is referenced by its UUID, as returned by 'xo pbd list'. The action
is asynchronous: the command prints the task id and returns. Track it with
'xo task get <task-id>' or 'xo task wait <task-id>'.

Examples:
  xo pbd unplug 550e8400-e29b-41d4-a716-446655440001
  xo pbd unplug <id> --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAction(cmd, args[0], "unplug", func(ctx context.Context, xo library.Library, id uuid.UUID) (string, error) {
				return xo.PBD().Unplug(ctx, id)
			})
		},
	}
	return cmd
}

// runAction implements the common flow shared by plug/unplug: load the
// client, resolve the target PBD (as an existence check), run the action, and
// print the resulting task ID.
func runAction(cmd *cobra.Command, idStr string, verb string, perform func(ctx context.Context, xo library.Library, id uuid.UUID) (string, error)) error {
	xo, cfg, err := newClient(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	id, err := parseID(idStr)
	if err != nil {
		return err
	}
	pbd, err := xo.PBD().Get(ctx, id)
	if err != nil {
		return cli.NotFound("PBD", "resolve", idStr, err, cfg.Insecure)
	}

	taskID, err := perform(ctx, xo, id)
	if err != nil {
		return cli.InsecureHint(fmt.Sprintf("cannot %s PBD %q: %v", verb, idStr, err), cfg.Insecure)
	}

	return renderActionResult(cmd, verb, describe(pbd), taskID)
}

// describe renders a PBD for messages: the PBD has no name_label, so the label
// links its host and SR.
func describe(pbd *payloads.PBD) string {
	return fmt.Sprintf("PBD %s (host %s, SR %s)", pbd.ID.String(), pbd.Host.String(), pbd.SR.String())
}

// renderActionResult prints the outcome of an async PBD action: a friendly
// line for the human formats and a small machine document for json/yaml. When
// the action was synchronous (empty task id), no task id is printed.
func renderActionResult(cmd *cobra.Command, verb, label, taskID string) error {
	format, err := output.ParseFormat(cli.OutputFormat(cmd))
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	switch format {
	case output.FormatJSON, output.FormatYAML:
		doc := map[string]any{"action": verb, "pbd": label}
		if taskID != "" {
			doc["task_id"] = taskID
		}
		raw, err := output.Normalize(doc)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, raw, nil)
	default:
		if taskID == "" {
			_, err := fmt.Fprintf(w, "Requested %s of %s\n", verb, label)
			return err
		}
		_, err := fmt.Fprintf(w, "Requested %s of %s (task %s)\n", verb, label, taskID)
		return err
	}
}
