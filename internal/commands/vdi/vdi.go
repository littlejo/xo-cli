// Package vdi implements the 'xo vdi' command group.
package vdi

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo vdi' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vdi",
		Short: "Manage virtual disks (VDIs)",
	}
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	return cmd
}
