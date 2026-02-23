# symbiont-ai-todoapp

AI-powered Todo application built with [Symbiont](https://github.com/cleitonmarx/symbiont), Go, React, PostgreSQL, and Pub/Sub.

## Features

- 📝 **Todo Management**: Create, update, delete, filter, sort, and paginate todos
- 🤖 **LLM Chat & Tools**: Streamed AI chat (SSE) with tool-calling for local and external tools
- 📦 **Batch Todo Actions**: Assistant-first bulk operations (`create_todos`, `update_todos`, `update_todos_due_date`, `delete_todos`)
- 🔗 **MCP Gateway Integration**: MCP-based external tools via `docker/mcp-gateway` (default setup includes DuckDuckGo tools)
- 🎛️ **Tool Definition Overrides**: MCP tool descriptions/inputs/hints can be overridden with YAML for tighter model behavior control
- 💬 **Conversation Management**: Conversation history with rename/delete and auto/LLM title generation
- 📌 **Board Summary**: AI-generated board summary from todo domain events
- 🧠 **Chat Summary**: Conversation-aware AI summaries from chat-message events
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
- **PostgreSQL** (`internal/adapters/outbound/postgres`): Primary data store with migrations and vector extension support
- **Vault Provider** (`internal/adapters/outbound/config/vault_provider.go`): Loads secret-backed config values (`DB_USER`, `DB_PASS`)
- **Assistant Client** (`internal/adapters/outbound/modelrunner`): OpenAI/DRM-compatible client for chat, summarization, embeddings, and model listing
- **Assistant Action Registries** (`internal/adapters/outbound/actionregistry`):
  - `local`: Built-in app actions (UI filters, fetch todos, and batch todo mutations)
  - `mcp`: MCP gateway-backed tool registry using `github.com/modelcontextprotocol/go-sdk/mcp`
  - `composite`: Aggregates local + MCP actions and ranks relevance with embeddings
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
- `PUBSUB_PROJECT_ID`, `PUBSUB_EMULATOR_HOST` (for local emulator), `TODO_EVENTS_SUBSCRIPTION_ID`, `CHAT_EVENTS_SUBSCRIPTION_ID`, `CHAT_TITLE_EVENTS_SUBSCRIPTION_ID`
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
