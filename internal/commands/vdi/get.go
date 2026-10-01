package vdi

import (
	"fmt"
	"io"
	"strings"

	"github.com/docker/go-units"
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
		Short: "Get a virtual disk (VDI)",
		Long: `Get a virtual disk (VDI) from Xen Orchestra.

The VDI is referenced by its UUID, as returned by 'xo vdi list' or
'xo vm vdis <id>'.

Examples:
  xo vdi get 11111111-1111-4111-8111-111111111111
  xo vdi get <id> --output json
  xo vdi get <id> --query 'name_label'`,
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

			vdi, err := xo.VDI().Get(cmd.Context(), id)
			if err != nil {
				return notFound(args[0], err, cfg.Insecure)
			}

			return renderVDI(cmd.OutOrStdout(), format, vdi, query)
		},
	}

	cmd.Flags().StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. 'name_label'")
	return cmd
}

// parseID converts a positional VDI identifier into a UUID.
func parseID(id string) (uuid.UUID, error) {
	u, err := uuid.FromString(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid VDI id %q (expected a UUID)", id)
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

// notFound turns a lookup failure into the concise "VDI not found" form when
// the API returned a 404, and keeps the original error otherwise. It passes
// the profile's insecure state through so the TLS hint is only suggested when
// it would actually help.
func notFound(id string, err error, insecure bool) error {
	if err != nil && strings.Contains(err.Error(), "404") {
		return fmt.Errorf("VDI %q not found", id)
	}
	return cli.InsecureHint(fmt.Sprintf("cannot get VDI %q: %v", id, err), insecure)
}

// renderVDI renders a single VDI in the requested format. The human format
// uses a single-row table with the same columns as 'xo vdi list'; the
// structured formats emit the normalized object (or its --query projection).
func renderVDI(w io.Writer, format output.Format, vdi *payloads.VDI, query string) error {
	if format == output.FormatTable && query == "" {
		table := output.Table{
			Headers: []string{"ID", "NAME", "TYPE", "SIZE", "USAGE", "SR"},
			Rows: [][]string{{
				vdi.ID.String(),
				vdi.NameLabel,
				string(vdi.VDIType),
				units.HumanSize(float64(vdi.Size)),
				units.HumanSize(float64(vdi.Usage)),
				vdi.SR.String(),
			}},
		}
		return output.Render(w, format, table, vdi, nil)
	}

	if query != "" {
		queryResult, err := output.Query(query, vdi)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, vdi, queryResult)
	}

	normalized, err := output.Normalize(vdi)
	if err != nil {
		return err
	}
	return output.Render(w, format, output.Table{}, normalized, nil)
}
