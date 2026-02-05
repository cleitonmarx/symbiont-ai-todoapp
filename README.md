# TodoApp - A Complete Symbiont Example

TodoApp is a comprehensive example application demonstrating the capabilities of Symbiont. It's a full-stack todo application with LLM-powered and event-driven architecture.

## Features

- 📝 **Todo Management**: Create, update, and track todos with due dates
- 🤖 **LLM Chat & Tools**: LLM chat that can use tools to create, update, and delete tasks
- 📌 **Board Summary**: Board with counters and an LLM-generated summary
- 🧰 **LLM Tools Integration**: Tool registry enables the chat agent to perform todo operations
- 🔔 **Event-Driven**: Pub/Sub architecture for asynchronous processing
- 🔒 **Secrets Management**: HashiCorp Vault integration for secure configuration
- 📊 **Observability**: OpenTelemetry tracing with Jaeger
- 🗄️ **PostgreSQL**: Persistent storage with migrations
- 🎨 **Modern UI**: React + TypeScript frontend
- 🗃️ **Batch GraphQL Operations**: Efficiently update or delete multiple todos in a single GraphQL request using aliases.
- 🧠 **Vector Search**: pgvector + embeddings for similarity search over todos

## Architecture

### Components

- **HTTP API**: RESTful API for todo operations (OpenAPI 3.0)
- **GraphQL**: Enables batch operations (multiple mutations/queries in one request) for efficient bulk updates and deletes
- **Workers**: Background worker for summary generation and message relay
- **PostgreSQL**: Database for todos and board summaries, enhanced with pgvector for AI-powered semantic search
- **Pub/Sub Emulator**: Google Cloud Pub/Sub emulator for event messaging
- **Vault**: Secret management for sensitive configuration
- **Docker Model Runner**: Local LLM inference for board summary generation and AI Chat
- **Jaeger**: Distributed tracing and monitoring using OpenTelemetry

### 🔗 Generated Dependency Graph (Initializers, Runners, Dependencies, Configs)

