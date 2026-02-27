# symbiont-ai-todoapp

AI-powered Todo application built with [Symbiont](https://github.com/cleitonmarx/symbiont), Go, React, PostgreSQL, and Pub/Sub.

## Features

- 📝 **Todo Management**: Create, update, delete, filter, sort, and paginate todos
- 🤖 **LLM Chat & Actions/Tools**: Streamed AI chat (SSE) with action/tool-calling for local and external actions/tools
- 🧩 **Skill-Based Action/Tool Routing**: Markdown runbooks (`skills/*.md`) used to decide which actions/tools are injected each turn
- ✅ **Action/Tools Approval Flow**: Human approval for sensitive/destructive action/tool execution (local actions and MCP tools)
- 🔗 **MCP Gateway Integration**: MCP-based external tools via `docker/mcp-gateway` (default setup includes DuckDuckGo tools)
- 💬 **Conversation Management**: Conversation history with rename/delete and auto/LLM title generation
- 📌 **Board Summary**: AI-generated board summary from todo domain events
- 🧠 **Conversation/Context Compression**: Conversation-aware AI summaries from chat-message events
- 🔔 **Event-Driven Workflow**: Outbox + Pub/Sub workers for asynchronous processing
- 🧠 **Vector Search**: PostgreSQL `pgvector` + embeddings for semantic todo search
- 🔌 **Dual APIs**: REST (OpenAPI) + GraphQL
- 📊 **Observability**: OpenTelemetry + Jaeger + Prometheus + Grafana
- 🎨 **Modern UI**: React webapp embedded in Docker image and available via Vite dev mode

## Architecture

### Components

- **HTTP API** (`internal/adapters/inbound/http`): Serves REST endpoints and static web assets on `HTTP_PORT` (default `8080`)
- **GraphQL API** (`internal/adapters/inbound/graphql`): Serves `/v1/query` and GraphQL playground (`/`) on `GRAPHQL_SERVER_PORT` (default `8085`)
- **Message Relay Worker** (`internal/adapters/inbound/workers/message_relay.go`): Publishes persisted outbox events to Pub/Sub
- **Board Summary Worker** (`internal/adapters/inbound/workers/board_summary_generator.go`): Batches todo events and triggers board-summary generation
- **Chat Summary Worker** (`internal/adapters/inbound/workers/chat_summary_generator.go`): Batches chat events by `ConversationID` and triggers one chat-summary generation per conversation window
- **Conversation Title Worker** (`internal/adapters/inbound/workers/conversation_title_generator.go`): Batches chat events by `ConversationID` and updates titles asynchronously
- **Action Approval Dispatcher Worker** (`internal/adapters/inbound/workers/action_approval_dispatcher.go`): Consumes approval decisions from Pub/Sub and forwards them to the in-memory action approval dispatcher, using a server-scoped subscription suffix for horizontal distribution
- **PostgreSQL** (`internal/adapters/outbound/postgres`): Primary data store with migrations and vector extension support
- **Vault Provider** (`internal/adapters/outbound/config/vault_provider.go`): Loads secret-backed config values (`DB_USER`, `DB_PASS`)
- **Assistant Client** (`internal/adapters/outbound/modelrunner`): OpenAI/DRM-compatible client for chat, summarization, embeddings, and model listing
- **Assistant Action/Tool Registries** (`internal/adapters/outbound/actionregistry`):
  - `local`: Built-in app actions (UI filters, fetch todos, and batch todo mutations)
  - `mcp`: MCP gateway-backed action/tool registry using `github.com/modelcontextprotocol/go-sdk/mcp`
  - `composite`: Aggregates local + MCP actions/tools
- **Assistant Skill Registry** (`internal/adapters/outbound/skillregistry`): Loads markdown skills and selects skills using turn context (current input, recent user inputs, and optional conversation summary)
- **Telemetry** (`internal/telemetry`): Traces and metrics instrumentation for HTTP, DB, Pub/Sub, and use cases

### Generated Introspection Graph

- Interactive graph endpoint: `http://localhost:8080/introspect`
- Full generated Mermaid graph: `docs/introspection.md`

## Async Batching Behavior

### ChatSummaryGenerator

- Decodes events in batch windows
- Ignores unrelated event types (Ack)
- Nacks invalid payloads
- Coalesces by `ConversationID` and generates one summary per conversation using the latest event
- Acks or Nacks grouped messages per conversation result

Tunable settings:

- `CHAT_SUMMARY_BATCH_INTERVAL` (default `3s`)
- `CHAT_SUMMARY_BATCH_SIZE` (default `50`)

### ConversationTitleGenerator

- Uses the same batch/coalescing strategy as chat summaries
- Generates/updates one title per conversation using the latest event in the batch

Tunable settings:

- `CHAT_TITLE_BATCH_INTERVAL` (default `3s`)
- `CHAT_TITLE_BATCH_SIZE` (default `50`)

## API Overview

REST endpoints are primarily under `/api/v1/...`.
GraphQL currently exposes todo operations (`listTodos`, `updateTodo`, `deleteTodo`) on `/v1/query`.

- OpenAPI spec: `api/openapi/openapi.yml`
- GraphQL schema: `api/graphql/schema.graphql`

## Action Approval Flow

Action Approval adds a human-in-the-loop safety step before sensitive actions are executed.

When an action is marked as requiring approval:

- The assistant pauses before running that action.
- The user sees a clear prompt with what is about to happen.
- The user can approve or reject (optionally with a reason).
- If approved, the assistant proceeds.
- If rejected or timed out, the action is not executed.

This is useful for destructive operations (for example, deleting todos) and external/network actions where explicit user confirmation is preferred.

### Configuring approval

- You can enable approval per action/tool.
- You can customize the prompt title/description, preview fields, and timeout.
- Local actions location: `internal/adapters/outbound/actionregistry/local/actions/`
- MCP tool YAML location: `internal/adapters/outbound/actionregistry/mcp/tool_overrides.yaml`

Example:

```yaml
tools:
  - name: fetch_content
    approval:
      required: true
      title: Confirm URL content fetch
      description: This action fetches content from an external URL. Please confirm before proceeding.
      preview_fields:
        - url
      timeout: 90s
```

## Skill-Based Routing

The assistant can use skills as lightweight runbooks to improve action/tool selection.

- Skill files live in: `internal/adapters/outbound/skillregistry/skills/*.md`
- Each skill has YAML frontmatter (`name`, `use_when`, `avoid_when`, `priority`, `tags`, `tools`) plus workflow instructions in markdown body
- At runtime, selected skills are converted into:
  - Action/tool allowlist for the turn (`tools` field)
  - System guidance prompt for action/tool workflow

Current selector uses weighted context:

- Current user input (highest weight)
- Recent user inputs
- Conversation summary (optional, lower weight)

## Runtime Profiles and Minimum Machine

You can run this app in two main modes:

### 1) Local models (Docker Model Runner / compatible local endpoint)

Recommended hardware profile for a smooth local experience:

- CPU: 12 high-performance cores (or equivalent compute throughput)
- GPU: 30+ cores
- Memory: 32 GB unified/system RAM
- Storage: 20+ GB free for models, caches, and containers

### 2) OpenAI API (lighter local machine requirements)

When using remote OpenAI models, local requirements are mostly for app services:

- CPU: 4 vCPU
- RAM: 8 GB
- Disk: 10+ GB free

Use this environment configuration:

```yaml
- LLM_MODEL_HOST=https://api.openai.com
- LLM_API_KEY=$(OPENAI_API_KEY)
- LLM_SUMMARY_MODEL=gpt-4.1-nano-2025-04-14
- LLM_CHAT_SUMMARY_MODEL=gpt-4.1-nano-2025-04-14
- LLM_CHAT_TITLE_MODEL=gpt-4.1-nano-2025-04-14
```

### Embedding Model

`embeddinggemma:300M-Q8_0` is highly recommended for this project due to its speed/capacity tradeoff. You can still use another embedding model if needed by updating `LLM_EMBEDDING_MODEL`.


## Quick Start (Docker Compose)

Prerequisites:

- Docker + Docker Compose
- Docker Model Runner (or compatible model endpoint) for `qwen3` and `embeddinggemma`

Run everything:

```bash
docker compose up --build
```

Useful local URLs:

- App + REST API + embedded UI: `http://localhost:8080`
- GraphQL playground: `http://localhost:8085`
- App dependency graph introspection: `http://localhost:8080/introspect`
- Jaeger: `http://localhost:16686`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (admin/admin)

Stop:

```bash
docker compose down
```

## Local Development

### Backend + dependencies

Start dependency stack:

```bash
docker compose -f docker-compose.deps.yml up -d
```

Run backend (local binary against local deps):

```bash
VAULT_ADDR=http://localhost:8200 \
VAULT_TOKEN=root-token \
VAULT_MOUNT_PATH=secret \
VAULT_SECRET_PATH=todoapp \
DB_HOST=localhost \
DB_PORT=5432 \
DB_NAME=todoappdb \
PUBSUB_EMULATOR_HOST=localhost:8681 \
PUBSUB_PROJECT_ID=local-dev \
TODO_EVENTS_SUBSCRIPTION_ID=todo_summary_generator \
CHAT_EVENTS_SUBSCRIPTION_ID=chat_message_summary_generator \
CHAT_TITLE_EVENTS_SUBSCRIPTION_ID=chat_message_title_generator \
ACTION_APPROVAL_EVENTS_SUBSCRIPTION_ID=action_approval_dispatcher \
LLM_MODEL_HOST=http://localhost:12434 \
LLM_SUMMARY_MODEL=qwen3:14B-Q6_K \
LLM_CHAT_SUMMARY_MODEL=qwen3:14B-Q6_K \
LLM_CHAT_TITLE_MODEL=qwen3:14B-Q6_K \
LLM_EMBEDDING_MODEL=embeddinggemma:300M-Q8_0 \
MCP_GATEWAY_ENDPOINT=http://localhost:8811 \
go run ./cmd/todoapp
```

### Web app in Vite dev mode

```bash
cd webapp
npm install
VITE_API_BASE_URL=http://localhost:8080 \
VITE_GRAPHQL_ENDPOINT=http://localhost:8085/v1/query \
npm run dev
```

Open `http://localhost:5173`.

## Testing

Run package tests:

```bash
go test ./...
```

Run integration tests:

```bash
go test -tags=integration -v -timeout 30m ./tests/integration/...
```

## Key Configuration

Required or commonly tuned variables:

- `HTTP_PORT` (default: `8080`)
- `GRAPHQL_SERVER_PORT` (default: `8085`)
- `DB_HOST`, `DB_PORT` (default: `5432`), `DB_NAME`
- `DB_USER`, `DB_PASS` (can be sourced from Vault)
- `VAULT_ADDR`, `VAULT_TOKEN`, `VAULT_MOUNT_PATH`, `VAULT_SECRET_PATH`
- `PUBSUB_PROJECT_ID`, `PUBSUB_EMULATOR_HOST` (for local emulator), `TODO_EVENTS_SUBSCRIPTION_ID`, `CHAT_EVENTS_SUBSCRIPTION_ID`, `CHAT_TITLE_EVENTS_SUBSCRIPTION_ID`, `ACTION_APPROVAL_EVENTS_SUBSCRIPTION_ID`
- `LLM_MODEL_HOST`, `LLM_SUMMARY_MODEL`, `LLM_CHAT_SUMMARY_MODEL`, `LLM_CHAT_TITLE_MODEL`, `LLM_EMBEDDING_MODEL`
- `MCP_GATEWAY_ENDPOINT` (e.g. `http://mcp-gateway:8811`)
- `MCP_GATEWAY_API_KEY` (default: `-`)
- `MCP_GATEWAY_API_KEY_HEADER` (default: `Authorization`)
- `MCP_GATEWAY_REQUEST_TIMEOUT` (default: `20s`)
- `MCP_GATEWAY_TOP_ACTIONS_PER_REGISTRY` (default: `2`)
- `LLM_MAX_ACTION_CYCLES` (default: `50`)
- `FETCH_OUTBOX_INTERVAL` (default: `500ms`)
- `SUMMARY_BATCH_INTERVAL` (default: `3s`), `SUMMARY_BATCH_SIZE` (default: `20`)
- `CHAT_SUMMARY_BATCH_INTERVAL` (default: `3s`), `CHAT_SUMMARY_BATCH_SIZE` (default: `50`)
- `CHAT_TITLE_BATCH_INTERVAL` (default: `3s`), `CHAT_TITLE_BATCH_SIZE` (default: `50`)
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`, `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT`

## Codegen

Regenerate server/client code when specs change:

```bash
go generate ./...
```

Regenerate web GraphQL types:

```bash
cd webapp
npm run generategql
```

## License

MIT. See `LICENSE`.
