//go:build integration

package host

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
// host fixtures. Without those variables the test is reported as skipped,
// never as passed.

func TestIntegrationHostList(t *testing.T) {
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
		t.Fatalf("xo host list against %s: %v", url, err)
	}

	var hosts []map[string]any
	if err := json.Unmarshal([]byte(out.String()), &hosts); err != nil {
		t.Fatalf("integration output is not valid JSON: %v\n%s", err, out.String())
	}
	if len(hosts) == 0 {
		t.Skip("no hosts available on the test instance")
	}
	t.Logf("listed %d hosts", len(hosts))
	for _, host := range hosts {
		if host["name_label"] == nil || host["id"] == nil {
			t.Fatalf("unexpected host payload: %v", host)
		}
	}

	// Fetch the first host by id to exercise the get path.
	id, _ := hosts[0]["id"].(string)
	if id == "" {
		t.Fatalf("host without id: %v", hosts[0])
	}

	getRoot := newGetTestRoot()
	var getOut strings.Builder
	getRoot.SetOut(&getOut)
	getRoot.SetErr(&getOut)
	getRoot.SetArgs([]string{"get", id, "--output", "json"})
	if err := getRoot.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("xo host get %s against %s: %v", id, url, err)
	}

	var host map[string]any
	if err := json.Unmarshal([]byte(getOut.String()), &host); err != nil {
		t.Fatalf("get output is not valid JSON: %v\n%s", err, getOut.String())
	}
	if host["id"] != id {
		t.Fatalf("expected host %s, got %v", id, host["id"])
	}
}
