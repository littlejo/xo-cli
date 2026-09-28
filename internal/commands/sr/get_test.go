package sr

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

const fixtureSR = `{
	"id": "11111111-1111-4111-8111-111111111111",
	"uuid": "11111111-1111-4111-8111-111111111111",
	"type": "SR",
	"name_label": "Local storage",
	"SR_type": "lvm",
	"size": 10737418240,
	"usage": 5368709120,
	"physical_usage": 3221225472,
	"content_type": "user",
	"shared": false,
	"$container": "aaaaaaaa-bbbb-cccc-dddd-000000000001"
}`

// fakeXOGet serves GET /rest/v0/srs/{id}.
func fakeXOGet(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/rest/v0/srs/") {
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
		_, _ = fmt.Fprint(w, fixtureSR)
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

func TestSRGetTable(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	out, err := runGet(t, "get", "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatalf("sr get: %v", err)
	}
	for _, expected := range []string{"ID", "NAME", "TYPE", "SIZE", "USAGE", "CONTAINER", "Local storage", "lvm", "10.74GB", "5.369GB"} {
		if !strings.Contains(out, expected) {
			t.Errorf("table output missing %q:\n%s", expected, out)
		}
	}
}

func TestSRGetJSON(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	out, err := runGet(t, "get", "11111111-1111-4111-8111-111111111111", "--output", "json")
	if err != nil {
		t.Fatalf("sr get --output json: %v", err)
	}

	var sr map[string]any
	if err := json.Unmarshal([]byte(out), &sr); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if sr["name_label"] != "Local storage" || sr["SR_type"] != "lvm" {
		t.Fatalf("unexpected SR payload: %s", out)
	}
}

func TestSRGetQuery(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	out, err := runGet(t, "get", "11111111-1111-4111-8111-111111111111", "--query", "name_label")
	if err != nil {
		t.Fatalf("sr get --query: %v", err)
	}
	if strings.TrimSpace(out) != "Local storage" {
		t.Fatalf("unexpected query output: %q", out)
	}
}

func TestSRGetNotFound(t *testing.T) {
	server := fakeXOGet(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message":"not found"}`)
	})
	defer server.Close()
	isolatePointers(t, server.URL)

	_, err := runGet(t, "get", "11111111-1111-4111-8111-111111111111")
	if err == nil {
		t.Fatal("expected an error when the SR does not exist")
	}
	if !strings.Contains(err.Error(), `SR "11111111-1111-4111-8111-111111111111" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSRGetAPIError(t *testing.T) {
	server := fakeXOGet(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"message":"boom"}`)
	})
	defer server.Close()
	isolatePointers(t, server.URL)

	_, err := runGet(t, "get", "11111111-1111-4111-8111-111111111111")
	if err == nil {
		t.Fatal("expected an error when the API fails")
	}
	if !strings.Contains(err.Error(), "cannot get SR") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSRGetBadID(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	if _, err := runGet(t, "get", "not-a-uuid"); err == nil {
		t.Fatal("expected an error for an invalid id")
	}
}
