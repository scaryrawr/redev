# Agent Development Guide

`redev` is a Go CLI for devpod-oriented remote development. Keep devpod-specific
workspace orchestration in this repo and keep generic SSH/session primitives in
`devssh`.

## Build and test

```bash
go test ./...
go build ./cmd/redev
```

## Project conventions

- Prefer small packages with explicit dependencies so `devssh` package
  integration stays testable.
- Do not persist credentials in remote files, shell profiles, command argv, or
  logs. Credential forwarding must be scoped, redacted, and cleaned up.
- `redev ssh` always forwards the active GitHub token. Keep forwarding safe by
  setting `GH_TOKEN` only in the local `devpod ssh` child process and using
  `--send-env GH_TOKEN`; avoid `--set-env` with token values because that puts
  secrets in command argv. When using devssh through DevPod stdio, keep token
  lookup inside the hidden proxy helper so ProxyCommand, SSH options, and config
  files stay token-free.
- In the default devssh-backed `redev ssh` path, DevPod is only the stdio SSH
  transport. Run the DevPod proxy with SSH services disabled so devssh owns port
  monitoring and forwarding and the two layers do not forward the same ports.
  Keep these devssh features on by default; do not add public disable/fallback
  flags unless explicitly requested.
- Preserve DevPod's configured SSH user for the stdio transport. Read the
  generated `<workspace>.devpod` SSH config user and pass it to both OpenSSH and
  the DevPod proxy; do not force root unless no configured user is available.
- Synthetic DevPod stdio SSH hosts can present unstable helper-generated host
  keys. Match DevPod's generated SSH config by disabling strict host key
  checking and using `/dev/null` for known hosts on that transport.
- Generate shell completions from the CLI source of truth; do not hand-maintain
  separate completion logic.
