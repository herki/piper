# Agents Guide

Instructions for AI coding agents working on the Piper codebase.

## Project Overview

Piper is a CLI workflow engine written in Go. It executes YAML-defined flows that chain steps across connectors (HTTP, shell, log, flow). The binary is called `flow`. It supports conditional steps, parallel execution, retry with backoff, flow composition, secret management, external plugins, and MCP compatibility.

## Build & Test

```bash
go build -o flow .
go test ./...
```

Tests must pass before any commit. There are no external test dependencies.

## Architecture

```
main.go                        → entry point, calls cmd.Execute()
cmd/                           → cobra CLI commands (run, list, describe, validate, serve, mcp, version)
cmd/registry.go                → registers built-in connectors + loads external plugins
internal/types/types.go        → all shared types (FlowDef, StepDef, StepResult, FlowResult, RetryConfig)
internal/loader/loader.go      → YAML parsing, loads flows recursively from directory
internal/engine/engine.go      → core execution: sequential, parallel, retry, flow composition, conditionals
internal/engine/context.go     → ${{ }} variable resolution, pipe functions, condition evaluation, secrets
internal/engine/validator.go   → pre-run validation (connectors, step refs, parallel, flow composition)
internal/engine/secrets.go     → .env file parser for secret management
internal/plugin/interface.go   → Connector interface (Name, Actions, Execute, Validate)
internal/plugin/registry.go    → thread-safe connector registry
internal/plugin/external.go    → external plugin loader (exec-based, JSON over stdin/stdout)
internal/plugin/builtin/       → http.go, shell.go, log.go, webhook.go
internal/server/webhook.go     → HTTP server mapping trigger paths to flows
internal/server/mcp.go         → MCP JSON-RPC server over stdin/stdout
flows/                         → example YAML flow definitions
flows/skills/                  → AI agent skill flows
```

## Key Patterns

- **All types live in `internal/types/types.go`** — never define domain types elsewhere.
- **Connectors implement `plugin.Connector`** — see `internal/plugin/interface.go`. Each connector has Name(), Actions(), Execute(), Validate().
- **New connectors** go in `internal/plugin/builtin/` and must be registered in `cmd/registry.go`.
- **Variable expressions** use `${{ }}` syntax, resolved in `internal/engine/context.go`. Supported roots: `input`, `steps`, `env`, `secret`. Pipe functions: `slugify`, `upper`, `lower`, `trim`.
- **Missing optional input fields resolve to empty string** — flows use shell defaults (`${VAR:-default}`) for optional parameters.
- **Error handling** is per-step via `on_error`: `abort` (default, stops flow), `continue` (marks failed, keeps going), `skip` (ignores), `retry` (retries with exponential backoff, requires `retry:` config).
- **Conditional steps** use `when:` field with `${{ }}` expressions. Supports `==`, `!=`, `>`, `<`, `>=`, `<=` operators and truthy evaluation. Evaluated in `context.go` `EvaluateCondition()`.
- **Parallel execution** uses `parallel:` field on a step containing a list of sub-steps. Executed via goroutines with `sync.WaitGroup` in `engine.go` `executeParallel()`.
- **Retry** uses `on_error: retry` + `retry:` config with `max_retries` and `backoff_seconds`. Exponential backoff (backoff * 2^attempt). Implemented in `engine.go` `executeStepWithRetry()`.
- **Flow composition** uses `connector: flow` + `flow:` field. Child flows are loaded via `Engine.FlowLoader` function. Implemented in `engine.go` `executeFlowStep()`.
- **Secret management** loads `.env` files via `secrets.go` `LoadSecrets()`. Secrets are passed to `RunWithSecrets()` and resolved via `${{ secret.KEY }}` in context.
- **External plugins** are executables that support `--describe` (returns JSON metadata) and receive JSON on stdin / write JSON to stdout. Loaded by `external.go` `LoadExternalPlugins()`.
- **MCP server** implements JSON-RPC over stdin/stdout in `internal/server/mcp.go`. Supports `initialize`, `tools/list`, `tools/call`.
- **CLI output** supports `--output json` (machine-readable) and `--output table` (human-readable, default).
- **Flow loading is recursive** — `loader.go` uses `filepath.WalkDir` to discover flows in subdirectories.

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
- `on_error` is optional (defaults to `abort`); values: `abort`, `continue`, `skip`, `retry`
- `when` is optional; evaluates `${{ }}` expressions to decide if step runs
- `retry` is optional; requires `max_retries` and `backoff_seconds`
- `parallel` contains a list of sub-steps that run concurrently
- `flow` specifies a child flow name for flow composition (used with `connector: flow`)
- `trigger` with `type: webhook` and `path` enables webhook serving
- `input.properties` defines the flow's input schema with `type`, `description`, `required`

## Testing Conventions

- Unit tests are colocated: `engine_test.go` next to `engine.go`
- Use `t.TempDir()` for file-based tests
- Connector tests should use the real connector (log, shell are side-effect-free enough)
- Server tests use `httptest.NewRecorder()` — no real HTTP server needed
- Test files: `context_test.go` (variable resolution, conditions, secrets), `engine_test.go` (execution, parallel, retry, composition), `secrets_test.go` (.env parsing), `validator_test.go` (validation), `loader_test.go` (YAML loading), `webhook_test.go` (HTTP server)

## Releasing

Tag a version to trigger the GoReleaser pipeline:

```bash
git tag v0.3.0
git push origin v0.3.0
```

Builds binaries for linux/darwin/windows (amd64/arm64). Version is injected via ldflags into `cmd.Version`.

## Dependencies

Intentionally minimal:
- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML parsing
- Standard library for everything else

Do not add new dependencies without strong justification.
