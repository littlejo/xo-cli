package vbd

import (
	"fmt"
	"io"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/config"
	"github.com/littlejo/xo-gocli/internal/output"
)

const flagQuery = "query"

func newGetCommand() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a virtual block device (VBD)",
		Long: `Get a virtual block device (VBD) from Xen Orchestra.

A VBD is the attachment point between a VM and a VDI (virtual disk). The VBD
is referenced by its UUID; it can be found in the VM's 'other_config' or by
listing VBDs and matching on the VM.

Examples:
  xo vbd get 33333333-3333-4333-8333-333333333333
  xo vbd get <id> --output json
  xo vbd get <id> --query 'VDI'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			if err := output.ValidateQuery(query); err != nil {
				return err
			}
			format, err := output.ParseFormat(cli.OutputFormat(cmd))
			if err != nil {
				return err
			}

			xo, cfg, err := newClient(cmd)
			if err != nil {
				return err
			}

			vbd, err := xo.VBD().Get(cmd.Context(), id)
			if err != nil {
				return notFound(args[0], err, cfg.Insecure)
			}

			return renderVBD(cmd.OutOrStdout(), format, vbd, query)
		},
	}

	cmd.Flags().StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. 'VDI'")
	return cmd
}

// parseID converts a positional VBD identifier into a UUID.
func parseID(id string) (uuid.UUID, error) {
	u, err := uuid.FromString(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid VBD id %q (expected a UUID)", id)
	}
	return u, nil
}

// newClient loads the selected profile and builds an authenticated SDK v2
// client. Commands must pass their cobra context to the SDK operations so that
// cancellation (Ctrl+C) reaches the HTTP layer.
func newClient(cmd *cobra.Command) (library.Library, *config.ClientConfig, error) {
	cfg, err := config.Load(cli.ProfileName(cmd))
	if err != nil {
		return nil, nil, err
	}
	xo, err := cli.NewClient(cmd, cfg)
	if err != nil {
		return nil, nil, err
	}
	return xo, cfg, nil
}

// notFound delegates to cli.NotFound, which reports a 404 concisely and keeps
// the raw API error as debug detail.
func notFound(id string, err error, insecure bool) error {
	return cli.NotFound("VBD", "get", id, err, insecure)
}

// renderVBD renders a single VBD in the requested format. The human format
// uses a single-row table with the same columns as 'xo vbd list'; the
// structured formats emit the normalized object (or its --query projection).
func renderVBD(w io.Writer, format output.Format, vbd *payloads.VBD, query string) error {
	if format == output.FormatTable && query == "" {
		table := output.Table{
			Headers: []string{"ID", "VM", "VDI", "DEVICE", "MODE", "ATTACHED"},
			Rows: [][]string{{
				vbd.ID.String(),
				vbd.VM.String(),
				vdiText(vbd.VDI),
				deviceText(vbd.Device),
				modeText(vbd),
				boolText(vbd.Attached),
			}},
		}
		return output.Render(w, format, table, vbd, nil)
	}

	if query != "" {
		queryResult, err := output.Query(query, vbd)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, vbd, queryResult)
	}

	normalized, err := output.Normalize(vbd)
	if err != nil {
		return err
	}
	return output.Render(w, format, output.Table{}, normalized, nil)
}

func vdiText(vdi *uuid.UUID) string {
	if vdi == nil || *vdi == uuid.Nil {
		return "-"
	}
	return vdi.String()
}

func deviceText(device *string) string {
	if device == nil || *device == "" {
		return "-"
	}
	return *device
}

func modeText(vbd *payloads.VBD) string {
	if vbd.ReadOnly {
		return "RO"
	}
	return "RW"
}

func boolText(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
