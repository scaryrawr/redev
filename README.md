# redev

`redev` is a dedicated CLI for smoother remote development with
[devpod](https://devpod.sh/). It is intended to combine devpod workspace
lifecycle commands with the richer SSH, port-forwarding, browser/OAuth, desktop
notification, and credential-forwarding primitives being extracted from
[`devssh`](https://github.com/scaryrawr/devssh).

`redev ssh` uses DevPod as the container transport and devssh for the interactive
SSH lifecycle.

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
redev ssh [flags] <workspace> [-- ssh-args...]
redev completion fish
```

`redev ssh` starts a devssh session through a DevPod `ssh --stdio` proxy. This
provides devssh's ControlMaster-based lifecycle, automatic remote port
monitoring, default reverse forwards for local services such as Ollama, LM
Studio, and Chrome DevTools, browser opening, notifications, and helper cleanup.
The DevPod stdio proxy runs with DevPod SSH services disabled so devssh owns
port forwarding and the two layers do not attempt to manage the same forwards.

The active `gh auth token` is still forwarded through DevPod without putting the
token value in command argv, SSH config, logs, or remote files. redev's hidden
ProxyCommand helper reads the token locally, sets it only on the local
`devpod ssh --stdio` child process as `GH_TOKEN`, and asks DevPod to forward
that environment variable into the workspace with `--send-env GH_TOKEN`.

No flags are required for the default devssh experience: port monitoring,
user-local xdg-open shim setup, browser opening, notifications, and default
reverse forwards are all enabled.

## Roadmap

- Add explicit, scoped GitHub credential forwarding through a short-lived broker
  instead of persistent remote environment files.
- Add devpod-aware workspace discovery, setup validation, cleanup, and richer
  generated shell completions.
