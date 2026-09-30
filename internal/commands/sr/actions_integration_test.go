//go:build integration

package sr

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
// In CI it is pointed at the xo-api-sim simulator, which answers the SR
// scan / reclaim_space action endpoints, so the test exercises both
// commands end to end. Without the variables the test is reported as
// skipped, never as passed.

func TestIntegrationSRActions(t *testing.T) {
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

	// run executes a command of the full 'xo sr' group and returns the
	// combined output.
	run := func(args ...string) (string, error) {
		t.Helper()
		root := newActionTestRoot()
		var out strings.Builder
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(append([]string{"sr"}, args...))
		err := root.ExecuteContext(context.Background())
		return out.String(), err
	}

	// Pick the first available SR.
	listOut, err := run("list", "--limit", "1", "--output", "json")
	if err != nil {
		t.Fatalf("xo sr list: %v\n%s", err, listOut)
	}
	var srs []map[string]any
	if err := json.Unmarshal([]byte(listOut), &srs); err != nil {
		t.Fatalf("sr list output is not valid JSON: %v\n%s", err, listOut)
	}
	if len(srs) == 0 {
		t.Skip("no SRs available on the test instance")
	}
	srID, _ := srs[0]["id"].(string)
	if srID == "" {
		t.Fatalf("SR without id: %v", srs[0])
	}
	t.Logf("using SR %s", srID)

	// Both actions are asynchronous: they must return a task id.
	for _, action := range []string{"scan", "reclaim-space"} {
		out, err := run(action, srID)
		if err != nil {
			t.Fatalf("xo sr %s: %v\n%s", action, err, out)
		}
		if !strings.Contains(out, "(task ") {
			t.Fatalf("%s output should carry a task id:\n%s", action, out)
		}
	}
}
