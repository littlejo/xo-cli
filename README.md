# xo Go CLI

A modern command line client for [Xen Orchestra](https://github.com/vates/xen-orchestra),
written in Go with an AWS-CLI-like experience.

`xo` is a thin UX layer over the official Go SDK
([`github.com/vatesfr/xenorchestra-go-sdk/v2`](https://github.com/vatesfr/xenorchestra-go-sdk)).
It talks to the Xen Orchestra **REST API** only — there is no second HTTP client
and no legacy JSON-RPC (v1) code path.

```text
   xo Go CLI (commands / output / query / config)
        │
        ▼
   xenorchestra-go-sdk/v2          ← the only Xen Orchestra client
        │
        ▼
   Xen Orchestra REST API
```

## Quick start

```sh
# 1. Install the binary
curl -fsSL https://raw.githubusercontent.com/littlejo/xo-gocli/main/install.sh | sh

# 2. Store a profile (interactive, or via flags)
xo configure --profile lab \
  --endpoint https://xo.example.com \
  --token <token>

# 3. List VMs
xo vm list --profile lab

# 4. Get machine-readable output
xo vm list --profile lab --output json | jq '.[].name_label'

# 5. Filter with a JMESPath query
xo vm list --profile lab --query '[?power_state==`Running`].name_label'
```

You can also select the profile with an environment variable:

```sh
export XO_PROFILE=lab
xo vm list
```

## Features

- Resource-oriented commands (`xo vm list`, `xo host list`, `xo template list`, `xo sr list`, `xo pool list`, …)
- Multiple connection profiles, AWS-style (`--profile`, `$XO_PROFILE`)
- Human-friendly default output plus `--output json|yaml|text` for scripting
- AWS-CLI-like `--query` using [JMESPath](https://jmespath.org/)
- `--insecure` escape hatch for self-signed / internal certificates
- Static, dependency-free binaries (releases are built for the common 64-bit platforms)

## Installation

### Install script

```sh
curl -fsSL https://raw.githubusercontent.com/littlejo/xo-gocli/main/install.sh | sh
```

The script detects your OS and architecture, downloads the latest release
artifact, verifies its checksum, and installs the binary to
`/usr/local/bin` (or `~/.local/bin` if you don't have write access).

### Manual download

Download `xo_<version>_<os>_<arch>.tar.gz` (or `.zip` on Windows) from the
[Releases page](https://github.com/littlejo/xo-gocli/releases), verify it against
`xo_<version>_checksums.txt`, extract it and put the `xo` binary on your
`PATH`.

### From source

Requires Go 1.26+ (a [mise](https://mise.jdx.dev/) config is included).

```sh
go build -o dist/xo ./cmd/xo
```

or, with mise:

```sh
mise install
mise run build      # -> dist/xo
```

## Configuration

Profiles are stored in `~/.config/xo/config` (override the location with
`$XO_CONFIG_FILE`). The file is written with `0600` permissions because it may
hold credentials.

```yaml
current: lab
profiles:
  - name: lab
    endpoint: https://xo.example.com
    token: <token>
    # OR: username: admin / password: <secret>
    # OR: insecure: true
```

### `xo configure`

```sh
xo configure                                   # interactive, default profile
xo configure --profile lab                     # interactive, named profile
xo configure --profile lab --endpoint https://xo.example.com --token <token>
xo configure --profile lab --username admin --password <secret>
xo configure --profile lab --insecure          # skip TLS verification
```

### Environment variables

Environment variables always take precedence over the stored profile:

| Variable       | Purpose                                   |
| -------------- | ----------------------------------------- |
| `XO_PROFILE`   | Select the active profile                 |
| `XO_ENDPOINT`  | Xen Orchestra base URL                    |
| `XO_TOKEN`     | Authentication token                      |
| `XO_USERNAME`  | Username (alternative to a token)         |
| `XO_PASSWORD`  | Password (alternative to a token)         |
| `XO_INSECURE`  | Skip TLS certificate verification         |
| `XO_CONFIG_FILE` | Location of the configuration file     |

Either a token, or a username + password, must be available to authenticate.

### Insecure mode

For self-signed or internally-issued certificates:

```sh
xo configure --profile lab --insecure          # stored in the profile
XO_INSECURE=1 xo vm list                        # per invocation
```

This disables certificate verification and is only appropriate for internal,
trusted networks.

## Commands

### Global flags

| Flag            | Description                                             |
| --------------- | ------------------------------------------------------- |
| `--profile`     | Configuration profile to use (or `$XO_PROFILE`)         |
| `--output`      | Output format: `table` (default), `json`, `yaml`, `text`|

### `xo vm`

```sh
# Read
xo vm list                          # all VMs
xo vm list --output json            # machine readable
xo vm list --power-state Running    # filter by power state
xo vm list --limit 10               # cap the number of results
xo vm list --query '[].name_label'  # project a single field
xo vm get <id>                      # one VM (table/json/yaml)

# Lifecycle (all return an async task id)
xo vm start <id>                    # power on
xo vm start <id> --host <host-id>   # pin to a host
xo vm stop <id>                     # clean shutdown (asks to confirm)
xo vm stop <id> --hard              # power off immediately
xo vm stop <id> --yes               # skip confirmation (automation)
xo vm reboot <id>                   # clean reboot
xo vm reboot <id> --hard            # force a hard reboot
xo vm snapshot <id>                 # take a snapshot
xo vm snapshot <id> --name backup   # label the snapshot
```

Destructive operations (`stop`) require confirmation; pass `--yes` to run
non-interactively.

### `xo host`

```sh
xo host list
xo host list --query '[].name_label'
xo host list --query '[?power_state==`Running`].name_label'
```

### `xo sr`

```sh
xo sr list
xo sr list --type lvm               # filter by SR type (lvm, nfs, ext, iso, …)
xo sr list --query '[?SR_type==`nfs`].name_label'
```

### `xo pool`

```sh
xo pool list
xo pool list --query '[?HA_enabled].name_label'
```

### `xo network`

```sh
xo network list
xo network list --query '[].name_label'
xo network get <id>                 # one network (table/json/yaml)
```

### `xo task`

```sh
xo task list                        # all asynchronous tasks
xo task list --status failure       # filter by status (pending, success, failure, interrupted)
xo task list --query '[].id'
xo task get <id>                    # one task (table/json/yaml)
```

### `xo token`

Manage the authentication tokens of the current user (the same value `xo
configure` stores). The token **id is the secret**, so it is masked in the
output by default — pass `--no-secret` to reveal it (use with care).

```sh
xo token list                         # your tokens (id masked)
xo token list --no-secret --query '[].id'
xo token get <id>                     # one token (resolved against the list)
xo token create                       # create a token, printed in full once
xo token create --description "ci" --expires-in "30 days"
xo token create --client-id my-cli    # reuse the token for a given client
```

`create` prints the token in full **once** (save it, e.g. into `xo
configure`); `list`/`get` only show a masked id. Deletion is not exposed by
the REST API and is therefore not implemented here.

### `xo template`

In Xen Orchestra, templates are first-class objects (REST resource
`vm-templates`), not part of the `vms` collection — so `xo vm list` does not
show them. `xo template list` reads that dedicated resource.

```sh
xo template list
xo template list --output json
xo template list --query '[].name_label'
xo template get <id>            # one template (table/json/yaml)
```

More resources and sub-commands (`get`, `start`, `stop`, …) are added on top of
the SDK as it evolves. See `xo <resource> --help` for the current surface.

## Output & querying

The pipeline is always: **SDK response → structured data → query → formatter**.
Queries operate on the structured data, never on rendered tables.

```sh
xo vm list --output json                      # full objects as JSON
xo vm list --query '[].name_label'            # one value per line
xo vm list --query '[?power_state==`Running`].name_label'
xo vm list --query 'length(@)'                # count
```

Backtick literals (`` `Running` ``) work as in the AWS CLI even though the
underlying JMESPath engine uses single quotes.

## Development

The repo is set up for AI-assisted and local development. See [AGENTS.md](AGENTS.md)
for the architecture rules and conventions.

```sh
mise install          # install the pinned Go toolchain
mise run build        # go build -o dist/xo ./cmd/xo
mise run test         # go test ./...
mise run lint         # go vet ./... + gofmt check
```

### Testing

Unit tests run without a Xen Orchestra instance (they use `httptest` servers and
fixtures). Integration tests are opt-in and only run when explicitly enabled:

```sh
export XO_TEST_URL=https://xo.example.com
export XO_TEST_TOKEN=<token>
go test -tags=integration ./...
```

Without those variables the integration tests are reported as **skipped**, never
as passed.

### CI / Release

- **CI** (`.github/workflows/ci.yml`): runs on push to `main` and on every PR —
  `gofmt`, `go vet`, `golangci-lint`, unit + integration tests, build.
- **Version** (`.github/workflows/version.yml`): on every push to `main`,
  computes the next semver tag from the conventional-commits history
  (`feat` → minor, anything else → patch), pushes it, and triggers the
  Release workflow.
- **Release** (`.github/workflows/release.yml`): for a `v*.*.*` tag, runs the
  test suite, then builds cross-platform binaries with
  [GoReleaser](https://goreleaser.com) and publishes them as a draft GitHub
  release. It can also be run manually from the Actions tab (optionally
  targeting a specific tag).

Normal flow: push to `main` — the tag and the release are created
automatically. To release a specific commit by hand:

```sh
git tag v1.0.0 && git push origin v1.0.0
```

## Roadmap

- [x] `xo configure` + profiles
- [x] `xo vm list`
- [x] `xo sr list`
- [x] `xo pool list`
- [x] `xo host list`
- [x] `xo network list / get`
- [x] `xo vm get / start / stop / reboot / snapshot`
- [x] `xo template list / get`
- [x] `xo task list / get` (asynchronous operations)
- [x] `xo token list / get / create`

## License

This project is licensed under the [MIT License](https://opensource.org/licenses/MIT),
the same license as the Xen Orchestra Go SDK.
