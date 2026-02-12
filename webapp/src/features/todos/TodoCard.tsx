import { useMemo, useState } from 'react';
import { Dialog } from '../../components/ui/Dialog';
import { Button } from '../../components/ui/Button';
import type { Todo, TodoStatus } from '../../types';

interface TodoCardProps {
  todo: Todo;
  onUpdate: (id: string, status?: TodoStatus, title?: string, due_date?: string) => void;
  onDelete: (id: string) => void;
}

const getMinDate = () => {
  const yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  yesterday.setHours(0, 0, 0, 0);
  const year = yesterday.getFullYear();
  const month = String(yesterday.getMonth() + 1).padStart(2, '0');
  const day = String(yesterday.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const formatDateTime = (value: string) => {
  const date = new Date(value);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};

const formatDueDate = (value: string) => {
  const [year, month, day] = value.split('-').map(Number);
  const date = new Date(year, month - 1, day);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
};

export const TodoCard = ({ todo, onUpdate, onDelete }: TodoCardProps) => {
  const [editOpen, setEditOpen] = useState(false);
  const [title, setTitle] = useState(todo.title);
  const [dueDate, setDueDate] = useState(todo.due_date);

  const dueTone = useMemo(() => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    const [year, month, day] = todo.due_date.split('-').map(Number);
    const due = new Date(year, month - 1, day);
    due.setHours(0, 0, 0, 0);

    if (todo.status === 'DONE') return 'done';
    if (due < today) return 'overdue';
    if (due.getTime() === today.getTime()) return 'today';
    return 'upcoming';
  }, [todo.due_date, todo.status]);

  const closeEdit = () => {
    setTitle(todo.title);
    setDueDate(todo.due_date);
    setEditOpen(false);
  };

  const saveEdit = () => {
    onUpdate(todo.id, undefined, title, dueDate);
    setEditOpen(false);
  };

  return (
    <article className="ui-todo-card">
      <header className="ui-todo-card-header">
        <h3>{todo.title}</h3>
        <span className={`ui-status-badge ${todo.status === 'DONE' ? 'done' : 'open'}`}>{todo.status}</span>
      </header>

      <div className="ui-todo-card-body">
        <div className="ui-meta-row">
          <span>Due</span>
          <strong className={`ui-due-text ${dueTone}`}>{formatDueDate(todo.due_date)}</strong>
        </div>
        <div className="ui-meta-grid">
          <p>
            <span>Created</span>
            <strong>{formatDateTime(todo.created_at)}</strong>
          </p>
          <p>
            <span>Updated</span>
            <strong>{formatDateTime(todo.updated_at)}</strong>
          </p>
        </div>
      </div>

      <footer className="ui-todo-card-footer">
        {todo.status === 'OPEN' ? (
          <Button type="button" variant="primary" onClick={() => onUpdate(todo.id, 'DONE')}>
            Mark done
          </Button>
        ) : (
          <Button type="button" variant="secondary" onClick={() => onUpdate(todo.id, 'OPEN')}>
            Re-open
          </Button>
        )}
        <Button type="button" variant="secondary" onClick={() => setEditOpen(true)}>
          Edit
        </Button>
        <Button type="button" variant="danger" onClick={() => onDelete(todo.id)}>
          Delete
        </Button>
      </footer>

      <Dialog
        open={editOpen}
        title="Edit Todo"
        onClose={closeEdit}
        footer={(
          <>
            <Button type="button" variant="secondary" onClick={closeEdit}>
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              onClick={saveEdit}
              disabled={!title.trim() || !dueDate || (title === todo.title && dueDate === todo.due_date)}
            >
              Save
            </Button>
          </>
        )}
      >
        <div className="ui-form-stack">
          <label className="ui-label" htmlFor={`todo-title-${todo.id}`}>
            Title
          </label>
          <input
            id={`todo-title-${todo.id}`}
            className="ui-input"
            type="text"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
          />
          <label className="ui-label" htmlFor={`todo-due-${todo.id}`}>
            Due date
          </label>
          <input
            id={`todo-due-${todo.id}`}
            className="ui-input"
            type="date"
            value={dueDate}
            min={getMinDate()}
            onChange={(event) => setDueDate(event.target.value)}
          />
        </div>
      </Dialog>
    </article>
  );
};
