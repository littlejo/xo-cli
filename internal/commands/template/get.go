package template

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
		Short: "Get a VM template",
		Long: `Get a VM template from Xen Orchestra.

The template is referenced by its id, as returned by 'xo template list'.

Examples:
  xo template get d31e47fd-a70e-d849-883e-c17193472710-6959dfe8-534c-4c58-8a8c-3c3792293543
  xo template get <id> --output json
  xo template get <id> --query 'name_label'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if id == "" || strings.Contains(id, "/") {
				return fmt.Errorf("invalid template id %q", id)
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

			var template map[string]any
			endpoint := templatesEndpoint + "/" + id
			if err := client.TypedGet(cmd.Context(), httpClient, endpoint, listParams{Fields: "*"}, &template); err != nil {
				return notFound(id, err, cfg.Insecure)
			}

			return renderTemplate(cmd.OutOrStdout(), format, template, query)
		},
	}

	cmd.Flags().StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. 'name_label'")
	return cmd
}

// notFound delegates to cli.NotFound, which reports a 404 concisely and keeps
// the raw API error as debug detail.
func notFound(id string, err error, insecure bool) error {
	return cli.NotFound("template", "get", id, err, insecure)
}

// renderTemplate renders a single template in the requested format. The human
// format uses a single-row table with the same columns as 'xo template list';
// the structured formats emit the normalized object (or its --query projection).
func renderTemplate(w io.Writer, format output.Format, t map[string]any, query string) error {
	if format == output.FormatTable && query == "" {
		table := output.Table{
			Headers: []string{"ID", "NAME", "DEFAULT", "MEMORY", "CPUS", "POOL"},
			Rows: [][]string{{
				strField(t, "id"),
				strField(t, "name_label"),
				fmt.Sprintf("%t", boolField(t, "isDefaultTemplate")),
				memoryText(t),
				fmt.Sprintf("%d", nestedInt(t, "CPUs", "number")),
				strField(t, "$pool"),
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
