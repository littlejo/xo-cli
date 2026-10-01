// Package network implements the 'xo network' command group.
package network

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo network' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage networks",
	}
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newTagCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newCreateInternalCommand())
	cmd.AddCommand(newCreateBondedCommand())
	cmd.AddCommand(newDeleteCommand())
	return cmd
}
