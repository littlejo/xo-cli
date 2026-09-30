//go:build integration

package task

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

func TestIntegrationTaskList(t *testing.T) {
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
		t.Fatalf("xo task list against %s: %v", url, err)
	}

	var tasks []map[string]any
	if err := json.Unmarshal([]byte(out.String()), &tasks); err != nil {
		t.Fatalf("integration output is not valid JSON: %v\n%s", err, out.String())
	}
	t.Logf("listed %d tasks", len(tasks))
	for _, task := range tasks {
		if task["id"] == nil || task["status"] == nil {
			t.Fatalf("unexpected task payload: %v", task)
		}
	}
}
