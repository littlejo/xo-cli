package commands

import (
	"bytes"
	"strings"
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

// TestRootTimeoutFlag pins that the global --timeout flag is defined on the
// real root (so it is available to every command) and is a duration. The
// precedence rules are covered by the cli package tests; this guards the
// wiring on NewRoot so the flag cannot silently disappear.
func TestRootTimeoutFlag(t *testing.T) {
	root := NewRoot()
	f := root.PersistentFlags().Lookup(cli.FlagTimeout)
	if f == nil {
		t.Fatalf("--%s is not defined on the root command", cli.FlagTimeout)
	}
	if f.Value.Type() != "duration" {
		t.Fatalf("--%s type = %q, want %q", cli.FlagTimeout, f.Value.Type(), "duration")
	}
}

// TestVersionCommand pins that 'xo version' exists and prints the same line
// as '--version', offline (no profile or connection needed).
func TestVersionCommand(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("xo version: %v", err)
	}
	want := "xo version " + cli.Version
	if got := strings.TrimSpace(out.String()); got != want {
		t.Fatalf("xo version = %q, want %q", got, want)
	}
}
