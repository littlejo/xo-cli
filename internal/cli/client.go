package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"

	xov2 "github.com/vatesfr/xenorchestra-go-sdk/v2"
	v2client "github.com/vatesfr/xenorchestra-go-sdk/v2/client"

	xoconfig "github.com/littlejo/xo-gocli/internal/config"
)

const (
	// FlagProfile is the global --profile flag.
	FlagProfile = "profile"
	// FlagOutput is the global --output flag.
	FlagOutput = "output"
	// FlagDebug is the global --debug flag.
	FlagDebug = "debug"
	// EnvDebug enables verbose SDK/API error diagnostics (like --debug).
	EnvDebug = "XOA_DEBUG"
)

// Version is set at build time with -ldflags "-X ...=x.y.z".
var Version = "dev"

// Execute runs the given root command with the provided context and returns
// the first error encountered, if any.
//
// The command's stderr is used for diagnostics: when --debug (or $XOA_DEBUG)
// is set, extra context is printed about any failure — which profile and
// endpoint resolved, and the raw SDK/API error carried by the command.
// Normal runs print nothing extra; the single "Error: …" line is emitted by
// the caller (main).
func Execute(ctx context.Context, root *cobra.Command, args []string) error {
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return nil
	}
	if Debug(root) {
		printDebugError(root.ErrOrStderr(), root, err)
	}
	return err
}

// Debug reports whether verbose SDK/API error diagnostics are requested,
// either through the global --debug flag or the $XOA_DEBUG environment
// variable. The flag wins; the variable uses the same truthy values as
// XOA_YES (1, true, yes, case-insensitive).
func Debug(cmd *cobra.Command) bool {
	if cmd != nil {
		if b, err := cmd.Root().PersistentFlags().GetBool(FlagDebug); err == nil && b {
			return true
		}
	}
	v := os.Getenv(EnvDebug)
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// detailError wraps an error with an extra diagnostic detail. The detail is
// not part of Error() (so normal output stays concise and stable for
// scripting); it is revealed only in debug mode, where Execute prints it.
type detailError struct {
	err    error
	detail string
}

func (e *detailError) Error() string { return e.err.Error() }
func (e *detailError) Unwrap() error { return e.err }

// Detail returns the hidden diagnostic detail, or "" if err carries none.
func Detail(err error) string {
	var de *detailError
	if errors.As(err, &de) {
		return de.detail
	}
	return ""
}

// NotFound turns a lookup failure into the concise "<kind> not found" message
// when the API returned a 404, keeping the raw SDK error as debug-only detail
// (see --debug / $XOA_DEBUG). For any other error it wraps it in
// "cannot <verb> <kind> <id>: …" and applies the TLS hint.
//
// verb is the verb shown for non-404 failures ("get" for a read, "resolve"
// for an existence check); it does not affect the 404 wording.
func NotFound(kind, verb, id string, err error, insecure bool) error {
	if err != nil && strings.Contains(err.Error(), "404") {
		return &detailError{
			err:    fmt.Errorf("%s %q not found", kind, id),
			detail: err.Error(),
		}
	}
	return InsecureHint(fmt.Sprintf("cannot %s %s %q: %v", verb, kind, id, err), insecure)
}

// printDebugError prints the --debug diagnostics for a failing run to w:
// which profile and endpoint resolved (best effort, never failing) and, when
// the command attached one, the raw SDK/API error behind the concise message
// (e.g. the "API error: 404 Not Found - …" line for a not-found lookup).
func printDebugError(w io.Writer, cmd *cobra.Command, err error) {
	if cfg, errLoad := xoconfig.Load(ProfileName(cmd)); errLoad == nil {
		_, _ = fmt.Fprintf(w, "debug: profile=%s endpoint=%s\n", cfg.Name, cfg.Endpoint)
	}
	if d := Detail(err); d != "" {
		_, _ = fmt.Fprintf(w, "debug: %s\n", d)
	}
}

// ProfileName returns the profile selected with --profile or $XOA_PROFILE.
func ProfileName(cmd *cobra.Command) string {
	name, _ := cmd.Flags().GetString(FlagProfile)
	return name
}

// OutputFormat returns the requested output format.
func OutputFormat(cmd *cobra.Command) string {
	format, _ := cmd.Flags().GetString(FlagOutput)
	return format
}

// SkipConfirm reports whether destructive operations should run without a
// confirmation prompt: either the --yes flag or the $XOA_YES environment
// variable. The variable exists so scripts and CI pipelines can confirm
// non-interactively without repeating --yes on every command; it is checked
// only by confirmations, so setting it has no other effect.
func SkipConfirm(cmd *cobra.Command) bool {
	if yes, _ := cmd.Flags().GetBool("yes"); yes {
		return true
	}
	if v := os.Getenv("XOA_YES"); v != "" {
		return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
	}
	return false
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

// NewHTTPClient builds the authenticated SDK v2 REST client.
//
// The v2 client exposes its HttpClient, BaseURL and AuthToken specifically so
// callers can talk to REST endpoints the SDK does not (yet) wrap with a typed
// service. It is the same single API boundary as NewClient: authentication,
// TLS and base URL handling all come from the SDK.
//
// Note: for resources the SDK already exposes, prefer NewClient and the typed
// service; use this only for endpoints that are a known gap in the SDK (the
// missing operation should be contributed upstream).
func NewHTTPClient(cmd *cobra.Command, cfg *xoconfig.ClientConfig) (*v2client.Client, error) {
	sdkConfig, err := buildSDKConfig(cfg)
	if err != nil {
		return nil, err
	}
	client, err := v2client.New(sdkConfig)
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
	return errors.New(msg + "\n\nthe server certificate could not be verified; if this is a self-signed or internal certificate, retry with 'xo configure --insecure' (or set XOA_INSECURE=1)")
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
