package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
