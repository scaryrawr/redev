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
  logs. Credential forwarding must be explicit, scoped, redacted, and cleaned up.
- Generate shell completions from the CLI source of truth; do not hand-maintain
  separate completion logic.
