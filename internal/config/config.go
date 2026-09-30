// Package config implements CLI-level configuration: named profiles,
// environment variables and the configuration file.
//
// It only stores connection settings. Authentication and HTTP handling are
// delegated to the Xen Orchestra SDK.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// EnvProfile selects the active profile (like AWS_PROFILE).
	EnvProfile = "XO_PROFILE"
	// EnvEndpoint overrides the profile endpoint.
	EnvEndpoint = "XO_ENDPOINT"
	// EnvToken overrides the profile authentication token.
	EnvToken = "XO_TOKEN"
	// EnvUsername overrides the profile username.
	EnvUsername = "XO_USERNAME"
	// EnvPassword overrides the profile password.
	EnvPassword = "XO_PASSWORD"
	// EnvInsecure overrides the profile insecure flag.
	EnvInsecure = "XO_INSECURE"
)

// DefaultProfile is used when no profile is selected.
const DefaultProfile = "default"

// Profile holds the connection settings for one Xen Orchestra instance.
type Profile struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Insecure bool   `yaml:"insecure"`
}

// File is the on-disk representation of the configuration.
type File struct {
	Current  string    `yaml:"current"`
	Profiles []Profile `yaml:"profiles"`
}

// ClientConfig is the resolved configuration for a single command run.
type ClientConfig struct {
	Name     string
	Endpoint string
	Token    string
	Username string
	Password string
	Insecure bool
}

// Load reads the configuration file, applies environment overrides and
// returns the resolved profile named profileName.
//
// Profile selection follows the AWS CLI precedence: an explicit profile name
// (flag) wins, then $XO_PROFILE, then the file's current profile, then the
// default profile.
func Load(profileName string) (*ClientConfig, error) {
	file, err := read()
	if err != nil {
		return nil, err
	}

	if profileName == "" {
		profileName = os.Getenv(EnvProfile)
	}
	if profileName == "" {
		profileName = file.Current
	}
	if profileName == "" {
		profileName = DefaultProfile
	}

	profile := findProfile(file, profileName)
	if profile == nil && profileName != DefaultProfile {
		return nil, fmt.Errorf("profile %q is not configured, run 'xo configure --profile %s'", profileName, profileName)
	}

	cfg := &ClientConfig{Name: profileName}
	if profile != nil {
		cfg.Endpoint = profile.Endpoint
		cfg.Token = profile.Token
		cfg.Username = profile.Username
		cfg.Password = profile.Password
		cfg.Insecure = profile.Insecure
	}

	cfg.applyEnv()

	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("no endpoint configured for profile %q, set the %s environment variable or run 'xo configure'", profileName, EnvEndpoint)
	}
	if cfg.Token == "" && (cfg.Username == "" || cfg.Password == "") {
		return nil, fmt.Errorf("no credentials configured for profile %q, run 'xo configure' or set %s (or %s and %s)", profileName, EnvToken, EnvUsername, EnvPassword)
	}

	return cfg, nil
}

func findProfile(file *File, name string) *Profile {
	for i := range file.Profiles {
		if file.Profiles[i].Name == name {
			return &file.Profiles[i]
		}
	}
	return nil
}

func (c *ClientConfig) applyEnv() {
	if v := os.Getenv(EnvEndpoint); v != "" {
		c.Endpoint = v
	}
	if v := os.Getenv(EnvToken); v != "" {
		c.Token = v
	}
	if v := os.Getenv(EnvUsername); v != "" {
		c.Username = v
	}
	if v := os.Getenv(EnvPassword); v != "" {
		c.Password = v
	}
	if v := os.Getenv(EnvInsecure); v != "" {
		c.Insecure = v == "1" || v == "true" || v == "yes"
	}
}

// read parses the configuration file. A missing file is not an error.
func read() (*File, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &File{Current: DefaultProfile}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read configuration file %s: %w", path, err)
	}

	file := &File{Current: DefaultProfile}
	if err := yaml.Unmarshal(data, file); err != nil {
		return nil, fmt.Errorf("cannot parse configuration file %s: %w", path, err)
	}
	return file, nil
}

// WriteFile stores profiles in the configuration file, creating it (and its
// directory) when needed. The file is written with 0600 permissions because it
// may contain credentials.
func WriteFile(file *File) (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("cannot create configuration directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("cannot write configuration file: %w", err)
	}
	return path, nil
}

// Upsert merges profile into file (matching by name) and updates the current
// profile marker. It returns the path of the written file.
func Upsert(profile Profile, makeCurrent bool) (string, error) {
	file, err := read()
	if err != nil {
		return "", err
	}

	found := false
	for i := range file.Profiles {
		if file.Profiles[i].Name == profile.Name {
			file.Profiles[i] = profile
			found = true
			break
		}
	}
	if !found {
		file.Profiles = append(file.Profiles, profile)
	}
	if makeCurrent {
		file.Current = profile.Name
	}
	return WriteFile(file)
}

// Path returns the location of the configuration file, honoring the
// XO_CONFIG_FILE environment variable used by the tests.
func Path() (string, error) {
	if p := os.Getenv("XO_CONFIG_FILE"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "xo", "config"), nil
}
