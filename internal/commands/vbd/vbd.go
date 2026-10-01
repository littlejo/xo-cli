// Package vbd implements the 'xo vbd' command group.
package vbd

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo vbd' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vbd",
		Short: "Manage virtual block devices (VBDs)",
	}
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newConnectCommand())
	cmd.AddCommand(newDisconnectCommand())
	return cmd
}
