package vm

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/output"
)

func newGetCommand() *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a virtual machine",
		Long: `Get a virtual machine from Xen Orchestra.

The VM is referenced by its UUID, as returned by 'xo vm list'.

Examples:
  xo vm get 550e8400-e29b-41d4-a716-446655440001
  xo vm get <id> --output json
  xo vm get <id> --query 'name_label'`,
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

			vm, err := xo.VM().GetByID(cmd.Context(), id)
			if err != nil {
				return notFound(args[0], err, cfg.Insecure)
			}

			return renderVM(cmd.OutOrStdout(), format, vm, query)
		},
	}

	cmd.Flags().StringVar(&query, flagQuery, "", "JMESPath expression applied to the result, e.g. 'name_label'")
	return cmd
}

// renderVM renders a single VM in the requested format. The human format uses a
// single-row table with the same columns as 'xo vm list'; the structured formats
// emit the normalized object (or its --query projection).
func renderVM(w io.Writer, format output.Format, vm *payloads.VM, query string) error {
	if format == output.FormatTable && query == "" {
		table := output.Table{
			Headers: []string{"ID", "NAME", "POWER STATE", "MEMORY", "CPUS", "HOST/POOL"},
			Rows: [][]string{{
				vm.ID.String(),
				vm.NameLabel,
				vm.PowerState,
				memoryText(vm),
				fmt.Sprintf("%d", vm.CPUs.Number),
				vm.Container.String(),
			}},
		}
		return output.Render(w, format, table, vm, nil)
	}

	if query != "" {
		queryResult, err := output.Query(query, vm)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, vm, queryResult)
	}

	normalized, err := output.Normalize(vm)
	if err != nil {
		return err
	}
	return output.Render(w, format, output.Table{}, normalized, nil)
}
