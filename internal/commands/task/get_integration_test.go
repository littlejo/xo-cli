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

func TestIntegrationTaskGet(t *testing.T) {
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

	// Fetch the list first to obtain a real task id.
	listRoot := newTestRoot()
	var listOut strings.Builder
	listRoot.SetOut(&listOut)
	listRoot.SetErr(&listOut)
	listRoot.SetArgs([]string{"list", "--limit", "1", "--output", "json"})
	if err := listRoot.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("xo task list against %s: %v", url, err)
	}

	var tasks []map[string]any
	if err := json.Unmarshal([]byte(listOut.String()), &tasks); err != nil {
		t.Fatalf("list output is not valid JSON: %v\n%s", err, listOut.String())
	}
	if len(tasks) == 0 {
		t.Skip("no tasks available on the test instance")
	}
	id, _ := tasks[0]["id"].(string)
	if id == "" {
		t.Fatalf("task without id: %v", tasks[0])
	}

	getRoot := newGetTestRoot()
	var out strings.Builder
	getRoot.SetOut(&out)
	getRoot.SetErr(&out)
	getRoot.SetArgs([]string{"get", id, "--output", "json"})
	if err := getRoot.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("xo task get %s against %s: %v", id, url, err)
	}

	var task map[string]any
	if err := json.Unmarshal([]byte(out.String()), &task); err != nil {
		t.Fatalf("integration output is not valid JSON: %v\n%s", err, out.String())
	}
	if task["id"] != id {
		t.Fatalf("expected task %s, got %v", id, task["id"])
	}
	t.Logf("fetched task %s (status %v)", id, task["status"])
}
