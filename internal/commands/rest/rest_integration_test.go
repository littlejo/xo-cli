//go:build integration

package rest

import (
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
// In CI it is pointed at the xo-api-sim simulator, which exposes the REST
// surface the escape hatch talks to. Without those variables the test is
// reported as skipped, never as passed.
//
// It exercises the raw REST path end to end: an authenticated GET with
// --param and --query, and a 404 error surfaced from the API.
//
// Every result is asserted in the human readable formats (default table and
// --output text) as well as JSON: the list endpoints answer with resource
// URIs when no fields= is given, and a regression in that rendering path
// used to print nothing at all. JSON alone would not have caught it.

func TestIntegrationRest(t *testing.T) {
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
	wantName := names[0]

	// The same object must render in the human formats: default (table) and
	// --output text. Both are the generic renderers shared by the raw REST
	// path, so they must not drop the data (a past bug printed nothing for
	// list results in these formats).
	tableOut, err := run(t, "rest", "get", "hosts", "--param", "limit=1", "--param", "fields=*")
	if err != nil {
		t.Fatalf("rest get hosts (table): %v\n%s", err, tableOut)
	}
	if !strings.Contains(tableOut, wantName) {
		t.Fatalf("table output should show the host name %q:\n%s", wantName, tableOut)
	}
	textOut, err := run(t, "rest", "get", "hosts", "--param", "limit=1", "--param", "fields=*", "--output", "text")
	if err != nil {
		t.Fatalf("rest get hosts (text): %v\n%s", err, textOut)
	}
	if !strings.Contains(textOut, wantName) {
		t.Fatalf("text output should show the host name %q:\n%s", wantName, textOut)
	}

	// Without fields= the list endpoint answers with resource URIs, not
	// objects. The URI list must be printed one per line, never dropped into
	// an empty table.
	uriOut, err := run(t, "rest", "get", "hosts", "--output", "text")
	if err != nil {
		t.Fatalf("rest get hosts (URI list, text): %v\n%s", err, uriOut)
	}
	if !strings.Contains(uriOut, "/rest/v0/hosts/") {
		t.Fatalf("text output should list the host URIs:\n%s", uriOut)
	}

	// A single object (GET /hosts/<id>) must render as key/value lines in
	// text and carry the id.
	getOut, err := run(t, "rest", "get", "hosts", "--param", "limit=1", "--param", "fields=*", "--output", "json", "--query", "[].id")
	if err != nil {
		t.Fatalf("rest get hosts (id): %v\n%s", err, getOut)
	}
	var hostIDs []string
	if err := json.Unmarshal([]byte(getOut), &hostIDs); err != nil {
		t.Fatalf("host id output is not valid JSON: %v\n%s", err, getOut)
	}
	if len(hostIDs) != 1 {
		t.Fatalf("expected one host id, got %s", getOut)
	}
	hostID := hostIDs[0]
	objOut, err := run(t, "rest", "get", "hosts/"+hostID, "--output", "text")
	if err != nil {
		t.Fatalf("rest get hosts/<id> (text): %v\n%s", err, objOut)
	}
	if !strings.Contains(objOut, hostID) {
		t.Fatalf("text output should carry the host id %q:\n%s", hostID, objOut)
	}

	// A missing resource must surface the API 404 as an error.
	_, err = run(t, "rest", "get", "vms/deadbeef-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("rest get on a missing resource must fail")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected a 404 error, got: %v", err)
	}
}
