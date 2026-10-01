package vbd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/config"
	"github.com/littlejo/xo-gocli/internal/output"
)

const (
	flagLimit = "limit"
	flagVM    = "vm"
)

func newListCommand() *cobra.Command {
	var (
		query string
		limit int
		vmID  string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List virtual block devices (VBDs)",
		Long: `List virtual block devices (VBDs) from Xen Orchestra.

A VBD is the attachment point between a VM and a VDI (virtual disk).

Examples:
  xo vbd list
  xo vbd list --vm <vm-id>
  xo vbd list --output json
  xo vbd list --query '[].VDI'
  xo vbd list --limit 10`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := output.ParseFormat(cli.OutputFormat(cmd))
			if err != nil {
				return err
			}
			if limit < 0 {
				return fmt.Errorf("--limit must be greater than or equal to 0")
			}
			if err := output.ValidateQuery(query); err != nil {
				return err
			}
			filter := ""
			if vmID != "" {
				vm, err := parseID(vmID)
				if err != nil {
					return err
				}
				filter = "VM:" + vm.String()
			}

			cfg, err := config.Load(cli.ProfileName(cmd))
			if err != nil {
				return err
			}
			xo, err := cli.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			vbds, err := xo.VBD().GetAll(cmd.Context(), limit, filter)
			if err != nil {
				return cli.InsecureHint(fmt.Sprintf("cannot list VBDs: %v", err), cfg.Insecure)
			}

			return renderVBDs(cmd.OutOrStdout(), format, vbds, query)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. '[].VDI'")
	flags.IntVar(&limit, flagLimit, 0, "maximum number of VBDs to return (0 for no limit)")
	flags.StringVar(&vmID, flagVM, "", "only list the VBDs of the given VM (UUID)")

	return cmd
}

// renderVBDs applies the optional --query expression and renders the result
// in the requested format.
func renderVBDs(w io.Writer, format output.Format, vbds []*payloads.VBD, query string) error {
	queryResult, err := output.Query(query, vbds)
	if err != nil {
		return err
	}

	table := output.Table{
		Headers: []string{"ID", "VM", "VDI", "DEVICE", "MODE", "ATTACHED"},
	}
	for _, vbd := range vbds {
		table.Rows = append(table.Rows, []string{
			vbd.ID.String(),
			vbd.VM.String(),
			vdiText(vbd.VDI),
			deviceText(vbd.Device),
			modeText(vbd),
			boolText(vbd.Attached),
		})
	}

	// For structured formats without a query, normalize the SDK types so the
	// output only contains the requested data.
	var raw any = vbds
	if format != output.FormatTable && (queryResult == nil || !queryResult.Present) {
		normalized, err := output.Normalize(vbds)
		if err != nil {
			return err
		}
		raw = normalized
	}

	return output.Render(w, format, table, raw, queryResult)
}
