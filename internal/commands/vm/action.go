package vm

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/config"
	"github.com/littlejo/xo-gocli/internal/output"
)

// flagYes skips the interactive confirmation for destructive operations. It is
// required whenever stdin is not a terminal so automation never blocks; the
// $XO_YES environment variable is an equivalent for scripts (cli.SkipConfirm).
const flagYes = "yes"

// parseID converts a positional VM identifier into a UUID.
func parseID(id string) (uuid.UUID, error) {
	u, err := uuid.FromString(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid VM id %q (expected a UUID)", id)
	}
	return u, nil
}

// nameOf resolves the name_label of the VM with the given id. It also acts as
// an existence check so actions fail with a clear "not found" error before the
// operation is performed.
func nameOf(ctx context.Context, xo library.Library, id uuid.UUID) (string, error) {
	vm, err := xo.VM().GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if vm.NameLabel == "" {
		return id.String(), nil
	}
	return vm.NameLabel, nil
}

// isTerminal reports whether in is connected to a terminal.
func isTerminal(in io.Reader) bool {
	file, ok := in.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

// confirm asks the user to confirm a destructive operation. When --yes is
// provided it always proceeds; when stdin is not a terminal it refuses so that
// automation never blocks, forcing the caller to pass --yes explicitly.
func confirm(cmd *cobra.Command, message string, yes bool) (bool, error) {
	if yes {
		return true, nil
	}
	in := cmd.InOrStdin()
	if !isTerminal(in) {
		return false, fmt.Errorf("confirmation required: re-run with --yes to proceed non-interactively")
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N]: ", message); err != nil {
		return false, err
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	if err != nil && answer == "" {
		return false, fmt.Errorf("cannot read confirmation: %w", err)
	}
	return answer == "y" || answer == "yes", nil
}

// newClient loads the selected profile and builds an authenticated SDK v2
// client. Commands must pass their cobra context to the SDK operations so that
// cancellation (Ctrl+C) reaches the HTTP layer.
func newClient(cmd *cobra.Command) (library.Library, *config.ClientConfig, error) {
	cfg, err := config.Load(cli.ProfileName(cmd))
	if err != nil {
		return nil, nil, err
	}
	xo, err := cli.NewClient(cmd, cfg)
	if err != nil {
		return nil, nil, err
	}
	return xo, cfg, nil
}

// notFound turns a lookup failure into the concise "VM not found" form when the
// API returned a 404, and keeps the original error otherwise. It passes the
// profile's insecure state through so the TLS hint is only suggested when it
// would actually help.
func notFound(id string, err error, insecure bool) error {
	if err != nil && strings.Contains(err.Error(), "404") {
		return fmt.Errorf("VM %q not found", id)
	}
	return cli.InsecureHint(fmt.Sprintf("cannot get VM %q: %v", id, err), insecure)
}

// actionSpec describes an async VM action so each command only supplies what
// differs: the verb shown to the user, whether it is destructive, and the SDK
// call to run.
type actionSpec struct {
	verb        string
	id          string
	destructive bool
	yes         bool
	perform     func(ctx context.Context, xo library.Library, id uuid.UUID) (string, error)
}

// runAction implements the common flow shared by start/stop/reboot/snapshot:
// load the client, resolve and confirm the target VM, run the action, and print
// the resulting task ID.
func runAction(cmd *cobra.Command, spec actionSpec) error {
	xo, cfg, err := newClient(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	id, err := parseID(spec.id)
	if err != nil {
		return err
	}
	name, err := nameOf(ctx, xo, id)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return fmt.Errorf("VM %q not found", spec.id)
		}
		return cli.InsecureHint(fmt.Sprintf("cannot resolve VM %q: %v", spec.id, err), cfg.Insecure)
	}

	if spec.destructive {
		ok, err := confirm(cmd, fmt.Sprintf("Are you sure you want to %s VM %q?", spec.verb, name), spec.yes)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("aborted")
		}
	}

	taskID, err := spec.perform(ctx, xo, id)
	if err != nil {
		return cli.InsecureHint(fmt.Sprintf("cannot %s VM %q: %v", spec.verb, name, err), cfg.Insecure)
	}

	return renderActionResult(cmd, spec.verb, name, taskID)
}

// renderActionResult prints the outcome of an async VM action: a friendly line
// for the human formats and a small machine document for json/yaml.
func renderActionResult(cmd *cobra.Command, verb, name, taskID string) error {
	format, err := output.ParseFormat(cli.OutputFormat(cmd))
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	switch format {
	case output.FormatJSON, output.FormatYAML:
		raw, err := output.Normalize(map[string]any{"action": verb, "vm": name, "task_id": taskID})
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, raw, nil)
	default:
		_, err := fmt.Fprintf(w, "Requested %s of %q (task %s)\n", verb, name, taskID)
		return err
	}
}
