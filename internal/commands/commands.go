// Package commands assembles the xo command tree.
package commands

import (
	"github.com/spf13/cobra"

	"github.com/vatesfr/xo-cli/internal/cli"
	"github.com/vatesfr/xo-cli/internal/commands/configure"
	"github.com/vatesfr/xo-cli/internal/commands/pool"
	"github.com/vatesfr/xo-cli/internal/commands/sr"
	"github.com/vatesfr/xo-cli/internal/commands/vm"
)

// NewRoot builds the root command with its global flags and subcommands.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "xo",
		Short: "Xen Orchestra command line client",
		Long: `xo is a command line client for Xen Orchestra.

It talks to the Xen Orchestra REST API through the official Go SDK (v2).
Configure a profile first with 'xo configure'.`,
		Version:       cli.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String(cli.FlagProfile, "", "name of the configuration profile to use (or $XO_PROFILE)")
	root.PersistentFlags().String(cli.FlagOutput, "table", "output format: table, json, yaml, text")

	root.AddCommand(configure.NewCommand())
	root.AddCommand(pool.NewCommand())
	root.AddCommand(sr.NewCommand())
	root.AddCommand(vm.NewCommand())

	return root
}
