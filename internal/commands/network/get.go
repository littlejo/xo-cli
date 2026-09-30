package network

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
		Short: "Get a network",
		Long: `Get a network from Xen Orchestra.

The network is referenced by its UUID, as returned by 'xo network list'.

Examples:
  xo network get 11111111-1111-4111-8111-111111111111
  xo network get <id> --output json
  xo network get <id> --query 'name_label'`,
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

			network, err := xo.Network().Get(cmd.Context(), id)
			if err != nil {
				return notFound(args[0], err, cfg.Insecure)
			}

			return renderNetwork(cmd.OutOrStdout(), format, network, query)
		},
	}

	cmd.Flags().StringVar(&query, flagQuery, "", "JMESPath expression applied to the result, e.g. 'name_label'")
	return cmd
}

// parseID converts a positional network identifier into a UUID.
func parseID(id string) (uuid.UUID, error) {
	u, err := uuid.FromString(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid network id %q (expected a UUID)", id)
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

// notFound turns a lookup failure into the concise "network not found" form
// when the API returned a 404, and keeps the original error otherwise. It
// passes the profile's insecure state through so the TLS hint is only
// suggested when it would actually help.
func notFound(id string, err error, insecure bool) error {
	if err != nil && strings.Contains(err.Error(), "404") {
		return fmt.Errorf("network %q not found", id)
	}
	return cli.InsecureHint(fmt.Sprintf("cannot get network %q: %v", id, err), insecure)
}

// renderNetwork renders a single network in the requested format. The human
// format uses a single-row table with the same columns as 'xo network list';
// the structured formats emit the normalized object (or its --query
// projection).
func renderNetwork(w io.Writer, format output.Format, n *payloads.Network, query string) error {
	if format == output.FormatTable && query == "" {
		table := output.Table{
			Headers: []string{"ID", "NAME", "BRIDGE", "TYPE", "MTU", "VIFS", "POOL"},
			Rows: [][]string{{
				n.ID.String(),
				n.NameLabel,
				n.Bridge,
				string(n.Type),
				fmt.Sprintf("%d", n.MTU),
				fmt.Sprintf("%d", len(n.VIFs)),
				n.Pool.String(),
			}},
		}
		return output.Render(w, format, table, n, nil)
	}

	if query != "" {
		queryResult, err := output.Query(query, n)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, n, queryResult)
	}

	normalized, err := output.Normalize(n)
	if err != nil {
		return err
	}
	return output.Render(w, format, output.Table{}, normalized, nil)
}
