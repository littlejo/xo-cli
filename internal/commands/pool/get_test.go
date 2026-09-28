package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/littlejo/xo-gocli/internal/cli"
)

const fixturePool = `{
	"id": "aaaaaaaa-bbbb-cccc-dddd-000000000001",
	"uuid": "aaaaaaaa-bbbb-cccc-dddd-000000000001",
	"type": "pool",
	"name_label": "Pool prod",
	"platform_version": "8.2",
	"cpus": {"cores": 16, "sockets": 2},
	"master": "11111111-1111-4111-8111-111111111111",
	"HA_enabled": true
}`

// fakeXOGet serves GET /rest/v0/pools/{id}.
func fakeXOGet(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/rest/v0/pools/") {
			http.NotFound(w, r)
			return
		}
		if handler != nil {
			handler(w, r)
			return
		}
		if cookie, err := r.Cookie("authenticationToken"); err != nil || cookie.Value != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprint(w, `{"message":"unauthorized"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, fixturePool)
	}))
}

func newGetTestRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "xo",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String(cli.FlagProfile, "", "")
	root.PersistentFlags().String(cli.FlagOutput, "table", "")
	root.AddCommand(newGetCommand())
	return root
}

func runGet(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newGetTestRoot()
	var out strings.Builder
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return out.String(), err
}

func TestPoolGetTable(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	out, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001")
	if err != nil {
		t.Fatalf("pool get: %v", err)
	}
	for _, expected := range []string{"ID", "NAME", "PLATFORM", "CORES", "SOCKETS", "MASTER", "HA", "Pool prod", "8.2", "16", "2", "true"} {
		if !strings.Contains(out, expected) {
			t.Errorf("table output missing %q:\n%s", expected, out)
		}
	}
}

func TestPoolGetJSON(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	out, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001", "--output", "json")
	if err != nil {
		t.Fatalf("pool get --output json: %v", err)
	}

	var pool map[string]any
	if err := json.Unmarshal([]byte(out), &pool); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if pool["name_label"] != "Pool prod" || pool["platform_version"] != "8.2" {
		t.Fatalf("unexpected pool payload: %s", out)
	}
}

func TestPoolGetQuery(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	out, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001", "--query", "name_label")
	if err != nil {
		t.Fatalf("pool get --query: %v", err)
	}
	if strings.TrimSpace(out) != "Pool prod" {
		t.Fatalf("unexpected query output: %q", out)
	}
}

func TestPoolGetNotFound(t *testing.T) {
	server := fakeXOGet(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message":"not found"}`)
	})
	defer server.Close()
	isolatePointers(t, server.URL)

	_, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001")
	if err == nil {
		t.Fatal("expected an error when the pool does not exist")
	}
	if !strings.Contains(err.Error(), `pool "aaaaaaaa-bbbb-cccc-dddd-000000000001" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPoolGetAPIError(t *testing.T) {
	server := fakeXOGet(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"message":"boom"}`)
	})
	defer server.Close()
	isolatePointers(t, server.URL)

	_, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001")
	if err == nil {
		t.Fatal("expected an error when the API fails")
	}
	if !strings.Contains(err.Error(), "cannot get pool") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPoolGetBadID(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	if _, err := runGet(t, "get", "not-a-uuid"); err == nil {
		t.Fatal("expected an error for an invalid id")
	}
}
