package host

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gofrs/uuid"

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
		Short: "Get a host",
		Long: `Get a host from Xen Orchestra.

The host is referenced by its UUID, as returned by 'xo host list'.

Examples:
  xo host get 550e8400-e29b-41d4-a716-446655440001
  xo host get <id> --output json
  xo host get <id> --query 'name_label'`,
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

			host, err := xo.Host().Get(cmd.Context(), id)
			if err != nil {
				return notFound(args[0], err, cfg.Insecure)
			}

			return renderHost(cmd.OutOrStdout(), format, host, query)
		},
	}

	cmd.Flags().StringVarP(&query, flagQuery, "q", "", "JMESPath expression applied to the result, e.g. 'name_label'")
	return cmd
}

// parseID converts a positional host identifier into a UUID.
func parseID(id string) (uuid.UUID, error) {
	u, err := uuid.FromString(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid host id %q (expected a UUID)", id)
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

// notFound turns a lookup failure into the concise "host not found" form when
// the API returned a 404, and keeps the original error otherwise. It passes the
// profile's insecure state through so the TLS hint is only suggested when it
// would actually help.
func notFound(id string, err error, insecure bool) error {
	if err != nil && strings.Contains(err.Error(), "404") {
		return fmt.Errorf("host %q not found", id)
	}
	return cli.InsecureHint(fmt.Sprintf("cannot get host %q: %v", id, err), insecure)
}

// renderHost renders a single host in the requested format. The human format
// uses a single-row table with the same columns as 'xo host list'; the
// structured formats emit the normalized object (or its --query projection).
func renderHost(w io.Writer, format output.Format, h *payloads.Host, query string) error {
	if format == output.FormatTable && query == "" {
		table := output.Table{
			Headers: []string{"ID", "NAME", "ADDRESS", "POWER STATE", "PLATFORM", "MEMORY", "VMS", "POOL"},
			Rows: [][]string{{
				h.ID.String(),
				h.NameLabel,
				h.Address,
				h.PowerState,
				h.Version,
				memoryText(h),
				fmt.Sprintf("%d", len(h.ResidentVMs)),
				h.Pool.String(),
			}},
		}
		return output.Render(w, format, table, h, nil)
	}

	if query != "" {
		queryResult, err := output.Query(query, h)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, h, queryResult)
	}

	normalized, err := output.Normalize(h)
	if err != nil {
		return err
	}
	return output.Render(w, format, output.Table{}, normalized, nil)
}
