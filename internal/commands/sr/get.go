package sr

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

func newGetCommand() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a storage repository",
		Long: `Get a storage repository (SR) from Xen Orchestra.

The SR is referenced by its UUID, as returned by 'xo sr list'.

Examples:
  xo sr get 550e8400-e29b-41d4-a716-446655440001
  xo sr get <id> --output json
  xo sr get <id> --query 'name_label'`,
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

			sr, err := xo.SR().Get(cmd.Context(), id)
			if err != nil {
				return notFound(args[0], err, cfg.Insecure)
			}

			return renderSR(cmd.OutOrStdout(), format, sr, query)
		},
	}

	cmd.Flags().StringVar(&query, flagQuery, "", "JMESPath expression applied to the result, e.g. 'name_label'")
	return cmd
}

// parseID converts a positional SR identifier into a UUID.
func parseID(id string) (uuid.UUID, error) {
	u, err := uuid.FromString(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid SR id %q (expected a UUID)", id)
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

// notFound turns a lookup failure into the concise "SR not found" form when the
// API returned a 404, and keeps the original error otherwise. It passes the
// profile's insecure state through so the TLS hint is only suggested when it
// would actually help.
func notFound(id string, err error, insecure bool) error {
	if err != nil && strings.Contains(err.Error(), "404") {
		return fmt.Errorf("SR %q not found", id)
	}
	return cli.InsecureHint(fmt.Sprintf("cannot get SR %q: %v", id, err), insecure)
}

// renderSR renders a single SR in the requested format. The human format uses
// a single-row table with the same columns as 'xo sr list'; the structured
// formats emit the normalized object (or its --query projection).
func renderSR(w io.Writer, format output.Format, sr *payloads.StorageRepository, query string) error {
	if format == output.FormatTable && query == "" {
		table := output.Table{
			Headers: []string{"ID", "NAME", "TYPE", "SIZE", "USAGE", "CONTAINER"},
			Rows: [][]string{{
				sr.ID.String(),
				sr.NameLabel,
				sr.SRType,
				units.HumanSize(sr.Size),
				units.HumanSize(sr.Usage),
				sr.Container.String(),
			}},
		}
		return output.Render(w, format, table, sr, nil)
	}

	if query != "" {
		queryResult, err := output.Query(query, sr)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, sr, queryResult)
	}

	normalized, err := output.Normalize(sr)
	if err != nil {
		return err
	}
	return output.Render(w, format, output.Table{}, normalized, nil)
}
