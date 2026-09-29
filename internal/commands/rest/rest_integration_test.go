//go:build integration

package rest

import (
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
// In CI it is pointed at the xo-api-sim simulator, which exposes the REST
// surface the escape hatch talks to. Without those variables the test is
// reported as skipped, never as passed.
//
// It exercises the raw REST path end to end: an authenticated GET with
// --param and --query, and a 404 error surfaced from the API.

func TestIntegrationRest(t *testing.T) {
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

	// GET /hosts with a limit and a JMESPath projection. Without fields=*
	// the list endpoints answer with resource URIs (real XO behavior,
	// mirrored by the simulator), so request the full objects explicitly.
	out, err := run(t, "rest", "get", "hosts", "--param", "limit=1", "--param", "fields=*", "--output", "json", "--query", "[].name_label")
	if err != nil {
		t.Fatalf("rest get hosts: %v\n%s", err, out)
	}
	var names []string
	if err := json.Unmarshal([]byte(out), &names); err != nil {
		t.Fatalf("rest get hosts output is not valid JSON: %v\n%s", err, out)
	}
	if len(names) != 1 || names[0] == "" {
		t.Fatalf("expected exactly one non-empty host name, got %s", out)
	}
	t.Logf("rest get hosts (limit=1): %s", strings.Join(names, ", "))

	// A missing resource must surface the API 404 as an error.
	_, err = run(t, "rest", "get", "vms/deadbeef-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("rest get on a missing resource must fail")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected a 404 error, got: %v", err)
	}
}
