package host

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

const fixtureHost = `{
	"id": "aaaaaaaa-bbbb-cccc-dddd-000000000001",
	"uuid": "aaaaaaaa-bbbb-cccc-dddd-000000000001",
	"type": "host",
	"name_label": "host-01",
	"address": "10.0.0.11",
	"power_state": "Running",
	"version": "8.2.0",
	"memory": {"size": 2147483648, "usage": 1073741824},
	"cpus": {"cores": 8, "sockets": 2},
	"$pool": "99999999-9999-4999-8999-999999999999",
	"residentVms": ["11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"]
}`

// fakeXOGet serves GET /rest/v0/hosts/{id}.
func fakeXOGet(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/rest/v0/hosts/") {
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
		_, _ = fmt.Fprint(w, fixtureHost)
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
	root.PersistentFlags().BoolP(cli.FlagDebug, "d", false, "")
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

// runGetExec drives the command through cli.Execute (like the real binary),
// so the global --debug handling on its error path is exercised. It returns
// whatever was written to stderr.
func runGetExec(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newGetTestRoot()
	var errOut strings.Builder
	root.SetErr(&errOut)
	err := cli.Execute(context.Background(), root, args)
	return errOut.String(), err
}

func TestHostGetTable(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	out, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001")
	if err != nil {
		t.Fatalf("host get: %v", err)
	}
	for _, expected := range []string{"ID", "NAME", "ADDRESS", "POWER STATE", "PLATFORM", "MEMORY", "VMS", "POOL", "host-01", "10.0.0.11", "Running", "8.2.0", "2.147GB"} {
		if !strings.Contains(out, expected) {
			t.Errorf("table output missing %q:\n%s", expected, out)
		}
	}
}

func TestHostGetJSON(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	out, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001", "--output", "json")
	if err != nil {
		t.Fatalf("host get --output json: %v", err)
	}

	var host map[string]any
	if err := json.Unmarshal([]byte(out), &host); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if host["name_label"] != "host-01" || host["address"] != "10.0.0.11" {
		t.Fatalf("unexpected host payload: %s", out)
	}
}

func TestHostGetQuery(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	out, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001", "--query", "name_label")
	if err != nil {
		t.Fatalf("host get --query: %v", err)
	}
	if strings.TrimSpace(out) != "host-01" {
		t.Fatalf("unexpected query output: %q", out)
	}
}

func TestHostGetNotFound(t *testing.T) {
	server := fakeXOGet(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message":"not found"}`)
	})
	defer server.Close()
	isolatePointers(t, server.URL)

	_, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001")
	if err == nil {
		t.Fatal("expected an error when the host does not exist")
	}
	if !strings.Contains(err.Error(), `host "aaaaaaaa-bbbb-cccc-dddd-000000000001" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHostGetNotFoundDebugRevealsAPIError(t *testing.T) {
	server := fakeXOGet(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message":"not found"}`)
	})
	defer server.Close()
	isolatePointers(t, server.URL)

	errOut, err := runGetExec(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001", "--debug")
	if err == nil {
		t.Fatal("expected an error when the host does not exist")
	}
	// The concise message must stay the user-facing error …
	if !strings.Contains(err.Error(), `host "aaaaaaaa-bbbb-cccc-dddd-000000000001" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
	// … while --debug adds the raw API error and the resolved profile.
	if !strings.Contains(errOut, "404 Not Found") {
		t.Fatalf("--debug must reveal the raw API error, got: %q", errOut)
	}
	if !strings.Contains(errOut, "debug: profile=") {
		t.Fatalf("--debug must report the resolved profile, got: %q", errOut)
	}
}

func TestHostGetNotFoundWithoutDebugHasNoDiag(t *testing.T) {
	server := fakeXOGet(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message":"not found"}`)
	})
	defer server.Close()
	isolatePointers(t, server.URL)

	errOut, err := runGetExec(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001")
	if err == nil {
		t.Fatal("expected an error when the host does not exist")
	}
	if strings.Contains(errOut, "debug:") {
		t.Fatalf("no debug diagnostics expected without --debug, got: %q", errOut)
	}
}

func TestHostGetAPIError(t *testing.T) {
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
	if !strings.Contains(err.Error(), "cannot get host") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHostGetBadID(t *testing.T) {
	server := fakeXOGet(t, nil)
	defer server.Close()
	isolatePointers(t, server.URL)

	if _, err := runGet(t, "get", "not-a-uuid"); err == nil {
		t.Fatal("expected an error for an invalid id")
	}
}

// The TLS hint must follow the profile's insecure state end to end, i.e. the
// value resolved by config.Load (file, flag or environment) must reach the
// error formatting. A server that answers with an x509-style error lets us
// observe both sides of the hint through the full command path.

func tlsFailServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"message":"x509: certificate signed by unknown authority"}`)
	}))
}

func TestHostGetTLSErrorHintsWhenInsecureOff(t *testing.T) {
	server := tlsFailServer()
	defer server.Close()
	isolatePointers(t, server.URL)

	_, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001")
	if err == nil {
		t.Fatal("expected an error from the failing API")
	}
	if !strings.Contains(err.Error(), "--insecure") {
		t.Fatalf("expected the --insecure hint when insecure is off: %v", err)
	}
}

func TestHostGetTLSErrorNoHintWhenInsecureOn(t *testing.T) {
	server := tlsFailServer()
	defer server.Close()
	isolatePointers(t, server.URL)
	t.Setenv("XOA_INSECURE", "1")

	_, err := runGet(t, "get", "aaaaaaaa-bbbb-cccc-dddd-000000000001")
	if err == nil {
		t.Fatal("expected an error from the failing API")
	}
	if strings.Contains(err.Error(), "--insecure") {
		t.Fatalf("no hint expected when XOA_INSECURE=1: %v", err)
	}
}
