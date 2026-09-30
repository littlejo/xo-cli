package vm

import (
	"fmt"
	"io"

	"github.com/docker/go-units"
	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/config"
	"github.com/littlejo/xo-gocli/internal/output"
)

const flagVDIType = "type"

func newVdisCommand() *cobra.Command {
	var (
		query   string
		limit   int
		vdiType string
	)

	cmd := &cobra.Command{
		Use:   "vdis <id>",
		Short: "List the virtual disks (VDIs) of a virtual machine",
		Long: `List the virtual disks (VDIs) attached to a virtual machine.

The VM is referenced by its UUID, as returned by 'xo vm list'.

Examples:
  xo vm vdis 550e8400-e29b-41d4-a716-446655440001
  xo vm vdis <id> --output json
  xo vm vdis <id> --query '[].name_label'
  xo vm vdis <id> --type user
  xo vm vdis <id> --limit 10`,
		Args: cobra.ExactArgs(1),
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
			filter := vdiType
			if filter != "" {
				filter = "VDI_type:" + vdiType
			}

			id, err := parseID(args[0])
			if err != nil {
				return err
			}

			cfg, err := config.Load(cli.ProfileName(cmd))
			if err != nil {
				return err
			}
			xo, err := cli.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			// Existence check through the typed service so a missing VM fails
			// with the usual "not found" error before listing its disks.
			if _, err := xo.VM().GetByID(cmd.Context(), id); err != nil {
				if isNotFound(err) {
					return fmt.Errorf("VM %q not found", args[0])
				}
				return cli.InsecureHint(fmt.Sprintf("cannot resolve VM %q: %v", args[0], err), cfg.Insecure)
			}

			vdis, err := xo.VM().GetVDIs(cmd.Context(), id, limit, filter)
			if err != nil {
				return cli.InsecureHint(fmt.Sprintf("cannot list VDIs of VM %q: %v", args[0], err), cfg.Insecure)
			}

			return renderVDIs(cmd.OutOrStdout(), format, vdis, query)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. '[].name_label'")
	flags.IntVar(&limit, flagLimit, 0, "maximum number of VDIs to return (0 for no limit)")
	flags.StringVar(&vdiType, flagVDIType, "", "filter by VDI type: user, system, suspend, rrd, …")

	return cmd
}

// renderVDIs applies the optional --query expression and renders the result in
// the requested format.
func renderVDIs(w io.Writer, format output.Format, vdis []*payloads.VDI, query string) error {
	queryResult, err := output.Query(query, vdis)
	if err != nil {
		return err
	}

	table := output.Table{
		Headers: []string{"ID", "NAME", "TYPE", "SIZE", "USAGE", "SR"},
	}
	for _, vdi := range vdis {
		table.Rows = append(table.Rows, []string{
			vdi.ID.String(),
			vdi.NameLabel,
			string(vdi.VDIType),
			units.HumanSize(float64(vdi.Size)),
			units.HumanSize(float64(vdi.Usage)),
			vdi.SR.String(),
		})
	}

	// For structured formats without a query, normalize the SDK types so the
	// output only contains the requested data.
	var raw any = vdis
	if format != output.FormatTable && (queryResult == nil || !queryResult.Present) {
		normalized, err := output.Normalize(vdis)
		if err != nil {
			return err
		}
		raw = normalized
	}

	return output.Render(w, format, table, raw, queryResult)
}
