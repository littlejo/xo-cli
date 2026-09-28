package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"

	xov2 "github.com/vatesfr/xenorchestra-go-sdk/v2"

	xoconfig "github.com/vatesfr/xo-cli/internal/config"
)

const (
	// FlagProfile is the global --profile flag.
	FlagProfile = "profile"
	// FlagOutput is the global --output flag.
	FlagOutput = "output"
)

// Version is set at build time with -ldflags "-X ...=x.y.z".
var Version = "dev"

// Execute runs the given root command with the provided context and returns
// the first error encountered, if any.
func Execute(ctx context.Context, root *cobra.Command, args []string) error {
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}

// ProfileName returns the profile selected with --profile or $XO_PROFILE.
func ProfileName(cmd *cobra.Command) string {
	name, _ := cmd.Flags().GetString(FlagProfile)
	return name
}

// OutputFormat returns the requested output format.
func OutputFormat(cmd *cobra.Command) string {
	format, _ := cmd.Flags().GetString(FlagOutput)
	return format
}

// NewClient resolves the selected profile and builds an authenticated SDK v2
// client. Commands must pass their cobra context to the SDK operations so
// that cancellation (Ctrl+C) reaches the HTTP layer; the SDK client enforces
// its own per-request timeout.
func NewClient(cmd *cobra.Command, cfg *xoconfig.ClientConfig) (library.Library, error) {
	sdkConfig, err := buildSDKConfig(cfg)
	if err != nil {
		return nil, err
	}

	client, err := xov2.New(sdkConfig)
	if err != nil {
		return nil, newConnectionError(cfg, err)
	}
	return client, nil
}

func newConnectionError(cfg *xoconfig.ClientConfig, err error) error {
	var msg string
	if cfg.Token == "" && cfg.Username != "" {
		msg = fmt.Sprintf("authentication to %s failed: %v", cfg.Endpoint, err)
	} else {
		msg = fmt.Sprintf("cannot connect to %s: %v", cfg.Endpoint, err)
	}
	return InsecureHint(msg, cfg.Insecure)
}

// InsecureHint returns msg unchanged, or appends a hint when the failure is a
// TLS certificate problem and insecure mode is not already enabled. Commands
// should run their SDK errors through it so users get the documented escape
// hatch instead of raw x509 internals.
func InsecureHint(msg string, alreadyInsecure bool) error {
	msg = cleanSDKArtifact(msg)
	if alreadyInsecure || !isTLSVerifyError(msg) {
		return errors.New(msg)
	}
	return errors.New(msg + "\n\nthe server certificate could not be verified; if this is a self-signed or internal certificate, retry with 'xo configure --insecure' (or set XO_INSECURE=1)")
}

// cleanSDKArtifact removes the trailing "%!(EXTRA ...)" marker that the Go
// runtime appends when the SDK formats an error with more arguments than
// format verbs. It is purely cosmetic and always appears at the end.
func cleanSDKArtifact(msg string) string {
	if i := strings.Index(msg, "%!(EXTRA"); i >= 0 {
		return strings.TrimSpace(msg[:i])
	}
	return msg
}

func isTLSVerifyError(msg string) bool {
	return strings.Contains(msg, "certificate") || strings.Contains(msg, "x509")
}

func buildSDKConfig(cfg *xoconfig.ClientConfig) (*config.Config, error) {
	sdk := &config.Config{
		Url:                cfg.Endpoint,
		Token:              cfg.Token,
		Username:           cfg.Username,
		Password:           cfg.Password,
		InsecureSkipVerify: cfg.Insecure,
		// Keep the SDK quiet: the CLI owns stdout/stderr.
		LogOutputPaths:      []string{"/dev/null"},
		LogErrorOutputPaths: []string{"/dev/null"},
	}
	return config.NewWithValues(sdk)
}
