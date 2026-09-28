package configure

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/vatesfr/xo-cli/internal/cli"
	"github.com/vatesfr/xo-cli/internal/config"
)

// newTestRoot mirrors the production root flags that configure relies on.
func newTestRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "xo",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String(cli.FlagProfile, "", "")
	root.PersistentFlags().String(cli.FlagOutput, "table", "")
	root.AddCommand(NewCommand())
	return root
}

func runConfigure(t *testing.T, args ...string) string {
	t.Helper()
	root := newTestRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(""))
	root.SetArgs(append([]string{"configure"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("configure %v: %v (output: %s)", args, err, out.String())
	}
	return out.String()
}

func runConfigureExpectError(t *testing.T, args ...string) string {
	t.Helper()
	root := newTestRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(""))
	root.SetArgs(append([]string{"configure"}, args...))
	err := root.Execute()
	if err == nil {
		t.Fatalf("configure %v: expected an error", args)
	}
	return err.Error()
}

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("XO_CONFIG_FILE", t.TempDir()+"/config")
	for _, key := range []string{"XO_PROFILE", "XO_ENDPOINT", "XO_TOKEN", "XO_USERNAME", "XO_PASSWORD", "XO_INSECURE"} {
		t.Setenv(key, "")
	}
}

func TestConfigureNonInteractiveSavesProfile(t *testing.T) {
	isolate(t)

	runConfigure(t, "--profile", "lab", "--endpoint", "https://xo.lab.example.com", "--token", "secret")

	cfg, err := config.Load("lab")
	if err != nil {
		t.Fatalf("Load after configure: %v", err)
	}
	if cfg.Endpoint != "https://xo.lab.example.com" || cfg.Token != "secret" {
		t.Fatalf("unexpected stored profile: %+v", cfg)
	}
}

func TestConfigureDefaultProfile(t *testing.T) {
	isolate(t)

	runConfigure(t, "--endpoint", "https://xo.example.com", "--token", "secret")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load after configure: %v", err)
	}
	if cfg.Name != config.DefaultProfile || cfg.Endpoint != "https://xo.example.com" {
		t.Fatalf("unexpected stored profile: %+v", cfg)
	}
}

func TestConfigureRejectsBadEndpoint(t *testing.T) {
	isolate(t)

	out := runConfigureExpectError(t, "--endpoint", "ftp://nope", "--token", "x")
	if !strings.Contains(out, "http") {
		t.Fatalf("unexpected error output: %s", out)
	}
}

func TestConfigureNonInteractiveRequiresCredentials(t *testing.T) {
	isolate(t)

	out := runConfigureExpectError(t, "--endpoint", "https://xo.example.com")
	if !strings.Contains(out, "token") && !strings.Contains(out, "username") {
		t.Fatalf("unexpected error output: %s", out)
	}
}

func TestConfigureUsernamePassword(t *testing.T) {
	isolate(t)

	runConfigure(t, "--endpoint", "https://xo.example.com", "--username", "admin", "--password", "s3cret")

	cfg, err := config.Load(config.DefaultProfile)
	if err != nil {
		t.Fatalf("Load after configure: %v", err)
	}
	if cfg.Username != "admin" || cfg.Password != "s3cret" || cfg.Token != "" {
		t.Fatalf("unexpected stored profile: %+v", cfg)
	}
}
