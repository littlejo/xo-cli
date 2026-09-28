package config

import (
	"os"
	"path/filepath"
	"testing"
)

func isolateConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config")
	t.Setenv("XO_CONFIG_FILE", path)
	// Clear any environment overrides so tests are deterministic.
	for _, key := range []string{EnvProfile, EnvEndpoint, EnvToken, EnvUsername, EnvPassword, EnvInsecure} {
		t.Setenv(key, "")
	}
	return path
}

func TestLoadWithoutAnyConfigurationFails(t *testing.T) {
	isolateConfig(t)

	if _, err := Load(""); err == nil {
		t.Fatal("expected an error when no endpoint is configured")
	}
}

func TestUpsertThenLoad(t *testing.T) {
	isolateConfig(t)

	profile := Profile{Name: "lab", Endpoint: "https://xo.lab.example.com", Token: "secret"}
	path, err := Upsert(profile, true)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config file was not written: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("config file is empty")
	}

	cfg, err := Load("lab")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Endpoint != profile.Endpoint || cfg.Token != profile.Token {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Name != "lab" {
		t.Fatalf("unexpected profile name: %q", cfg.Name)
	}
}

func TestUpsertReplacesExistingProfile(t *testing.T) {
	isolateConfig(t)

	if _, err := Upsert(Profile{Name: "lab", Endpoint: "https://old.example.com", Token: "a"}, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := Upsert(Profile{Name: "lab", Endpoint: "https://new.example.com", Token: "b"}, false); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	cfg, err := Load("lab")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Endpoint != "https://new.example.com" || cfg.Token != "b" {
		t.Fatalf("profile was not replaced: %+v", cfg)
	}
}

func TestLoadUnknownProfileFails(t *testing.T) {
	isolateConfig(t)

	if _, err := Upsert(Profile{Name: "lab", Endpoint: "https://xo.lab.example.com", Token: "a"}, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if _, err := Load("missing"); err == nil {
		t.Fatal("expected an error for an unknown profile")
	}
}

func TestEnvironmentOverridesFile(t *testing.T) {
	isolateConfig(t)

	if _, err := Upsert(Profile{Name: "lab", Endpoint: "https://file.example.com", Token: "file-token"}, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	t.Setenv(EnvEndpoint, "https://env.example.com")
	t.Setenv(EnvToken, "env-token")

	cfg, err := Load("lab")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Endpoint != "https://env.example.com" || cfg.Token != "env-token" {
		t.Fatalf("environment variables must win over the file: %+v", cfg)
	}
}

func TestProfileFromEnvironment(t *testing.T) {
	isolateConfig(t)

	if _, err := Upsert(Profile{Name: "prod", Endpoint: "https://prod.example.com", Token: "t"}, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	t.Setenv(EnvProfile, "prod")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Name != "prod" || cfg.Endpoint != "https://prod.example.com" {
		t.Fatalf("XO_PROFILE was not honored: %+v", cfg)
	}
}

func TestLoadRequiresCredentials(t *testing.T) {
	isolateConfig(t)

	t.Setenv(EnvEndpoint, "https://xo.example.com")
	if _, err := Load(""); err == nil {
		t.Fatal("expected an error when no credentials are configured")
	}
}