```mermaid
---
  config:
    layout: elk
---
graph TD
	HTTP_PORT["<b><span style='font-size:16px'>HTTP_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	LLM_MODEL_HOST["<b><span style='font-size:16px'>LLM_MODEL_HOST</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	usecases_CreateTodoImpl["<b><span style='font-size:16px'>usecases.CreateTodo</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.CreateTodoImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitCreateTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/create_todo.go:57)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	ptr_usecases_DeleteConversationImpl["<b><span style='font-size:16px'>usecases.DeleteConversation</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 *usecases.DeleteConversationImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitDeleteConversation.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/delete_conversation.go:39)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	postgres_ChatMessageRepository["<b><span style='font-size:16px'>domain.ChatMessageRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 postgres.ChatMessageRepository</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ postgres.InitChatMessageRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(postgres/chat.go:164)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	LLM_MODEL["<b><span style='font-size:16px'>LLM_MODEL</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	time_CurrentTimeProvider["<b><span style='font-size:16px'>domain.CurrentTimeProvider</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 time.CurrentTimeProvider</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ time.InitCurrentTimeProvider.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(time/provider.go:25)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	PUBSUB_PROJECT_ID["<b><span style='font-size:16px'>PUBSUB_PROJECT_ID</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	usecases_UpdateTodoImpl["<b><span style='font-size:16px'>usecases.UpdateTodo</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.UpdateTodoImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitUpdateTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/update_todo.go:63)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	DB_PORT["<b><span style='font-size:16px'>DB_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	usecases_TodoCreatorImpl["<b><span style='font-size:16px'>usecases.TodoCreator</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.TodoCreatorImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitTodoCreator.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/todo_creator.go:88)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	postgres_BoardSummaryRepository["<b><span style='font-size:16px'>domain.BoardSummaryRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 postgres.BoardSummaryRepository</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ postgres.InitBoardSummaryRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(postgres/board_summary.go:216)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	ptr_postgres_UnitOfWork["<b><span style='font-size:16px'>domain.UnitOfWork</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 *postgres.UnitOfWork</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ postgres.InitUnitOfWork.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(postgres/unit_work.go:74)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	usecases_ListChatMessagesImpl["<b><span style='font-size:16px'>usecases.ListChatMessages</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.ListChatMessagesImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitListChatMessages.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/list_chat_messages.go:56)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	LLM_MAX_TOOL_CYCLES["<b><span style='font-size:16px'>LLM_MAX_TOOL_CYCLES</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	usecases_StreamChatImpl["<b><span style='font-size:16px'>usecases.StreamChat</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.StreamChatImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitStreamChat.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/stream_chat.go:346)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	usecases_RelayOutboxImpl["<b><span style='font-size:16px'>usecases.RelayOutbox</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.RelayOutboxImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitRelayOutbox.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/relay_outbox.go:79)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	DB_HOST["<b><span style='font-size:16px'>DB_HOST</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	ptr_log_Logger["<b><span style='font-size:16px'>*log.Logger</span></b><br/><span style='color:darkblue;font-size:11px;'>🏗️ log.InitLogger.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(log/logger.go:16)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	postgres_TodoRepository["<b><span style='font-size:16px'>domain.TodoRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 postgres.TodoRepository</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ postgres.InitTodoRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(postgres/todo.go:225)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	usecases_DeleteTodoImpl["<b><span style='font-size:16px'>usecases.DeleteTodo</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.DeleteTodoImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitDeleteTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/delete_todo.go:50)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	DB_PASS["<b><span style='font-size:16px'>DB_PASS</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.VaultProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	usecases_TodoUpdaterImpl["<b><span style='font-size:16px'>usecases.TodoUpdater</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.TodoUpdaterImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitTodoUpdater.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/todo_updater.go:109)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	SUMMARY_BATCH_SIZE["<b><span style='font-size:16px'>SUMMARY_BATCH_SIZE</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	ptr_http_Client["<b><span style='font-size:16px'>*http.Client</span></b><br/><span style='color:darkblue;font-size:11px;'>🏗️ tracing.InitHttpClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(tracing/tracing.go:157)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	ptr_sql_DB["<b><span style='font-size:16px'>*sql.DB</span></b><br/><span style='color:darkblue;font-size:11px;'>🏗️ postgres.(*InitDB).Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(postgres/init_db.go:76)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	PUBSUB_SUBSCRIPTION_ID["<b><span style='font-size:16px'>PUBSUB_SUBSCRIPTION_ID</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	FETCH_OUTBOX_INTERVAL["<b><span style='font-size:16px'>FETCH_OUTBOX_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	SUMMARY_BATCH_INTERVAL["<b><span style='font-size:16px'>SUMMARY_BATCH_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	ptr_pubsub_Client["<b><span style='font-size:16px'>*pubsub.Client</span></b><br/><span style='color:darkblue;font-size:11px;'>🏗️ pubsub.(*InitClient).Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(pubsub/client.go:27)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	usecases_GetBoardSummaryImpl["<b><span style='font-size:16px'>usecases.GetBoardSummary</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.GetBoardSummaryImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitGetBoardSummary.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/get_board_summary.go:46)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	usecases_LLMToolManager["<b><span style='font-size:16px'>domain.LLMToolRegistry</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.LLMToolManager</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitLLMToolRegistry.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/llm_tools.go:590)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	usecases_GenerateBoardSummaryImpl["<b><span style='font-size:16px'>usecases.GenerateBoardSummary</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.GenerateBoardSummaryImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitGenerateBoardSummary.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/generate_board_summary.go:187)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	DB_NAME["<b><span style='font-size:16px'>DB_NAME</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	GRAPHQL_SERVER_PORT["<b><span style='font-size:16px'>GRAPHQL_SERVER_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	LLM_EMBEDDING_MODEL["<b><span style='font-size:16px'>LLM_EMBEDDING_MODEL</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	usecases_TodoDeleterImpl["<b><span style='font-size:16px'>usecases.TodoDeleter</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.TodoDeleterImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitTodoDeleter.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/todo_deleter.go:60)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	DB_USER["<b><span style='font-size:16px'>DB_USER</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.VaultProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
	pubsub_PubSubEventPublisher["<b><span style='font-size:16px'>domain.TodoEventPublisher</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 pubsub.PubSubEventPublisher</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ pubsub.(*InitPublisher).Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(pubsub/publisher.go:54)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	modelrunner_LLMClient["<b><span style='font-size:16px'>domain.LLMClient</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 modelrunner.LLMClient</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ modelrunner.InitLLMClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(modelrunner/llm_client.go:216)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	usecases_ListTodosImpl["<b><span style='font-size:16px'>usecases.ListTodos</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.ListTodosImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitListTodos.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/list_todos.go:52)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
	ptr_postgres_InitTodoRepository["<b><span style='font-size:15px'>*postgres.InitTodoRepository</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitTodoCreator["<b><span style='font-size:15px'>*usecases.InitTodoCreator</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitTodoDeleter["<b><span style='font-size:15px'>*usecases.InitTodoDeleter</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitListTodos["<b><span style='font-size:15px'>*usecases.InitListTodos</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_postgres_InitDB["<b><span style='font-size:15px'>*postgres.InitDB</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_tracing_InitHttpClient["<b><span style='font-size:15px'>*tracing.InitHttpClient</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitListChatMessages["<b><span style='font-size:15px'>*usecases.InitListChatMessages</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_postgres_InitChatMessageRepository["<b><span style='font-size:15px'>*postgres.InitChatMessageRepository</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitDeleteConversation["<b><span style='font-size:15px'>*usecases.InitDeleteConversation</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitLLMToolRegistry["<b><span style='font-size:15px'>*usecases.InitLLMToolRegistry</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_modelrunner_InitLLMClient["<b><span style='font-size:15px'>*modelrunner.InitLLMClient</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_postgres_InitUnitOfWork["<b><span style='font-size:15px'>*postgres.InitUnitOfWork</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_pubsub_InitPublisher["<b><span style='font-size:15px'>*pubsub.InitPublisher</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_tracing_InitOpenTelemetry["<b><span style='font-size:15px'>*tracing.InitOpenTelemetry</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitRelayOutbox["<b><span style='font-size:15px'>*usecases.InitRelayOutbox</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_log_InitLogger["<b><span style='font-size:16px'>*log.InitLogger</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitUpdateTodo["<b><span style='font-size:15px'>*usecases.InitUpdateTodo</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitTodoUpdater["<b><span style='font-size:15px'>*usecases.InitTodoUpdater</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitGenerateBoardSummary["<b><span style='font-size:15px'>*usecases.InitGenerateBoardSummary</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_app_ReportLoggerIntrospector["<b><span style='font-size:15px'>*app.ReportLoggerIntrospector</span></b><br/><span style='color:gray;font-size:11px;'>📍(symbiont@v0.1.0/symbiont.go:95)</span>"]
	ptr_config_InitVaultProvider["<b><span style='font-size:16px'>*config.InitVaultProvider</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_time_InitCurrentTimeProvider["<b><span style='font-size:16px'>*time.InitCurrentTimeProvider</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitCreateTodo["<b><span style='font-size:15px'>*usecases.InitCreateTodo</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitStreamChat["<b><span style='font-size:15px'>*usecases.InitStreamChat</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_pubsub_InitClient["<b><span style='font-size:15px'>*pubsub.InitClient</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitDeleteTodo["<b><span style='font-size:15px'>*usecases.InitDeleteTodo</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_usecases_InitGetBoardSummary["<b><span style='font-size:15px'>*usecases.InitGetBoardSummary</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_postgres_InitBoardSummaryRepository["<b><span style='font-size:15px'>*postgres.InitBoardSummaryRepository</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
	ptr_http_TodoAppServer["<b><span style='font-size:16px'>*http.TodoAppServer</span></b><br/><span style='color:green;font-size:11px;'>⚙️ <b>Runnable</b></span>"]
	ptr_workers_TodoEventSubscriber["<b><span style='font-size:16px'>*workers.TodoEventSubscriber</span></b><br/><span style='color:green;font-size:11px;'>⚙️ <b>Runnable</b></span>"]
	ptr_workers_MessageRelay["<b><span style='font-size:16px'>*workers.MessageRelay</span></b><br/><span style='color:green;font-size:11px;'>⚙️ <b>Runnable</b></span>"]
	ptr_graphql_TodoGraphQLServer["<b><span style='font-size:16px'>*graphql.TodoGraphQLServer</span></b><br/><span style='color:green;font-size:11px;'>⚙️ <b>Runnable</b></span>"]
	SymbiontApp["<b><span style='font-size:20px;color:white'>🚀 Symbiont App</span></b>"]
	ptr_graphql_TodoGraphQLServer --- SymbiontApp
	ptr_http_Client -.-> ptr_modelrunner_InitLLMClient
	ptr_http_TodoAppServer --- SymbiontApp
	ptr_log_InitLogger --o ptr_log_Logger
	ptr_log_Logger -.-> ptr_app_ReportLoggerIntrospector
	ptr_log_Logger -.-> ptr_graphql_TodoGraphQLServer
	ptr_log_Logger -.-> ptr_http_TodoAppServer
	ptr_log_Logger -.-> ptr_postgres_InitDB
	ptr_log_Logger -.-> ptr_pubsub_InitClient
	ptr_log_Logger -.-> ptr_tracing_InitHttpClient
	ptr_log_Logger -.-> ptr_tracing_InitOpenTelemetry
	ptr_log_Logger -.-> ptr_usecases_InitRelayOutbox
	ptr_log_Logger -.-> ptr_workers_MessageRelay
	ptr_log_Logger -.-> ptr_workers_TodoEventSubscriber
	ptr_modelrunner_InitLLMClient --o modelrunner_LLMClient
	ptr_postgres_InitBoardSummaryRepository --o postgres_BoardSummaryRepository
	ptr_postgres_InitChatMessageRepository --o postgres_ChatMessageRepository
	ptr_postgres_InitDB --o ptr_sql_DB
	ptr_postgres_InitTodoRepository --o postgres_TodoRepository
	ptr_postgres_InitUnitOfWork --o ptr_postgres_UnitOfWork
	ptr_postgres_UnitOfWork -.-> ptr_usecases_InitCreateTodo
	ptr_postgres_UnitOfWork -.-> ptr_usecases_InitDeleteTodo
	ptr_postgres_UnitOfWork -.-> ptr_usecases_InitLLMToolRegistry
	ptr_postgres_UnitOfWork -.-> ptr_usecases_InitRelayOutbox
	ptr_postgres_UnitOfWork -.-> ptr_usecases_InitTodoCreator
	ptr_postgres_UnitOfWork -.-> ptr_usecases_InitTodoDeleter
	ptr_postgres_UnitOfWork -.-> ptr_usecases_InitTodoUpdater
	ptr_postgres_UnitOfWork -.-> ptr_usecases_InitUpdateTodo
	ptr_pubsub_Client -.-> ptr_pubsub_InitPublisher
	ptr_pubsub_Client -.-> ptr_workers_TodoEventSubscriber
	ptr_pubsub_InitClient --o ptr_pubsub_Client
	ptr_pubsub_InitPublisher --o pubsub_PubSubEventPublisher
	ptr_sql_DB -.-> ptr_postgres_InitBoardSummaryRepository
	ptr_sql_DB -.-> ptr_postgres_InitChatMessageRepository
	ptr_sql_DB -.-> ptr_postgres_InitTodoRepository
	ptr_sql_DB -.-> ptr_postgres_InitUnitOfWork
	ptr_time_InitCurrentTimeProvider --o time_CurrentTimeProvider
	ptr_tracing_InitHttpClient --o ptr_http_Client
	ptr_usecases_DeleteConversationImpl -.-> ptr_http_TodoAppServer
	ptr_usecases_InitCreateTodo --o usecases_CreateTodoImpl
	ptr_usecases_InitDeleteConversation --o ptr_usecases_DeleteConversationImpl
	ptr_usecases_InitDeleteTodo --o usecases_DeleteTodoImpl
	ptr_usecases_InitGenerateBoardSummary --o usecases_GenerateBoardSummaryImpl
	ptr_usecases_InitGetBoardSummary --o usecases_GetBoardSummaryImpl
	ptr_usecases_InitLLMToolRegistry --o usecases_LLMToolManager
	ptr_usecases_InitListChatMessages --o usecases_ListChatMessagesImpl
	ptr_usecases_InitListTodos --o usecases_ListTodosImpl
	ptr_usecases_InitRelayOutbox --o usecases_RelayOutboxImpl
	ptr_usecases_InitStreamChat --o usecases_StreamChatImpl
	ptr_usecases_InitTodoCreator --o usecases_TodoCreatorImpl
	ptr_usecases_InitTodoDeleter --o usecases_TodoDeleterImpl
	ptr_usecases_InitTodoUpdater --o usecases_TodoUpdaterImpl
	ptr_usecases_InitUpdateTodo --o usecases_UpdateTodoImpl
	ptr_workers_MessageRelay --- SymbiontApp
	ptr_workers_TodoEventSubscriber --- SymbiontApp
	DB_HOST -.-> ptr_postgres_InitDB
	DB_NAME -.-> ptr_postgres_InitDB
	DB_PASS -.-> ptr_postgres_InitDB
	DB_PORT -.-> ptr_postgres_InitDB
	DB_USER -.-> ptr_postgres_InitDB
	FETCH_OUTBOX_INTERVAL -.-> ptr_workers_MessageRelay
	GRAPHQL_SERVER_PORT -.-> ptr_graphql_TodoGraphQLServer
	HTTP_PORT -.-> ptr_http_TodoAppServer
	LLM_EMBEDDING_MODEL -.-> ptr_usecases_InitLLMToolRegistry
	LLM_EMBEDDING_MODEL -.-> ptr_usecases_InitStreamChat
	LLM_EMBEDDING_MODEL -.-> ptr_usecases_InitTodoCreator
	LLM_EMBEDDING_MODEL -.-> ptr_usecases_InitTodoUpdater
	LLM_MAX_TOOL_CYCLES -.-> ptr_usecases_InitStreamChat
	LLM_MODEL -.-> ptr_usecases_InitGenerateBoardSummary
	LLM_MODEL -.-> ptr_usecases_InitStreamChat
	LLM_MODEL_HOST -.-> ptr_modelrunner_InitLLMClient
	PUBSUB_PROJECT_ID -.-> ptr_pubsub_InitClient
	PUBSUB_SUBSCRIPTION_ID -.-> ptr_workers_TodoEventSubscriber
	SUMMARY_BATCH_INTERVAL -.-> ptr_workers_TodoEventSubscriber
	SUMMARY_BATCH_SIZE -.-> ptr_workers_TodoEventSubscriber
	modelrunner_LLMClient -.-> ptr_usecases_InitGenerateBoardSummary
	modelrunner_LLMClient -.-> ptr_usecases_InitLLMToolRegistry
	modelrunner_LLMClient -.-> ptr_usecases_InitStreamChat
	modelrunner_LLMClient -.-> ptr_usecases_InitTodoCreator
	modelrunner_LLMClient -.-> ptr_usecases_InitTodoUpdater
	postgres_BoardSummaryRepository -.-> ptr_usecases_InitGenerateBoardSummary
	postgres_BoardSummaryRepository -.-> ptr_usecases_InitGetBoardSummary
	postgres_ChatMessageRepository -.-> ptr_usecases_InitDeleteConversation
	postgres_ChatMessageRepository -.-> ptr_usecases_InitListChatMessages
	postgres_ChatMessageRepository -.-> ptr_usecases_InitStreamChat
	postgres_TodoRepository -.-> ptr_usecases_InitLLMToolRegistry
	postgres_TodoRepository -.-> ptr_usecases_InitListTodos
	pubsub_PubSubEventPublisher -.-> ptr_usecases_InitRelayOutbox
	time_CurrentTimeProvider -.-> ptr_usecases_InitGenerateBoardSummary
	time_CurrentTimeProvider -.-> ptr_usecases_InitLLMToolRegistry
	time_CurrentTimeProvider -.-> ptr_usecases_InitStreamChat
	time_CurrentTimeProvider -.-> ptr_usecases_InitTodoCreator
	time_CurrentTimeProvider -.-> ptr_usecases_InitTodoDeleter
	time_CurrentTimeProvider -.-> ptr_usecases_InitTodoUpdater
	usecases_CreateTodoImpl -.-> ptr_http_TodoAppServer
	usecases_DeleteTodoImpl -.-> ptr_graphql_TodoGraphQLServer
	usecases_DeleteTodoImpl -.-> ptr_http_TodoAppServer
	usecases_GenerateBoardSummaryImpl -.-> ptr_workers_TodoEventSubscriber
	usecases_GetBoardSummaryImpl -.-> ptr_http_TodoAppServer
	usecases_LLMToolManager -.-> ptr_usecases_InitStreamChat
	usecases_ListChatMessagesImpl -.-> ptr_http_TodoAppServer
	usecases_ListTodosImpl -.-> ptr_graphql_TodoGraphQLServer
	usecases_ListTodosImpl -.-> ptr_http_TodoAppServer
	usecases_RelayOutboxImpl -.-> ptr_workers_MessageRelay
	usecases_StreamChatImpl -.-> ptr_http_TodoAppServer
	usecases_TodoCreatorImpl -.-> ptr_usecases_InitCreateTodo
	usecases_TodoCreatorImpl -.-> ptr_usecases_InitLLMToolRegistry
	usecases_TodoDeleterImpl -.-> ptr_usecases_InitDeleteTodo
	usecases_TodoDeleterImpl -.-> ptr_usecases_InitLLMToolRegistry
	usecases_TodoUpdaterImpl -.-> ptr_usecases_InitLLMToolRegistry
	usecases_TodoUpdaterImpl -.-> ptr_usecases_InitUpdateTodo
	usecases_UpdateTodoImpl -.-> ptr_graphql_TodoGraphQLServer
	usecases_UpdateTodoImpl -.-> ptr_http_TodoAppServer
	style HTTP_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_MODEL_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_CreateTodoImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style ptr_usecases_DeleteConversationImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style postgres_ChatMessageRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style time_CurrentTimeProvider fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style PUBSUB_PROJECT_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_UpdateTodoImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style DB_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_TodoCreatorImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style postgres_BoardSummaryRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style ptr_postgres_UnitOfWork fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_ListChatMessagesImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_MAX_TOOL_CYCLES fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_StreamChatImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_RelayOutboxImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style DB_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_log_Logger fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style postgres_TodoRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_DeleteTodoImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style DB_PASS fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_TodoUpdaterImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style SUMMARY_BATCH_SIZE fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_http_Client fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style ptr_sql_DB fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style PUBSUB_SUBSCRIPTION_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style FETCH_OUTBOX_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style SUMMARY_BATCH_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_pubsub_Client fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_GetBoardSummaryImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_LLMToolManager fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_GenerateBoardSummaryImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style DB_NAME fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style GRAPHQL_SERVER_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_EMBEDDING_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_TodoDeleterImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style DB_USER fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style pubsub_PubSubEventPublisher fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style modelrunner_LLMClient fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_ListTodosImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style ptr_postgres_InitTodoRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitTodoCreator fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitTodoDeleter fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitListTodos fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitDB fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_tracing_InitHttpClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitListChatMessages fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitChatMessageRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitDeleteConversation fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitLLMToolRegistry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_modelrunner_InitLLMClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitUnitOfWork fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_pubsub_InitPublisher fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_tracing_InitOpenTelemetry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitRelayOutbox fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_log_InitLogger fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitUpdateTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitTodoUpdater fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitGenerateBoardSummary fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_app_ReportLoggerIntrospector fill:#fff3e0,stroke:#f57c00,stroke-width:2px,color:#222222
	style ptr_config_InitVaultProvider fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_time_InitCurrentTimeProvider fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitCreateTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitStreamChat fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_pubsub_InitClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitDeleteTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitGetBoardSummary fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitBoardSummaryRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_http_TodoAppServer fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_TodoEventSubscriber fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_MessageRelay fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_graphql_TodoGraphQLServer fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style SymbiontApp fill:#0f56c4,stroke:#68a4eb,stroke-width:6px,color:#ffffff,font-weight:bold
```


