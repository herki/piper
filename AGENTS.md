# Agents Guide

Instructions for AI coding agents working on the Piper codebase.

## Project Overview

Piper is a CLI workflow engine written in Go. It executes YAML-defined flows that chain steps across connectors (HTTP, shell, log). The binary is called `flow`.

## Build & Test

```bash
go build -o flow .
go test ./...
```

Tests must pass before any commit. There are no external test dependencies.

## Architecture

```
main.go                     → entry point, calls cmd.Execute()
cmd/                        → cobra CLI commands (run, list, describe, validate, serve, version)
cmd/registry.go             → registers built-in connectors
internal/types/types.go     → all shared types (FlowDef, StepDef, StepResult, FlowResult)
internal/loader/loader.go   → YAML parsing, loads flows from directory
internal/engine/engine.go   → core execution: runs steps sequentially, handles on_error
internal/engine/context.go  → ${{ }} variable resolution, pipe functions
internal/engine/validator.go→ pre-run validation (connectors exist, step refs valid, etc.)
internal/plugin/interface.go→ Connector interface (Name, Actions, Execute, Validate)
internal/plugin/registry.go → thread-safe connector registry
internal/plugin/builtin/    → http.go, shell.go, log.go, webhook.go
internal/server/webhook.go  → HTTP server mapping trigger paths to flows
flows/                      → example YAML flow definitions
```

## Key Patterns

- **All types live in `internal/types/types.go`** — never define domain types elsewhere.
- **Connectors implement `plugin.Connector`** — see `internal/plugin/interface.go`. Each connector has Name(), Actions(), Execute(), Validate().
- **New connectors** go in `internal/plugin/builtin/` and must be registered in `cmd/registry.go`.
- **Variable expressions** use `${{ }}` syntax, resolved in `internal/engine/context.go`. Supported roots: `input`, `steps`, `env`. Pipe functions: `slugify`, `upper`, `lower`, `trim`.
- **Missing optional input fields resolve to empty string** — flows use shell defaults (`${VAR:-default}`) for optional parameters.
- **Error handling** is per-step via `on_error`: `abort` (default, stops flow), `continue` (marks failed, keeps going), `skip` (ignores).
- **CLI output** supports `--output json` (machine-readable) and `--output table` (human-readable, default).

## Adding a New Connector

1. Create `internal/plugin/builtin/<name>.go`
2. Implement the `plugin.Connector` interface
3. Register it in `cmd/registry.go` inside `defaultRegistry()`
4. Add tests

## Adding a New CLI Command

1. Create `cmd/<command>.go`
2. Define cobra command, register with `rootCmd.AddCommand()` in `init()`
3. Use `flowsDir` and `outputFormat` from `cmd/root.go` for consistency

## Writing Flow YAML

- `name` and at least one step are required
- Each step needs: `name`, `connector`, `action`, `input`
- `on_error` is optional (defaults to `abort`)
- `trigger` with `type: webhook` and `path` enables webhook serving
- `input.properties` defines the flow's input schema with `type`, `description`, `required`

## Testing Conventions

- Unit tests are colocated: `engine_test.go` next to `engine.go`
- Use `t.TempDir()` for file-based tests
- Connector tests should use the real connector (log, shell are side-effect-free enough)
- Server tests use `httptest.NewRecorder()` — no real HTTP server needed

## Dependencies

Intentionally minimal:
- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML parsing
- Standard library for everything else

Do not add new dependencies without strong justification.
