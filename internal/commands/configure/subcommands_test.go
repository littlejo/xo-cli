package configure

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/littlejo/xo-gocli/internal/config"
)

func seedProfiles(t *testing.T) {
	t.Helper()
	profiles := []config.Profile{
		{Name: "lab", Endpoint: "https://lab.example.com", Token: "lab-token-secret", Insecure: true},
		{Name: "ci", Endpoint: "https://ci.example.com", Username: "ci-user", Password: "ci-password-secret"},
	}
	for _, p := range profiles {
		if _, err := config.Upsert(p, p.Name == "lab"); err != nil {
			t.Fatalf("Upsert %s: %v", p.Name, err)
		}
	}
}

// runConfigureSub runs a 'configure <sub>' command and returns (output, error).
func runConfigureSub(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newTestRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(""))
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

// --- list --------------------------------------------------------------------

func TestConfigureListTable(t *testing.T) {
	isolate(t)
	seedProfiles(t)

	out, err := runConfigureSub(t, "configure", "list")
	if err != nil {
		t.Fatalf("configure list: %v", err)
	}
	for _, expected := range []string{"NAME", "ENDPOINT", "lab", "ci", "https://lab.example.com", "****cret", "ci-user"} {
		if !strings.Contains(out, expected) {
			t.Errorf("list output missing %q:\n%s", expected, out)
		}
	}
	// Full secrets must not leak.
	for _, secret := range []string{"lab-token-secret", "ci-password-secret"} {
		if strings.Contains(out, secret) {
			t.Errorf("list leaked the full secret %q:\n%s", secret, out)
		}
	}
}

func TestConfigureListJSON(t *testing.T) {
	isolate(t)
	seedProfiles(t)

	out, err := runConfigureSub(t, "configure", "list", "--output", "json")
	if err != nil {
		t.Fatalf("configure list --output json: %v", err)
	}

	var profiles []map[string]any
	if err := json.Unmarshal([]byte(out), &profiles); err != nil {
		t.Fatalf("list output is not valid JSON: %v\n%s", err, out)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d: %s", len(profiles), out)
	}
}

func TestConfigureListEmpty(t *testing.T) {
	isolate(t)

	out, err := runConfigureSub(t, "configure", "list")
	if err != nil {
		t.Fatalf("configure list on empty config: %v", err)
	}
	if strings.Contains(out, "lab") {
		t.Fatalf("unexpected profile in empty list:\n%s", out)
	}
}

// --- show --------------------------------------------------------------------

func TestConfigureShowRevealsSecrets(t *testing.T) {
	isolate(t)
	seedProfiles(t)

	out, err := runConfigureSub(t, "configure", "show", "lab")
	if err != nil {
		t.Fatalf("configure show lab: %v", err)
	}
	if !strings.Contains(out, "lab-token-secret") {
		t.Fatalf("show must reveal the full token:\n%s", out)
	}
	if !strings.Contains(out, "https://lab.example.com") {
		t.Fatalf("show missing the endpoint:\n%s", out)
	}
}

func TestConfigureShowDefaultsToSelectedProfile(t *testing.T) {
	isolate(t)
	seedProfiles(t)

	// Without a name or --profile the current profile is shown.
	out, err := runConfigureSub(t, "configure", "show")
	if err != nil {
		t.Fatalf("configure show: %v", err)
	}
	if !strings.Contains(out, "https://lab.example.com") {
		t.Fatalf("show must default to the current profile (lab):\n%s", out)
	}

	// --profile selects the profile to show.
	out, err = runConfigureSub(t, "configure", "show", "--profile", "ci")
	if err != nil {
		t.Fatalf("configure show --profile ci: %v", err)
	}
	if !strings.Contains(out, "https://ci.example.com") {
		t.Fatalf("show --profile ci must show the ci profile:\n%s", out)
	}
}

func TestConfigureShowUnknownFails(t *testing.T) {
	isolate(t)
	seedProfiles(t)

	_, err := runConfigureSub(t, "configure", "show", "nope")
	if err == nil {
		t.Fatal("expected an error for an unknown profile")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigureShowJSON(t *testing.T) {
	isolate(t)
	seedProfiles(t)

	out, err := runConfigureSub(t, "configure", "show", "lab", "--output", "json")
	if err != nil {
		t.Fatalf("configure show lab --output json: %v", err)
	}

	var profile map[string]any
	if err := json.Unmarshal([]byte(out), &profile); err != nil {
		t.Fatalf("show output is not valid JSON: %v\n%s", err, out)
	}
	if profile["token"] != "lab-token-secret" {
		t.Fatalf("expected the full token in JSON output, got %v", profile["token"])
	}
}

// --- remove ------------------------------------------------------------------

func TestConfigureRemoveYes(t *testing.T) {
	isolate(t)
	seedProfiles(t)

	out, err := runConfigureSub(t, "configure", "remove", "ci", "--yes")
	if err != nil {
		t.Fatalf("configure remove ci --yes: %v", out)
	}
	if !strings.Contains(out, "removed") {
		t.Fatalf("remove did not report success:\n%s", out)
	}
	if _, err := config.GetProfile("ci"); err == nil {
		t.Fatal("ci should have been removed")
	}
	// The current profile (lab) is untouched.
	if _, err := config.GetProfile("lab"); err != nil {
		t.Fatalf("lab must still exist: %v", err)
	}
}

func TestConfigureRemoveRequiresConfirmation(t *testing.T) {
	isolate(t)
	seedProfiles(t)

	_, err := runConfigureSub(t, "configure", "remove", "ci")
	if err == nil {
		t.Fatal("expected an error when no confirmation is given and stdin is not a terminal")
	}
	if _, err := config.GetProfile("ci"); err != nil {
		t.Fatalf("ci must not be removed without confirmation: %v", err)
	}
}

func TestConfigureRemoveUnknownFails(t *testing.T) {
	isolate(t)
	seedProfiles(t)

	_, err := runConfigureSub(t, "configure", "remove", "nope", "--yes")
	if err == nil {
		t.Fatal("expected an error when removing an unknown profile")
	}
}
