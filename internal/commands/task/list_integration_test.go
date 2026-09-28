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
// XO_TEST_URL and XO_TEST_TOKEN environment variables are set. It is
// explicitly enabled with:
//
//	go test -tags=integration ./...
//
// Without those variables the test is reported as skipped, never as passed.

func TestIntegrationTaskList(t *testing.T) {
	url := os.Getenv("XO_TEST_URL")
	token := os.Getenv("XO_TEST_TOKEN")
	if url == "" || token == "" {
		t.Skip("integration test skipped: set XO_TEST_URL and XO_TEST_TOKEN to run against a real Xen Orchestra instance")
	}

	t.Setenv("XO_CONFIG_FILE", t.TempDir()+"/config")
	for _, key := range []string{"XO_PROFILE", "XO_ENDPOINT", "XO_TOKEN", "XO_USERNAME", "XO_PASSWORD", "XO_INSECURE"} {
		t.Setenv(key, "")
	}
	t.Setenv("XO_ENDPOINT", url)
	t.Setenv("XO_TOKEN", token)

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
