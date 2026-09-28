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
	cmd.AddCommand(newListCommand())
	return cmd
}
