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
