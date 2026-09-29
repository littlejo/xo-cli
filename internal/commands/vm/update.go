package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"

	"github.com/littlejo/xo-gocli/internal/cli"
	"github.com/littlejo/xo-gocli/internal/config"
	"github.com/littlejo/xo-gocli/internal/output"
)

const (
	flagDescription = "description"
)

// vmEndpointPrefix is the REST resource holding VMs. The SDK v2 exposes a
// typed service for it, but the Update operation is not implemented yet in the
// SDK (VM().Update returns "not yet implemented"). The REST API does support a
// partial update (PATCH /vms/{id}), so this command reaches it through the
// SDK's own REST client (cli.NewHTTPClient) — the same single API boundary, not
// a second HTTP client. The missing typed operation should be contributed to
// the SDK.
const vmEndpointPrefix = "vms"

// authCookieName is the cookie the SDK v2 client uses to carry the token.
const authCookieName = "authenticationToken"

func newUpdateCommand() *cobra.Command {
	var (
		name        string
		description string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update the name or description of a virtual machine",
		Long: `Update the name_label and/or name_description of a virtual machine.

At least one of --name or --description is required. Only the fields that are
given are changed; the others are left untouched.

The VM is referenced by its UUID, as returned by 'xo vm list'.

Note: this operation is not exposed by the SDK v2 typed service yet, so it is
performed through the SDK's own REST client against PATCH /vms/<id>.

Examples:
  xo vm update 550e8400-e29b-41d4-a716-446655440001 --name web-01
  xo vm update <id> --description "primary web server"
  xo vm update <id> --name web-01 --description "primary web server"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && description == "" {
				return fmt.Errorf("nothing to update: provide --name and/or --description")
			}
			format, err := output.ParseFormat(cli.OutputFormat(cmd))
			if err != nil {
				return err
			}

			id, err := parseID(args[0])
			if err != nil {
				return err
			}

			cfg, err := config.Load(cli.ProfileName(cmd))
			if err != nil {
				return err
			}

			body := map[string]any{}
			if name != "" {
				body["nameLabel"] = name
			}
			if description != "" {
				body["nameDescription"] = description
			}
			encoded, err := json.Marshal(body)
			if err != nil {
				return err
			}

			httpClient, err := cli.NewHTTPClient(cmd, cfg)
			if err != nil {
				return err
			}

			if err := patchVM(cmd.Context(), httpClient, id.String(), encoded); err != nil {
				return cli.InsecureHint(fmt.Sprintf("cannot update VM %q: %v", args[0], err), cfg.Insecure)
			}

			// Re-fetch through the typed SDK service so the output reflects the
			// updated object and uses the SDK's response types.
			xo, err := cli.NewClient(cmd, cfg)
			if err != nil {
				return err
			}
			vm, err := xo.VM().GetByID(cmd.Context(), id)
			if err != nil {
				return cli.InsecureHint(fmt.Sprintf("cannot get updated VM %q: %v", args[0], err), cfg.Insecure)
			}

			return renderUpdatedVM(cmd, format, vm)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&name, "name", "", "new name_label for the VM")
	flags.StringVar(&description, flagDescription, "", "new name_description for the VM")

	return cmd
}

// patchVM sends a JSON PATCH to /rest/v0/vms/<id> using the SDK v2 REST client.
// The SDK client's BaseURL is already suffixed with /rest/v0; the endpoint is
// joined to it and the auth cookie is attached.
func patchVM(ctx context.Context, c *client.Client, id string, body []byte) error {
	u := *c.BaseURL
	u.Path = path.Join(c.BaseURL.Path, vmEndpointPrefix, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: string(c.AuthToken)})

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	respBody, err := io.ReadAll(resp.Body)
	if cerr := resp.Body.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error: %s - %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// renderUpdatedVM prints the updated VM in the requested format, using the same
// columns as 'xo vm get'.
func renderUpdatedVM(cmd *cobra.Command, format output.Format, vm *payloads.VM) error {
	w := cmd.OutOrStdout()
	switch format {
	case output.FormatJSON, output.FormatYAML:
		normalized, err := output.Normalize(vm)
		if err != nil {
			return err
		}
		return output.Render(w, format, output.Table{}, normalized, nil)
	default:
		_, err := fmt.Fprintf(w, "VM %q updated:\n  id:     %s\n  name:   %s\n  desc:   %s\n  state:  %s\n",
			vm.NameLabel, vm.ID.String(), vm.NameLabel, vm.NameDescription, vm.PowerState)
		return err
	}
}
