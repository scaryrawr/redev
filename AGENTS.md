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

- Prefer small packages with explicit dependencies so future `devssh` package
  integration stays testable.
- Do not persist credentials in remote files, shell profiles, command argv, or
  logs. Credential forwarding must be scoped, redacted, and cleaned up.
- `redev ssh` always forwards the active GitHub token. Keep forwarding safe by
  setting `GH_TOKEN` only in the local `devpod ssh` child process and using
  `--send-env GH_TOKEN`; avoid `--set-env` with token values because that puts
  secrets in command argv.
- Generate shell completions from the CLI source of truth; do not hand-maintain
  separate completion logic.
