package sr

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

const (
	flagQuery = "query"
	flagLimit = "limit"
	flagType  = "type"
)

func newListCommand() *cobra.Command {
	var (
		query  string
		limit  int
		srType string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List storage repositories",
		Long: `List storage repositories (SRs) from Xen Orchestra.

Examples:
  xo sr list
  xo sr list --output json
  xo sr list --query '[].name_label'
  xo sr list --type lvm
  xo sr list --limit 10`,
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
			filter := srType
			if filter != "" {
				filter = "SR_type:" + srType
			}

			cfg, err := config.Load(cli.ProfileName(cmd))
			if err != nil {
				return err
			}

			xo, err := cli.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			srs, err := xo.SR().GetAll(cmd.Context(), limit, filter)
			if err != nil {
				return cli.InsecureHint(fmt.Sprintf("cannot list storage repositories: %v", err), cfg.Insecure)
			}

			return renderSRs(cmd.OutOrStdout(), format, srs, query)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&query, flagQuery, "", "JMESPath expression applied to the result, e.g. '[].name_label'")
	flags.IntVar(&limit, flagLimit, 0, "maximum number of SRs to return (0 for no limit)")
	flags.StringVar(&srType, flagType, "", "filter by SR type (XO live-filter: case-insensitive substring, so 'lvm' also matches 'lvmoiscsi')")

	return cmd
}

// renderSRs applies the optional --query expression and renders the result in
// the requested format.
func renderSRs(w io.Writer, format output.Format, srs []*payloads.StorageRepository, query string) error {
	queryResult, err := output.Query(query, srs)
	if err != nil {
		return err
	}

	table := output.Table{
		Headers: []string{"ID", "NAME", "TYPE", "SIZE", "USAGE", "CONTAINER"},
	}
	for _, sr := range srs {
		table.Rows = append(table.Rows, []string{
			sr.ID.String(),
			sr.NameLabel,
			sr.SRType,
			units.HumanSize(sr.Size),
			units.HumanSize(sr.Usage),
			sr.Container.String(),
		})
	}

	// For structured formats without a query, normalize the SDK types so the
	// output only contains the requested data.
	var raw any = srs
	if format != output.FormatTable && (queryResult == nil || !queryResult.Present) {
		normalized, err := output.Normalize(srs)
		if err != nil {
			return err
		}
		raw = normalized
	}

	return output.Render(w, format, table, raw, queryResult)
}
