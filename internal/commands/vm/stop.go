package vm

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
)

const flagHard = "hard"

func newStopCommand() *cobra.Command {
	var (
		hard bool
		yes  bool
	)

	cmd := &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a virtual machine",
		Long: `Stop a virtual machine.

By default this performs a clean shutdown (the guest is asked to power off).
Use --hard to power off the VM immediately without a clean shutdown.

This is a destructive operation and asks for confirmation unless --yes is
given. The VM is referenced by its UUID, as returned by 'xo vm list'.

Examples:
  xo vm stop 550e8400-e29b-41d4-a716-446655440001
  xo vm stop <id> --hard
  xo vm stop <id> --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			verb := "cleanly stop"
			if hard {
				verb = "hard stop"
			}
			return runAction(cmd, actionSpec{
				verb:        verb,
				id:          args[0],
				destructive: true,
				yes:         yes,
				perform: func(ctx context.Context, xo library.Library, id uuid.UUID) (string, error) {
					if hard {
						return xo.VM().HardShutdown(ctx, id)
					}
					return xo.VM().CleanShutdown(ctx, id)
				},
			})
		},
	}

	cmd.Flags().BoolVar(&hard, flagHard, false, "power off the VM immediately instead of a clean shutdown")
	cmd.Flags().BoolVar(&yes, flagYes, false, "do not ask for confirmation")
	return cmd
}
