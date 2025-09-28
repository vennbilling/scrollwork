# CLAUDE.md

## Project Overview

Scrollwork is an AI billing risk assessment tool that monitors token usage and evaluates prompt costs in real-time. It uses a Unix socket interface to assess billing risk before AI prompts are executed. Scrollwork can run either as a sidecar or a standalone service.

We support Anthropic and Open AI Models.

### Architecture

Scrollwork has two major components: the agent and the worker.

#### Agent

The agent is responsible for processing prompt requests. Using a UNIX socket, messages are sent to the agent over the UNIX Socket. We use `buf` to manage message shape and serde.

The agent takes a prompt (in the form a list of messages) and counts the number of input tokens in the prompt based on the select LLM model. We do not calculate the number of output tokens as that is only known by the LLM providers.

Once a token count is returned, we calculate the percentage increase in total input tokens used. Using three user-configurable risk levels and the percentage, we return one of the following values:

- low
- medium
- high

The clients then determine what to do based on this risk using business logic in their own codebases. The agent, itself, does not execute the prompt against the LLM.

#### Worker

The worker (known internally as the usage worker) is responsible for fetching and tracking the current input token usage for a given organization. This fetching is configurable by the user to avoid rate limiting.

#### Startup

Starting scrollwork, be it via `go run` or the container itself is done in the following steps:

1. Scrollwork validates all flags and config values. If they are invalid, we fail to start.
2. The agent is initialized with the appropriate configuration. As part of this initializing, the worker is also initialized
3. The function `func (a *Agent) Start()` is called. This function is more of "setup for running" function that does the following:
   a. It first checks that the provided LLM model is supported.
   b. It then configures the appropriate LLM API client with the provided API key
   c. We then attempt to start the worker in a which is also a "setup up for running" function". The agent listens on a "worker ready" channel for worker to indicate it is ready. We use a `ctx.WithTimeout` to manage this startup.
   d. The worker does the following before sending `true` on the "worker ready" channel.
   1. Performs a healthcheck against the LLM API to make the client is valid
   2. Fetches the current input token usage for the organization
   3. Stores the current input token usage, in memory,
   4. `true` is sent on the "worker ready" channel.
4. Once the worker is ready, the agent is also ready
5. We then call `func (a *Agent) Run` to run the agent. This also starts up the worker in a separate goroutine to begin periodically fetching organization usage.
6. Once the agent is running, messages can be sent to the unix socket `/tmp/scrollwork.sock`

Shutting scrollwork down follows a similar process but in reverse. The worker gets shut down first, followed by the agent.

There is no support for soft restarts.

#### Command line flags

- `--model`: AI model identifier (required). Multi-model not supported at this time. Currently assuming Anthropic.
- `--apiKey`: Provider API key for non-admin API requests (required)
- `--adminKey`: Provider admin key for Admin API requests (required). This key should be present for Anthropic models given their API permission structure.
- `--refreshRate`: Usage worker sync interval in minutes (default: 1)
- `--lowRiskThreshold`: Token percentage threshold for low risk level (default: 50)
- `--mediumRiskThreshold`: Token percentage threshold for medium risk level (default: 75)
- `--highRiskThreshold`: Token percentage threshold for high risk level (default: 100)

Ideally, this could also be configured via YAML.

#### Multi-model support

We should also consider what a multi-model world. In this thinking, we would count the tokens of the prompt with _all_ LLM providers. We would then return a risk assesment for each model. The only models we would care about are the ones configured.

For example, we could support the flags `--model=claude-3-5-sonnet-20241022` and `--model=gpt-4o`. That would make all risk assesments check against Anthropic and OpenAI.

The challenges:

- Managing multiple API Keys
- Initializing API clients based on the models provided
- Performance impact, specifically rate limiting, of counting tokens for two providers

We should be designing with the mindset of multi-model support but we will only support a list of "1" to start.

## Architecture Overview

### Project Structure

```
.
├── api
│   └── proto
│       └── scrollwork
│           └── v1
├── cmd
│   └── scrollwork
└── internal
    ├── llm
    ├── scrollwork
    └── usage
```

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

### Client Data Flow

1. Client connects via socket be it UNIX or Network
2. Agent receives prompt which is a list of messages defined by our protobufs. Scrollwork deserializes those messages using `buf` tooling + golang serde.
3. Based on the configured model, the Agent counts the number of input tokens.
4. Agent assesses risk level using current usage + token count vs thresholds
5. Agent returns usage stats and risk level to client defined by our protobufs.

### Style guide

All contributions to this codebase are expected to follow these guidelines:

#### Coding style and format

- Changes should be small and focused. We don't want to see a giant PR with 1000+ lines of code.

```
GOOD PR: "This PR implements a way to consume $API_NAME from the LLM Providers"
BAD PR: "This PR implements a way to consume $API_NAME, updates the agent, and worker and does everything"
BAD PR: "Implement $FEATURE"
```

- Focus primarily on the code change / task at hand and less about what the next coding steps could be. Adding a new struct field, for example, doesn't imply all (if any) functions we will need
- Functions that are public should be commented
- When in doubt, start with small changes that don't touch the `scrollwork` package. Once verified, start a new set of changes that integrate them into the agent or worker.
- We should avoid rewriting things when things don't make sense. Instead, think about the flow of data and ask if there is a better way
- Bonus fixes are tempting but don't sneak them in unless they are very small. New PRs are easy to create.
- Run relevant tests after making changes

#### Logging rules

- Never log sensitive information
- `log.Printf` should be used to log meaningful operations performed by the agent or worker
- `log.Fatal` should only be used the `main` package and only be called when an `error` is returned.

#### Go Specific

- `go fmt` handles all code formatting
- Comments for public functions but no comments in functions
- External libraries should not be pulled in automatically
- `fmt.Errorf` should be used to return an error when a function returns an error
- `context.Context` should be passed in when appropriate especially when making API requests. The context we share is defined in `main.go`
- Avoid pointers where we can to avoid heap allocations

#### Testing Patterns

- Use `_test.go` files alongside source files
- Tests use testify/assert for assertions
- Mock external dependencies (AI provider APIs) for unit tests
- Integration tests should test the full agent lifecycle
- Always run go fmt when touching go files
- When a test fails, call it out but don't spend too much time automatically investigating. Dont attempt to fix unrelated changes to make the test pass either

#### Testing Strategy

- Code changes should always run relevant unit tests.
- Prior to pushing to origin, run integration and smoke tests
- unit: Run all unit tests with `go test`
- integration: Verify the agent starts after touching anything outside of cmd. Go through all the flag scenarios
- smoke: Run scrollwork with valid flag values and wait at least a minute to verify "fetching usage" logging appears. There is a `mise` task to help with this. We should also connect to the unix socket and verify some response comes back.
- e2e: Always verify the Docker image builds and it runs and shows the same output as the smoke test. You should tag images with scrollwork:latest

## Development Commands

### Building and Running

```bash
# Build the project
mise run build
# or: go build -o ./bin/scrollwork

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
