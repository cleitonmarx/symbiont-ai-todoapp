import type { Todo, CreateTodoRequest, UpdateTodoRequest, ListTodosResponse, TodoStatus } from '../types';
import { apiClient } from './httpClient';

export type TodoSort =
  | 'createdAtAsc'
  | 'createdAtDesc'
  | 'dueDateAsc'
  | 'dueDateDesc'
  | 'similarityAsc'
  | 'similarityDesc';

export const getTodos = async (
  status?: TodoStatus,
  query?: string,
  page = 1,
  pageSize = 50,
  dateRange?: { dueAfter?: string; dueBefore?: string },
  sort?: TodoSort,
): Promise<ListTodosResponse> => {
  const params: Record<string, string | number> = { page, pageSize };

  if (status) {
    params.status = status;
  }

  if (query) {
    params.query = query;
  }

  if (dateRange?.dueAfter) {
    params['dateRange[dueAfter]'] = dateRange.dueAfter;
  }

  if (dateRange?.dueBefore) {
    params['dateRange[dueBefore]'] = dateRange.dueBefore;
  }

  if (sort) {
    params.sort = sort;
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
