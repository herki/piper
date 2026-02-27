# Piper

[![CI](https://github.com/herki/piper/actions/workflows/ci.yml/badge.svg)](https://github.com/herki/piper/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/herki/piper)](https://github.com/herki/piper/releases/latest)
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
- **Conditional steps** -- skip or run steps based on `when:` expressions
- **Parallel execution** -- run independent steps concurrently
- **Retry with backoff** -- automatic retries with exponential backoff on failure
- **Flow composition** -- parent flows call child flows as steps
- **Secret management** -- load secrets from `.env` files, reference via `${{ secret.KEY }}`
- **External plugins** -- extend with custom connectors via subprocess protocol
- **MCP compatible** -- expose flows as tools for AI agents via Model Context Protocol
- **Minimal dependencies** -- Go standard library + cobra + yaml.v3, nothing else

## Install

**From releases (recommended):**

```bash
# Linux (amd64)
curl -Lo piper.tar.gz https://github.com/herki/piper/releases/latest/download/piper_0.1.0_linux_amd64.tar.gz
tar xzf piper.tar.gz
chmod +x flow && sudo mv flow /usr/local/bin/

# macOS (Apple Silicon)
curl -Lo piper.tar.gz https://github.com/herki/piper/releases/latest/download/piper_0.1.0_darwin_arm64.tar.gz
tar xzf piper.tar.gz
chmod +x flow && mv flow /usr/local/bin/
```

**From source:**

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

# Run with secrets
flow run my-flow --input '{}' --secrets-file .env

# Start webhook server
flow serve --port 8080

# Trigger via HTTP
curl -X POST http://localhost:8080/demo \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://httpbin.org/get"}'

# Start MCP server (for AI agent tool discovery)
flow mcp
```

## CLI Commands

| Command | Description |
|---|---|
| `flow run <name> --input '{}'` | Execute a flow with JSON input |
| `flow run <name> --dry-run` | Show what would execute without running |
| `flow run <name> --secrets-file .env` | Run with secrets loaded from file |
| `flow list` | List all available flows |
| `flow describe <name>` | Show flow details: input schema, steps, connectors |
| `flow validate <file>` | Validate a YAML flow file |
| `flow serve --port 8080` | Start webhook server |
| `flow mcp` | Start MCP server over stdin/stdout |
| `flow version` | Print version |

All commands support `--output json` for machine-readable output.

## Writing Flows

Flows are YAML files in the `flows/` directory (loaded recursively):

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

Reference inputs, previous step outputs, environment variables, and secrets:

| Expression | Description |
|---|---|
| `${{ input.name }}` | Flow input value |
| `${{ steps.step-name.output.field }}` | Output from a previous step |
| `${{ steps.step-name.status }}` | Status of a previous step |
| `${{ env.API_KEY }}` | Environment variable |
| `${{ secret.API_KEY }}` | Secret from `.env` file |

Pipe functions for transformations:

| Pipe | Example | Result |
|---|---|---|
| `slugify` | `${{ input.name \| slugify }}` | `acme-corp` |
| `upper` | `${{ input.name \| upper }}` | `ACME CORP` |
| `lower` | `${{ input.name \| lower }}` | `acme corp` |
| `trim` | `${{ input.name \| trim }}` | trimmed whitespace |

### Conditional Steps

Run or skip steps based on expressions using the `when:` field:

```yaml
steps:
  - name: deploy-staging
    connector: shell
    action: run
    input:
      command: "deploy.sh staging"
    when: ${{ input.environment == "staging" }}

  - name: deploy-production
    connector: shell
    action: run
    input:
      command: "deploy.sh production"
    when: ${{ input.environment == "production" }}

  - name: notify-on-success
    connector: log
    action: print
    input:
      message: "Deploy succeeded"
    when: ${{ steps.deploy-staging.status == "success" }}
```

Supported operators: `==`, `!=`, `>`, `<`, `>=`, `<=`. Truthy evaluation for simple values (`"true"`, non-empty strings = true; `"false"`, `""` = false).

### Parallel Execution

Run independent steps concurrently using `parallel:`:

```yaml
steps:
  - name: check-all
    parallel:
      - name: check-api
        connector: http
        action: request
        input:
          url: "https://api.example.com/health"
          method: GET

      - name: check-db
        connector: shell
        action: run
        input:
          command: "pg_isready -h localhost"

      - name: check-cache
        connector: shell
        action: run
        input:
          command: "redis-cli ping"

  - name: summary
    connector: log
    action: print
    input:
      message: "API=${{ steps.check-api.output.status_code }}, DB=${{ steps.check-db.output.stdout }}"
```

All parallel steps run simultaneously via goroutines. Their outputs are available to subsequent steps.

### Retry with Backoff

Automatically retry failed steps with exponential backoff:

```yaml
steps:
  - name: call-flaky-api
    connector: http
    action: request
    input:
      url: "https://api.example.com/data"
      method: GET
    on_error: retry
    retry:
      max_retries: 3
      backoff_seconds: 1  # 1s, 2s, 4s (exponential)
```

After exhausting retries, the step falls through to abort behavior. The `retries` count is included in the step result.

### Flow Composition

Call one flow from another using the `flow` connector:

```yaml
# parent-flow.yaml
name: parent-flow
steps:
  - name: setup
    connector: flow
    flow: child-setup
    input:
      project: "client-${{ input.name | slugify }}"

  - name: notify
    connector: log
    action: print
    input:
      message: "Setup status: ${{ steps.setup.output.flow_status }}"
```

The child flow runs with its own input context. The parent receives `flow_status`, `steps` count, and the last step's `stdout`/`stderr` as output.

### Secret Management

Load secrets from `.env` files and reference them in flows:

```bash
# .env file
DB_PASSWORD="super secret"
API_KEY='sk-test-123'
SLACK_TOKEN=xoxb-abc-123
```

```yaml
steps:
  - name: call-api
    connector: http
    action: request
    input:
      url: "https://api.example.com/data"
      method: GET
      headers:
        Authorization: "Bearer ${{ secret.API_KEY }}"
```

```bash
flow run my-flow --input '{}' --secrets-file .env
```

Secrets support single quotes, double quotes, and unquoted values. Comments and blank lines are ignored.

### Error Handling

Each step has an `on_error` policy:

- **`abort`** (default) -- stop the flow, mark as failed
- **`continue`** -- mark step as failed, keep running subsequent steps
- **`skip`** -- ignore the error entirely
- **`retry`** -- retry with backoff (requires `retry:` config)

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
      Authorization: "Bearer ${{ secret.API_TOKEN }}"
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

### `flow` -- Flow Composition

```yaml
- name: run-child
  connector: flow
  flow: child-flow-name
  input:
    param: "value"
```

Output: `flow_status`, `steps`, `stdout`, `stderr`

## External Plugins

Extend Piper with custom connectors as standalone executables. Plugins communicate via JSON over stdin/stdout.

Place executables in the `plugins/` directory (or `--plugins-dir`):

```
plugins/
├── slack
├── github
└── jira
```

Each plugin must support `--describe` to report its name and actions:

```bash
$ ./plugins/slack --describe
{
  "name": "slack",
  "actions": [
    {"name": "send-message", "description": "Send a message to a channel"}
  ]
}
```

When invoked, the plugin receives JSON on stdin and writes JSON to stdout:

```json
// stdin
{"action": "send-message", "input": {"channel": "#general", "text": "Hello"}}

// stdout
{"status": "success", "output": {"ts": "1234567890.123456"}}
```

## MCP Compatibility

Piper can expose all flows as [Model Context Protocol](https://modelcontextprotocol.io/) tools, allowing AI agents to discover and call flows directly:

```bash
flow mcp
```

This starts a JSON-RPC server over stdin/stdout that supports:

- `initialize` -- MCP handshake
- `tools/list` -- returns all flows as tools with JSON Schema input definitions
- `tools/call` -- executes a flow and returns the result

Configure in your AI agent's MCP settings:

```json
{
  "mcpServers": {
    "piper": {
      "command": "flow",
      "args": ["mcp", "--flows-dir", "./flows"]
    }
  }
}
```

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
      "output": {"status_code": 200, "body": {}, "headers": {}},
      "duration_ms": 450,
      "retries": 0
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
| `conditional-deploy` | Deploy with conditional steps based on environment |
| `parallel-health` | Check multiple services in parallel, then summarize |
| `retry-fetch` | Fetch data with retry on failure and exponential backoff |
| `parent-onboard` | Flow composition -- parent calls child-setup flow |

**AI Agent Skills** (`flows/skills/`):

| Flow | Description |
|---|---|
| `summarize-url` | Fetch and summarize a web page |
| `research-topic` | Multi-source research on a topic |
| `analyze-repo` | Analyze a Git repository's structure |
| `json-transform` | Transform JSON data with jq |
| `monitor-endpoint` | Monitor an endpoint with repeated checks |
| `notify-multi` | Send notifications to multiple channels |
| `file-convert` | Convert files between formats |
| `git-repo-stats` | Gather Git repository statistics |

## Webhook Server

`flow serve` starts an HTTP server that maps trigger paths to flows:

```bash
flow serve --port 8080

# Output:
# Starting webhook server on :8080
# Loaded 12 flow(s)
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
│   ├── root.go                 # Flags: --flows-dir, --output, --plugins-dir
│   ├── run.go                  # flow run (--secrets-file)
│   ├── list.go                 # flow list
│   ├── describe.go             # flow describe
│   ├── validate.go             # flow validate
│   ├── serve.go                # flow serve
│   ├── mcp.go                  # flow mcp
│   └── version.go              # flow version
├── internal/
│   ├── engine/                 # Execution engine
│   │   ├── engine.go           # Step execution, parallel, retry, composition
│   │   ├── context.go          # Variable resolution, conditions, secrets
│   │   ├── validator.go        # Pre-run validation
│   │   └── secrets.go          # .env file parser
│   ├── loader/                 # YAML parser (recursive)
│   │   └── loader.go
│   ├── plugin/                 # Connector system
│   │   ├── interface.go        # Connector interface
│   │   ├── registry.go         # Plugin registry
│   │   ├── external.go         # External plugin loader
│   │   └── builtin/            # http, shell, log, webhook
│   ├── server/
│   │   ├── webhook.go          # Webhook HTTP server
│   │   └── mcp.go              # MCP JSON-RPC server
│   └── types/                  # Shared types
│       └── types.go
├── flows/                      # Example flow definitions
│   └── skills/                 # AI agent skill flows
├── plugins/                    # External plugin directory
├── .github/workflows/
│   ├── ci.yml                  # Test, lint, validate
│   └── release.yml             # GoReleaser on tag push
├── .goreleaser.yaml
├── main.go
└── go.mod
```

## Testing

```bash
go test ./...
```

## Releasing

Tag a version to trigger the release pipeline:

```bash
git tag v0.2.0
git push origin v0.2.0
```

GoReleaser builds binaries for linux/darwin/windows (amd64/arm64) and publishes them to [GitHub Releases](https://github.com/herki/piper/releases).

## Roadmap

- Real service connectors (GitHub, Slack, Google Calendar) as separate plugin repos
- Web UI for flow visualization and monitoring
- NATS message bus for distributed execution
- Flow versioning and rollback
