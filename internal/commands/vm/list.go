package vm

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
	flagQuery      = "query"
	flagLimit      = "limit"
	flagPowerState = "power-state"
)

func newListCommand() *cobra.Command {
	var (
		query      string
		limit      int
		powerState string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List virtual machines",
		Long: `List virtual machines from Xen Orchestra.

Examples:
  xo vm list
  xo vm list --output json
  xo vm list --query '[].name_label'
  xo vm list --power-state Running
  xo vm list --limit 10`,
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
			filter := powerState
			if filter != "" {
				filter = "power_state:" + powerState
			}

			cfg, err := config.Load(cli.ProfileName(cmd))
			if err != nil {
				return err
			}

			xo, err := cli.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			vms, err := xo.VM().GetAll(cmd.Context(), limit, filter)
			if err != nil {
				return cli.InsecureHint(fmt.Sprintf("cannot list VMs: %v", err), cfg.Insecure)
			}

			return renderVMs(cmd.OutOrStdout(), format, vms, query)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&query, flagQuery, "", "JMESPath expression applied to the result, e.g. '[].name_label'")
	flags.IntVar(&limit, flagLimit, 0, "maximum number of VMs to return (0 for no limit)")
	flags.StringVar(&powerState, flagPowerState, "", "filter by power state: Running, Halted, Paused, Suspended")

	return cmd
}

// renderVMs applies the optional --query expression and renders the result in
// the requested format.
func renderVMs(w io.Writer, format output.Format, vms []*payloads.VM, query string) error {
	queryResult, err := output.Query(query, vms)
	if err != nil {
		return err
	}

	table := output.Table{
		Headers: []string{"ID", "NAME", "POWER STATE", "MEMORY", "CPUS", "HOST/POOL"},
	}
	for _, vm := range vms {
		table.Rows = append(table.Rows, []string{
			vm.ID.String(),
			vm.NameLabel,
			vm.PowerState,
			memoryText(vm),
			fmt.Sprintf("%d", vm.CPUs.Number),
			vm.Container.String(),
		})
	}

	// For structured formats without a query, normalize the SDK types so the
	// output only contains the requested data.
	var raw any = vms
	if format != output.FormatTable && (queryResult == nil || !queryResult.Present) {
		normalized, err := output.Normalize(vms)
		if err != nil {
			return err
		}
		raw = normalized
	}

	return output.Render(w, format, table, raw, queryResult)
}

func memoryText(vm *payloads.VM) string {
	if vm.Memory.Size == 0 {
		return ""
	}
	return units.HumanSize(float64(vm.Memory.Size))
}
