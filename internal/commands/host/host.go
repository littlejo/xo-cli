// Package host implements the 'xo host' command group.
package host

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo host' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host",
		Short: "Manage hosts",
	}
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	return cmd
}
