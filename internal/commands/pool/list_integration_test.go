//go:build integration

package pool

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
// In CI it is pointed at the xo-api-sim simulator instead, which always has
// pool fixtures. Without those variables the test is reported as skipped,
// never as passed.

func TestIntegrationPoolList(t *testing.T) {
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
	root.SetArgs([]string{"list", "--output", "json"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("xo pool list against %s: %v", url, err)
	}

	var pools []map[string]any
	if err := json.Unmarshal([]byte(out.String()), &pools); err != nil {
		t.Fatalf("integration output is not valid JSON: %v\n%s", err, out.String())
	}
	if len(pools) == 0 {
		t.Skip("no pools available on the test instance")
	}
	t.Logf("listed %d pools", len(pools))
	for _, pool := range pools {
		if pool["name_label"] == nil || pool["id"] == nil {
			t.Fatalf("unexpected pool payload: %v", pool)
		}
	}

	// Fetch the first pool by id to exercise the get path.
	id, _ := pools[0]["id"].(string)
	if id == "" {
		t.Fatalf("pool without id: %v", pools[0])
	}

	getRoot := newGetTestRoot()
	var getOut strings.Builder
	getRoot.SetOut(&getOut)
	getRoot.SetErr(&getOut)
	getRoot.SetArgs([]string{"get", id, "--output", "json"})
	if err := getRoot.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("xo pool get %s against %s: %v", id, url, err)
	}

	var pool map[string]any
	if err := json.Unmarshal([]byte(getOut.String()), &pool); err != nil {
		t.Fatalf("get output is not valid JSON: %v\n%s", err, getOut.String())
	}
	if pool["id"] != id {
		t.Fatalf("expected pool %s, got %v", id, pool["id"])
	}
}
