// src/types/index.ts

export type TodoStatus = 'OPEN' | 'DONE';
export type TodoSearchMode = 'TITLE' | 'SIMILARITY';
export type TodoSortOption =
  | 'createdAtAsc'
  | 'createdAtDesc'
  | 'dueDateAsc'
  | 'dueDateDesc'
  | 'similarityAsc'
  | 'similarityDesc';

export interface Todo {
  id: string;
  title: string;
  status: TodoStatus;
  due_date: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTodoRequest {
  title: string;
  due_date: string;
}

export interface UpdateTodoRequest {
  title?: string;
  status?: TodoStatus;
  due_date?: string;
}

export interface ListTodosResponse {
  items: Todo[];
  page: number;
  previous_page: number | null;
  next_page: number | null;
}

export interface ErrorResponse {
  error: {
    code: 'BAD_REQUEST' | 'NOT_FOUND' | 'INTERNAL_ERROR';
    message: string;
  };
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  created_at: string;
}

export interface ChatHistoryResp {
  conversation_id: string;
  messages: ChatMessage[];
  page: number;
  next_page: number | null;
  previous_page: number | null;
}

export interface ChatStreamRequest {
  message: string;
  model: string;
  conversation_id?: string;
}

export interface ModelListResponse {
  models: string[];
}

export type ConversationTitleSource = 'user' | 'llm' | 'auto';

export interface Conversation {
  id: string;
  title: string;
  title_source: ConversationTitleSource;
  created_at: string;
  updated_at: string;
}

export interface ConversationListResp {
  conversations: Conversation[];
  page: number;
  next_page: number | null;
  previous_page: number | null;
}

export interface ChatStreamMeta {
  conversation_id: string;
  user_message_id: string;
  assistant_message_id: string;
  conversation_created: boolean;
}

export interface ChatStreamDelta {
  text: string;
}

export interface ChatStreamDone {
  assistant_message_id: string;
  completed_at: string;
  usage: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  };
}

export interface AssistantTodoFilters {
  status?: TodoStatus;
  searchQuery?: string;
  searchType?: TodoSearchMode;
  sortBy?: TodoSortOption;
  dueAfter?: string;
  dueBefore?: string;
  page?: number;
  pageSize?: number;
}
