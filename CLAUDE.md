# CLAUDE.md

## Project Overview

Scrollwork is an AI billing risk assessment tool that monitors token usage and evaluates prompt costs in real-time. It uses a Unix socket interface to assess billing risk before AI prompts are executed.

## Development Commands

### Building and Running

```bash
# Build the project
mise run build
# or: go build -o ./scrollwork

# Run the application
mise run start
# or: go run cmd/scrollwork/main.go

# Run with required flags
go run cmd/scrollwork/main.go -model="claude-3-5-sonnet-20241022" -apiKey="sk-ant-..." -adminKey="sk-ant-..."
```

### Testing

```bash
# Run all tests
mise run test
# or: go test -count=1 ./...

# Run tests for specific package
go test ./internal/usage/
go test ./internal/llm/
```

### Docker

```bash
# Build Docker image
mise run docker-build
# or: docker build . -t scrollwork:latest
```

## Architecture Overview

### Core Components

**Agent (`internal/scrollwork/agent.go`)**

- Main orchestrator that manages the Unix socket listener and usage worker
- Handles incoming connections on `/tmp/scrollwork.sock`
- Coordinates risk assessment of prompts using current usage data and configurable thresholds
- Manages lifecycle of worker processes

**Usage Worker (`internal/scrollwork/worker.go`)**

- Background process that periodically fetches organization usage from AI provider APIs
- Runs on configurable intervals (default: 1 minute)
- Sends usage updates to the agent via channels
- Performs health checks on AI provider connections

**LLM Integration (`internal/llm/`)**

- `anthropic.go`: Anthropic API client for token counting and usage reporting
- `model.go`: Model validation and provider detection utilities
- Currently supports Anthropic models only; OpenAI support is planned

**Usage & Risk Assessment (`internal/usage/`)**

- `risk.go`: Risk level calculation based on token thresholds (Low/Medium/High/Unknown)
- `tokens.go`: Token usage tracking and management
- Risk thresholds are configurable via command-line flags

### Style guide

- Comments for public functions but no comments in functions
- `go fmt` handles all code formatting
- External libraries should not be pulled in automatically

### Data Flow

1. Agent starts and initializes usage worker with AI provider client
2. Worker performs health check and fetches initial usage snapshot
3. Worker runs periodic usage sync, sending updates to agent via channels
4. Client connects via socket
5. Agent receives prompt
   a. Anthropic queries use the Messages API. Client must provide valid Message structure.
   b. TODO: OpenAI queries
6. Agent assesses risk level using current usage + token count vs thresholds
7. Agent returns usage stats and risk level to client

### Connection Methods

- UNIX Socket
- TODO: TCP Socket via gRPC and buf

### Command line flags

- `--model`: AI model identifier (required). Multi-model not supported. Currently assuming Anthropic.
- `--apiKey`: Provider API key for non-admin API requests (required)
- `--adminKey`: Provider admin key for Admin API requests (required)
- `--refreshRate`: Usage worker sync interval in minutes (default: 1)

Risk threshold flags exist but are currently hardcoded to 0 in `cmd/scrollwork/main.go:60-62`. These will be configured via flags.

### Testing Patterns

- Use `_test.go` files alongside source files
- Tests use testify/assert for assertions
- Mock external dependencies (AI provider APIs) for unit tests
- Integration tests should test the full agent lifecycle
