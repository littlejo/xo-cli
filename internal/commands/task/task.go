// Package task implements the 'xo task' command group.
package task

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo task' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage asynchronous tasks",
	}
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newWaitCommand())
	cmd.AddCommand(newAbortCommand())
	return cmd
}
