# Miscellaneous

Things that don't fit the [usage](usage.md), [development](development.md) or
[comparison](comparison.md) pages.

## Architecture

`xo` is a thin UX layer over the official Go SDK
([`github.com/vatesfr/xenorchestra-go-sdk/v2`](https://github.com/vatesfr/xenorchestra-go-sdk)).
It talks to the Xen Orchestra **REST API** only — there is no second HTTP
client and no legacy JSON-RPC (v1) code path:

```text
   xo Go CLI (commands / output / query / config)
        │
        ▼
   xenorchestra-go-sdk/v2          ← the only Xen Orchestra client
        │
        ▼
   Xen Orchestra REST API
```

Commands stay thin: they resolve flags, call the SDK, and hand the result to
the output layer. SDK gaps (for example the missing typed VM update) are
reached through the SDK's own HTTP facilities, documented in the code, and
contributed upstream where appropriate. The full rules are in
[AGENTS.md](../AGENTS.md).

## Roadmap

Done:

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

Planned, as the SDK v2 exposes them (see also the
[comparison](comparison.md)):

- [ ] VM import / export (XVA)
- [ ] Remaining VM lifecycle operations (`delete`, `pause`, `resume`, `suspend`)
- [ ] `xo watch` (live events)
- [ ] More resources: VDI, VBD, servers, users, groups, backups

## Versioning

Releases follow [semantic versioning](https://semver.org/) and
[Conventional Commits](https://www.conventionalcommits.org/): a `feat` commit
bumps the minor version, anything else bumps the patch version. Tags are
created automatically on push to `main`; see
[Development → CI / Release](development.md#ci--release).

## License

This project is licensed under the [MIT License](https://opensource.org/licenses/MIT),
the same license as the Xen Orchestra Go SDK.
