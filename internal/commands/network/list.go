package network

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
		Short: "List networks",
		Long: `List networks from Xen Orchestra.

Examples:
  xo network list
  xo network list --output json
  xo network list --query '[].name_label'
  xo network list --limit 10`,
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

			networks, err := xo.Network().GetAll(cmd.Context(), limit, "")
			if err != nil {
				return cli.InsecureHint(fmt.Sprintf("cannot list networks: %v", err), cfg.Insecure)
			}

			return renderNetworks(cmd.OutOrStdout(), format, networks, query)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. '[].name_label'")
	flags.IntVar(&limit, flagLimit, 0, "maximum number of networks to return (0 for no limit)")

	return cmd
}

// renderNetworks applies the optional --query expression and renders the
// result in the requested format.
func renderNetworks(w io.Writer, format output.Format, networks []*payloads.Network, query string) error {
	queryResult, err := output.Query(query, networks)
	if err != nil {
		return err
	}

	table := output.Table{
		Headers: []string{"ID", "NAME", "BRIDGE", "TYPE", "MTU", "VIFS", "POOL"},
	}
	for _, n := range networks {
		table.Rows = append(table.Rows, []string{
			n.ID.String(),
			n.NameLabel,
			n.Bridge,
			string(n.Type),
			fmt.Sprintf("%d", n.MTU),
			fmt.Sprintf("%d", len(n.VIFs)),
			n.Pool.String(),
		})
	}

	// For structured formats without a query, normalize the SDK types so the
	// output only contains the requested data.
	var raw any = networks
	if format != output.FormatTable && (queryResult == nil || !queryResult.Present) {
		normalized, err := output.Normalize(networks)
		if err != nil {
			return err
		}
		raw = normalized
	}

	return output.Render(w, format, table, raw, queryResult)
}
