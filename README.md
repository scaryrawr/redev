# redev

`redev` is a dedicated CLI for smoother remote development with
[devpod](https://devpod.sh/). It is intended to combine devpod workspace
lifecycle commands with the richer SSH, port-forwarding, browser/OAuth, desktop
notification, and credential-forwarding primitives being extracted from
[`devssh`](https://github.com/scaryrawr/devssh).

This repository starts with a small, working frontend around `devpod`; the
deeper `devssh` integration will follow once those primitives are importable.

## Install from source

```bash
go install ./cmd/redev
```

## Commands

```bash
redev doctor
redev list
redev list --json
redev open <workspace>
redev open --ide <name> <workspace>
redev ssh <workspace> [-- ssh-args...]
redev completion fish
```

## Roadmap

- Use shared `devssh` packages for ControlMaster lifecycle, forwarding,
  browser opening, notifications, and remote helper upload.
- Add explicit, scoped GitHub credential forwarding through a short-lived broker
  instead of persistent remote environment files.
- Add devpod-aware workspace discovery, setup validation, cleanup, and richer
  generated shell completions.
