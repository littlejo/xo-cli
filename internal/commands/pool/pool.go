// Package pool implements the 'xo pool' command group.
package pool

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo pool' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool",
		Short: "Manage pools",
	}
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	return cmd
}