## Prerequisites

- Docker v29.1.5
- Docker Compose v5.0.1
- Docker Model Runner v1.0.7
- Go 1.25 or higher (for local development)
- Node.js 18+ (for webapp development)

## Quick Start

### 1. Start the Application

```bash
docker compose up -d
```

The application will be available at:
- **Web UI**: http://localhost:8080
- **API**: http://localhost:8080/api/v1
- **GraphQL Playground**: http://localhost:8085/
- **GraphQL API**: http://localhost:8085/v1/query
- **Jaeger Tracing**: http://localhost:16686

### Running Tests

#### Unit Tests

```bash
# Run all unit tests
go test ./...
```

#### Integration Tests

Integration tests use testcontainers to automatically spin up and manage all required dependencies (PostgreSQL, Vault, Pub/Sub, Docker Model Runner). No manual setup needed.

```bash
go test -tags=integration ./tests/integration/... -v
```

Testcontainers automatically:
- Pulls and starts required Docker images and models.
- Waits for services to be healthy
- Cleans up containers after tests complete
- Isolates tests from each other


## Observability

### Application Logs

View real-time application logs:

```bash
# View logs from Docker container
docker-compose logs -f todoapp

```

### Jaeger Tracing

Access Jaeger UI at http://localhost:16686 to view:
- Request traces across services
- API call latencies
- Database query performance
- Pub/Sub message flow
- LLM generation traces

Search for traces by:
- Service name: `todoapp`
- Operation: `HTTP POST /api/v1/todos`, `GenerateBoardSummary`, etc.
- Tags: `http.status_code`, `db.statement`, etc.

## Technical Details

### Symbiont Features

This example demonstrates Symbiont's key features:

1. **Dependency Injection**: All components are wired through Symbiont's DI container
2. **Configuration Loading**: Type-safe configuration from multiple sources (environment variables, Vault, defaults)
3. **Introspection**: Auto-generated dependency graphs for visualizing application architecture
4. **Multi-Process Hosting**: Run multiple Runners/servers/workers within the same deployable for simplified ops
