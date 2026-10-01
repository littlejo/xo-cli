package vbd

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/output"
)

func newConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect <id>",
		Short: "Hot-plug a VBD (attach the disk to a running VM)",
		Long: `Hot-plug a VBD, dynamically attaching its VDI to the running VM.

The VBD is referenced by its UUID, as returned by 'xo vbd list'. The VM must
be running for a hot-plug; if it is halted the disk is attached at boot
instead (no action needed).

Examples:
  xo vbd connect 33333333-3333-4333-8333-333333333333`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAction(cmd, args[0], "connect", func(ctx context.Context, xo library.Library, id uuid.UUID) (string, error) {
				return xo.VBD().Connect(ctx, id)
			})
		},
	}
	return cmd
}

func newDisconnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disconnect <id>",
		Short: "Hot-unplug a VBD (detach the disk from a running VM)",
		Long: `Hot-unplug a VBD, dynamically detaching its VDI from the running VM.

The VBD is referenced by its UUID, as returned by 'xo vbd list'. Use
'xo vbd delete <id>' instead to remove the attachment entirely (keeping the
VDI).

Examples:
  xo vbd disconnect 33333333-3333-4333-8333-333333333333`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAction(cmd, args[0], "disconnect", func(ctx context.Context, xo library.Library, id uuid.UUID) (string, error) {
				return xo.VBD().Disconnect(ctx, id)
			})
		},
	}
	return cmd
}

// runAction implements the common flow shared by connect/disconnect: load the
// client, resolve the target VBD (as an existence check), run the action, and
// print the resulting task ID (or a plain confirmation when the action is
// synchronous).
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
	vbd, err := xo.VBD().Get(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return fmt.Errorf("VBD %q not found", idStr)
		}
		return cli.InsecureHint(fmt.Sprintf("cannot resolve VBD %q: %v", idStr, err), cfg.Insecure)
	}
	label := describe(vbd)

	taskID, err := perform(ctx, xo, id)
	if err != nil {
		return cli.InsecureHint(fmt.Sprintf("cannot %s VBD %s: %v", verb, label, err), cfg.Insecure)
	}

	return renderActionResult(cmd, verb, label, taskID)
}

// renderActionResult prints the outcome of an async VBD action: a friendly
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
		doc := map[string]any{"action": verb, "vbd": label}
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
