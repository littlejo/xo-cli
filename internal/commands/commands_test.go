package commands

import (
	"testing"

	"github.com/littlejo/xo-gocli/internal/cli"
)

// TestRootOutputShorthand pins the -o shortcut for --output. The long form is
// exercised end-to-end by the per-resource list tests; this guards the wiring
// of the shortcut on the real root so it cannot silently regress to a
// long-only flag.
func TestRootOutputShorthand(t *testing.T) {
	root := NewRoot()
	f := root.PersistentFlags().Lookup(cli.FlagOutput)
	if f == nil {
		t.Fatalf("--%s is not defined on the root command", cli.FlagOutput)
	}
	if f.Shorthand != "o" {
		t.Fatalf("--%s shorthand = %q, want %q", cli.FlagOutput, f.Shorthand, "o")
	}
}
