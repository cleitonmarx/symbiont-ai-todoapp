import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTodos, createTodo, updateTodo, getBoardSummary, deleteTodo as deleteTodoApi } from '../services/api';
import type { Todo, CreateTodoRequest, TodoStatus } from '../types';
import { useState, useEffect } from 'react';

interface UseTodosReturn {
  todos: Todo[];
  loading: boolean;
  error: string | null;
  createTodo: (title: string, due_date: string) => void;
  updateTodo: (id: string, status?: TodoStatus, title?: string, due_date?: string) => void;
  boardSummary: any;
  statusFilter: TodoStatus | 'ALL';
  setStatusFilter: (status: TodoStatus | 'ALL') => void;
  page: number;
  previousPage: number | null;
  nextPage: number | null;
  goToPage: (page: number) => void;
  deleteTodo: (id: string) => void;
}

export const useTodos = (): UseTodosReturn & { refetch: () => void } => {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilterState] = useState<TodoStatus | 'ALL'>('ALL');
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [mutationError, setMutationError] = useState<string | null>(null);
  const [boardSummary, setBoardSummary] = useState<any>(null);

  // Reset page to 1 whenever status filter changes
  useEffect(() => {
    setCurrentPage(1);
  }, [statusFilter]);

  const { 
    data: response, 
    isLoading: loading, 
    error, 
    refetch 
  } = useQuery({
    queryKey: ['todos', statusFilter, currentPage],
    queryFn: () => getTodos(
      statusFilter === 'ALL' ? undefined : statusFilter,
      currentPage,
      4
    ),
    retry: 1,
  });

  const todos = response?.items || [];
  const page = response?.page ?? 1;
  const previousPage = response?.previous_page ?? null;
  const nextPage = response?.next_page ?? null;

  const errorMessage = error 
    ? error instanceof Error 
      ? error.message 
      : String(error)
    : mutationError;

  const createMutation = useMutation({
    mutationFn: (request: CreateTodoRequest) => createTodo(request),
    onSuccess: () => {
      setMutationError(null);
      setCurrentPage(1);
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
    onError: (err: Error) => {
      setMutationError(err.message);
    },
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: TodoStatus }) => 
      updateTodo(id, { status }),
    onSuccess: () => {
      setMutationError(null);
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
    onError: (err: Error) => {
      setMutationError(err.message);
    },
  });

  const updateTitleMutation = useMutation({
    mutationFn: ({ id, title, due_date }: { id: string; title: string; due_date: string }) => 
      updateTodo(id, { title, due_date }),
    onSuccess: () => {
      setMutationError(null);
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
    onError: (err: Error) => {
      setMutationError(err.message);
    },
  });

  const fetchBoardSummary = async () => {
    try {
      const summary = await getBoardSummary();
      setBoardSummary(summary);
    } catch (err) {
      console.error('Failed to fetch board summary:', err);
    }
  };

  useEffect(() => {
    fetchBoardSummary();
    const interval = setInterval(() => {
      fetchBoardSummary();
    }, 3000);

    return () => clearInterval(interval);
  }, []);

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteTodoApi(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
    onError: (err: Error) => {
      setMutationError(err.message);
    },
  });

  return {
    todos,
    boardSummary,
    loading,
    error: errorMessage,
    createTodo: (title: string, due_date: string) => 
      createMutation.mutate({ title, due_date }),
    updateTodo: (id: string, status?: TodoStatus, title?: string, due_date?: string) => {
      if (status !== undefined) {
        updateStatusMutation.mutate({ id, status });
      } else if (title !== undefined && due_date !== undefined) {
        updateTitleMutation.mutate({ id, title, due_date });
      }
    },
    deleteTodo: (id: string) => deleteMutation.mutate(id),
    statusFilter,
    setStatusFilter: setStatusFilterState,
    page,
    previousPage,
    nextPage,
    goToPage: setCurrentPage,
    refetch,
  };
};