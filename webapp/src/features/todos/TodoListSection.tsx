import { memo } from 'react';
import { Button } from '../../components/ui/Button';
import type { Todo, TodoStatus } from '../../types';
import { TodoCard } from './TodoCard';

interface TodoListSectionProps {
  todos: Todo[];
  loading: boolean;
  currentPage: number;
  previousPage: number | null;
  nextPage: number | null;
  onPreviousPage: () => void;
  onNextPage: () => void;
  onUpdate: (id: string, status?: TodoStatus, title?: string, due_date?: string) => void;
  onDelete: (id: string) => void;
}

const TodoListSectionComponent = ({
  todos,
  loading,
  currentPage,
  previousPage,
  nextPage,
  onPreviousPage,
  onNextPage,
  onUpdate,
  onDelete,
}: TodoListSectionProps) => {
  if (loading) {
    return <div className="ui-state">Loading todos...</div>;
  }

  if (todos.length === 0) {
    return <div className="ui-state">No todos found for this view.</div>;
  }

  return (
    <section>
      <div className="ui-todos-grid">
        {todos.map((todo) => (
          <TodoCard key={todo.id} todo={todo} onUpdate={onUpdate} onDelete={onDelete} />
        ))}
      </div>

      <footer className="ui-pagination">
        <span>Page {currentPage || 1}</span>
        <div>
          <Button type="button" variant="secondary" onClick={onPreviousPage} disabled={previousPage === null}>
            Previous
          </Button>
          <Button type="button" variant="secondary" onClick={onNextPage} disabled={nextPage === null}>
            Next
          </Button>
        </div>
      </footer>
    </section>
  );
};

export const TodoListSection = memo(TodoListSectionComponent);
