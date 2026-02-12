import React, { memo } from 'react';
import type { TodoStatus } from '../types';
import type { Todo } from '../services/api';
import TodoItem from './TodoItem';

interface TodoListProps {
  todos: Todo[];
  onUpdate: (id: string, status?: TodoStatus, title?: string, due_date?: string) => void;
  onDelete: (id: string) => void;
  currentPage: number;
  previousPage: number | null;
  nextPage: number | null;
  onPreviousPage: () => void;
  onNextPage: () => void;
}

const TodoListComponent: React.FC<TodoListProps> = ({
  todos,
  onUpdate,
  currentPage,
  previousPage,
  nextPage,
  onPreviousPage,
  onNextPage,
  onDelete,
}) => {
  return (
    <div className="todo-list">
      <div className="todo-list-header">
        <div>
          <h2>Todos</h2>
        </div>
      </div>

      {todos.length === 0 && (
        <div className="empty-state">
          <p>No todos found.</p>
        </div>
      )}

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
  );
};

const arePropsEqual = (prev: TodoListProps, next: TodoListProps): boolean => {
  if (prev.currentPage !== next.currentPage) {
    return false;
  }

  if (
    prev.onUpdate !== next.onUpdate ||
    prev.onDelete !== next.onDelete ||
    prev.onPreviousPage !== next.onPreviousPage ||
    prev.onNextPage !== next.onNextPage
  ) {
    return false;
  }

  return true;
};

export const TodoList = memo(TodoListComponent, arePropsEqual);
export default TodoList;