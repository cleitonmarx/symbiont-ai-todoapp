import React from 'react';
import type { TodoStatus } from '../types';
import type { Todo, BoardSummary } from '../services/api';
import TodoItem from './TodoItem';
import { BoardSummary as BoardSummaryCard } from './BoardSummary';

interface TodoListProps {
  todos: Todo[];
  boardSummary: BoardSummary | null;
  onUpdate: (id: string, status?: TodoStatus, title?: string, due_date?: string) => void;
  onDelete: (id: string) => void;
  statusFilter: TodoStatus | 'ALL';
  onStatusFilterChange: (status: TodoStatus | 'ALL') => void;
  currentPage: number;
  previousPage: number | null;
  nextPage: number | null;
  onPreviousPage: () => void;
  onNextPage: () => void;
  loading: boolean;
  error: string | null;
}

export const TodoList: React.FC<TodoListProps> = ({
  todos,
  boardSummary,
  loading,
  error,
  onUpdate,
  statusFilter,
  onStatusFilterChange,
  currentPage,
  previousPage,
  nextPage,
  onPreviousPage,
  onNextPage,
  onDelete,
}) => {
  return (
    <div className="todo-list-container">
      {boardSummary && <BoardSummaryCard data={boardSummary} />}
      <div className="todo-list">
        <div className="todo-list-header">
          <div>
            <h2>Todos</h2>
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <div className="error">
            {error}
          </div>
        )}

        {/* Filters */}
        <div className="filter-bar">
          <div className="filter-group">
            <label>Status:</label>
            <div className="filter-buttons">
              {(['ALL', 'OPEN', 'DONE'] as const).map((status) => (
                <button
                  key={status}
                  className={`filter-button ${statusFilter === status ? 'active' : ''}`}
                  onClick={() => onStatusFilterChange(status)}
                >
                  {status}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Loading */}
        {loading && todos.length === 0 && (
          <div className="loading">Loading todos...</div>
        )}

        {/* Empty State */}
        {!loading && todos.length === 0 && !error && (
          <div className="empty-state">
            <p>No todos yet. Create one to get started!</p>
          </div>
        )}

        {/* Todos Grid */}
        {todos.length > 0 && (
          <>
            <div className="todos-grid">
              {todos.map((todo) => (
                <TodoItem
                  key={todo.id}
                  todo={todo}
                  onUpdate={onUpdate}
                  onDelete={onDelete}
                />
              ))}
            </div>

            {/* Pagination */}
            <div className="pagination">
              <div className="pagination-info">
                Page {currentPage || 1}
              </div>
              <div className="pagination-buttons">
                <button
                  className="btn-primary"
                  onClick={onPreviousPage}
                  disabled={previousPage === null}
                >
                  ←
                </button>
                <button
                  className="btn-primary"
                  onClick={onNextPage}
                  disabled={nextPage === null}
                >
                  →
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export default TodoList;