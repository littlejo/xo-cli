# Comparison with `xo-cli`

[XO's reference CLI](https://github.com/vatesfr/xen-orchestra/blob/master/packages/xo-cli/README.md)
(`xo-cli`, a Node.js package) and this tool have different scopes: `xo-cli` is a
general-purpose, introspection-based client of `xo-server` (JSON-RPC over
WebSocket + a raw REST wrapper), described upstream as a *debug and power-user
tool*. This Go CLI is a typed, REST-only client with an AWS-CLI-like UX.

## Side-by-side

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
| VM lifecycle               | All methods (`vm.start`, `vm.stop`, `vm.reboot`, `vm.pause`, …)       | `create`, `start` (host pinning), `stop` (clean/hard, confirm + `--yes`), `reboot` (clean/hard), `snapshot`, `tag add/remove`, `update` (name/description/tags) |
| VM import / export (XVA)   | `vm.import` / `vm.export` (file streaming with `@=`)                  | Not implemented (SDK v2 exposes VDI import/export)                                |
| Token management           | `create-token` (accepts same params as `register`)                    | `token list / get / create` with secret masking in output                         |
| Untyped parameters         | `param=value`, JSON values via `json:` prefix                          | Typed flags per command (validated at parse time)                                 |
| Destructive operations     | No confirmation prompt                                                | Confirmation prompt + non-interactive `--yes`                                     |
| TLS                        | `--allowUnauthorized` / `--au`                                        | `--insecure` flag, per profile, or `$XO_INSECURE`                                 |
| Testing / CI               | Upstream tests only                                                   | Unit tests (httptest fixtures) + opt-in integration tests against a live XO       |

## Feature coverage

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

## Summary

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
  [architecture rules](../AGENTS.md).
