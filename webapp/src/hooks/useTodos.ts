import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTodos, createTodo, updateTodo, getBoardSummary, deleteTodo as deleteTodoApi } from '../services/api';
import type { Todo, CreateTodoRequest, TodoStatus } from '../types';
import { useState, useEffect, useCallback, useMemo } from 'react';

type TodoSort =
  | 'createdAtAsc'
  | 'createdAtDesc'
  | 'dueDateAsc'
  | 'dueDateDesc'
  | 'similarityAsc'
  | 'similarityDesc';

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
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  sortBy: TodoSort;
  setSortBy: (sort: TodoSort) => void;
  refetch: () => void;
}

const MAX_TODO_PAGE_SIZE = 16;

export const useTodos = (): UseTodosReturn => {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilterState] = useState<TodoStatus | 'ALL'>('ALL');
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState<string>('');
  const [sortBy, setSortBy] = useState<TodoSort>('createdAtDesc');
  const [mutationError, setMutationError] = useState<string | null>(null);
  const [boardSummary, setBoardSummary] = useState<any>(null);

  // Debounce search query - only update after 500ms of no typing
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearchQuery(searchQuery);
    }, 500);

    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Reset page to 1 whenever status filter, debounced search, or sort changes
  useEffect(() => {
    setCurrentPage(1);
  }, [statusFilter, debouncedSearchQuery, sortBy]);

  const { 
    data: response, 
    isLoading: loading, 
    error, 
    refetch 
  } = useQuery({
    queryKey: ['todos', statusFilter, currentPage, debouncedSearchQuery, sortBy],
    queryFn: () => getTodos(
      statusFilter === 'ALL' ? undefined : statusFilter,
      debouncedSearchQuery || undefined,
      currentPage,
      MAX_TODO_PAGE_SIZE,
      undefined,
      sortBy
    ),
    retry: 1,
  });

  // Memoize todos array to prevent unnecessary re-renders
  const todos = useMemo(() => response?.items || [], [response?.items]);
  
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

  const handleSetSearchQuery = useCallback((query: string) => {
    setSearchQuery(query);
  }, []);

  const handleSetStatusFilter = useCallback((status: TodoStatus | 'ALL') => {
    setStatusFilterState(status);
  }, []);

  const handleSetSortBy = useCallback((sort: TodoSort) => {
    setSortBy(sort);
  }, []);

  const handleGoToPage = useCallback((page: number) => {
    setCurrentPage(page);
  }, []);

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
    setStatusFilter: handleSetStatusFilter,
    page,
    previousPage,
    nextPage,
    goToPage: handleGoToPage,
    refetch,
    searchQuery,
    setSearchQuery: handleSetSearchQuery,
    sortBy,
    setSortBy: handleSetSortBy,
  };
};