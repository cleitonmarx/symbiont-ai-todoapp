# Introspection Graph

This document contains the full generated Mermaid graph for the TodoApp Symbiont composition.

Interactive endpoint when running locally: `http://localhost:8080/introspect`

```mermaid
---
  config:
    layout: elk
---
graph TD
	domain_ChatMessageRepository____postgres_ChatMessageRepository["<b><span style='font-size:16px'>domain.ChatMessageRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.ChatMessageRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitChatMessageRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/chat.go:256)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_AssistantModelCatalog____modelrunner_AssistantClient["<b><span style='font-size:16px'>domain.AssistantModelCatalog</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© modelrunner.AssistantClient</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ modelrunner.InitAssistantClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(modelrunner/assistant_client.go:391)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	LLM_SUMMARY_MODEL["<b><span style='font-size:16px'>LLM_SUMMARY_MODEL</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_PORT["<b><span style='font-size:16px'>DB_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	usecases_TodoCreator____usecases_TodoCreatorImpl["<b><span style='font-size:16px'>usecases.TodoCreator</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.TodoCreatorImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitTodoCreator.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/todo_creator.go:90)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_ListAvailableModels____ptr_usecases_ListAvailableModelsImpl["<b><span style='font-size:16px'>usecases.ListAvailableModels</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *usecases.ListAvailableModelsImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitListAvailableModels.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/list_available_models.go:56)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	CHAT_TITLE_BATCH_INTERVAL["<b><span style='font-size:16px'>CHAT_TITLE_BATCH_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ptr_http_Client____ptr_http_Client["<b><span style='font-size:16px'>*http.Client</span></b><br/><span style='color:darkblue;font-size:11px;'>ðï¸ telemetry.InitHttpClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(telemetry/init.go:108)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_GenerateConversationTitle____usecases_GenerateConversationTitleImpl["<b><span style='font-size:16px'>usecases.GenerateConversationTitle</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.GenerateConversationTitleImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitGenerateConversationTitle.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/generate_conversation_title.go:407)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_UpdateTodo____usecases_UpdateTodoImpl["<b><span style='font-size:16px'>usecases.UpdateTodo</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.UpdateTodoImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitUpdateTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/update_todo.go:63)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	SUMMARY_BATCH_INTERVAL["<b><span style='font-size:16px'>SUMMARY_BATCH_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	TODO_EVENTS_SUBSCRIPTION_ID["<b><span style='font-size:16px'>TODO_EVENTS_SUBSCRIPTION_ID</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	usecases_ListTodos____usecases_ListTodosImpl["<b><span style='font-size:16px'>usecases.ListTodos</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.ListTodosImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitListTodos.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/list_todos.go:139)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	LLM_EMBEDDING_MODEL_HOST["<b><span style='font-size:16px'>LLM_EMBEDDING_MODEL_HOST</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ptr_sql_DB____ptr_sql_DB["<b><span style='font-size:16px'>*sql.DB</span></b><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.(*InitDB).Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/init_db.go:96)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	MCP_GATEWAY_REQUEST_TIMEOUT["<b><span style='font-size:16px'>MCP_GATEWAY_REQUEST_TIMEOUT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	domain_Assistant____modelrunner_AssistantClient["<b><span style='font-size:16px'>domain.Assistant</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© modelrunner.AssistantClient</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ modelrunner.InitAssistantClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(modelrunner/assistant_client.go:389)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	DB_USER["<b><span style='font-size:16px'>DB_USER</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.VaultProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	domain_ConversationRepository____postgres_ConversationRepository["<b><span style='font-size:16px'>domain.ConversationRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.ConversationRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitConversationRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/conversation.go:233)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_AssistantActionRegistry__local__local_LocalRegistry["<b><span style='font-size:16px'>domain.AssistantActionRegistry</span></b><br/><span style='color:#b26a00;font-size:12px;'>name: local</span><br/><span style='color:darkgray;font-size:11px;'>ð§© local.LocalRegistry</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ local.InitLocalActionRegistry.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(local/registry.go:111)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_SubmitActionApproval____ptr_usecases_SubmitActionApprovalImpl["<b><span style='font-size:16px'>usecases.SubmitActionApproval</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *usecases.SubmitActionApprovalImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitSubmitActionApproval.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/submit_action_approval.go:82)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	CHAT_SUMMARY_BATCH_SIZE["<b><span style='font-size:16px'>CHAT_SUMMARY_BATCH_SIZE</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_PASS["<b><span style='font-size:16px'>DB_PASS</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.VaultProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	domain_CurrentTimeProvider____time_CurrentTimeProvider["<b><span style='font-size:16px'>domain.CurrentTimeProvider</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© time.CurrentTimeProvider</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ time.InitCurrentTimeProvider.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(time/provider.go:25)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	LLM_MODEL_HOST["<b><span style='font-size:16px'>LLM_MODEL_HOST</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	MCP_GATEWAY_API_KEY_HEADER["<b><span style='font-size:16px'>MCP_GATEWAY_API_KEY_HEADER</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ptr_log_Logger____ptr_log_Logger["<b><span style='font-size:16px'>*log.Logger</span></b><br/><span style='color:darkblue;font-size:11px;'>ðï¸ log.InitLogger.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(log/logger.go:16)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_CreateTodo____usecases_CreateTodoImpl["<b><span style='font-size:16px'>usecases.CreateTodo</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.CreateTodoImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitCreateTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/create_todo.go:57)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_RelayOutbox____usecases_RelayOutboxImpl["<b><span style='font-size:16px'>usecases.RelayOutbox</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.RelayOutboxImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitRelayOutbox.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/relay_outbox.go:79)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	DB_HOST["<b><span style='font-size:16px'>DB_HOST</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	usecases_GetBoardSummary____usecases_GetBoardSummaryImpl["<b><span style='font-size:16px'>usecases.GetBoardSummary</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.GetBoardSummaryImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitGetBoardSummary.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/get_board_summary.go:46)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_ConversationSummaryRepository____postgres_ConversationSummaryRepository["<b><span style='font-size:16px'>domain.ConversationSummaryRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.ConversationSummaryRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitConversationSummaryRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/conversation_summary.go:116)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_TodoDeleter____usecases_TodoDeleterImpl["<b><span style='font-size:16px'>usecases.TodoDeleter</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.TodoDeleterImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitTodoDeleter.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/todo_deleter.go:60)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_AssistantActionRegistry__mcp__ptr_mcp_MCPRegistry["<b><span style='font-size:16px'>domain.AssistantActionRegistry</span></b><br/><span style='color:#b26a00;font-size:12px;'>name: mcp</span><br/><span style='color:darkgray;font-size:11px;'>ð§© *mcp.MCPRegistry</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ mcp.InitMCPActionRegistry.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(mcp/registry.go:1029)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_DeleteConversation____ptr_usecases_DeleteConversationImpl["<b><span style='font-size:16px'>usecases.DeleteConversation</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *usecases.DeleteConversationImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitDeleteConversation.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/delete_conversation.go:69)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_ListChatMessages____usecases_ListChatMessagesImpl["<b><span style='font-size:16px'>usecases.ListChatMessages</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.ListChatMessagesImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitListChatMessages.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/list_chat_messages.go:57)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_TodoRepository____postgres_TodoRepository["<b><span style='font-size:16px'>domain.TodoRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.TodoRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitTodoRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/todo.go:268)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_StreamChat____usecases_StreamChatImpl["<b><span style='font-size:16px'>usecases.StreamChat</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.StreamChatImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitStreamChat.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/stream_chat.go:1122)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_GenerateBoardSummary____usecases_GenerateBoardSummaryImpl["<b><span style='font-size:16px'>usecases.GenerateBoardSummary</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.GenerateBoardSummaryImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitGenerateBoardSummary.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/generate_board_summary.go:211)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_AssistantActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher["<b><span style='font-size:16px'>domain.AssistantActionApprovalDispatcher</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *approvaldispatcher.Dispatcher</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ approvaldispatcher.InitDispatcher.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(approvaldispatcher/dispatcher.go:90)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	SUMMARY_BATCH_SIZE["<b><span style='font-size:16px'>SUMMARY_BATCH_SIZE</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ACTION_APPROVAL_EVENTS_SUBSCRIPTION_ID["<b><span style='font-size:16px'>ACTION_APPROVAL_EVENTS_SUBSCRIPTION_ID</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_CHAT_TITLE_MODEL["<b><span style='font-size:16px'>LLM_CHAT_TITLE_MODEL</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	domain_BoardSummaryRepository____postgres_BoardSummaryRepository["<b><span style='font-size:16px'>domain.BoardSummaryRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.BoardSummaryRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitBoardSummaryRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/board_summary.go:216)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	FETCH_OUTBOX_INTERVAL["<b><span style='font-size:16px'>FETCH_OUTBOX_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	usecases_GenerateChatSummary____usecases_GenerateChatSummaryImpl["<b><span style='font-size:16px'>usecases.GenerateChatSummary</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.GenerateChatSummaryImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitGenerateChatSummary.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/generate_chat_summary.go:562)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_EventPublisher____pubsub_PubSubEventPublisher["<b><span style='font-size:16px'>domain.EventPublisher</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© pubsub.PubSubEventPublisher</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ pubsub.(*InitPublisher).Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(pubsub/publisher.go:54)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_AssistantSkillRegistry____skillregistry_Registry["<b><span style='font-size:16px'>domain.AssistantSkillRegistry</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© skillregistry.Registry</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ skillregistry.InitLocalSkillRegistry.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(skillregistry/registry.go:587)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	CHAT_EVENTS_SUBSCRIPTION_ID["<b><span style='font-size:16px'>CHAT_EVENTS_SUBSCRIPTION_ID</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ptr_pubsub_Client____ptr_pubsub_Client["<b><span style='font-size:16px'>*pubsub.Client</span></b><br/><span style='color:darkblue;font-size:11px;'>ðï¸ pubsub.(*InitClient).Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(pubsub/client.go:30)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	MCP_GATEWAY_ENDPOINT["<b><span style='font-size:16px'>MCP_GATEWAY_ENDPOINT</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	domain_AssistantActionRegistry____composite_CompositeActionRegistry["<b><span style='font-size:16px'>domain.AssistantActionRegistry</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© composite.CompositeActionRegistry</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ composite.InitCompositeActionRegistry.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(composite/registry.go:77)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_UnitOfWork____ptr_postgres_UnitOfWork["<b><span style='font-size:16px'>domain.UnitOfWork</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *postgres.UnitOfWork</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitUnitOfWork.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/unit_work.go:92)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	domain_SemanticEncoder____modelrunner_AssistantClient["<b><span style='font-size:16px'>domain.SemanticEncoder</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© modelrunner.AssistantClient</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ modelrunner.InitAssistantClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(modelrunner/assistant_client.go:390)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_TodoUpdater____usecases_TodoUpdaterImpl["<b><span style='font-size:16px'>usecases.TodoUpdater</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.TodoUpdaterImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitTodoUpdater.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/todo_updater.go:112)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	usecases_ListConversations____ptr_usecases_ListConversationsImpl["<b><span style='font-size:16px'>usecases.ListConversations</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *usecases.ListConversationsImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitListConversations.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/list_conversations.go:49)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	MCP_GATEWAY_API_KEY["<b><span style='font-size:16px'>MCP_GATEWAY_API_KEY</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	HTTP_PORT["<b><span style='font-size:16px'>HTTP_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_EMBEDDING_API_KEY["<b><span style='font-size:16px'>LLM_EMBEDDING_API_KEY</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	usecases_DeleteTodo____usecases_DeleteTodoImpl["<b><span style='font-size:16px'>usecases.DeleteTodo</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© usecases.DeleteTodoImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitDeleteTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/delete_todo.go:50)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	LLM_API_KEY["<b><span style='font-size:16px'>LLM_API_KEY</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_MAX_ACTION_CYCLES["<b><span style='font-size:16px'>LLM_MAX_ACTION_CYCLES</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	PUBSUB_PROJECT_ID["<b><span style='font-size:16px'>PUBSUB_PROJECT_ID</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	CHAT_TITLE_EVENTS_SUBSCRIPTION_ID["<b><span style='font-size:16px'>CHAT_TITLE_EVENTS_SUBSCRIPTION_ID</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_NAME["<b><span style='font-size:16px'>DB_NAME</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	usecases_UpdateConversation____ptr_usecases_UpdateConversationImpl["<b><span style='font-size:16px'>usecases.UpdateConversation</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *usecases.UpdateConversationImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ usecases.InitUpdateConversation.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(usecases/update_conversation.go:69)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	CHAT_SUMMARY_BATCH_INTERVAL["<b><span style='font-size:16px'>CHAT_SUMMARY_BATCH_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	GRAPHQL_SERVER_PORT["<b><span style='font-size:16px'>GRAPHQL_SERVER_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_CHAT_SUMMARY_MODEL["<b><span style='font-size:16px'>LLM_CHAT_SUMMARY_MODEL</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	CHAT_TITLE_BATCH_SIZE["<b><span style='font-size:16px'>CHAT_TITLE_BATCH_SIZE</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_EMBEDDING_MODEL["<b><span style='font-size:16px'>LLM_EMBEDDING_MODEL</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ptr_usecases_InitListTodos["<b><span style='font-size:15px'>*usecases.InitListTodos</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitTodoUpdater["<b><span style='font-size:15px'>*usecases.InitTodoUpdater</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitConversationSummaryRepository["<b><span style='font-size:15px'>*postgres.InitConversationSummaryRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitTodoCreator["<b><span style='font-size:15px'>*usecases.InitTodoCreator</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitCreateTodo["<b><span style='font-size:15px'>*usecases.InitCreateTodo</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitBoardSummaryRepository["<b><span style='font-size:15px'>*postgres.InitBoardSummaryRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_local_InitLocalActionRegistry["<b><span style='font-size:15px'>*local.InitLocalActionRegistry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitGenerateChatSummary["<b><span style='font-size:15px'>*usecases.InitGenerateChatSummary</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitStreamChat["<b><span style='font-size:15px'>*usecases.InitStreamChat</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitConversationRepository["<b><span style='font-size:15px'>*postgres.InitConversationRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_pubsub_InitPublisher["<b><span style='font-size:15px'>*pubsub.InitPublisher</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitDB["<b><span style='font-size:15px'>*postgres.InitDB</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitUnitOfWork["<b><span style='font-size:15px'>*postgres.InitUnitOfWork</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_time_InitCurrentTimeProvider["<b><span style='font-size:16px'>*time.InitCurrentTimeProvider</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_skillregistry_InitLocalSkillRegistry["<b><span style='font-size:15px'>*skillregistry.InitLocalSkillRegistry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitGenerateBoardSummary["<b><span style='font-size:15px'>*usecases.InitGenerateBoardSummary</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitTodoRepository["<b><span style='font-size:15px'>*postgres.InitTodoRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitChatMessageRepository["<b><span style='font-size:15px'>*postgres.InitChatMessageRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_approvaldispatcher_InitDispatcher["<b><span style='font-size:16px'>*approvaldispatcher.InitDispatcher</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitUpdateTodo["<b><span style='font-size:15px'>*usecases.InitUpdateTodo</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_telemetry_InitOpenTelemetry["<b><span style='font-size:15px'>*telemetry.InitOpenTelemetry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_config_InitVaultProvider["<b><span style='font-size:16px'>*config.InitVaultProvider</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitGetBoardSummary["<b><span style='font-size:15px'>*usecases.InitGetBoardSummary</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_mcp_InitMCPActionRegistry["<b><span style='font-size:15px'>*mcp.InitMCPActionRegistry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitGenerateConversationTitle["<b><span style='font-size:15px'>*usecases.InitGenerateConversationTitle</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_pubsub_InitClient["<b><span style='font-size:15px'>*pubsub.InitClient</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_log_InitLogger["<b><span style='font-size:16px'>*log.InitLogger</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitSubmitActionApproval["<b><span style='font-size:15px'>*usecases.InitSubmitActionApproval</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitTodoDeleter["<b><span style='font-size:15px'>*usecases.InitTodoDeleter</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitDeleteConversation["<b><span style='font-size:15px'>*usecases.InitDeleteConversation</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitListChatMessages["<b><span style='font-size:15px'>*usecases.InitListChatMessages</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitRelayOutbox["<b><span style='font-size:15px'>*usecases.InitRelayOutbox</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_composite_InitCompositeActionRegistry["<b><span style='font-size:15px'>*composite.InitCompositeActionRegistry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitListAvailableModels["<b><span style='font-size:15px'>*usecases.InitListAvailableModels</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_modelrunner_InitAssistantClient["<b><span style='font-size:15px'>*modelrunner.InitAssistantClient</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_telemetry_InitHttpClient["<b><span style='font-size:15px'>*telemetry.InitHttpClient</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitDeleteTodo["<b><span style='font-size:15px'>*usecases.InitDeleteTodo</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitListConversations["<b><span style='font-size:15px'>*usecases.InitListConversations</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_usecases_InitUpdateConversation["<b><span style='font-size:15px'>*usecases.InitUpdateConversation</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_http_TodoAppServer["<b><span style='font-size:16px'>*http.TodoAppServer</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_workers_ChatSummaryGenerator["<b><span style='font-size:16px'>*workers.ChatSummaryGenerator</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_workers_ConversationTitleGenerator["<b><span style='font-size:16px'>*workers.ConversationTitleGenerator</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_graphql_TodoGraphQLServer["<b><span style='font-size:16px'>*graphql.TodoGraphQLServer</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_workers_ActionApprovalDispatcher["<b><span style='font-size:16px'>*workers.ActionApprovalDispatcher</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_workers_MessageRelay["<b><span style='font-size:16px'>*workers.MessageRelay</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_workers_BoardSummaryGenerator["<b><span style='font-size:16px'>*workers.BoardSummaryGenerator</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	SymbiontApp["<b><span style='font-size:20px;color:white'>ð Symbiont App</span></b>"]
	ptr_approvaldispatcher_InitDispatcher --o domain_AssistantActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher
	ptr_composite_InitCompositeActionRegistry --o domain_AssistantActionRegistry____composite_CompositeActionRegistry
	ptr_graphql_TodoGraphQLServer --- SymbiontApp
	ptr_http_Client____ptr_http_Client -.-> ptr_mcp_InitMCPActionRegistry
	ptr_http_Client____ptr_http_Client -.-> ptr_modelrunner_InitAssistantClient
	ptr_http_TodoAppServer --- SymbiontApp
	ptr_local_InitLocalActionRegistry --o domain_AssistantActionRegistry__local__local_LocalRegistry
	ptr_log_InitLogger --o ptr_log_Logger____ptr_log_Logger
	ptr_log_Logger____ptr_log_Logger -.-> ptr_graphql_TodoGraphQLServer
	ptr_log_Logger____ptr_log_Logger -.-> ptr_http_TodoAppServer
	ptr_log_Logger____ptr_log_Logger -.-> ptr_mcp_InitMCPActionRegistry
	ptr_log_Logger____ptr_log_Logger -.-> ptr_postgres_InitDB
	ptr_log_Logger____ptr_log_Logger -.-> ptr_pubsub_InitClient
	ptr_log_Logger____ptr_log_Logger -.-> ptr_telemetry_InitHttpClient
	ptr_log_Logger____ptr_log_Logger -.-> ptr_telemetry_InitOpenTelemetry
	ptr_log_Logger____ptr_log_Logger -.-> ptr_usecases_InitRelayOutbox
	ptr_log_Logger____ptr_log_Logger -.-> ptr_usecases_InitStreamChat
	ptr_log_Logger____ptr_log_Logger -.-> ptr_workers_ActionApprovalDispatcher
	ptr_log_Logger____ptr_log_Logger -.-> ptr_workers_BoardSummaryGenerator
	ptr_log_Logger____ptr_log_Logger -.-> ptr_workers_ChatSummaryGenerator
	ptr_log_Logger____ptr_log_Logger -.-> ptr_workers_ConversationTitleGenerator
	ptr_log_Logger____ptr_log_Logger -.-> ptr_workers_MessageRelay
	ptr_mcp_InitMCPActionRegistry --o domain_AssistantActionRegistry__mcp__ptr_mcp_MCPRegistry
	ptr_modelrunner_InitAssistantClient --o domain_Assistant____modelrunner_AssistantClient
	ptr_modelrunner_InitAssistantClient --o domain_AssistantModelCatalog____modelrunner_AssistantClient
	ptr_modelrunner_InitAssistantClient --o domain_SemanticEncoder____modelrunner_AssistantClient
	ptr_postgres_InitBoardSummaryRepository --o domain_BoardSummaryRepository____postgres_BoardSummaryRepository
	ptr_postgres_InitChatMessageRepository --o domain_ChatMessageRepository____postgres_ChatMessageRepository
	ptr_postgres_InitConversationRepository --o domain_ConversationRepository____postgres_ConversationRepository
	ptr_postgres_InitConversationSummaryRepository --o domain_ConversationSummaryRepository____postgres_ConversationSummaryRepository
	ptr_postgres_InitDB --o ptr_sql_DB____ptr_sql_DB
	ptr_postgres_InitTodoRepository --o domain_TodoRepository____postgres_TodoRepository
	ptr_postgres_InitUnitOfWork --o domain_UnitOfWork____ptr_postgres_UnitOfWork
	ptr_pubsub_Client____ptr_pubsub_Client -.-> ptr_pubsub_InitPublisher
	ptr_pubsub_Client____ptr_pubsub_Client -.-> ptr_workers_ActionApprovalDispatcher
	ptr_pubsub_Client____ptr_pubsub_Client -.-> ptr_workers_BoardSummaryGenerator
	ptr_pubsub_Client____ptr_pubsub_Client -.-> ptr_workers_ChatSummaryGenerator
	ptr_pubsub_Client____ptr_pubsub_Client -.-> ptr_workers_ConversationTitleGenerator
	ptr_pubsub_InitClient --o ptr_pubsub_Client____ptr_pubsub_Client
	ptr_pubsub_InitPublisher --o domain_EventPublisher____pubsub_PubSubEventPublisher
	ptr_skillregistry_InitLocalSkillRegistry --o domain_AssistantSkillRegistry____skillregistry_Registry
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitBoardSummaryRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitChatMessageRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitConversationRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitConversationSummaryRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitTodoRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitUnitOfWork
	ptr_telemetry_InitHttpClient --o ptr_http_Client____ptr_http_Client
	ptr_time_InitCurrentTimeProvider --o domain_CurrentTimeProvider____time_CurrentTimeProvider
	ptr_usecases_InitCreateTodo --o usecases_CreateTodo____usecases_CreateTodoImpl
	ptr_usecases_InitDeleteConversation --o usecases_DeleteConversation____ptr_usecases_DeleteConversationImpl
	ptr_usecases_InitDeleteTodo --o usecases_DeleteTodo____usecases_DeleteTodoImpl
	ptr_usecases_InitGenerateBoardSummary --o usecases_GenerateBoardSummary____usecases_GenerateBoardSummaryImpl
	ptr_usecases_InitGenerateChatSummary --o usecases_GenerateChatSummary____usecases_GenerateChatSummaryImpl
	ptr_usecases_InitGenerateConversationTitle --o usecases_GenerateConversationTitle____usecases_GenerateConversationTitleImpl
	ptr_usecases_InitGetBoardSummary --o usecases_GetBoardSummary____usecases_GetBoardSummaryImpl
	ptr_usecases_InitListAvailableModels --o usecases_ListAvailableModels____ptr_usecases_ListAvailableModelsImpl
	ptr_usecases_InitListChatMessages --o usecases_ListChatMessages____usecases_ListChatMessagesImpl
	ptr_usecases_InitListConversations --o usecases_ListConversations____ptr_usecases_ListConversationsImpl
	ptr_usecases_InitListTodos --o usecases_ListTodos____usecases_ListTodosImpl
	ptr_usecases_InitRelayOutbox --o usecases_RelayOutbox____usecases_RelayOutboxImpl
	ptr_usecases_InitStreamChat --o usecases_StreamChat____usecases_StreamChatImpl
	ptr_usecases_InitSubmitActionApproval --o usecases_SubmitActionApproval____ptr_usecases_SubmitActionApprovalImpl
	ptr_usecases_InitTodoCreator --o usecases_TodoCreator____usecases_TodoCreatorImpl
	ptr_usecases_InitTodoDeleter --o usecases_TodoDeleter____usecases_TodoDeleterImpl
	ptr_usecases_InitTodoUpdater --o usecases_TodoUpdater____usecases_TodoUpdaterImpl
	ptr_usecases_InitUpdateConversation --o usecases_UpdateConversation____ptr_usecases_UpdateConversationImpl
	ptr_usecases_InitUpdateTodo --o usecases_UpdateTodo____usecases_UpdateTodoImpl
	ptr_workers_ActionApprovalDispatcher --- SymbiontApp
	ptr_workers_BoardSummaryGenerator --- SymbiontApp
	ptr_workers_ChatSummaryGenerator --- SymbiontApp
	ptr_workers_ConversationTitleGenerator --- SymbiontApp
	ptr_workers_MessageRelay --- SymbiontApp
	ACTION_APPROVAL_EVENTS_SUBSCRIPTION_ID -.-> ptr_workers_ActionApprovalDispatcher
	CHAT_EVENTS_SUBSCRIPTION_ID -.-> ptr_workers_ChatSummaryGenerator
	CHAT_SUMMARY_BATCH_INTERVAL -.-> ptr_workers_ChatSummaryGenerator
	CHAT_SUMMARY_BATCH_SIZE -.-> ptr_workers_ChatSummaryGenerator
	CHAT_TITLE_BATCH_INTERVAL -.-> ptr_workers_ConversationTitleGenerator
	CHAT_TITLE_BATCH_SIZE -.-> ptr_workers_ConversationTitleGenerator
	CHAT_TITLE_EVENTS_SUBSCRIPTION_ID -.-> ptr_workers_ConversationTitleGenerator
	DB_HOST -.-> ptr_postgres_InitDB
	DB_NAME -.-> ptr_postgres_InitDB
	DB_PASS -.-> ptr_postgres_InitDB
	DB_PORT -.-> ptr_postgres_InitDB
	DB_USER -.-> ptr_postgres_InitDB
	FETCH_OUTBOX_INTERVAL -.-> ptr_workers_MessageRelay
	GRAPHQL_SERVER_PORT -.-> ptr_graphql_TodoGraphQLServer
	HTTP_PORT -.-> ptr_http_TodoAppServer
	LLM_API_KEY -.-> ptr_modelrunner_InitAssistantClient
	LLM_CHAT_SUMMARY_MODEL -.-> ptr_usecases_InitGenerateChatSummary
	LLM_CHAT_TITLE_MODEL -.-> ptr_usecases_InitGenerateConversationTitle
	LLM_EMBEDDING_API_KEY -.-> ptr_modelrunner_InitAssistantClient
	LLM_EMBEDDING_MODEL -.-> ptr_local_InitLocalActionRegistry
	LLM_EMBEDDING_MODEL -.-> ptr_skillregistry_InitLocalSkillRegistry
	LLM_EMBEDDING_MODEL -.-> ptr_usecases_InitListTodos
	LLM_EMBEDDING_MODEL -.-> ptr_usecases_InitStreamChat
	LLM_EMBEDDING_MODEL -.-> ptr_usecases_InitTodoCreator
	LLM_EMBEDDING_MODEL -.-> ptr_usecases_InitTodoUpdater
	LLM_EMBEDDING_MODEL_HOST -.-> ptr_modelrunner_InitAssistantClient
	LLM_MAX_ACTION_CYCLES -.-> ptr_usecases_InitStreamChat
	LLM_MODEL_HOST -.-> ptr_modelrunner_InitAssistantClient
	LLM_SUMMARY_MODEL -.-> ptr_usecases_InitGenerateBoardSummary
	MCP_GATEWAY_API_KEY -.-> ptr_mcp_InitMCPActionRegistry
	MCP_GATEWAY_API_KEY_HEADER -.-> ptr_mcp_InitMCPActionRegistry
	MCP_GATEWAY_ENDPOINT -.-> ptr_mcp_InitMCPActionRegistry
	MCP_GATEWAY_REQUEST_TIMEOUT -.-> ptr_mcp_InitMCPActionRegistry
	PUBSUB_PROJECT_ID -.-> ptr_pubsub_InitClient
	PUBSUB_PROJECT_ID -.-> ptr_workers_ActionApprovalDispatcher
	SUMMARY_BATCH_INTERVAL -.-> ptr_workers_BoardSummaryGenerator
	SUMMARY_BATCH_SIZE -.-> ptr_workers_BoardSummaryGenerator
	TODO_EVENTS_SUBSCRIPTION_ID -.-> ptr_workers_BoardSummaryGenerator
	domain_Assistant____modelrunner_AssistantClient -.-> ptr_usecases_InitGenerateBoardSummary
	domain_Assistant____modelrunner_AssistantClient -.-> ptr_usecases_InitGenerateChatSummary
	domain_Assistant____modelrunner_AssistantClient -.-> ptr_usecases_InitGenerateConversationTitle
	domain_Assistant____modelrunner_AssistantClient -.-> ptr_usecases_InitStreamChat
	domain_AssistantActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher -.-> ptr_usecases_InitStreamChat
	domain_AssistantActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher -.-> ptr_workers_ActionApprovalDispatcher
	domain_AssistantActionRegistry____composite_CompositeActionRegistry -.-> ptr_usecases_InitStreamChat
	domain_AssistantActionRegistry__local__local_LocalRegistry -.-> ptr_composite_InitCompositeActionRegistry
	domain_AssistantActionRegistry__mcp__ptr_mcp_MCPRegistry -.-> ptr_composite_InitCompositeActionRegistry
	domain_AssistantModelCatalog____modelrunner_AssistantClient -.-> ptr_usecases_InitListAvailableModels
	domain_AssistantSkillRegistry____skillregistry_Registry -.-> ptr_usecases_InitStreamChat
	domain_BoardSummaryRepository____postgres_BoardSummaryRepository -.-> ptr_usecases_InitGenerateBoardSummary
	domain_BoardSummaryRepository____postgres_BoardSummaryRepository -.-> ptr_usecases_InitGetBoardSummary
	domain_ChatMessageRepository____postgres_ChatMessageRepository -.-> ptr_usecases_InitGenerateChatSummary
	domain_ChatMessageRepository____postgres_ChatMessageRepository -.-> ptr_usecases_InitGenerateConversationTitle
	domain_ChatMessageRepository____postgres_ChatMessageRepository -.-> ptr_usecases_InitListChatMessages
	domain_ChatMessageRepository____postgres_ChatMessageRepository -.-> ptr_usecases_InitStreamChat
	domain_ConversationRepository____postgres_ConversationRepository -.-> ptr_usecases_InitGenerateConversationTitle
	domain_ConversationRepository____postgres_ConversationRepository -.-> ptr_usecases_InitListConversations
	domain_ConversationRepository____postgres_ConversationRepository -.-> ptr_usecases_InitStreamChat
	domain_ConversationSummaryRepository____postgres_ConversationSummaryRepository -.-> ptr_usecases_InitGenerateChatSummary
	domain_ConversationSummaryRepository____postgres_ConversationSummaryRepository -.-> ptr_usecases_InitGenerateConversationTitle
	domain_ConversationSummaryRepository____postgres_ConversationSummaryRepository -.-> ptr_usecases_InitStreamChat
	domain_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_local_InitLocalActionRegistry
	domain_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_usecases_InitGenerateBoardSummary
	domain_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_usecases_InitGenerateChatSummary
	domain_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_usecases_InitGenerateConversationTitle
	domain_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_usecases_InitStreamChat
	domain_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_usecases_InitTodoCreator
	domain_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_usecases_InitTodoDeleter
	domain_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_usecases_InitTodoUpdater
	domain_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_usecases_InitUpdateConversation
	domain_EventPublisher____pubsub_PubSubEventPublisher -.-> ptr_usecases_InitRelayOutbox
	domain_EventPublisher____pubsub_PubSubEventPublisher -.-> ptr_usecases_InitSubmitActionApproval
	domain_SemanticEncoder____modelrunner_AssistantClient -.-> ptr_local_InitLocalActionRegistry
	domain_SemanticEncoder____modelrunner_AssistantClient -.-> ptr_skillregistry_InitLocalSkillRegistry
	domain_SemanticEncoder____modelrunner_AssistantClient -.-> ptr_usecases_InitListTodos
	domain_SemanticEncoder____modelrunner_AssistantClient -.-> ptr_usecases_InitTodoCreator
	domain_SemanticEncoder____modelrunner_AssistantClient -.-> ptr_usecases_InitTodoUpdater
	domain_TodoRepository____postgres_TodoRepository -.-> ptr_local_InitLocalActionRegistry
	domain_TodoRepository____postgres_TodoRepository -.-> ptr_usecases_InitListTodos
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_local_InitLocalActionRegistry
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitCreateTodo
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitDeleteConversation
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitDeleteTodo
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitRelayOutbox
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitStreamChat
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitTodoCreator
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitTodoDeleter
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitTodoUpdater
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitUpdateConversation
	domain_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_usecases_InitUpdateTodo
	usecases_CreateTodo____usecases_CreateTodoImpl -.-> ptr_http_TodoAppServer
	usecases_DeleteConversation____ptr_usecases_DeleteConversationImpl -.-> ptr_http_TodoAppServer
	usecases_DeleteTodo____usecases_DeleteTodoImpl -.-> ptr_graphql_TodoGraphQLServer
	usecases_DeleteTodo____usecases_DeleteTodoImpl -.-> ptr_http_TodoAppServer
	usecases_GenerateBoardSummary____usecases_GenerateBoardSummaryImpl -.-> ptr_workers_BoardSummaryGenerator
	usecases_GenerateChatSummary____usecases_GenerateChatSummaryImpl -.-> ptr_workers_ChatSummaryGenerator
	usecases_GenerateConversationTitle____usecases_GenerateConversationTitleImpl -.-> ptr_workers_ConversationTitleGenerator
	usecases_GetBoardSummary____usecases_GetBoardSummaryImpl -.-> ptr_http_TodoAppServer
	usecases_ListAvailableModels____ptr_usecases_ListAvailableModelsImpl -.-> ptr_http_TodoAppServer
	usecases_ListChatMessages____usecases_ListChatMessagesImpl -.-> ptr_http_TodoAppServer
	usecases_ListConversations____ptr_usecases_ListConversationsImpl -.-> ptr_http_TodoAppServer
	usecases_ListTodos____usecases_ListTodosImpl -.-> ptr_graphql_TodoGraphQLServer
	usecases_ListTodos____usecases_ListTodosImpl -.-> ptr_http_TodoAppServer
	usecases_RelayOutbox____usecases_RelayOutboxImpl -.-> ptr_workers_MessageRelay
	usecases_StreamChat____usecases_StreamChatImpl -.-> ptr_http_TodoAppServer
	usecases_SubmitActionApproval____ptr_usecases_SubmitActionApprovalImpl -.-> ptr_http_TodoAppServer
	usecases_TodoCreator____usecases_TodoCreatorImpl -.-> ptr_local_InitLocalActionRegistry
	usecases_TodoCreator____usecases_TodoCreatorImpl -.-> ptr_usecases_InitCreateTodo
	usecases_TodoDeleter____usecases_TodoDeleterImpl -.-> ptr_local_InitLocalActionRegistry
	usecases_TodoDeleter____usecases_TodoDeleterImpl -.-> ptr_usecases_InitDeleteTodo
	usecases_TodoUpdater____usecases_TodoUpdaterImpl -.-> ptr_local_InitLocalActionRegistry
	usecases_TodoUpdater____usecases_TodoUpdaterImpl -.-> ptr_usecases_InitUpdateTodo
	usecases_UpdateConversation____ptr_usecases_UpdateConversationImpl -.-> ptr_http_TodoAppServer
	usecases_UpdateTodo____usecases_UpdateTodoImpl -.-> ptr_graphql_TodoGraphQLServer
	usecases_UpdateTodo____usecases_UpdateTodoImpl -.-> ptr_http_TodoAppServer
	style domain_ChatMessageRepository____postgres_ChatMessageRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_AssistantModelCatalog____modelrunner_AssistantClient fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_SUMMARY_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_TodoCreator____usecases_TodoCreatorImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_ListAvailableModels____ptr_usecases_ListAvailableModelsImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style CHAT_TITLE_BATCH_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_http_Client____ptr_http_Client fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_GenerateConversationTitle____usecases_GenerateConversationTitleImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_UpdateTodo____usecases_UpdateTodoImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style SUMMARY_BATCH_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style TODO_EVENTS_SUBSCRIPTION_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_ListTodos____usecases_ListTodosImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_EMBEDDING_MODEL_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_sql_DB____ptr_sql_DB fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style MCP_GATEWAY_REQUEST_TIMEOUT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style domain_Assistant____modelrunner_AssistantClient fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style DB_USER fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style domain_ConversationRepository____postgres_ConversationRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_AssistantActionRegistry__local__local_LocalRegistry fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_SubmitActionApproval____ptr_usecases_SubmitActionApprovalImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style CHAT_SUMMARY_BATCH_SIZE fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_PASS fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style domain_CurrentTimeProvider____time_CurrentTimeProvider fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_MODEL_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style MCP_GATEWAY_API_KEY_HEADER fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_log_Logger____ptr_log_Logger fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_CreateTodo____usecases_CreateTodoImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_RelayOutbox____usecases_RelayOutboxImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style DB_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_GetBoardSummary____usecases_GetBoardSummaryImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_ConversationSummaryRepository____postgres_ConversationSummaryRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_TodoDeleter____usecases_TodoDeleterImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_AssistantActionRegistry__mcp__ptr_mcp_MCPRegistry fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_DeleteConversation____ptr_usecases_DeleteConversationImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_ListChatMessages____usecases_ListChatMessagesImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_TodoRepository____postgres_TodoRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_StreamChat____usecases_StreamChatImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_GenerateBoardSummary____usecases_GenerateBoardSummaryImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_AssistantActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style SUMMARY_BATCH_SIZE fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ACTION_APPROVAL_EVENTS_SUBSCRIPTION_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_CHAT_TITLE_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style domain_BoardSummaryRepository____postgres_BoardSummaryRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style FETCH_OUTBOX_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_GenerateChatSummary____usecases_GenerateChatSummaryImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_EventPublisher____pubsub_PubSubEventPublisher fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_AssistantSkillRegistry____skillregistry_Registry fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style CHAT_EVENTS_SUBSCRIPTION_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_pubsub_Client____ptr_pubsub_Client fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style MCP_GATEWAY_ENDPOINT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style domain_AssistantActionRegistry____composite_CompositeActionRegistry fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_UnitOfWork____ptr_postgres_UnitOfWork fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style domain_SemanticEncoder____modelrunner_AssistantClient fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_TodoUpdater____usecases_TodoUpdaterImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style usecases_ListConversations____ptr_usecases_ListConversationsImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style MCP_GATEWAY_API_KEY fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style HTTP_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_EMBEDDING_API_KEY fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_DeleteTodo____usecases_DeleteTodoImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_API_KEY fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_MAX_ACTION_CYCLES fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style PUBSUB_PROJECT_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style CHAT_TITLE_EVENTS_SUBSCRIPTION_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_NAME fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style usecases_UpdateConversation____ptr_usecases_UpdateConversationImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style CHAT_SUMMARY_BATCH_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style GRAPHQL_SERVER_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_CHAT_SUMMARY_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style CHAT_TITLE_BATCH_SIZE fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_EMBEDDING_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_usecases_InitListTodos fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitTodoUpdater fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitConversationSummaryRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitTodoCreator fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitCreateTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitBoardSummaryRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_local_InitLocalActionRegistry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitGenerateChatSummary fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitStreamChat fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitConversationRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_pubsub_InitPublisher fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitDB fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitUnitOfWork fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_time_InitCurrentTimeProvider fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_skillregistry_InitLocalSkillRegistry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitGenerateBoardSummary fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitTodoRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitChatMessageRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_approvaldispatcher_InitDispatcher fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitUpdateTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_telemetry_InitOpenTelemetry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_config_InitVaultProvider fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitGetBoardSummary fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_mcp_InitMCPActionRegistry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitGenerateConversationTitle fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_pubsub_InitClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_log_InitLogger fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitSubmitActionApproval fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitTodoDeleter fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitDeleteConversation fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitListChatMessages fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitRelayOutbox fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_composite_InitCompositeActionRegistry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitListAvailableModels fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_modelrunner_InitAssistantClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_telemetry_InitHttpClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitDeleteTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitListConversations fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_usecases_InitUpdateConversation fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_http_TodoAppServer fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_ChatSummaryGenerator fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_ConversationTitleGenerator fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_graphql_TodoGraphQLServer fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_ActionApprovalDispatcher fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_MessageRelay fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_BoardSummaryGenerator fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style SymbiontApp fill:#0f56c4,stroke:#68a4eb,stroke-width:6px,color:#ffffff,font-weight:bold
```
