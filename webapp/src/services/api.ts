import axios from 'axios';
import type { Todo, CreateTodoRequest, UpdateTodoRequest, ListTodosResponse, TodoStatus } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add response interceptor to handle errors
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error);
    
    if (error.response) {
      const errorData = error.response.data?.error;
      const message = errorData?.message || error.response.statusText || 'An error occurred';
      const status = error.response.status;
      throw new Error(`[${status}] ${message}`);
    } else if (error.request) {
      throw new Error('No response from server');
    } else {
      throw new Error(error.message);
    }
  }
);

export const getTodos = async (
  status?: string,
  page: number = 1,
  pagesize: number = 50
): Promise<ListTodosResponse> => {
  const params: Record<string, any> = {
    page,
    pagesize,
  };
  
  if (status) {
    params.status = status;
  }

  const response = await apiClient.get<ListTodosResponse>('/api/v1/todos', { params });
  return response.data;
};

export const createTodo = async (request: CreateTodoRequest): Promise<Todo> => {
  const response = await apiClient.post<Todo>('/api/v1/todos', {
    title: request.title,
    due_date: request.due_date,
  });
  return response.data;
};

export const updateTodo = async (id: string, request: UpdateTodoRequest): Promise<Todo> => {
  const payload: UpdateTodoRequest = {};
  if (request.title !== undefined) payload.title = request.title;
  if (request.due_date !== undefined) payload.due_date = request.due_date;
  if (request.status !== undefined) payload.status = request.status;
  
  const response = await apiClient.patch<Todo>(`/api/v1/todos/${id}`, payload);
  return response.data;
};

export const deleteTodo = async (id: string): Promise<void> => {
  await apiClient.delete(`/api/v1/todos/${id}`);
};

export interface BoardSummary {
  counts: {
    OPEN: number;
    DONE: number;
  };
  next_up: Array<{
    title: string;
    reason: string;
  }>;
  overdue: string[];
  near_deadline: string[];
  summary: string;
}

export const getBoardSummary = async (): Promise<BoardSummary | null> => {
  try {
    const response = await apiClient.get<BoardSummary>('/api/v1/board/summary');
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.status === 404) {
      return null;
    }
    console.error('Error fetching board summary:', error);
    return null;
  }
};

// Function to stream chat responses. NOTE: axious does not support streaming natively.
// export const streamChat = async (message: string) => {
//   const response = await apiClient.post('/api/v1/chat', { message });
//   return response.data;
// };

// Function to stream chat responses (using fetch for streaming support)
export const streamChat = async (message: string) => {
  const response = await fetch(`${API_BASE_URL}/api/v1/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message }),
  });
  // The consumer should handle response.body as a stream
  return response;
};

// Function to fetch chat history
export const fetchChatMessages = async (page: number, pageSize: number) => {
  const response = await apiClient.get('/api/v1/chat/messages', {
    params: { page, pagesize: pageSize },
  });
  return response.data;
};

// Function to clear chat history
export const clearChatMessages = async () => {
  await apiClient.delete('/api/v1/chat/messages');
};

export type { TodoStatus, Todo, ListTodosResponse };