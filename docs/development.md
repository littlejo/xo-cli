# Development

How to build, test, and release `xo`. The repository is set up for AI-assisted
and local development — see [AGENTS.md](../AGENTS.md) for the architecture
rules and conventions (the important one: `xenorchestra-go-sdk/v2` is the only
Xen Orchestra API layer).

## Toolchain

A [mise](https://mise.jdx.dev/) config pins the Go toolchain (`mise.toml`):

```sh
mise install          # install the pinned Go toolchain
mise run build        # go build -o dist/xo ./cmd/xo
mise run test         # go test ./...
mise run lint         # go vet ./... + gofmt check
```

Without mise, a Go 1.26+ toolchain works directly:

```sh
go build -o dist/xo ./cmd/xo
go test ./...
go vet ./... && test -z "$(gofmt -l .)"
```

`golangci-lint` (v2.14.0) is the reference linter; CI runs it on every push.

## Testing

Unit tests run without a Xen Orchestra instance (they use `httptest` servers
and deterministic fixtures). Integration tests are opt-in and only run when
explicitly enabled:

```sh
export XO_TEST_URL=https://xo.example.com
export XO_TEST_TOKEN=<token>
go test -tags=integration ./...
```

Without those variables the integration tests are reported as **skipped**,
never as passed. Integration credentials must never be committed; they are
configured through the environment.

### Functional tests against the simulator

Functional tests run the same `-tags=integration` suite against the
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
> `ci/xo-api-sim-cookie-auth.patch` closes that gap and is applied in CI; run
> the simulator from that patched source (or the fix contributed upstream)
> locally.

## CI / Release

- **CI** (`.github/workflows/ci.yml`): runs on push to `main` and on every PR —
  `gofmt`, `go vet`, `golangci-lint`, unit + integration tests (including a
  `-race` pass to catch data races early), build. A second `functional` job
  spins up the xo-api-sim REST simulator (pinned commit, cookie-auth patch) and
  runs the integration suite against it, so every push is tested end-to-end
  over real HTTP without a live instance.
- **Version** (`.github/workflows/version.yml`): on every push to `main`,
  computes the next semver tag from the conventional-commits history
  (`feat` → minor, anything else → patch), pushes it, and triggers the Release
  workflow.
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

## Repository layout

```text
cmd/xo/                  main entrypoint
internal/
  cli/                   SDK client construction, global flags, error hints
  commands/              one package per resource group (vm, host, pool, …)
    configure/           profile management
  config/                profiles, environment overrides, config file
  output/                table/json/yaml/text rendering, JMESPath queries
```

## SDK v2: what we build on

This is the result of inspecting `xenorchestra-go-sdk` (pinned in `go.mod`)
before implementing each command, per the SDK-first rule in
[AGENTS.md](../AGENTS.md). The AGENTS.md *SDK versioning* checklist requires
this section to be refreshed whenever the SDK is upgraded.

### Module layout

The SDK is a **single Go module** (`github.com/vatesfr/xenorchestra-go-sdk`,
pinned at `v1.19.0` in `go.mod`) that contains **both** APIs:

| Path | API | Used by xo-gocli |
| ---- | --- | ---------------- |
| `client/`, `pkg/services/jsonrpc/` | v1: JSON-RPC over WebSocket | **never** |
| `v2/` | v2: REST client + typed services | **yes — the only API layer** |
| `pkg/config/` | shared config type | yes (via `v2`) |
| `pkg/payloads/` | shared REST response types | yes (returned by the typed services) |

There is no `v2/go.mod`: `v2` is a plain subdirectory, so the import path
`github.com/vatesfr/xenorchestra-go-sdk/v2` is a subpackage of the v1 module
— the `/v2` suffix is a directory name, not a Go major-version path. The
v1.19.0 version number is the *module* version, not the REST API version
(the REST API itself is `/rest/v0`).

### Two entry points

1. **`v2.New(cfg) → library.Library`** — the typed facade. Returns
   `VM()`, `Host()`, `Pool()`, `SR()`, `Network()`, `Task()`, `VDI()`, `VBD()`,
   `PBD()`. Every service method is a REST call through the internal
   `v2/client` (`client.TypedGet` & co.), returning `pkg/payloads` structs.
   This is what `internal/cli.NewClient` builds and what most commands use.
2. **`v2/client.New(cfg) → *client.Client`** — the raw REST client. The SDK
   *exports* `HttpClient`, `BaseURL` and `AuthToken` precisely so callers can
   reach endpoints the SDK does not (yet) wrap. `internal/cli.NewHTTPClient`
   is built on it and powers `xo rest`, `xo token`, `xo template`, `xo task`
   and the VM PATCH in `xo vm update`. That is the documented escape hatch,
   not a second REST client.

### Authentication & transport

- Token mode: every request carries the token as an
  `authenticationToken` **cookie** (set by `doRequest`, the SDK's own
  mechanism). The CLI never sends `Authorization` headers itself.
- Username/password mode: `client.New` logs in via
  `POST /auth/login` (base URL **without** `/rest/v0`) and keeps the token
  cookie from the response.
- `ws`/`wss` endpoints are transparently rewritten to `http(s)`;
  `InsecureSkipVerify` is applied on a cloned transport; default HTTP timeout
  is 30 s (`cfg.ClientTimeout`).
- `v2/xo.go` also holds a **lazy** v1 client (`V1Client()`), created only if
  JSON-RPC is actually requested. The CLI never calls it, so no WebSocket is
  ever opened.

### Request model (as the SDK sends it)

- Base URL is `<endpoint>/rest/v0`; endpoints beginning with `api/` are sent
  without that prefix.
- Collection reads: `GET /<resource>` with `fields`, `limit`, `filter`
  (the XO live-filter syntax) query parameters.
- Single object: `GET /<resource>/<id>`.
- Actions: `POST /<resource>/<id>/actions/<name>` → `{"taskId": …}` (202,
  asynchronous); e.g. `start`, `clean_shutdown`, `hard_shutdown`,
  `clean_reboot`, `hard_reboot`, `snapshot`.
- Tags: `PUT`/`DELETE /vms/<id>/tags/<tag>`.
- Partial update: `PATCH /<resource>/<id>` with **camelCase** JSON fields
  (`nameLabel`, `nameDescription`, …) — responses, in contrast, use
  snake_case.
- Non-2xx responses surface as `API error: <status> - <body>`; the CLI maps
  the `404` substring to a concise "not found" message.
- `pkg/config.NewWithValues` (what we use) takes explicit values and reads no
  environment variables; `pkg/config.New` (env `XOA_*`) is not used.

### Known SDK gaps the CLI works around

| Gap | CLI workaround |
| --- | -------------- |
| `VM().Update` returns `not yet implemented` | `xo vm update` sends the PATCH itself via the exported `*client.Client` (documented in `vm/update.go`; to contribute upstream) |
| No typed service for `vm-templates`, tasks or user tokens | `TypedGet` directly (`xo template`, `xo task`, `xo token`) |
| `users/me` 307-redirects to the user id | handled by `net/http` following the redirect; `doTokensRequest` relies on 307 body replay for POST |
| `v2` package `init()` runs `gotenv.Load()` (reads a `.env` in the CWD) | harmless: we build the config with `NewWithValues`, which reads no env vars |

### Which command uses which SDK surface

| Command | SDK surface |
| ------- | ----------- |
| `vm list/get/create/start/stop/reboot/snapshot/tag` | typed `library.VM` |
| `host/pool/sr/network list/get` | typed `library.{Host,Pool,SR,Network}` |
| `vm update` | raw `*client.Client` (PATCH — SDK gap) |
| `template list/get`, `task list/get` | raw `client.TypedGet` |
| `token list/get/create` | raw `*client.Client` (GET/POST, 307 redirect) |
| `rest` | raw `*client.Client` (full method set) |

