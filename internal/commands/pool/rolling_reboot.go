package pool

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
)

func newRollingRebootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rolling-reboot <id>",
		Short: "Reboot the pool's hosts one by one",
		Long: `Reboot all the hosts of a pool, one at a time.

The hosts are rebooted in turn: the VMs that run on the host being rebooted
are moved away first (live migration when possible), the host is rebooted,
and it rejoins the pool before the next one goes down. The pool stays
available the whole time, but VMs that cannot be live-migrated away (e.g.
without an available target) suffer a brief downtime on their host's turn.

This is a destructive operation and asks for confirmation unless --yes is
given. The command is synchronous: it waits until the backing task completes.
The pool is referenced by its UUID, as returned by 'xo pool list'.

Examples:
  xo pool rolling-reboot aaaaaaaa-bbbb-cccc-dddd-000000000001
  xo pool rolling-reboot <id> --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAction(cmd, args[0], actionSpec{
				verb:        "rolling reboot",
				past:        "rolling reboot done",
				destructive: true,
				do: func(ctx context.Context, xo library.Library, id uuid.UUID) error {
					return xo.Pool().RollingReboot(ctx, id)
				},
			})
		},
	}
	cmd.Flags().Bool(flagYes, false, "do not ask for confirmation (or set XOA_YES=1)")
	return cmd
}
