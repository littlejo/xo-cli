package pool

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
)

func newEmergencyShutdownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emergency-shutdown <id>",
		Short: "Shut down every host of the pool immediately",
		Long: `Shut down every host of a pool, without moving the VMs away first.

Unlike a rolling reboot, nothing is evacuated: every host (and every VM
running on it) is powered off at once. The pool is down afterwards and the
hosts must be restarted manually to recover. Use it only when the pool is in
a bad state and cannot be handled by any other means.

This is a destructive operation and asks for confirmation unless --yes is
given. The command is synchronous: it waits until the backing task completes.
The pool is referenced by its UUID, as returned by 'xo pool list'.

Examples:
  xo pool emergency-shutdown aaaaaaaa-bbbb-cccc-dddd-000000000001
  xo pool emergency-shutdown <id> --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAction(cmd, args[0], actionSpec{
				verb:        "emergency shutdown",
				past:        "emergency shutdown done",
				destructive: true,
				do: func(ctx context.Context, xo library.Library, id uuid.UUID) error {
					return xo.Pool().EmergencyShutdown(ctx, id)
				},
			})
		},
	}
	cmd.Flags().Bool(flagYes, false, "do not ask for confirmation (or set XOA_YES=1)")
	return cmd
}
