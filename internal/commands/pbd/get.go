package pbd

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

func newGetCommand() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a physical block device (PBD)",
		Long: `Get a physical block device (PBD) from Xen Orchestra.

The PBD is referenced by its UUID, as returned by 'xo pbd list'.

Examples:
  xo pbd get 550e8400-e29b-41d4-a716-446655440001
  xo pbd get <id> --output json
  xo pbd get <id> --query 'device_config'`,
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

			pbd, err := xo.PBD().Get(cmd.Context(), id)
			if err != nil {
				return notFound(args[0], err, cfg.Insecure)
			}

			return renderPBD(cmd.OutOrStdout(), format, pbd, query)
		},
	}

	cmd.Flags().StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. 'device_config'")
	return cmd
}

// parseID converts a positional PBD identifier into a UUID.
func parseID(id string) (uuid.UUID, error) {
	u, err := uuid.FromString(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid PBD id %q (expected a UUID)", id)
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
	return cli.NotFound("PBD", "get", id, err, insecure)
}

// renderPBD renders a single PBD in the requested format. The human format
// uses a single-row table with the same columns as 'xo pbd list'; the
// structured formats emit the normalized object (or its --query projection).
func renderPBD(w io.Writer, format output.Format, pbd *payloads.PBD, query string) error {
	if format == output.FormatTable && query == "" {
		table := output.Table{
			Headers: []string{"ID", "HOST", "SR", "POOL", "ATTACHED", "DEVICE"},
			Rows: [][]string{{
				pbd.ID.String(),
				pbd.Host.String(),
				pbd.SR.String(),
				pbd.Pool.String(),
				boolText(pbd.Attached),
				deviceOf(pbd),
			}},
		}
		return output.Render(w, format, table, pbd, nil)
	}

	if query != "" {
		queryResult, err := output.Query(query, pbd)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, pbd, queryResult)
	}

	normalized, err := output.Normalize(pbd)
	if err != nil {
		return err
	}
	return output.Render(w, format, output.Table{}, normalized, nil)
}
