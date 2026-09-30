package sr

import (
	"errors"
	"strings"
	"testing"
)

// notFound must report a 404 as a concise "not found" error and pass the
// profile's insecure state through to the TLS hint (regression: the hint
// logic used to always see "insecure off").

func TestNotFoundReports404(t *testing.T) {
	err := notFound("abc", errors.New("API error: 404 Not Found - not found"), false)
	if !strings.Contains(err.Error(), `SR "abc" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNotFoundHintsWhenInsecureOff(t *testing.T) {
	err := notFound("abc", errors.New("x509: certificate signed by unknown authority"), false)
	if !strings.Contains(err.Error(), "--insecure") {
		t.Fatalf("expected the --insecure hint: %v", err)
	}
	if !strings.Contains(err.Error(), "cannot get SR") {
		t.Fatalf("original error must be kept: %v", err)
	}
}

func TestNotFoundNoHintWhenAlreadyInsecure(t *testing.T) {
	err := notFound("abc", errors.New("x509: certificate signed by unknown authority"), true)
	if strings.Contains(err.Error(), "--insecure") {
		t.Fatalf("no hint expected when already in insecure mode: %v", err)
	}
	if !strings.Contains(err.Error(), "cannot get SR") {
		t.Fatalf("original error must be kept: %v", err)
	}
}
