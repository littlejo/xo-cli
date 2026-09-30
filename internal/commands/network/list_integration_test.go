//go:build integration

package network

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// This test runs against a real Xen Orchestra instance when the
// XOA_TEST_URL and XOA_TEST_TOKEN environment variables are set. It is
// explicitly enabled with:
//
//	go test -tags=integration ./...
//
// Without those variables the test is reported as skipped, never as passed.

func TestIntegrationNetworkList(t *testing.T) {
	url := os.Getenv("XOA_TEST_URL")
	token := os.Getenv("XOA_TEST_TOKEN")
	if url == "" || token == "" {
		t.Skip("integration test skipped: set XOA_TEST_URL and XOA_TEST_TOKEN to run against a real Xen Orchestra instance")
	}

	t.Setenv("XOA_CONFIG_FILE", t.TempDir()+"/config")
	for _, key := range []string{"XOA_PROFILE", "XOA_ENDPOINT", "XOA_TOKEN", "XOA_USERNAME", "XOA_PASSWORD", "XOA_INSECURE"} {
		t.Setenv(key, "")
	}
	t.Setenv("XOA_ENDPOINT", url)
	t.Setenv("XOA_TOKEN", token)

	root := newTestRoot()
	var out strings.Builder
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"list", "--limit", "1", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("xo network list against %s: %v", url, err)
	}

	var networks []map[string]any
	if err := json.Unmarshal([]byte(out.String()), &networks); err != nil {
		t.Fatalf("integration output is not valid JSON: %v\n%s", err, out.String())
	}
	if len(networks) == 0 {
		t.Skip("no networks available on the test instance")
	}
	t.Logf("listed %d networks", len(networks))
	for _, network := range networks {
		if network["name_label"] == nil || network["id"] == nil {
			t.Fatalf("unexpected network payload: %v", network)
		}
	}

	// Fetch the first network by id to exercise the get path.
	id, _ := networks[0]["id"].(string)
	if id == "" {
		t.Fatalf("network without id: %v", networks[0])
	}

	getRoot := newGetTestRoot()
	var getOut strings.Builder
	getRoot.SetOut(&getOut)
	getRoot.SetErr(&getOut)
	getRoot.SetArgs([]string{"get", id, "--output", "json"})
	if err := getRoot.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("xo network get %s against %s: %v", id, url, err)
	}

	var network map[string]any
	if err := json.Unmarshal([]byte(getOut.String()), &network); err != nil {
		t.Fatalf("get output is not valid JSON: %v\n%s", err, getOut.String())
	}
	if network["id"] != id {
		t.Fatalf("expected network %s, got %v", id, network["id"])
	}
}
