# Introspection Graph

This document contains the full generated Mermaid graph for the TodoApp Symbiont composition.

Interactive endpoint when running locally: `http://localhost:8080/introspect/`

```mermaid
---
  config:
    layout: elk
---
graph TD
	DB_HOST["<b><span style='font-size:16px'>DB_HOST</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_PORT["<b><span style='font-size:16px'>DB_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	todo_Deleter____todo_DeleterImpl["<b><span style='font-size:16px'>todo.Deleter</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© todo.DeleterImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ todo.InitDeleter.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(todo/init.go:86)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	semantic_Encoder____modelrunner_SemanticEncoder["<b><span style='font-size:16px'>semantic.Encoder</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© modelrunner.SemanticEncoder</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ modelrunner.InitEncoderClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(modelrunner/init.go:41)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	SUMMARY_BATCH_SIZE["<b><span style='font-size:16px'>SUMMARY_BATCH_SIZE</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	core_CurrentTimeProvider____time_CurrentTimeProvider["<b><span style='font-size:16px'>core.CurrentTimeProvider</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© time.CurrentTimeProvider</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ time.InitCurrentTimeProvider.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(time/init.go:15)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_UpdateConversation____ptr_chat_UpdateConversationImpl["<b><span style='font-size:16px'>chat.UpdateConversation</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *chat.UpdateConversationImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitUpdateConversation.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:246)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	LLM_MAX_ACTION_CYCLES["<b><span style='font-size:16px'>LLM_MAX_ACTION_CYCLES</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	todo_List____todo_ListImpl["<b><span style='font-size:16px'>todo.List</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© todo.ListImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ todo.InitListTodos.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(todo/init.go:73)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_StreamChat____chat_StreamChatImpl["<b><span style='font-size:16px'>chat.StreamChat</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© chat.StreamChatImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitStreamChat.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:151)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_ListAvailableSkills____ptr_chat_ListAvailableSkillsImpl["<b><span style='font-size:16px'>chat.ListAvailableSkills</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *chat.ListAvailableSkillsImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitListAvailableSkills.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:95)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	MCP_GATEWAY_API_KEY_HEADER["<b><span style='font-size:16px'>MCP_GATEWAY_API_KEY_HEADER</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	todo_Update____todo_UpdateImpl["<b><span style='font-size:16px'>todo.Update</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© todo.UpdateImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ todo.InitUpdateTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(todo/init.go:104)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	MCP_GATEWAY_API_KEY["<b><span style='font-size:16px'>MCP_GATEWAY_API_KEY</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	TODO_EVENTS_SUBSCRIPTION_ID["<b><span style='font-size:16px'>TODO_EVENTS_SUBSCRIPTION_ID</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	assistant_Tokenizer____tokenizer_DefaultTokenizer["<b><span style='font-size:16px'>assistant.Tokenizer</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© tokenizer.DefaultTokenizer</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ tokenizer.InitTokenizer.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(tokenizer/init.go:15)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_ActionPipeline____chat_ActionPipelineImpl["<b><span style='font-size:16px'>chat.ActionPipeline</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© chat.ActionPipelineImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitActionPipeline.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:180)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	todo_Repository____postgres_TodoRepository["<b><span style='font-size:16px'>todo.Repository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.TodoRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitTodoRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/init.go:76)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	LLM_CHAT_TITLE_MODEL["<b><span style='font-size:16px'>LLM_CHAT_TITLE_MODEL</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_MODEL_HOST["<b><span style='font-size:16px'>LLM_MODEL_HOST</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	CHAT_TITLE_BATCH_SIZE["<b><span style='font-size:16px'>CHAT_TITLE_BATCH_SIZE</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	CHAT_TITLE_EVENTS_SUBSCRIPTION_ID["<b><span style='font-size:16px'>CHAT_TITLE_EVENTS_SUBSCRIPTION_ID</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ACTION_APPROVAL_EVENTS_SUBSCRIPTION_PREFIX["<b><span style='font-size:16px'>ACTION_APPROVAL_EVENTS_SUBSCRIPTION_PREFIX</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_EMBEDDING_MODEL["<b><span style='font-size:16px'>LLM_EMBEDDING_MODEL</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	MCP_GATEWAY_ENDPOINT["<b><span style='font-size:16px'>MCP_GATEWAY_ENDPOINT</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	assistant_ConversationRepository____postgres_ConversationRepository["<b><span style='font-size:16px'>assistant.ConversationRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.ConversationRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitConversationRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/init.go:43)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	assistant_ActionRegistry__mcp__ptr_mcp_ActionRegistry["<b><span style='font-size:16px'>assistant.ActionRegistry</span></b><br/><span style='color:#b26a00;font-size:12px;'>name: mcp</span><br/><span style='color:darkgray;font-size:11px;'>ð§© *mcp.ActionRegistry</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ mcp.(*InitActionRegistry).Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(mcp/init.go:43)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_ConversationCompactor____chat_ConversationCompactorImpl["<b><span style='font-size:16px'>chat.ConversationCompactor</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© chat.ConversationCompactorImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitConversationCompactor.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:44)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	ptr_sql_DB____ptr_sql_DB["<b><span style='font-size:16px'>*sql.DB</span></b><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.(*InitDB).Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/init_db.go:131)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	board_GenerateBoardSummary____board_GenerateBoardSummaryImpl["<b><span style='font-size:16px'>board.GenerateBoardSummary</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© board.GenerateBoardSummaryImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ board.InitGenerateBoardSummary.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(board/init.go:24)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	VAULT_TOKEN["<b><span style='font-size:16px'>VAULT_TOKEN</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	assistant_Assistant____modelrunner_AssistantClient["<b><span style='font-size:16px'>assistant.Assistant</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© modelrunner.AssistantClient</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ modelrunner.InitAssistantClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(modelrunner/init.go:24)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	todo_Creator____todo_CreatorImpl["<b><span style='font-size:16px'>todo.Creator</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© todo.CreatorImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ todo.InitCreator.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(todo/init.go:80)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	CHAT_TITLE_BATCH_INTERVAL["<b><span style='font-size:16px'>CHAT_TITLE_BATCH_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_HEALTH_CHECK_PERIOD["<b><span style='font-size:16px'>DB_HEALTH_CHECK_PERIOD</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_EMBEDDING_API_KEY["<b><span style='font-size:16px'>LLM_EMBEDDING_API_KEY</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	outbox_EventPublisher____pubsub_PubSubEventPublisher["<b><span style='font-size:16px'>outbox.EventPublisher</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© pubsub.PubSubEventPublisher</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ pubsub.(*InitPublisher).Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(pubsub/init.go:52)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	assistant_ActionRegistry____composite_ActionRegistry["<b><span style='font-size:16px'>assistant.ActionRegistry</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© composite.ActionRegistry</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ composite.InitActionRegistry.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(composite/init.go:19)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	todo_Delete____todo_DeleteImpl["<b><span style='font-size:16px'>todo.Delete</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© todo.DeleteImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ todo.InitDeleteTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(todo/init.go:67)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	LLM_SUMMARY_MODEL["<b><span style='font-size:16px'>LLM_SUMMARY_MODEL</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	chat_SubmitActionApproval____ptr_chat_SubmitActionApprovalImpl["<b><span style='font-size:16px'>chat.SubmitActionApproval</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *chat.SubmitActionApprovalImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitSubmitActionApproval.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:234)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	assistant_ModelCatalog____modelrunner_AssistantClient["<b><span style='font-size:16px'>assistant.ModelCatalog</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© modelrunner.AssistantClient</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ modelrunner.InitAssistantClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(modelrunner/init.go:25)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	todo_BoardSummaryRepository____postgres_BoardSummaryRepository["<b><span style='font-size:16px'>todo.BoardSummaryRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.BoardSummaryRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitBoardSummaryRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/init.go:21)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	core_Locker____postgres_AdvisoryLocker["<b><span style='font-size:16px'>core.Locker</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.AdvisoryLocker</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitLocker.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/init.go:54)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	SUMMARY_BATCH_INTERVAL["<b><span style='font-size:16px'>SUMMARY_BATCH_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	CHAT_COMPACTION_TRIGGER_TOKENS["<b><span style='font-size:16px'>CHAT_COMPACTION_TRIGGER_TOKENS</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_MAX_OPEN_CONNS["<b><span style='font-size:16px'>DB_MAX_OPEN_CONNS</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	GRAPHQL_SERVER_PORT["<b><span style='font-size:16px'>GRAPHQL_SERVER_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	OTEL_EXPORTER_OTLP_TRACES_ENDPOINT["<b><span style='font-size:16px'>OTEL_EXPORTER_OTLP_TRACES_ENDPOINT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ptr_http_Client__streaming__ptr_http_Client["<b><span style='font-size:16px'>*http.Client</span></b><br/><span style='color:#b26a00;font-size:12px;'>name: streaming</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ telemetry.InitHttpClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(telemetry/init.go:138)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	assistant_ActionRegistry__local__local_ActionRegistry["<b><span style='font-size:16px'>assistant.ActionRegistry</span></b><br/><span style='color:#b26a00;font-size:12px;'>name: local</span><br/><span style='color:darkgray;font-size:11px;'>ð§© local.ActionRegistry</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ local.InitActionRegistry.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(local/init.go:58)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	assistant_ConversationSummaryRepository____postgres_ConversationSummaryRepository["<b><span style='font-size:16px'>assistant.ConversationSummaryRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.ConversationSummaryRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitConversationSummaryRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/init.go:65)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	API_SERVER_PORT["<b><span style='font-size:16px'>API_SERVER_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_CHAT_SUMMARY_MODEL["<b><span style='font-size:16px'>LLM_CHAT_SUMMARY_MODEL</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	VAULT_ADDR["<b><span style='font-size:16px'>VAULT_ADDR</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	chat_TurnRunner____chat_TurnRunnerImpl["<b><span style='font-size:16px'>chat.TurnRunner</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© chat.TurnRunnerImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitTurnRunner.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:198)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_ListConversations____ptr_chat_ListConversationsImpl["<b><span style='font-size:16px'>chat.ListConversations</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *chat.ListConversationsImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitListConversations.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:119)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	LLM_API_KEY["<b><span style='font-size:16px'>LLM_API_KEY</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	todo_Create____todo_CreateImpl["<b><span style='font-size:16px'>todo.Create</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© todo.CreateImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ todo.InitCreateTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(todo/init.go:60)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_GenerateConversationTitle____chat_GenerateConversationTitleImpl["<b><span style='font-size:16px'>chat.GenerateConversationTitle</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© chat.GenerateConversationTitleImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitGenerateConversationTitle.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:62)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	MCP_GATEWAY_REQUEST_TIMEOUT["<b><span style='font-size:16px'>MCP_GATEWAY_REQUEST_TIMEOUT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	todo_Updater____todo_UpdaterImpl["<b><span style='font-size:16px'>todo.Updater</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© todo.UpdaterImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ todo.InitUpdater.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(todo/init.go:97)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	assistant_SkillRegistry____md_Registry["<b><span style='font-size:16px'>assistant.SkillRegistry</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© md.Registry</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ md.InitSkillRegistry.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(md/init.go:25)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	VAULT_MOUNT_PATH["<b><span style='font-size:16px'>VAULT_MOUNT_PATH</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ptr_log_Logger____ptr_log_Logger["<b><span style='font-size:16px'>*log.Logger</span></b><br/><span style='color:darkblue;font-size:11px;'>ðï¸ log.InitLogger.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(log/init.go:16)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	transaction_UnitOfWork____ptr_postgres_UnitOfWork["<b><span style='font-size:16px'>transaction.UnitOfWork</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *postgres.UnitOfWork</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitUnitOfWork.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/init.go:87)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	assistant_ActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher["<b><span style='font-size:16px'>assistant.ActionApprovalDispatcher</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *approvaldispatcher.Dispatcher</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ approvaldispatcher.InitDispatcher.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(approvaldispatcher/init.go:15)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	DB_PASS["<b><span style='font-size:16px'>DB_PASS</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.VaultProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	chat_ConversationTranscriptWriter____chat_ConversationTranscriptWriterImpl["<b><span style='font-size:16px'>chat.ConversationTranscriptWriter</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© chat.ConversationTranscriptWriterImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitConversationTranscriptWriter.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:163)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_ListChatMessages____chat_ListChatMessagesImpl["<b><span style='font-size:16px'>chat.ListChatMessages</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© chat.ListChatMessagesImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitListChatMessages.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:108)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	PUBSUB_PROJECT_ID["<b><span style='font-size:16px'>PUBSUB_PROJECT_ID</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	ptr_http_Client__standard__ptr_http_Client["<b><span style='font-size:16px'>*http.Client</span></b><br/><span style='color:#b26a00;font-size:12px;'>name: standard</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ telemetry.InitHttpClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(telemetry/init.go:137)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_ListAvailableModels____ptr_chat_ListAvailableModelsImpl["<b><span style='font-size:16px'>chat.ListAvailableModels</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *chat.ListAvailableModelsImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitListAvailableModels.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:82)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	outbox_Relay____outbox_RelayImpl["<b><span style='font-size:16px'>outbox.Relay</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© outbox.RelayImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ outbox.InitRelay.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(outbox/init.go:21)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	DB_MAX_IDLE_CONNS["<b><span style='font-size:16px'>DB_MAX_IDLE_CONNS</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_CONN_MAX_LIFETIME["<b><span style='font-size:16px'>DB_CONN_MAX_LIFETIME</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	assistant_ChatMessageRepository____postgres_ChatMessageRepository["<b><span style='font-size:16px'>assistant.ChatMessageRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© postgres.ChatMessageRepository</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ postgres.InitChatMessageRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(postgres/init.go:32)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	VAULT_SECRET_PATH["<b><span style='font-size:16px'>VAULT_SECRET_PATH</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	board_GetBoardSummary____board_GetBoardSummaryImpl["<b><span style='font-size:16px'>board.GetBoardSummary</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© board.GetBoardSummaryImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ board.InitGetBoardSummary.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(board/init.go:37)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	chat_DeleteConversation____ptr_chat_DeleteConversationImpl["<b><span style='font-size:16px'>chat.DeleteConversation</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© *chat.DeleteConversationImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitDeleteConversation.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:22)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	ptr_pubsub_Client____ptr_pubsub_Client["<b><span style='font-size:16px'>*pubsub.Client</span></b><br/><span style='color:darkblue;font-size:11px;'>ðï¸ pubsub.(*InitClient).Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(pubsub/init.go:33)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	CHAT_COMPACTION_TIMEOUT["<b><span style='font-size:16px'>CHAT_COMPACTION_TIMEOUT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_CONN_MAX_IDLE_TIME["<b><span style='font-size:16px'>DB_CONN_MAX_IDLE_TIME</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_USER["<b><span style='font-size:16px'>DB_USER</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.VaultProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	LLM_EMBEDDING_MODEL_HOST["<b><span style='font-size:16px'>LLM_EMBEDDING_MODEL_HOST</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_MIN_CONNS["<b><span style='font-size:16px'>DB_MIN_CONNS</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	DB_NAME["<b><span style='font-size:16px'>DB_NAME</span></b><br/><span style='color:green;font-size:11px;'>ð«´ð½ config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	FETCH_OUTBOX_INTERVAL["<b><span style='font-size:16px'>FETCH_OUTBOX_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	OTEL_EXPORTER_OTLP_METRICS_ENDPOINT["<b><span style='font-size:16px'>OTEL_EXPORTER_OTLP_METRICS_ENDPOINT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>ð <b>Config</b></span>"]
	chat_TurnStateBuilder____chat_TurnStateBuilderImpl["<b><span style='font-size:16px'>chat.TurnStateBuilder</span></b><br/><span style='color:darkgray;font-size:11px;'>ð§© chat.TurnStateBuilderImpl</span><br/><span style='color:darkblue;font-size:11px;'>ðï¸ chat.InitTurnStateBuilder.Initialize</span><br/><span style='color:gray;font-size:11px;'>ð(chat/init.go:217)</span><br/><span style='color:green;font-size:11px;'>ð <b>Dependency</b></span>"]
	ptr_mcp_InitActionRegistry["<b><span style='font-size:15px'>*mcp.InitActionRegistry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_todo_InitUpdateTodo["<b><span style='font-size:15px'>*todo.InitUpdateTodo</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitListAvailableSkills["<b><span style='font-size:15px'>*chat.InitListAvailableSkills</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_modelrunner_InitEncoderClient["<b><span style='font-size:15px'>*modelrunner.InitEncoderClient</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitActionPipeline["<b><span style='font-size:15px'>*chat.InitActionPipeline</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitListChatMessages["<b><span style='font-size:15px'>*chat.InitListChatMessages</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_telemetry_InitHttpClient["<b><span style='font-size:15px'>*telemetry.InitHttpClient</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitTurnRunner["<b><span style='font-size:15px'>*chat.InitTurnRunner</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_pubsub_InitClient["<b><span style='font-size:15px'>*pubsub.InitClient</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitBoardSummaryRepository["<b><span style='font-size:15px'>*postgres.InitBoardSummaryRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_board_InitGenerateBoardSummary["<b><span style='font-size:15px'>*board.InitGenerateBoardSummary</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitChatMessageRepository["<b><span style='font-size:15px'>*postgres.InitChatMessageRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_composite_InitActionRegistry["<b><span style='font-size:15px'>*composite.InitActionRegistry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitConversationRepository["<b><span style='font-size:15px'>*postgres.InitConversationRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitTurnStateBuilder["<b><span style='font-size:15px'>*chat.InitTurnStateBuilder</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_todo_InitDeleter["<b><span style='font-size:15px'>*todo.InitDeleter</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitSubmitActionApproval["<b><span style='font-size:15px'>*chat.InitSubmitActionApproval</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_todo_InitCreator["<b><span style='font-size:15px'>*todo.InitCreator</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_local_InitActionRegistry["<b><span style='font-size:15px'>*local.InitActionRegistry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_config_InitVaultProvider["<b><span style='font-size:16px'>*config.InitVaultProvider</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_pubsub_InitPublisher["<b><span style='font-size:15px'>*pubsub.InitPublisher</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitListConversations["<b><span style='font-size:15px'>*chat.InitListConversations</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_outbox_InitRelay["<b><span style='font-size:15px'>*outbox.InitRelay</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitConversationCompactor["<b><span style='font-size:15px'>*chat.InitConversationCompactor</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_log_InitLogger["<b><span style='font-size:16px'>*log.InitLogger</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitUnitOfWork["<b><span style='font-size:15px'>*postgres.InitUnitOfWork</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitConversationSummaryRepository["<b><span style='font-size:15px'>*postgres.InitConversationSummaryRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_time_InitCurrentTimeProvider["<b><span style='font-size:16px'>*time.InitCurrentTimeProvider</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_todo_InitCreateTodo["<b><span style='font-size:15px'>*todo.InitCreateTodo</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_board_InitGetBoardSummary["<b><span style='font-size:15px'>*board.InitGetBoardSummary</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_telemetry_InitOpenTelemetry["<b><span style='font-size:15px'>*telemetry.InitOpenTelemetry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitStreamChat["<b><span style='font-size:15px'>*chat.InitStreamChat</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitDeleteConversation["<b><span style='font-size:15px'>*chat.InitDeleteConversation</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitDB["<b><span style='font-size:15px'>*postgres.InitDB</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_md_InitSkillRegistry["<b><span style='font-size:15px'>*md.InitSkillRegistry</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_todo_InitDeleteTodo["<b><span style='font-size:15px'>*todo.InitDeleteTodo</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitTodoRepository["<b><span style='font-size:15px'>*postgres.InitTodoRepository</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitConversationTranscriptWriter["<b><span style='font-size:15px'>*chat.InitConversationTranscriptWriter</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitGenerateConversationTitle["<b><span style='font-size:15px'>*chat.InitGenerateConversationTitle</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitListAvailableModels["<b><span style='font-size:15px'>*chat.InitListAvailableModels</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_chat_InitUpdateConversation["<b><span style='font-size:15px'>*chat.InitUpdateConversation</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_approvaldispatcher_InitDispatcher["<b><span style='font-size:16px'>*approvaldispatcher.InitDispatcher</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_modelrunner_InitAssistantClient["<b><span style='font-size:15px'>*modelrunner.InitAssistantClient</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_todo_InitUpdater["<b><span style='font-size:15px'>*todo.InitUpdater</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_tokenizer_InitTokenizer["<b><span style='font-size:16px'>*tokenizer.InitTokenizer</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_todo_InitListTodos["<b><span style='font-size:15px'>*todo.InitListTodos</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_postgres_InitLocker["<b><span style='font-size:15px'>*postgres.InitLocker</span></b><br/><span style='color:green;font-size:11px;'>ð¦ <b>Initializer</b></span>"]
	ptr_workers_MessageRelay["<b><span style='font-size:16px'>*workers.MessageRelay</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_graphql_TodoGraphQLServer["<b><span style='font-size:16px'>*graphql.TodoGraphQLServer</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_http_TodoAppServer["<b><span style='font-size:16px'>*http.TodoAppServer</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_workers_BoardSummaryGenerator["<b><span style='font-size:16px'>*workers.BoardSummaryGenerator</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_workers_ActionApprovalDispatcher["<b><span style='font-size:16px'>*workers.ActionApprovalDispatcher</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	ptr_workers_ConversationTitleGenerator["<b><span style='font-size:16px'>*workers.ConversationTitleGenerator</span></b><br/><span style='color:green;font-size:11px;'>âï¸ <b>Runnable</b></span>"]
	SymbiontApp["<b><span style='font-size:20px;color:white'>ð Symbiont App</span></b>"]
	ptr_approvaldispatcher_InitDispatcher --o assistant_ActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher
	ptr_board_InitGenerateBoardSummary --o board_GenerateBoardSummary____board_GenerateBoardSummaryImpl
	ptr_board_InitGetBoardSummary --o board_GetBoardSummary____board_GetBoardSummaryImpl
	ptr_chat_InitActionPipeline --o chat_ActionPipeline____chat_ActionPipelineImpl
	ptr_chat_InitConversationCompactor --o chat_ConversationCompactor____chat_ConversationCompactorImpl
	ptr_chat_InitConversationTranscriptWriter --o chat_ConversationTranscriptWriter____chat_ConversationTranscriptWriterImpl
	ptr_chat_InitDeleteConversation --o chat_DeleteConversation____ptr_chat_DeleteConversationImpl
	ptr_chat_InitGenerateConversationTitle --o chat_GenerateConversationTitle____chat_GenerateConversationTitleImpl
	ptr_chat_InitListAvailableModels --o chat_ListAvailableModels____ptr_chat_ListAvailableModelsImpl
	ptr_chat_InitListAvailableSkills --o chat_ListAvailableSkills____ptr_chat_ListAvailableSkillsImpl
	ptr_chat_InitListChatMessages --o chat_ListChatMessages____chat_ListChatMessagesImpl
	ptr_chat_InitListConversations --o chat_ListConversations____ptr_chat_ListConversationsImpl
	ptr_chat_InitStreamChat --o chat_StreamChat____chat_StreamChatImpl
	ptr_chat_InitSubmitActionApproval --o chat_SubmitActionApproval____ptr_chat_SubmitActionApprovalImpl
	ptr_chat_InitTurnRunner --o chat_TurnRunner____chat_TurnRunnerImpl
	ptr_chat_InitTurnStateBuilder --o chat_TurnStateBuilder____chat_TurnStateBuilderImpl
	ptr_chat_InitUpdateConversation --o chat_UpdateConversation____ptr_chat_UpdateConversationImpl
	ptr_composite_InitActionRegistry --o assistant_ActionRegistry____composite_ActionRegistry
	ptr_graphql_TodoGraphQLServer --- SymbiontApp
	ptr_http_Client__standard__ptr_http_Client -.-> ptr_mcp_InitActionRegistry
	ptr_http_Client__streaming__ptr_http_Client -.-> ptr_modelrunner_InitAssistantClient
	ptr_http_Client__streaming__ptr_http_Client -.-> ptr_modelrunner_InitEncoderClient
	ptr_http_TodoAppServer --- SymbiontApp
	ptr_local_InitActionRegistry --o assistant_ActionRegistry__local__local_ActionRegistry
	ptr_log_InitLogger --o ptr_log_Logger____ptr_log_Logger
	ptr_log_Logger____ptr_log_Logger -.-> ptr_chat_InitStreamChat
	ptr_log_Logger____ptr_log_Logger -.-> ptr_chat_InitTurnRunner
	ptr_log_Logger____ptr_log_Logger -.-> ptr_graphql_TodoGraphQLServer
	ptr_log_Logger____ptr_log_Logger -.-> ptr_http_TodoAppServer
	ptr_log_Logger____ptr_log_Logger -.-> ptr_mcp_InitActionRegistry
	ptr_log_Logger____ptr_log_Logger -.-> ptr_outbox_InitRelay
	ptr_log_Logger____ptr_log_Logger -.-> ptr_postgres_InitDB
	ptr_log_Logger____ptr_log_Logger -.-> ptr_pubsub_InitClient
	ptr_log_Logger____ptr_log_Logger -.-> ptr_telemetry_InitHttpClient
	ptr_log_Logger____ptr_log_Logger -.-> ptr_telemetry_InitOpenTelemetry
	ptr_log_Logger____ptr_log_Logger -.-> ptr_workers_ActionApprovalDispatcher
	ptr_log_Logger____ptr_log_Logger -.-> ptr_workers_BoardSummaryGenerator
	ptr_log_Logger____ptr_log_Logger -.-> ptr_workers_ConversationTitleGenerator
	ptr_log_Logger____ptr_log_Logger -.-> ptr_workers_MessageRelay
	ptr_mcp_InitActionRegistry --o assistant_ActionRegistry__mcp__ptr_mcp_ActionRegistry
	ptr_md_InitSkillRegistry --o assistant_SkillRegistry____md_Registry
	ptr_modelrunner_InitAssistantClient --o assistant_Assistant____modelrunner_AssistantClient
	ptr_modelrunner_InitAssistantClient --o assistant_ModelCatalog____modelrunner_AssistantClient
	ptr_modelrunner_InitEncoderClient --o semantic_Encoder____modelrunner_SemanticEncoder
	ptr_outbox_InitRelay --o outbox_Relay____outbox_RelayImpl
	ptr_postgres_InitBoardSummaryRepository --o todo_BoardSummaryRepository____postgres_BoardSummaryRepository
	ptr_postgres_InitChatMessageRepository --o assistant_ChatMessageRepository____postgres_ChatMessageRepository
	ptr_postgres_InitConversationRepository --o assistant_ConversationRepository____postgres_ConversationRepository
	ptr_postgres_InitConversationSummaryRepository --o assistant_ConversationSummaryRepository____postgres_ConversationSummaryRepository
	ptr_postgres_InitDB --o ptr_sql_DB____ptr_sql_DB
	ptr_postgres_InitLocker --o core_Locker____postgres_AdvisoryLocker
	ptr_postgres_InitTodoRepository --o todo_Repository____postgres_TodoRepository
	ptr_postgres_InitUnitOfWork --o transaction_UnitOfWork____ptr_postgres_UnitOfWork
	ptr_pubsub_Client____ptr_pubsub_Client -.-> ptr_pubsub_InitPublisher
	ptr_pubsub_Client____ptr_pubsub_Client -.-> ptr_workers_ActionApprovalDispatcher
	ptr_pubsub_Client____ptr_pubsub_Client -.-> ptr_workers_BoardSummaryGenerator
	ptr_pubsub_Client____ptr_pubsub_Client -.-> ptr_workers_ConversationTitleGenerator
	ptr_pubsub_InitClient --o ptr_pubsub_Client____ptr_pubsub_Client
	ptr_pubsub_InitPublisher --o outbox_EventPublisher____pubsub_PubSubEventPublisher
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitBoardSummaryRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitChatMessageRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitConversationRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitConversationSummaryRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitLocker
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitTodoRepository
	ptr_sql_DB____ptr_sql_DB -.-> ptr_postgres_InitUnitOfWork
	ptr_telemetry_InitHttpClient --o ptr_http_Client__standard__ptr_http_Client
	ptr_telemetry_InitHttpClient --o ptr_http_Client__streaming__ptr_http_Client
	ptr_time_InitCurrentTimeProvider --o core_CurrentTimeProvider____time_CurrentTimeProvider
	ptr_todo_InitCreateTodo --o todo_Create____todo_CreateImpl
	ptr_todo_InitCreator --o todo_Creator____todo_CreatorImpl
	ptr_todo_InitDeleteTodo --o todo_Delete____todo_DeleteImpl
	ptr_todo_InitDeleter --o todo_Deleter____todo_DeleterImpl
	ptr_todo_InitListTodos --o todo_List____todo_ListImpl
	ptr_todo_InitUpdateTodo --o todo_Update____todo_UpdateImpl
	ptr_todo_InitUpdater --o todo_Updater____todo_UpdaterImpl
	ptr_tokenizer_InitTokenizer --o assistant_Tokenizer____tokenizer_DefaultTokenizer
	ptr_workers_ActionApprovalDispatcher --- SymbiontApp
	ptr_workers_BoardSummaryGenerator --- SymbiontApp
	ptr_workers_ConversationTitleGenerator --- SymbiontApp
	ptr_workers_MessageRelay --- SymbiontApp
	ACTION_APPROVAL_EVENTS_SUBSCRIPTION_PREFIX -.-> ptr_workers_ActionApprovalDispatcher
	API_SERVER_PORT -.-> ptr_http_TodoAppServer
	CHAT_COMPACTION_TIMEOUT -.-> ptr_chat_InitStreamChat
	CHAT_COMPACTION_TRIGGER_TOKENS -.-> ptr_chat_InitStreamChat
	CHAT_COMPACTION_TRIGGER_TOKENS -.-> ptr_http_TodoAppServer
	CHAT_TITLE_BATCH_INTERVAL -.-> ptr_workers_ConversationTitleGenerator
	CHAT_TITLE_BATCH_SIZE -.-> ptr_workers_ConversationTitleGenerator
	CHAT_TITLE_EVENTS_SUBSCRIPTION_ID -.-> ptr_workers_ConversationTitleGenerator
	DB_CONN_MAX_IDLE_TIME -.-> ptr_postgres_InitDB
	DB_CONN_MAX_LIFETIME -.-> ptr_postgres_InitDB
	DB_HEALTH_CHECK_PERIOD -.-> ptr_postgres_InitDB
	DB_HOST -.-> ptr_postgres_InitDB
	DB_MAX_IDLE_CONNS -.-> ptr_postgres_InitDB
	DB_MAX_OPEN_CONNS -.-> ptr_postgres_InitDB
	DB_MIN_CONNS -.-> ptr_postgres_InitDB
	DB_NAME -.-> ptr_postgres_InitDB
	DB_PASS -.-> ptr_postgres_InitDB
	DB_PORT -.-> ptr_postgres_InitDB
	DB_USER -.-> ptr_postgres_InitDB
	FETCH_OUTBOX_INTERVAL -.-> ptr_workers_MessageRelay
	GRAPHQL_SERVER_PORT -.-> ptr_graphql_TodoGraphQLServer
	LLM_API_KEY -.-> ptr_modelrunner_InitAssistantClient
	LLM_CHAT_SUMMARY_MODEL -.-> ptr_chat_InitConversationCompactor
	LLM_CHAT_TITLE_MODEL -.-> ptr_chat_InitGenerateConversationTitle
	LLM_EMBEDDING_API_KEY -.-> ptr_modelrunner_InitEncoderClient
	LLM_EMBEDDING_MODEL -.-> ptr_local_InitActionRegistry
	LLM_EMBEDDING_MODEL -.-> ptr_md_InitSkillRegistry
	LLM_EMBEDDING_MODEL -.-> ptr_todo_InitCreator
	LLM_EMBEDDING_MODEL -.-> ptr_todo_InitListTodos
	LLM_EMBEDDING_MODEL -.-> ptr_todo_InitUpdater
	LLM_EMBEDDING_MODEL_HOST -.-> ptr_modelrunner_InitEncoderClient
	LLM_MAX_ACTION_CYCLES -.-> ptr_chat_InitStreamChat
	LLM_MODEL_HOST -.-> ptr_modelrunner_InitAssistantClient
	LLM_SUMMARY_MODEL -.-> ptr_board_InitGenerateBoardSummary
	MCP_GATEWAY_API_KEY -.-> ptr_mcp_InitActionRegistry
	MCP_GATEWAY_API_KEY_HEADER -.-> ptr_mcp_InitActionRegistry
	MCP_GATEWAY_ENDPOINT -.-> ptr_mcp_InitActionRegistry
	MCP_GATEWAY_REQUEST_TIMEOUT -.-> ptr_mcp_InitActionRegistry
	OTEL_EXPORTER_OTLP_METRICS_ENDPOINT -.-> ptr_telemetry_InitOpenTelemetry
	OTEL_EXPORTER_OTLP_TRACES_ENDPOINT -.-> ptr_telemetry_InitOpenTelemetry
	PUBSUB_PROJECT_ID -.-> ptr_pubsub_InitClient
	PUBSUB_PROJECT_ID -.-> ptr_workers_ActionApprovalDispatcher
	SUMMARY_BATCH_INTERVAL -.-> ptr_workers_BoardSummaryGenerator
	SUMMARY_BATCH_SIZE -.-> ptr_workers_BoardSummaryGenerator
	TODO_EVENTS_SUBSCRIPTION_ID -.-> ptr_workers_BoardSummaryGenerator
	VAULT_ADDR -.-> ptr_config_InitVaultProvider
	VAULT_MOUNT_PATH -.-> ptr_config_InitVaultProvider
	VAULT_SECRET_PATH -.-> ptr_config_InitVaultProvider
	VAULT_TOKEN -.-> ptr_config_InitVaultProvider
	assistant_ActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher -.-> ptr_chat_InitActionPipeline
	assistant_ActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher -.-> ptr_workers_ActionApprovalDispatcher
	assistant_ActionRegistry____composite_ActionRegistry -.-> ptr_chat_InitActionPipeline
	assistant_ActionRegistry____composite_ActionRegistry -.-> ptr_chat_InitTurnStateBuilder
	assistant_ActionRegistry__local__local_ActionRegistry -.-> ptr_composite_InitActionRegistry
	assistant_ActionRegistry__mcp__ptr_mcp_ActionRegistry -.-> ptr_composite_InitActionRegistry
	assistant_Assistant____modelrunner_AssistantClient -.-> ptr_board_InitGenerateBoardSummary
	assistant_Assistant____modelrunner_AssistantClient -.-> ptr_chat_InitConversationCompactor
	assistant_Assistant____modelrunner_AssistantClient -.-> ptr_chat_InitGenerateConversationTitle
	assistant_Assistant____modelrunner_AssistantClient -.-> ptr_chat_InitTurnRunner
	assistant_ChatMessageRepository____postgres_ChatMessageRepository -.-> ptr_chat_InitConversationCompactor
	assistant_ChatMessageRepository____postgres_ChatMessageRepository -.-> ptr_chat_InitGenerateConversationTitle
	assistant_ChatMessageRepository____postgres_ChatMessageRepository -.-> ptr_chat_InitListChatMessages
	assistant_ChatMessageRepository____postgres_ChatMessageRepository -.-> ptr_chat_InitTurnStateBuilder
	assistant_ConversationRepository____postgres_ConversationRepository -.-> ptr_chat_InitGenerateConversationTitle
	assistant_ConversationRepository____postgres_ConversationRepository -.-> ptr_chat_InitListConversations
	assistant_ConversationRepository____postgres_ConversationRepository -.-> ptr_chat_InitStreamChat
	assistant_ConversationRepository____postgres_ConversationRepository -.-> ptr_http_TodoAppServer
	assistant_ConversationSummaryRepository____postgres_ConversationSummaryRepository -.-> ptr_chat_InitConversationCompactor
	assistant_ConversationSummaryRepository____postgres_ConversationSummaryRepository -.-> ptr_chat_InitGenerateConversationTitle
	assistant_ConversationSummaryRepository____postgres_ConversationSummaryRepository -.-> ptr_chat_InitTurnStateBuilder
	assistant_ModelCatalog____modelrunner_AssistantClient -.-> ptr_chat_InitListAvailableModels
	assistant_SkillRegistry____md_Registry -.-> ptr_chat_InitListAvailableSkills
	assistant_SkillRegistry____md_Registry -.-> ptr_chat_InitTurnStateBuilder
	assistant_Tokenizer____tokenizer_DefaultTokenizer -.-> ptr_chat_InitConversationTranscriptWriter
	board_GenerateBoardSummary____board_GenerateBoardSummaryImpl -.-> ptr_workers_BoardSummaryGenerator
	board_GetBoardSummary____board_GetBoardSummaryImpl -.-> ptr_http_TodoAppServer
	chat_ActionPipeline____chat_ActionPipelineImpl -.-> ptr_chat_InitTurnRunner
	chat_ConversationCompactor____chat_ConversationCompactorImpl -.-> ptr_chat_InitStreamChat
	chat_ConversationTranscriptWriter____chat_ConversationTranscriptWriterImpl -.-> ptr_chat_InitActionPipeline
	chat_ConversationTranscriptWriter____chat_ConversationTranscriptWriterImpl -.-> ptr_chat_InitStreamChat
	chat_DeleteConversation____ptr_chat_DeleteConversationImpl -.-> ptr_http_TodoAppServer
	chat_GenerateConversationTitle____chat_GenerateConversationTitleImpl -.-> ptr_workers_ConversationTitleGenerator
	chat_ListAvailableModels____ptr_chat_ListAvailableModelsImpl -.-> ptr_http_TodoAppServer
	chat_ListAvailableSkills____ptr_chat_ListAvailableSkillsImpl -.-> ptr_http_TodoAppServer
	chat_ListChatMessages____chat_ListChatMessagesImpl -.-> ptr_http_TodoAppServer
	chat_ListConversations____ptr_chat_ListConversationsImpl -.-> ptr_http_TodoAppServer
	chat_StreamChat____chat_StreamChatImpl -.-> ptr_http_TodoAppServer
	chat_SubmitActionApproval____ptr_chat_SubmitActionApprovalImpl -.-> ptr_http_TodoAppServer
	chat_TurnRunner____chat_TurnRunnerImpl -.-> ptr_chat_InitStreamChat
	chat_TurnStateBuilder____chat_TurnStateBuilderImpl -.-> ptr_chat_InitStreamChat
	chat_UpdateConversation____ptr_chat_UpdateConversationImpl -.-> ptr_http_TodoAppServer
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_board_InitGenerateBoardSummary
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_chat_InitActionPipeline
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_chat_InitConversationCompactor
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_chat_InitGenerateConversationTitle
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_chat_InitStreamChat
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_chat_InitTurnStateBuilder
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_chat_InitUpdateConversation
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_local_InitActionRegistry
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_todo_InitCreator
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_todo_InitDeleter
	core_CurrentTimeProvider____time_CurrentTimeProvider -.-> ptr_todo_InitUpdater
	core_Locker____postgres_AdvisoryLocker -.-> ptr_board_InitGenerateBoardSummary
	core_Locker____postgres_AdvisoryLocker -.-> ptr_chat_InitGenerateConversationTitle
	outbox_EventPublisher____pubsub_PubSubEventPublisher -.-> ptr_chat_InitSubmitActionApproval
	outbox_EventPublisher____pubsub_PubSubEventPublisher -.-> ptr_outbox_InitRelay
	outbox_Relay____outbox_RelayImpl -.-> ptr_workers_MessageRelay
	semantic_Encoder____modelrunner_SemanticEncoder -.-> ptr_local_InitActionRegistry
	semantic_Encoder____modelrunner_SemanticEncoder -.-> ptr_md_InitSkillRegistry
	semantic_Encoder____modelrunner_SemanticEncoder -.-> ptr_todo_InitCreator
	semantic_Encoder____modelrunner_SemanticEncoder -.-> ptr_todo_InitListTodos
	semantic_Encoder____modelrunner_SemanticEncoder -.-> ptr_todo_InitUpdater
	todo_BoardSummaryRepository____postgres_BoardSummaryRepository -.-> ptr_board_InitGenerateBoardSummary
	todo_BoardSummaryRepository____postgres_BoardSummaryRepository -.-> ptr_board_InitGetBoardSummary
	todo_Create____todo_CreateImpl -.-> ptr_http_TodoAppServer
	todo_Creator____todo_CreatorImpl -.-> ptr_local_InitActionRegistry
	todo_Creator____todo_CreatorImpl -.-> ptr_todo_InitCreateTodo
	todo_Delete____todo_DeleteImpl -.-> ptr_graphql_TodoGraphQLServer
	todo_Delete____todo_DeleteImpl -.-> ptr_http_TodoAppServer
	todo_Deleter____todo_DeleterImpl -.-> ptr_local_InitActionRegistry
	todo_Deleter____todo_DeleterImpl -.-> ptr_todo_InitDeleteTodo
	todo_List____todo_ListImpl -.-> ptr_graphql_TodoGraphQLServer
	todo_List____todo_ListImpl -.-> ptr_http_TodoAppServer
	todo_Repository____postgres_TodoRepository -.-> ptr_local_InitActionRegistry
	todo_Repository____postgres_TodoRepository -.-> ptr_todo_InitListTodos
	todo_Update____todo_UpdateImpl -.-> ptr_graphql_TodoGraphQLServer
	todo_Update____todo_UpdateImpl -.-> ptr_http_TodoAppServer
	todo_Updater____todo_UpdaterImpl -.-> ptr_local_InitActionRegistry
	todo_Updater____todo_UpdaterImpl -.-> ptr_todo_InitUpdateTodo
	transaction_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_chat_InitConversationTranscriptWriter
	transaction_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_chat_InitDeleteConversation
	transaction_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_chat_InitUpdateConversation
	transaction_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_local_InitActionRegistry
	transaction_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_outbox_InitRelay
	transaction_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_todo_InitCreateTodo
	transaction_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_todo_InitDeleteTodo
	transaction_UnitOfWork____ptr_postgres_UnitOfWork -.-> ptr_todo_InitUpdateTodo
	style DB_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style todo_Deleter____todo_DeleterImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style semantic_Encoder____modelrunner_SemanticEncoder fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style SUMMARY_BATCH_SIZE fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style core_CurrentTimeProvider____time_CurrentTimeProvider fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_UpdateConversation____ptr_chat_UpdateConversationImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_MAX_ACTION_CYCLES fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style todo_List____todo_ListImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_StreamChat____chat_StreamChatImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_ListAvailableSkills____ptr_chat_ListAvailableSkillsImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style MCP_GATEWAY_API_KEY_HEADER fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style todo_Update____todo_UpdateImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style MCP_GATEWAY_API_KEY fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style TODO_EVENTS_SUBSCRIPTION_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style assistant_Tokenizer____tokenizer_DefaultTokenizer fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_ActionPipeline____chat_ActionPipelineImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style todo_Repository____postgres_TodoRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_CHAT_TITLE_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_MODEL_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style CHAT_TITLE_BATCH_SIZE fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style CHAT_TITLE_EVENTS_SUBSCRIPTION_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ACTION_APPROVAL_EVENTS_SUBSCRIPTION_PREFIX fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_EMBEDDING_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style MCP_GATEWAY_ENDPOINT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style assistant_ConversationRepository____postgres_ConversationRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style assistant_ActionRegistry__mcp__ptr_mcp_ActionRegistry fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_ConversationCompactor____chat_ConversationCompactorImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style ptr_sql_DB____ptr_sql_DB fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style board_GenerateBoardSummary____board_GenerateBoardSummaryImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style VAULT_TOKEN fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style assistant_Assistant____modelrunner_AssistantClient fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style todo_Creator____todo_CreatorImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style CHAT_TITLE_BATCH_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_HEALTH_CHECK_PERIOD fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_EMBEDDING_API_KEY fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style outbox_EventPublisher____pubsub_PubSubEventPublisher fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style assistant_ActionRegistry____composite_ActionRegistry fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style todo_Delete____todo_DeleteImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_SUMMARY_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style chat_SubmitActionApproval____ptr_chat_SubmitActionApprovalImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style assistant_ModelCatalog____modelrunner_AssistantClient fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style todo_BoardSummaryRepository____postgres_BoardSummaryRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style core_Locker____postgres_AdvisoryLocker fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style SUMMARY_BATCH_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style CHAT_COMPACTION_TRIGGER_TOKENS fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_MAX_OPEN_CONNS fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style GRAPHQL_SERVER_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style OTEL_EXPORTER_OTLP_TRACES_ENDPOINT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_http_Client__streaming__ptr_http_Client fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style assistant_ActionRegistry__local__local_ActionRegistry fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style assistant_ConversationSummaryRepository____postgres_ConversationSummaryRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style API_SERVER_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_CHAT_SUMMARY_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style VAULT_ADDR fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style chat_TurnRunner____chat_TurnRunnerImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_ListConversations____ptr_chat_ListConversationsImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style LLM_API_KEY fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style todo_Create____todo_CreateImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_GenerateConversationTitle____chat_GenerateConversationTitleImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style MCP_GATEWAY_REQUEST_TIMEOUT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style todo_Updater____todo_UpdaterImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style assistant_SkillRegistry____md_Registry fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style VAULT_MOUNT_PATH fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_log_Logger____ptr_log_Logger fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style transaction_UnitOfWork____ptr_postgres_UnitOfWork fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style assistant_ActionApprovalDispatcher____ptr_approvaldispatcher_Dispatcher fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style DB_PASS fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style chat_ConversationTranscriptWriter____chat_ConversationTranscriptWriterImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_ListChatMessages____chat_ListChatMessagesImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style PUBSUB_PROJECT_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style ptr_http_Client__standard__ptr_http_Client fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_ListAvailableModels____ptr_chat_ListAvailableModelsImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style outbox_Relay____outbox_RelayImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style DB_MAX_IDLE_CONNS fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_CONN_MAX_LIFETIME fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style assistant_ChatMessageRepository____postgres_ChatMessageRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style VAULT_SECRET_PATH fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style board_GetBoardSummary____board_GetBoardSummaryImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style chat_DeleteConversation____ptr_chat_DeleteConversationImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style ptr_pubsub_Client____ptr_pubsub_Client fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style CHAT_COMPACTION_TIMEOUT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_CONN_MAX_IDLE_TIME fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_USER fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style LLM_EMBEDDING_MODEL_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_MIN_CONNS fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style DB_NAME fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style FETCH_OUTBOX_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style OTEL_EXPORTER_OTLP_METRICS_ENDPOINT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
	style chat_TurnStateBuilder____chat_TurnStateBuilderImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
	style ptr_mcp_InitActionRegistry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_todo_InitUpdateTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitListAvailableSkills fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_modelrunner_InitEncoderClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitActionPipeline fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitListChatMessages fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_telemetry_InitHttpClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitTurnRunner fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_pubsub_InitClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitBoardSummaryRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_board_InitGenerateBoardSummary fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitChatMessageRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_composite_InitActionRegistry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitConversationRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitTurnStateBuilder fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_todo_InitDeleter fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitSubmitActionApproval fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_todo_InitCreator fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_local_InitActionRegistry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_config_InitVaultProvider fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_pubsub_InitPublisher fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitListConversations fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_outbox_InitRelay fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitConversationCompactor fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_log_InitLogger fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitUnitOfWork fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitConversationSummaryRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_time_InitCurrentTimeProvider fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_todo_InitCreateTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_board_InitGetBoardSummary fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_telemetry_InitOpenTelemetry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitStreamChat fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitDeleteConversation fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitDB fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_md_InitSkillRegistry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_todo_InitDeleteTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitTodoRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitConversationTranscriptWriter fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitGenerateConversationTitle fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitListAvailableModels fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_chat_InitUpdateConversation fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_approvaldispatcher_InitDispatcher fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_modelrunner_InitAssistantClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_todo_InitUpdater fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_tokenizer_InitTokenizer fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_todo_InitListTodos fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_postgres_InitLocker fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
	style ptr_workers_MessageRelay fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_graphql_TodoGraphQLServer fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_http_TodoAppServer fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_BoardSummaryGenerator fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_ActionApprovalDispatcher fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style ptr_workers_ConversationTitleGenerator fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
	style SymbiontApp fill:#0f56c4,stroke:#68a4eb,stroke-width:6px,color:#ffffff,font-weight:bold
```
