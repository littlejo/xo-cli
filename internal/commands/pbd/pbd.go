// Package pbd implements the 'xo pbd' command group.
package pbd

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo pbd' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pbd",
		Short: "Manage physical block devices (PBDs)",
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newPlugCommand())
	cmd.AddCommand(newUnplugCommand())
	return cmd
}
