# symbiont-ai-todoapp

AI-powered Todo application built with [Symbiont](https://github.com/cleitonmarx/symbiont), Go, React, PostgreSQL, and Pub/Sub.

## Features

- 📝 **Todo Management**: Create, update, delete, filter, and sort todos with pagination
- 🤖 **LLM Chat & Tools**: Streamed AI chat (SSE) with tool-calling for todo operations
- 📌 **Board Summary**: AI-generated board summary from todo domain events
- 🧠 **Chat Summary**: Conversation-aware AI summaries from chat message events
- 🔔 **Event-Driven Workflow**: Outbox + Pub/Sub workers for asynchronous processing
- 🧠 **Vector Search**: PostgreSQL `pgvector` + embeddings for semantic todo search
- 🔌 **Dual APIs**: REST (OpenAPI) + GraphQL for flexible and batch-friendly operations
- 📊 **Observability**: OpenTelemetry + Jaeger + Prometheus + Grafana
- 🎨 **Modern UI**: React webapp embedded in Docker image and available via Vite dev mode

## Architecture

### Components

- **HTTP API** (`internal/adapters/inbound/http`): Serves REST endpoints (`/api/v1/...`) and static web assets
- **GraphQL API** (`internal/adapters/inbound/graphql`): Serves `/v1/query` with batch mutation/query support
- **Message Relay Worker** (`internal/adapters/inbound/workers/message_relay.go`): Publishes persisted outbox events to Pub/Sub
- **Todo Event Subscriber** (`internal/adapters/inbound/workers/todo_event_subscriber.go`): Batches todo events and triggers board-summary generation
- **Chat Event Subscriber** (`internal/adapters/inbound/workers/chat_event_subscriber.go`): Batches chat events by `ConversationID` and triggers one chat-summary generation per conversation window
- **PostgreSQL** (`internal/adapters/outbound/postgres`): Primary data store with migrations and vector extension support
- **Vault Provider** (`internal/adapters/outbound/config/vault_provider.go`): Loads secret-backed config values (`DB_USER`, `DB_PASS`)
- **LLM Client** (`internal/adapters/outbound/modelrunner`): OpenAI-compatible client for chat, summarization, and embeddings
- **Telemetry** (`internal/telemetry`): Traces and metrics instrumentation for HTTP, DB, Pub/Sub, and use cases

### 🔗 Generated Introspection Graph (Initializers, Runners, Dependencies, Configs)

- Interactive graph endpoint: `http://localhost:8080/introspect`
- Full generated Mermaid graph: `docs/introspection.md`

## Chat Summary Batching Behavior

`ChatEventSubscriber` is designed to reduce over-triggering and LLM cost:

- Decodes events in batch windows
- Ignores unrelated event types (Ack)
- Nacks invalid payloads
- Coalesces by `ConversationID` and generates one summary per conversation using the latest event
- Acks or Nacks grouped messages per conversation result

Tunable settings:

- `CHAT_SUMMARY_BATCH_INTERVAL` (default `3s`)
- `CHAT_SUMMARY_BATCH_SIZE` (default `50`)

## Quick Start (Docker Compose)

Prerequisites:

- Docker + Docker Compose + Docker Model Runner
- LLM endpoint compatible with this project (the provided compose file uses Docker Model Runner settings)

Run everything:

```bash
docker compose up --build
```

Useful local URLs:

- App + REST API: `http://localhost:8080`
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

Run backend:

```bash
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

Then open `http://localhost:5173`.

## Testing

Run unit/integration-lite package tests:

```bash
go test ./...
```

Run integration tests (requires Docker and local dependencies):

```bash
go test -tags integration ./tests/integration/...
```

## Key Configuration

Required or commonly tuned variables:

- `HTTP_PORT` (default: `8080`)
- `GRAPHQL_SERVER_PORT` (default: `8085`)
- `DB_HOST`, `DB_PORT` (default: `5432`), `DB_NAME`
- `DB_USER`, `DB_PASS` (usually sourced from Vault)
- `VAULT_ADDR`, `VAULT_TOKEN`, `VAULT_MOUNT_PATH`, `VAULT_SECRET_PATH`
- `PUBSUB_PROJECT_ID`, `TODO_EVENTS_SUBSCRIPTION_ID`, `CHAT_EVENTS_SUBSCRIPTION_ID`
- `LLM_MODEL_HOST`, `LLM_SUMMARY_MODEL`, `LLM_CHAT_SUMMARY_MODEL`, `LLM_EMBEDDING_MODEL`
- `LLM_MAX_TOOL_CYCLES` (default: `50`)
- `FETCH_OUTBOX_INTERVAL` (default: `500ms`)
- `SUMMARY_BATCH_INTERVAL` (default: `3s`), `SUMMARY_BATCH_SIZE` (default: `20`)
- `CHAT_SUMMARY_BATCH_INTERVAL` (default: `3s`), `CHAT_SUMMARY_BATCH_SIZE` (default: `50`)
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`, `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT`

## API Specs

- OpenAPI: `api/openapi/openapi.yml`
- GraphQL schema: `api/graphql/schema.graphql`

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
