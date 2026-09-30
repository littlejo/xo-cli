# Using `xo`

This page is the detailed reference: installation, configuration, every
command, and the output/querying model. For the overview and a five-minute
quickstart, see the [README](../README.md).

## Table of contents

- [Installation](#installation)
- [Configuration](#configuration)
  - [`xo configure`](#xo-configure)
  - [Environment variables](#environment-variables)
  - [Insecure mode](#insecure-mode)
- [Commands](#commands)
  - [Global flags](#global-flags)
  - [`xo version`](#xo-version)
  - [Shell completion](#shell-completion)
  - [`xo vm`](#xo-vm)
  - [`xo host`](#xo-host)
  - [`xo sr`](#xo-sr)
  - [`xo pool`](#xo-pool)
  - [`xo network`](#xo-network)
  - [`xo task`](#xo-task)
  - [`xo token`](#xo-token)
  - [`xo template`](#xo-template)
  - [`xo rest`](#xo-rest)
- [Output & querying](#output--querying)

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

Requires Go 1.26+ (a [mise](https://mise.jdx.dev/) config is included) — see
[Development → Toolchain](development.md#toolchain) for the build commands.

## Configuration

Profiles are stored in `~/.config/xo/config` (override the location with
`$XOA_CONFIG_FILE`). The file is written with `0600` permissions because it may
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

The active profile is resolved in this order: `--profile`, then `$XOA_PROFILE`,
then the file's `current` entry, then the `default` profile.

### `xo configure`

Manage configuration profiles: store or update one, and list, inspect or
remove the stored profiles.

```sh
# Store or update the profile selected with --profile
xo configure                                   # interactive, default profile
xo configure --profile lab                     # interactive, named profile
xo configure --profile lab --endpoint https://xo.example.com --token <token>
xo configure --profile lab --username admin --password <secret>
xo configure --profile lab -k                  # skip TLS verification (--insecure)

# Inspect and manage the stored profiles
xo configure list                              # all profiles (secrets masked)
xo configure list --output json
xo configure show lab                          # one profile, full secrets
xo configure remove lab                        # destructive: asks unless --yes
```

Values given with flags win over environment variables; unset values fall back
to the environment (`XOA_ENDPOINT`, `XOA_TOKEN`, `XOA_USERNAME`, `XOA_PASSWORD`),
then to an interactive prompt when stdin is a terminal.

### Environment variables

Environment variables always take precedence over the stored profile at run
time:

| Variable         | Purpose                                   |
| ---------------- | ----------------------------------------- |
| `XOA_PROFILE`     | Select the active profile                 |
| `XOA_ENDPOINT`    | Xen Orchestra base URL                    |
| `XOA_TOKEN`       | Authentication token                      |
| `XOA_USERNAME`    | Username (alternative to a token)         |
| `XOA_PASSWORD`    | Password (alternative to a token)         |
| `XOA_INSECURE`    | Skip TLS certificate verification         |
| `XOA_YES`         | Skip confirmation prompts (like `--yes`)  |
| `XOA_CONFIG_FILE` | Location of the configuration file        |

Either a token, or a username + password, must be available to authenticate.

`XOA_YES` is meant for scripts and CI: `XOA_YES=1 xo vm stop <id>` behaves like
`xo vm stop <id> --yes` without having to pass the flag everywhere.

### Insecure mode

For self-signed or internally-issued certificates:

```sh
xo configure --profile lab --insecure          # stored in the profile
XOA_INSECURE=1 xo vm list                        # per invocation
```

This disables certificate verification and is only appropriate for internal,
trusted networks. When a connection fails on certificate verification and
insecure mode is not enabled, the error points at this escape hatch.

## Commands

### Global flags

| Flag             | Description                                             |
| ---------------- | ------------------------------------------------------- |
| `-p`, `--profile`| Configuration profile to use (or `$XOA_PROFILE`)         |
| `-o`, `--output` | Output format: `table` (default), `json`, `yaml`, `text`|
| `--version`      | Print the CLI version and exit                          |

Commands that return data also accept `--query` / `-q`
(see [Output & querying](#output--querying)). Every command accepts `--help`.

### `xo version`

Print the CLI version, offline:

```sh
xo version          # or: xo --version, xo -v
```

The Xen Orchestra server version is not shown: the REST API does not expose
it (only the legacy JSON-RPC API does, which this CLI never uses).

### Shell completion

Completion is generated by cobra for bash, zsh, fish and PowerShell:

```sh
# bash (>= 4.4)
xo completion bash > /etc/bash_completion.d/xo     # or ~/.bash_completion.d/xo

# zsh
xo completion zsh > "${fpath[1]}/_xo"

# fish
xo completion fish > ~/.config/fish/completions/xo.fish

# PowerShell
xo completion powershell > xo.ps1
```

Each `xo completion <shell>` command prints its own install instructions with
`--help`.

### `xo vm`

Manage virtual machines.

```sh
# Read
xo vm list                          # all VMs
xo vm list --output json            # machine readable
xo vm list --power-state Running    # filter by power state
xo vm list --limit 10               # cap the number of results
xo vm list --query '[].name_label'  # project a single field
xo vm get <id>                      # one VM (table/json/yaml)

# Create
xo vm create web-02 --pool <pool-id> --template <template-id>
xo vm create web-02 --pool <pool-id> --template <template-id> --memory 4G
xo vm create web-02 --pool <pool-id> --template <template-id> --boot

# Update
xo vm update <id> --name web-01
xo vm update <id> --description "primary web server"
xo vm update <id> --tags production,web     # replaces the full tag list

# Tags
xo vm tag add <id> production
xo vm tag remove <id> production

# Disks
xo vm vdis <id>                     # list the VM's VDIs
xo vm vdis <id> --type user         # filter by VDI type
xo vm vdis <id> --query '[].name_label'

# Lifecycle (async actions return a task id; delete is synchronous)
xo vm start <id>                    # power on
xo vm start <id> --host <host-id>   # pin to a host
xo vm stop <id>                     # clean shutdown (asks to confirm)
xo vm stop <id> --hard              # power off immediately
xo vm stop <id> --yes               # skip confirmation (automation)
xo vm reboot <id>                   # clean reboot
xo vm reboot <id> --hard            # force a hard reboot
xo vm pause <id>                    # pause the vCPUs (state: Paused)
xo vm unpause <id>                  # resume a paused VM
xo vm suspend <id>                  # save to disk and release memory (state: Suspended)
xo vm resume <id>                   # restore a suspended VM
xo vm snapshot <id>                 # take a snapshot
xo vm snapshot <id> --name backup   # label the snapshot
xo vm export <id> > vm.xva          # export to XVA (or --file, --format ova)
xo vm import vm.xva --pool <pool>   # import an XVA into a pool
xo vm delete <id>                   # delete the VM (asks to confirm)
xo vm delete <id> --yes             # skip confirmation (automation)
```

`--memory` accepts bytes or human-readable sizes (`2G`, `512M`).

Destructive operations (`stop`, `delete`) require confirmation; pass `--yes`
(or set `XOA_YES`) to run non-interactively. Without either, a non-terminal
stdin is rejected rather than hanging, so automation never blocks. The
reversible actions (`pause`, `unpause`, `suspend`, `resume`) never prompt.

`vm export` streams the archive to stdout by default (use `--file` for a
file, `--format ova` for OVA, `--compress=false` to disable XVA
compression); `vm import` reads the XVA from a file or stdin (`-`) and
requires `--pool`. Both are performed through the SDK's own REST client
because the typed SDK service does not expose them yet (see
[development](development.md#known-sdk-gaps-the-cli-works-around)).

### `xo host`

Manage hosts.

```sh
xo host list
xo host list --query '[].name_label'
xo host list --query '[?power_state==`Running`].name_label'
xo host get <id>
```

### `xo sr`

Manage storage repositories.

```sh
xo sr list
xo sr list --type lvm               # filter by SR type (see note below)
xo sr list --query '[?SR_type==`nfs`].name_label'
xo sr get <id>
```

`--type` is passed to the XO live-filter engine, which is a case-insensitive
substring match: `--type lvm` also matches `lvmoiscsi`. For an exact type,
project with `--query` instead.

### `xo pool`

Manage pools.

```sh
xo pool list
xo pool list --query '[?HA_enabled].name_label'
xo pool get <id>
```

### `xo network`

Manage networks.

```sh
xo network list
xo network list --query '[].name_label'
xo network get <id>                 # one network (table/json/yaml)
```

### `xo task`

Manage asynchronous tasks.

```sh
xo task list                        # all asynchronous tasks
xo task list --status failure       # filter by status (pending, success, failure, interrupted)
xo task list --query '[].id'
xo task get <id>                    # one task (table/json/yaml)
```

Asynchronous operations (`vm start`, `vm create`, …) return a task id; follow
it with `xo task get <id>`.

### `xo token`

Manage authentication tokens.

These are the tokens of the current user (the same value `xo configure`
stores). The token **id is the secret**, so it is masked in the output by
default — pass `--no-secret` to reveal it (use with care).

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

Manage VM templates.

In Xen Orchestra, templates are first-class objects (REST resource
`vm-templates`), not part of the `vms` collection — so `xo vm list` does not
show them. `xo template list` reads that dedicated resource.

```sh
xo template list
xo template list --output json
xo template list --query '[].name_label'
xo template get <id>            # one template (table/json/yaml)
```

### `xo rest`

Call a raw Xen Orchestra REST endpoint.

Low-level escape hatch for Xen Orchestra REST endpoints that have no typed
command yet. Use it for what the typed commands don't cover — VDI/VBD
management, users and groups, SR actions, … — not to duplicate `xo vm list`
when `xo vm list` exists. The request goes through the SDK v2 HTTP client
(same authentication, base URL and TLS handling), so it is not a second REST
client. Prefer the typed commands when they cover what you need.

The path is relative to the REST API root (`/rest/v0`), and the full OpenAPI
spec at `<endpoint>/rest/v0/docs` lists every endpoint and its fields. See
also the [official REST API documentation](https://docs.xen-orchestra.com/automation/restapi).

```sh
# Resources with no typed command yet
xo rest get vdis                               # list disks (no 'xo vdi' yet)
xo rest get vdis/<id>                          # GET a single disk
xo rest get users --output json                # list users
xo rest get groups                             # list groups

# Actions without a typed command
xo rest post srs/<id>/actions/scan             # rescan an SR
xo rest post srs/<id>/actions/reclaim_space    # reclaim free space on an SR
xo rest post vbds/<id>/actions/connect         # attach a disk
xo rest post vbds/<id>/actions/disconnect      # detach a disk

# Request building
xo rest get vdis --param limit=10              # add query parameters
xo rest post vdis --data '{"name_label":"data"}' --param sr=<id>   # JSON body
xo rest post vdis --data - < vdi.json          # read the body from stdin
xo rest delete vdis/<id> --yes                 # destructive: asks unless --yes
xo rest get vdis --output json --query '[].name_label'
xo rest get vdis -i                             # status line + headers on stderr
```

Flags: `--data/-d` (JSON body, `-` = stdin), `--param KEY=VALUE` (repeatable),
`--header KEY: VALUE` (repeatable), `--query/-q`, `--yes`, `--include/-i`.

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

Format behavior:

- `table` (default): aligned columns for humans; for `get` a single-row table.
- `json` / `yaml`: the full structured data (or the `--query` projection).
  Machine-readable output is the only thing on stdout; errors go to stderr, so
  `xo vm list --output json | jq '.[].name_label'` always works.
- `text`: a compact key/value form; a list of objects becomes an auto-column
  table, a list of scalars one value per line.
