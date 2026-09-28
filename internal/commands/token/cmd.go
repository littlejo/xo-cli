// Package token implements the 'xo token' command group.
package token

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo token' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage authentication tokens",
	}
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	return cmd
}
