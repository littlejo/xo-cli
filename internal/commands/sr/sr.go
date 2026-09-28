// Package sr implements the 'xo sr' command group.
package sr

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo sr' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sr",
		Short: "Manage storage repositories",
	}
	cmd.AddCommand(newListCommand())
	return cmd
}
