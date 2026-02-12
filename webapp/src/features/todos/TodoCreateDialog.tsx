import { useMemo, useState } from 'react';
import { Dialog } from '../../components/ui/Dialog';
import { Button } from '../../components/ui/Button';

interface TodoCreateDialogProps {
  open: boolean;
  onClose: () => void;
  onCreateTodo: (title: string, due_date: string) => void;
}

const getTodayDate = () => new Date().toISOString().split('T')[0];

const getMinDate = () => {
  const yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  yesterday.setHours(0, 0, 0, 0);
  const year = yesterday.getFullYear();
  const month = String(yesterday.getMonth() + 1).padStart(2, '0');
  const day = String(yesterday.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

export const TodoCreateDialog = ({ open, onClose, onCreateTodo }: TodoCreateDialogProps) => {
  const [title, setTitle] = useState('');
  const [dueDate, setDueDate] = useState(getTodayDate());

  const minDate = useMemo(() => getMinDate(), []);

  const close = () => {
    setTitle('');
    setDueDate(getTodayDate());
    onClose();
  };

  const submit = () => {
    if (!title.trim() || !dueDate) {
      return;
    }
    onCreateTodo(title.trim(), dueDate);
    close();
  };

  return (
    <Dialog
      open={open}
      title="Create Todo"
      onClose={close}
      footer={(
        <>
          <Button type="button" variant="secondary" onClick={close}>
            Cancel
          </Button>
          <Button type="button" variant="primary" onClick={submit} disabled={!title.trim() || !dueDate}>
            Create
          </Button>
        </>
      )}
    >
      <div className="ui-form-stack">
        <label className="ui-label" htmlFor="create-todo-title">
          Title
        </label>
        <input
          id="create-todo-title"
          className="ui-input"
          type="text"
          value={title}
          placeholder="Enter todo title"
          onChange={(event) => setTitle(event.target.value)}
        />

        <label className="ui-label" htmlFor="create-todo-due-date">
          Due date
        </label>
        <input
          id="create-todo-due-date"
          className="ui-input"
          type="date"
          value={dueDate}
          min={minDate}
          onChange={(event) => setDueDate(event.target.value)}
        />
      </div>
    </Dialog>
  );
};
