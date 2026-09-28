package pool

import (
	"fmt"
	"io"
	"strings"

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
		Short: "Get a pool",
		Long: `Get a pool from Xen Orchestra.

The pool is referenced by its UUID, as returned by 'xo pool list'.

Examples:
  xo pool get 550e8400-e29b-41d4-a716-446655440001
  xo pool get <id> --output json
  xo pool get <id> --query 'name_label'`,
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

			xo, _, err := newClient(cmd)
			if err != nil {
				return err
			}

			pool, err := xo.Pool().Get(cmd.Context(), id)
			if err != nil {
				return notFound(args[0], err)
			}

			return renderPool(cmd.OutOrStdout(), format, pool, query)
		},
	}

	cmd.Flags().StringVar(&query, flagQuery, "", "JMESPath expression applied to the result, e.g. 'name_label'")
	return cmd
}

// parseID converts a positional pool identifier into a UUID.
func parseID(id string) (uuid.UUID, error) {
	u, err := uuid.FromString(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid pool id %q (expected a UUID)", id)
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

// notFound turns a lookup failure into the concise "pool not found" form when
// the API returned a 404, and keeps the original error otherwise.
func notFound(id string, err error) error {
	if err != nil && strings.Contains(err.Error(), "404") {
		return fmt.Errorf("pool %q not found", id)
	}
	return cli.InsecureHint(fmt.Sprintf("cannot get pool %q: %v", id, err), false)
}

// renderPool renders a single pool in the requested format. The human format
// uses a single-row table with the same columns as 'xo pool list'; the
// structured formats emit the normalized object (or its --query projection).
func renderPool(w io.Writer, format output.Format, p *payloads.Pool, query string) error {
	if format == output.FormatTable && query == "" {
		table := output.Table{
			Headers: []string{"ID", "NAME", "PLATFORM", "CORES", "SOCKETS", "MASTER", "HA"},
			Rows: [][]string{{
				p.ID.String(),
				p.NameLabel,
				p.PlatformVersion,
				fmt.Sprintf("%d", p.CPUs.Cores),
				fmt.Sprintf("%d", p.CPUs.Sockets),
				p.Master.String(),
				fmt.Sprintf("%t", p.HAEnabled),
			}},
		}
		return output.Render(w, format, table, p, nil)
	}

	if query != "" {
		queryResult, err := output.Query(query, p)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, p, queryResult)
	}

	normalized, err := output.Normalize(p)
	if err != nil {
		return err
	}
	return output.Render(w, format, output.Table{}, normalized, nil)
}
