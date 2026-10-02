package pbd

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
	flagQuery = "query"
	flagLimit = "limit"
)

func newListCommand() *cobra.Command {
	var (
		query string
		limit int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List physical block devices (PBDs)",
		Long: `List physical block devices (PBDs) from Xen Orchestra.

A PBD is the connection between a host and a storage repository (SR); it is
what "plugs" an SR into a host. Use 'xo pbd plug' / 'xo pbd unplug' to
connect or disconnect it.

Examples:
  xo pbd list
  xo pbd list --output json
  xo pbd list --query '[].attached'
  xo pbd list --limit 10`,
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

			cfg, err := config.Load(cli.ProfileName(cmd))
			if err != nil {
				return err
			}
			xo, err := cli.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			pbds, err := xo.PBD().GetAll(cmd.Context(), limit, "")
			if err != nil {
				return cli.InsecureHint(fmt.Sprintf("cannot list PBDs: %v", err), cfg.Insecure)
			}

			return renderPBDs(cmd.OutOrStdout(), format, pbds, query)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. '[].attached'")
	flags.IntVar(&limit, flagLimit, 0, "maximum number of PBDs to return (0 for no limit)")
	return cmd
}

// deviceOf returns the block device name from the PBD's device_config
// (e.g. "/dev/sda") or "-" when the PBD has no such entry (e.g. some network
// or iSCSI SRs use server/serverpath keys instead).
func deviceOf(pbd *payloads.PBD) string {
	if dev, ok := pbd.DeviceConfig["device"]; ok && dev != "" {
		return dev
	}
	return "-"
}

func boolText(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// renderPBDs applies the optional --query expression and renders the result in
// the requested format.
func renderPBDs(w io.Writer, format output.Format, pbds []*payloads.PBD, query string) error {
	queryResult, err := output.Query(query, pbds)
	if err != nil {
		return err
	}

	table := output.Table{
		Headers: []string{"ID", "HOST", "SR", "POOL", "ATTACHED", "DEVICE"},
	}
	for _, pbd := range pbds {
		table.Rows = append(table.Rows, []string{
			pbd.ID.String(),
			pbd.Host.String(),
			pbd.SR.String(),
			pbd.Pool.String(),
			boolText(pbd.Attached),
			deviceOf(pbd),
		})
	}

	// For structured formats without a query, normalize the SDK types so the
	// output only contains the requested data.
	var raw any = pbds
	if format != output.FormatTable && (queryResult == nil || !queryResult.Present) {
		normalized, err := output.Normalize(pbds)
		if err != nil {
			return err
		}
		raw = normalized
	}

	return output.Render(w, format, table, raw, queryResult)
}
