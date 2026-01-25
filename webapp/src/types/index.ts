// src/types/index.ts

export type TodoStatus = 'OPEN' | 'DONE';

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
  role: 'user' | 'assistant';
  content: string;
  created_at: string;
}

export interface ChatHistoryResp {
  messages: ChatMessage[];
  page: number;
  next_page: number | null;
  previous_page: number | null;
}

export interface ChatStreamRequest {
  message: string;
}

export interface ChatStreamMeta {
  conversation_id: string;
  user_message_id: string;
  assistant_message_id: string;
  started_at: string;
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