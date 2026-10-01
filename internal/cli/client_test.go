package cli

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	xoconfig "github.com/littlejo/xo-gocli/internal/config"
)

// fakeXOVMS serves the VM list endpoint behind the self-signed TLS
// certificate of httptest.NewTLSServer.
func fakeXOVMS(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/v0/vms" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"550e8400-e29b-41d4-a716-446655440001","name_label":"web-01","power_state":"Running","memory":{"size":1024},"CPUs":{"number":1},"boot":{},"type":"vm"}]`))
	}))
}

func newInsecureTestConfig(url string, insecure bool) *xoconfig.ClientConfig {
	return &xoconfig.ClientConfig{Endpoint: url, Token: "t", Insecure: insecure}
}

func TestNewClientRejectsSelfSignedWithoutInsecure(t *testing.T) {
	server := fakeXOVMS(t)
	defer server.Close()

	client, err := NewClient(nil, newInsecureTestConfig(server.URL, false))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.VM().GetAll(context.Background(), 0, ""); err == nil {
		t.Fatal("expected a TLS verification error")
	}
}

func TestNewClientAcceptsSelfSignedWithInsecure(t *testing.T) {
	server := fakeXOVMS(t)
	defer server.Close()

	client, err := NewClient(nil, newInsecureTestConfig(server.URL, true))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	vms, err := client.VM().GetAll(context.Background(), 0, "")
	if err != nil {
		t.Fatalf("GetAll with Insecure: %v", err)
	}
	if len(vms) != 1 || vms[0].NameLabel != "web-01" {
		t.Fatalf("unexpected VMs: %+v", vms)
	}
}

func TestWithInsecureHint(t *testing.T) {
	if err := InsecureHint("certificate is not valid for any names", false); err == nil ||
		!strings.Contains(err.Error(), "--insecure") {
		t.Errorf("expected an insecure hint, got: %v", err)
	}
	if err := InsecureHint("certificate is not valid for any names", true); err == nil ||
		strings.Contains(err.Error(), "--insecure") {
		t.Errorf("no hint should be added when insecure is already on: %v", err)
	}
	if err := InsecureHint("connection refused", false); err == nil ||
		strings.Contains(err.Error(), "--insecure") {
		t.Errorf("no hint should be added for non-TLS errors: %v", err)
	}
}

func newDebugTestRoot(fail error) *cobra.Command {
	root := &cobra.Command{Use: "xo", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().BoolP(FlagDebug, "d", false, "")
	root.AddCommand(&cobra.Command{
		Use:  "fail",
		RunE: func(*cobra.Command, []string) error { return fail },
	})
	return root
}

// isolateDebugEnv points the config resolution at an empty file with fixed
// credentials so the debug profile line is deterministic.
func isolateDebugEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XOA_CONFIG_FILE", t.TempDir()+"/config")
	for _, key := range []string{"XOA_PROFILE", "XOA_ENDPOINT", "XOA_TOKEN", "XOA_USERNAME", "XOA_PASSWORD", "XOA_INSECURE", "XOA_YES"} {
		t.Setenv(key, "")
	}
	t.Setenv("XOA_ENDPOINT", "https://xoa.test")
	t.Setenv("XOA_TOKEN", "test-token")
}

func TestDebugFromFlag(t *testing.T) {
	// The flag value is only populated once cobra parses it, i.e. inside
	// Execute, so the test must go through it before asking Debug.
	root := newDebugTestRoot(nil)
	if err := Execute(context.Background(), root, []string{"fail", "--debug"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !Debug(root) {
		t.Fatal("--debug must enable debug mode")
	}
}

func TestDebugFromEnvVar(t *testing.T) {
	for _, value := range []string{"1", "true", "yes", "TRUE"} {
		t.Setenv(EnvDebug, value)
		root := newDebugTestRoot(nil)
		if !Debug(root) {
			t.Fatalf("$%s=%q must enable debug mode", EnvDebug, value)
		}
	}
}

func TestDebugOffByDefault(t *testing.T) {
	t.Setenv(EnvDebug, "")
	root := newDebugTestRoot(nil)
	if Debug(root) {
		t.Fatal("debug mode must be off by default")
	}
}

func TestNotFound404IsConciseButCarriesDetail(t *testing.T) {
	err := NotFound("host", "get", "abc", errors.New("API error: 404 Not Found - not found"), false)
	if !strings.Contains(err.Error(), `host "abc" not found`) {
		t.Fatalf("expected a concise not-found message: %v", err)
	}
	if strings.Contains(err.Error(), "API error") {
		t.Fatalf("the raw API error must not leak into the normal message: %v", err)
	}
	if d := Detail(err); !strings.Contains(d, "404 Not Found") {
		t.Fatalf("the debug detail must carry the raw API error, got: %q", d)
	}
}

func TestNotFoundOtherErrorKeepsFullMessage(t *testing.T) {
	err := NotFound("VM", "resolve", "abc", errors.New("x509: certificate signed by unknown authority"), false)
	if !strings.Contains(err.Error(), `cannot resolve VM "abc"`) {
		t.Fatalf("expected the cannot-resolve form: %v", err)
	}
	if Detail(err) != "" {
		t.Fatalf("non-404 errors must not carry a hidden detail: %q", Detail(err))
	}
}

func TestExecutePrintsDebugDetails(t *testing.T) {
	isolateDebugEnv(t)
	t.Setenv(EnvDebug, "")
	raw := errors.New("API error: 404 Not Found - not found")
	root := newDebugTestRoot(NotFound("host", "get", "abc", raw, false))

	var errOut strings.Builder
	root.SetErr(&errOut)
	if err := Execute(context.Background(), root, []string{"fail", "--debug"}); err == nil {
		t.Fatal("expected the injected error")
	}
	got := errOut.String()
	if !strings.Contains(got, "debug: profile=default endpoint=https://xoa.test") {
		t.Errorf("debug output must name the resolved profile and endpoint:\n%s", got)
	}
	if !strings.Contains(got, "404 Not Found") {
		t.Errorf("debug output must reveal the raw API error:\n%s", got)
	}
}

func TestExecutePrintsNothingWithoutDebug(t *testing.T) {
	isolateDebugEnv(t)
	t.Setenv(EnvDebug, "")
	raw := errors.New("API error: 404 Not Found - not found")
	root := newDebugTestRoot(NotFound("host", "get", "abc", raw, false))

	var errOut strings.Builder
	root.SetErr(&errOut)
	if err := Execute(context.Background(), root, []string{"fail"}); err == nil {
		t.Fatal("expected the injected error")
	}
	if errOut.String() != "" {
		t.Errorf("no diagnostics expected without --debug, got:\n%s", errOut.String())
	}
}

func TestExecuteDebugViaEnvVar(t *testing.T) {
	isolateDebugEnv(t)
	t.Setenv(EnvDebug, "1")
	raw := errors.New("API error: 404 Not Found - not found")
	root := newDebugTestRoot(NotFound("host", "get", "abc", raw, false))

	var errOut strings.Builder
	root.SetErr(&errOut)
	if err := Execute(context.Background(), root, []string{"fail"}); err == nil {
		t.Fatal("expected the injected error")
	}
	if !strings.Contains(errOut.String(), "404 Not Found") {
		t.Errorf("$XOA_DEBUG must reveal the raw API error:\n%s", errOut.String())
	}
}
