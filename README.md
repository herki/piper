# Piper

[![CI](https://github.com/herki/piper/actions/workflows/ci.yml/badge.svg)](https://github.com/herki/piper/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI-first workflow engine where AI agents (or humans) define multi-step automation flows in YAML. A single command or webhook triggers a chain of actions across services without the caller needing to know each service's API.

```
flow run health-check --input '{"target": "https://api.example.com"}'
```

## Why Piper

An AI agent only needs to know: "run the onboard-client flow with a name and email." Piper handles the rest -- calling APIs, running scripts, chaining outputs between steps, and returning structured JSON.

- **YAML-defined flows** -- version-controlled, human-readable, AI-discoverable
- **CLI-first** -- `flow run`, `flow list`, `flow describe`, structured JSON output
- **Webhook server** -- `flow serve` maps HTTP endpoints to flows
- **Step chaining** -- each step's output is available to subsequent steps via `${{ steps.name.output.field }}`
- **Error handling** -- per-step `on_error` policy (abort, continue, skip)
- **Minimal dependencies** -- Go standard library + cobra + yaml.v3, nothing else

## Install

```bash
go build -o flow .
```

## Quick Start

```bash
# List available flows
flow list

# Describe a flow (see input schema, steps, connectors)
flow describe health-check

# Dry run (resolve variables, validate, don't execute)
flow run demo --input '{"url": "https://httpbin.org/get"}' --dry-run

# Run a flow
flow run demo --input '{"url": "https://httpbin.org/get"}'

# Start webhook server
flow serve --port 8080

# Trigger via HTTP
curl -X POST http://localhost:8080/demo \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://httpbin.org/get"}'
```

## CLI Commands

| Command | Description |
|---|---|
| `flow run <name> --input '{}'` | Execute a flow with JSON input |
| `flow run <name> --dry-run` | Show what would execute without running |
| `flow list` | List all available flows |
| `flow describe <name>` | Show flow details: input schema, steps, connectors |
| `flow validate <file>` | Validate a YAML flow file |
| `flow serve --port 8080` | Start webhook server |
| `flow version` | Print version |

All commands support `--output json` for machine-readable output.

## Writing Flows

Flows are YAML files in the `flows/` directory:

```yaml
name: health-check
version: "1.0"
description: "Check health of a service and notify Slack"

input:
  properties:
    target:
      type: string
      description: "URL to check"
      required: true
    slack_webhook:
      type: string
      description: "Slack webhook URL"
      required: false

trigger:
  type: webhook
  path: /health-check

steps:
  - name: check-endpoint
    connector: http
    action: request
    input:
      url: "${{ input.target }}"
      method: GET
    on_error: continue

  - name: measure-latency
    connector: shell
    action: run
    input:
      command: "curl -o /dev/null -s -w '%{time_total}' ${{ input.target }}"
    on_error: continue

  - name: notify
    connector: http
    action: request
    input:
      url: "${{ input.slack_webhook }}"
      method: POST
      body:
        text: "Health: ${{ input.target }} returned ${{ steps.check-endpoint.output.status_code }} in ${{ steps.measure-latency.output.stdout }}s"
    on_error: skip
```

### Variable Expressions

Reference inputs, previous step outputs, and environment variables:

| Expression | Description |
|---|---|
| `${{ input.name }}` | Flow input value |
| `${{ steps.step-name.output.field }}` | Output from a previous step |
| `${{ steps.step-name.status }}` | Status of a previous step |
| `${{ env.API_KEY }}` | Environment variable |

Pipe functions for transformations:

| Pipe | Example | Result |
|---|---|---|
| `slugify` | `${{ input.name \| slugify }}` | `acme-corp` |
| `upper` | `${{ input.name \| upper }}` | `ACME CORP` |
| `lower` | `${{ input.name \| lower }}` | `acme corp` |
| `trim` | `${{ input.name \| trim }}` | trimmed whitespace |

### Error Handling

Each step has an `on_error` policy:

- **`abort`** (default) -- stop the flow, mark as failed
- **`continue`** -- mark step as failed, keep running subsequent steps
- **`skip`** -- ignore the error entirely

## Built-in Connectors

### `http` -- HTTP Requests

```yaml
- name: call-api
  connector: http
  action: request
  input:
    url: "https://api.example.com/data"
    method: POST
    headers:
      Authorization: "Bearer ${{ env.API_TOKEN }}"
    body:
      key: "value"
```

Output: `status_code`, `body`, `headers`

### `shell` -- Shell Commands

```yaml
- name: run-script
  connector: shell
  action: run
  input:
    command: "echo hello world"
    dir: "/tmp"  # optional working directory
```

Output: `stdout`, `stderr`, `exit_code`

### `log` -- Debug Output

```yaml
- name: debug
  connector: log
  action: print
  input:
    message: "Current status: ${{ steps.previous.output.status_code }}"
```

Output: `message`

## Execution Output

Every flow returns structured JSON:

```json
{
  "flow": "health-check",
  "status": "success",
  "started_at": "2026-02-27T10:00:00Z",
  "completed_at": "2026-02-27T10:00:01Z",
  "input": {"target": "https://httpbin.org/get"},
  "steps": [
    {
      "name": "check-endpoint",
      "connector": "http",
      "action": "request",
      "status": "success",
      "output": {"status_code": 200, "body": {...}, "headers": {...}},
      "duration_ms": 450
    }
  ]
}
```

## Example Flows

| Flow | Description |
|---|---|
| `demo` | Fetch a URL and log the result |
| `shell-demo` | Run shell commands and chain outputs |
| `health-check` | Probe endpoint, measure response time, notify Slack |
| `git-deploy` | Git pull, build, restart systemd service |
| `github-issue-to-slack` | Fetch GitHub issue via API, post to Slack |
| `system-report` | Collect disk/memory/CPU/uptime, POST to webhook |
| `data-pipeline` | Fetch JSON, transform with jq, forward to destination |
| `onboard-client` | Multi-step client onboarding (API calls + notifications) |

## Webhook Server

`flow serve` starts an HTTP server that maps trigger paths to flows:

```bash
flow serve --port 8080

# Output:
# Starting webhook server on :8080
# Loaded 8 flow(s)
#   POST /demo -> demo
#   POST /health-check -> health-check
#   POST /deploy -> git-deploy
#   ...
```

Endpoints:

- `GET /health` -- health check (`{"status": "ok"}`)
- `GET /flows` -- list available flows with input schemas
- `POST /<trigger-path>` -- trigger a flow

## Project Structure

```
piper/
├── cmd/                        # CLI commands (cobra)
│   ├── root.go                 # Flags: --flows-dir, --output
│   ├── run.go                  # flow run
│   ├── list.go                 # flow list
│   ├── describe.go             # flow describe
│   ├── validate.go             # flow validate
│   ├── serve.go                # flow serve
│   └── version.go              # flow version
├── internal/
│   ├── engine/                 # Execution engine
│   │   ├── engine.go           # Sequential step execution
│   │   ├── context.go          # Variable resolution (${{ }})
│   │   └── validator.go        # Pre-run validation
│   ├── loader/                 # YAML parser
│   │   └── loader.go
│   ├── plugin/                 # Connector system
│   │   ├── interface.go        # Connector interface
│   │   ├── registry.go         # Plugin registry
│   │   └── builtin/            # http, shell, log, webhook
│   ├── server/                 # Webhook HTTP server
│   │   └── webhook.go
│   └── types/                  # Shared types
│       └── types.go
├── flows/                      # Example flow definitions
├── main.go
└── go.mod
```

## Testing

```bash
go test ./...
```

## Roadmap

- External plugin system (subprocess-based, JSON over stdin/stdout)
- Parallel step execution
- Conditional steps (`when:` expressions)
- Real service connectors (GitHub, Slack, Google Calendar) as separate plugins
- Flow composition (one flow calls another)
- Secret management
- MCP compatibility layer
