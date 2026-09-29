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

## Comparison with `xo-cli`

[XO's reference CLI](https://github.com/vatesfr/xen-orchestra/blob/master/packages/xo-cli/README.md)
(`xo-cli`, a Node.js package) and this tool have different scopes: `xo-cli` is a
general-purpose, introspection-based client of `xo-server` (JSON-RPC over
WebSocket + a raw REST wrapper), described upstream as a *debug and power-user
tool*. This Go CLI is a typed, REST-only client with an AWS-CLI-like UX.

### Side-by-side

| Aspect                     | `xo-cli` (Node.js)                                                    | `xo` Go CLI (this project)                                                        |
| -------------------------- | --------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Protocol                   | JSON-RPC over WebSocket + REST wrapper                                | REST API only, via `xenorchestra-go-sdk/v2`                                       |
| Runtime                    | Node.js (npm package)                                                 | Static Go binary, no runtime dependencies                                         |
| Command model              | Dynamic: `xo-cli <method> <param>=<value>` for **every** server method, discovered at runtime (`list-commands`) | Static, resource-oriented: `xo <resource> <operation>` (discovered via `--help`) |
| Configuration              | Single registered instance (`xo-cli register` / `unregister`), token stored only, `--url` per-invocation override | Multiple named profiles (`xo configure --profile`), `$XO_PROFILE` / `XO_*` env vars, 0600 config file |
| Authentication             | username/password, token, OTP (`--otp`), token validity control (`--expiresIn`) | username/password or token (via SDK v2)                                           |
| Object listing             | `list-objects` on all object types, with property filters             | Typed `list` per resource (`vm`, `host`, `pool`, `sr`, `network`, `task`, `template`) with `--limit` and resource filters (`--power-state`, `--type`, `--status`) |
| Output formats             | Plain text or `--json`                                                | `table` (default), `json`, `yaml`, `text`                                         |
| Filtering / projection     | `filter=` / `fields=` parameters (XO filter syntax)                   | AWS-CLI-like `--query` with JMESPath (incl. backtick literals)                    |
| Live events                | `xo-cli watch [--ndjson]` (stream of notifications)                   | Not implemented                                                                   |
| Raw REST escape hatch      | `xo-cli rest get/post/patch/put/del` for any endpoint                 | `xo rest <method> <path>` (on top of SDK v2 HTTP facilities, not a second client) |
| Task management            | `rest get tasks/<id> wait[=result]`, `rest post tasks/<id>/actions/abort` | `task list` / `task get` (list filterable by `--status`); lifecycle actions return the task id |
| VM lifecycle               | All methods (`vm.start`, `vm.stop`, `vm.reboot`, `vm.pause`, …) | `create`, `start` (host pinning), `stop` (clean/hard, confirm + `--yes`), `reboot` (clean/hard), `snapshot`, `tag add/remove`, `update` (name/description/tags) |
| VM import / export (XVA)   | `vm.import` / `vm.export` (file streaming with `@=`)                  | Not implemented (SDK v2 exposes VDI import/export)                                |
| Token management           | `create-token` (accepts same params as `register`)                    | `token list / get / create` with secret masking in output                         |
| Untyped parameters         | `param=value`, JSON values via `json:` prefix                          | Typed flags per command (validated at parse time)                                 |
| Destructive operations     | No confirmation prompt                                                | Confirmation prompt + non-interactive `--yes`                                     |
| TLS                        | `--allowUnauthorized` / `--au`                                        | `--insecure` flag, per profile, or `$XO_INSECURE`                                 |
| Testing / CI               | Upstream tests only                                                   | Unit tests (httptest fixtures) + opt-in integration tests against a live XO       |

### Feature coverage

| Capability                                        | `xo-cli` | `xo` Go CLI |
| ------------------------------------------------- | -------- | ----------- |
| List VMs / hosts / pools / SRs / networks / tasks | ✅ (via `list-objects`) | ✅ |
| List VM templates                                 | ✅ (`list-objects type=VM-template`) | ✅ (`xo template list`) |
| Get a single object                               | ✅ (`rest get vms/<id>`) | ✅ (`xo <resource> get <id>`) |
| Start / stop / reboot a VM                        | ✅ | ✅ |
| Create a VM (from a template)                     | ✅ | ✅ (`xo vm create`) |
| Update a VM (name, description)                   | ✅ | ✅ (`xo vm update`) |
| Manage VM tags                                    | ✅ | ✅ (`xo vm tag add/remove`) |
| Pause / resume / suspend / delete a VM            | ✅ | ⬜ (in SDK v2, not yet exposed) |
| Snapshot a VM                                     | ✅ | ✅ |
| Import / export VM (XVA)                          | ✅ | ⬜ |
| Manage tokens                                     | ✅ (`create-token`) | ✅ (`token list/get/create`, masking) |
| Watch notifications in real time                  | ✅ | ⬜ |
| Raw REST access (any endpoint)                    | ✅ | ✅ (`xo rest`) |
| Call any other server method (server, users, groups, backups, VDI, VBD, PBD, …) | ✅ | ⬜ (added resource by resource on top of the SDK) |
| Multiple instances / profiles                     | ⬜ (single registered instance) | ✅ |
| Human-friendly tables                             | ⬜ | ✅ |
| JMESPath querying                                 | ⬜ | ✅ |
| Static binary, no Node.js                         | ⬜ | ✅ |

Legend: ✅ available — ⬜ not available.

### Summary

- **Where `xo-cli` is broader**: it can call *every* `xo-server` method
  (hundreds: servers, users, groups, backups, VDI/VBD, network creation, …),
  stream live events, and reach any REST endpoint directly. For one-off or
  exotic operations it remains the most complete tool.
- **Where this CLI is better**: predictable and typed commands, multiple
  profiles, several output formats, JMESPath querying, non-interactive
  destructive operations, a static binary, and a REST-only architecture with
  no JSON-RPC/WebSocket dependency — better suited for automation, CI, and
  shell scripting.
- **What is still missing** (roadmap): `watch`, VM import/export, the
  remaining VM lifecycle operations (`delete`, `pause`, `resume`, `suspend`),
  and more resources (VDI, VBD, servers, users, groups, backups). These are
  added as they are exposed by the Go SDK v2, per the
  [architecture rules](AGENTS.md).

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
xo host get <id>
```

### `xo sr`

```sh
xo sr list
xo sr list --type lvm               # filter by SR type (lvm, nfs, ext, iso, …)
xo sr list --query '[?SR_type==`nfs`].name_label'
xo sr get <id>
```

### `xo pool`

```sh
xo pool list
xo pool list --query '[?HA_enabled].name_label'
xo pool get <id>
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

### `xo rest`

Low-level escape hatch for any Xen Orchestra REST endpoint the SDK does not
(yet) wrap in a typed command. The request goes through the SDK v2 HTTP client
(same authentication, base URL and TLS handling), so it is not a second REST
client. Prefer the typed commands when they cover what you need.

The path is relative to the REST API root (`/rest/v0`).

```sh
xo rest get vms                                # list VMs (any endpoint works)
xo rest get vms --param limit=10               # add query parameters
xo rest get vms/<id>                           # GET a single object
xo rest post vms --data '{"name_label":"web-01"}'   # send a JSON body
xo rest patch vms/<id> --data '{"name_label":"x"}'  # partial update
xo rest delete vms/<id> --yes                  # destructive: asks unless --yes
xo rest post vms --data - < vm.json            # read the body from stdin
xo rest get vms --output json --query '[].name_label'
xo rest get vms -i                             # status line + headers on stderr
```

Flags: `--data/-d` (JSON body, `-` = stdin), `--param KEY=VALUE` (repeatable),
`--header KEY: VALUE` (repeatable), `--query`, `--yes`, `--include/-i`.

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

**Functional tests** run the same `-tags=integration` suite against the
[xo-api-sim](https://github.com/vatesfr/xo-api-sim) REST API simulator, so the
CLI is exercised end-to-end with real HTTP requests but without a live Xen
Orchestra instance or any credential. In CI this is the `functional` job;
locally you can point the suite at a simulator you run yourself:

```sh
# terminal 1: start the simulator (see the xo-api-sim repo)
npm ci && npm run build
PORT=3001 AUTH_TOKEN=test-token node dist/index.js

# terminal 2: run the functional suite against it
XO_TEST_URL=http://localhost:3001 XO_TEST_TOKEN=test-token go test -tags=integration ./...
```

> Note: the SDK v2 client authenticates with an `authenticationToken` cookie,
> while xo-api-sim (as released) only reads the `Authorization: Bearer` header.
> `ci/xo-api-sim-cookie-auth.patch` closes that gap and is applied in CI; run the
> simulator from that patched source (or the fix contributed upstream) locally.

### CI / Release

- **CI** (`.github/workflows/ci.yml`): runs on push to `main` and on every PR —
  `gofmt`, `go vet`, `golangci-lint`, unit + integration tests, build. A second
  `functional` job spins up the xo-api-sim REST simulator (pinned commit,
  cookie-auth patch) and runs the integration suite against it, so every push is
  tested end-to-end over real HTTP without a live instance.
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
- [x] `xo host list / get`
- [x] `xo sr list / get`
- [x] `xo pool list / get`
- [x] `xo network list / get`
- [x] `xo vm get / start / stop / reboot / snapshot`
- [x] `xo vm create / update / tag add / tag remove`
- [x] `xo template list / get`
- [x] `xo task list / get` (asynchronous operations)
- [x] `xo token list / get / create`
- [x] `xo rest` (raw REST escape hatch on top of SDK v2)

## License

This project is licensed under the [MIT License](https://opensource.org/licenses/MIT),
the same license as the Xen Orchestra Go SDK.
