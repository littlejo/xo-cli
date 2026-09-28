// Package template implements the 'xo template' command group.
//
// In Xen Orchestra, VM templates are first-class objects exposed by the
// dedicated REST resource "vm-templates"; they are not part of the "vms"
// collection (so 'xo vm list' does not show them). The SDK v2 does not (yet)
// wrap that resource with a typed service, so this command reaches it through
// the SDK's own REST client (cli.NewHTTPClient) — the same single API
// boundary, not a second HTTP client. The missing operation should be
// contributed to the SDK.
package template

import (
	"github.com/spf13/cobra"
)

// NewCommand builds the 'xo template' command group.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage VM templates",
	}
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	return cmd
}
