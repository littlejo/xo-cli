package pool

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
	"github.com/littlejo/xo-gocli/internal/output"
)

// flagYes skips the interactive confirmation for destructive pool operations.
// It is required whenever stdin is not a terminal so automation never blocks;
// the $XOA_YES environment variable is an equivalent for scripts
// (cli.SkipConfirm).
const flagYes = "yes"

// actionSpec describes a pool maintenance action so each command only supplies
// what differs: the verbs used in messages and the SDK call to run. The three
// actions map to the SDK's PoolAction service (POST /pools/{id}/actions/<name>),
// which is synchronous: it polls the backing task until it completes.
type actionSpec struct {
	verb        string // base form, used in error messages ("cannot rolling update ...")
	past        string // past form, used in the success line ("rolling update done")
	destructive bool   // takes the pool (or its hosts) down; asks for confirmation
	do          func(ctx context.Context, xo library.Library, id uuid.UUID) error
}

// runAction implements the common flow shared by rolling-update,
// rolling-reboot and emergency-shutdown: load the client, resolve the target
// pool (as an existence check), confirm when destructive, run the action, and
// print the outcome.
func runAction(cmd *cobra.Command, idStr string, spec actionSpec) error {
	xo, cfg, err := newClient(cmd)
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	id, err := parseID(idStr)
	if err != nil {
		return err
	}
	name, err := poolNameOf(ctx, xo, id)
	if err != nil {
		return cli.NotFound("pool", "resolve", idStr, err, cfg.Insecure)
	}

	if spec.destructive {
		ok, err := confirm(cmd, fmt.Sprintf("Are you sure you want to %s pool %q?", spec.verb, name), cli.SkipConfirm(cmd))
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("aborted")
		}
	}

	// All three actions wait for the backing task to finish; the SDK call can
	// take a long time, so warn on stderr (stdout stays clean for scripting).
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Running %s on pool %q, this may take a while…\n", spec.verb, name); err != nil {
		return err
	}

	if err := spec.do(ctx, xo, id); err != nil {
		return cli.InsecureHint(fmt.Sprintf("cannot %s pool %q: %v", spec.verb, name, err), cfg.Insecure)
	}

	return renderActionResult(cmd, spec, name)
}

// poolNameOf resolves the name_label of the pool with the given id. It also
// acts as an existence check so actions fail with a clear "not found" error
// before the operation is performed.
func poolNameOf(ctx context.Context, xo library.Library, id uuid.UUID) (string, error) {
	p, err := xo.Pool().Get(ctx, id)
	if err != nil {
		return "", err
	}
	if p.NameLabel == "" {
		return id.String(), nil
	}
	return p.NameLabel, nil
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

// renderActionResult prints the outcome of a pool action: a friendly line for
// the human formats and a small machine document for json/yaml.
func renderActionResult(cmd *cobra.Command, spec actionSpec, name string) error {
	format, err := output.ParseFormat(cli.OutputFormat(cmd))
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	switch format {
	case output.FormatJSON, output.FormatYAML:
		raw, err := output.Normalize(map[string]any{"action": spec.verb, "pool": name, "status": "done"})
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, raw, nil)
	default:
		_, err := fmt.Fprintf(w, "Pool %q %s\n", name, spec.past)
		return err
	}
}
