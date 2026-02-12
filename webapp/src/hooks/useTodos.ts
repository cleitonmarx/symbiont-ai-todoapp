import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTodos, createTodo, updateTodo, deleteTodo as deleteTodoApi, type TodoSort } from '../services/todosApi';
import { getBoardSummary, type BoardSummary } from '../services/boardApi';
import type { Todo, CreateTodoRequest, TodoStatus } from '../types';
import { useState, useEffect, useCallback, useMemo } from 'react';

export interface UseTodosReturn {
  todos: Todo[];
  loading: boolean;
  error: string | null;
  createTodo: (title: string, due_date: string) => void;
  updateTodo: (id: string, status?: TodoStatus, title?: string, due_date?: string) => void;
  boardSummary: BoardSummary | null;
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
  pageSize: number;
  setPageSize: (size: number) => void;
  dueAfter: string;
  setDueAfter: (date: string) => void;
  dueBefore: string;
  setDueBefore: (date: string) => void;
  clearDateRange: () => void;
  refetch: () => void;
}

const DEFAULT_TODO_PAGE_SIZE = 25;

export const useTodos = (): UseTodosReturn => {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilterState] = useState<TodoStatus | 'ALL'>('OPEN');
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState<string>('');
  const [sortBy, setSortBy] = useState<TodoSort>('dueDateAsc');
  const [pageSize, setPageSize] = useState<number>(DEFAULT_TODO_PAGE_SIZE);
  const [dueAfter, setDueAfterState] = useState<string>('');
  const [dueBefore, setDueBeforeState] = useState<string>('');
  const [mutationError, setMutationError] = useState<string | null>(null);
  const [boardSummary, setBoardSummary] = useState<BoardSummary | null>(null);

  useEffect(() => {
    if (!searchQuery && (sortBy === 'similarityAsc' || sortBy === 'similarityDesc')) {
      setSortBy('dueDateAsc');
    }
  }, [searchQuery, sortBy]);

  const effectiveSortBy: TodoSort = useMemo(() => {
    if (!debouncedSearchQuery && (sortBy === 'similarityAsc' || sortBy === 'similarityDesc')) {
      return 'dueDateAsc';
    }
    return sortBy;
  }, [debouncedSearchQuery, sortBy]);

  const effectiveDateRange = useMemo(() => {
    if (!dueAfter || !dueBefore) {
      return undefined;
    }

    if (dueBefore < dueAfter) {
      return undefined;
    }

    return { dueAfter, dueBefore };
  }, [dueAfter, dueBefore]);

  // Debounce search query - only update after 500ms of no typing
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearchQuery(searchQuery);
    }, 500);

    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Reset page to 1 whenever status filter, debounced search, sort, or page size changes
  useEffect(() => {
    setCurrentPage(1);
  }, [statusFilter, debouncedSearchQuery, sortBy, pageSize, effectiveDateRange?.dueAfter, effectiveDateRange?.dueBefore]);

  const { 
    data: response, 
    isLoading: loading, 
    error, 
    refetch 
  } = useQuery({
    queryKey: [
      'todos',
      statusFilter,
      currentPage,
      debouncedSearchQuery,
      effectiveSortBy,
      pageSize,
      effectiveDateRange?.dueAfter,
      effectiveDateRange?.dueBefore,
    ],
    queryFn: () => getTodos(
      statusFilter === 'ALL' ? undefined : statusFilter,
      debouncedSearchQuery || undefined,
      currentPage,
      pageSize,
      effectiveDateRange,
      effectiveSortBy
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

  const handleSetDueAfter = useCallback((date: string) => {
    setDueAfterState(date);
    if (date && dueBefore && dueBefore < date) {
      setDueBeforeState(date);
    }
  }, [dueBefore]);

  const handleSetDueBefore = useCallback((date: string) => {
    if (date && dueAfter && date < dueAfter) {
      setDueBeforeState(dueAfter);
      return;
    }
    setDueBeforeState(date);
  }, [dueAfter]);

  const handleClearDateRange = useCallback(() => {
    setDueAfterState('');
    setDueBeforeState('');
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
    pageSize,
    setPageSize,
    dueAfter,
    setDueAfter: handleSetDueAfter,
    dueBefore,
    setDueBefore: handleSetDueBefore,
    clearDateRange: handleClearDateRange,
  };
};
