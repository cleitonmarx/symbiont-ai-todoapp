import React, { useState, useEffect, useRef } from 'react';
import { gqlListTodos, gqlBatchUpdateTodos, gqlBatchDeleteTodos } from '../services/batchGraphqlApi';
import type { ListTodosQuery, TodoStatus, TodoSortBy } from '../types/graphql';
import { DateRangePicker } from './ui/DateRangePicker';

interface BatchModalProps {
  open: boolean;
  onClose: () => void;
  onBatchComplete: () => void;
  hideTrigger?: boolean;
}

const DEFAULT_PAGE_SIZE = 50;

const BatchModal: React.FC<BatchModalProps> = ({
  open,
  onClose,
  onBatchComplete,
  hideTrigger = false,
}) => {
  const [show, setShow] = useState(false);
  const [todos, setTodos] = useState<ListTodosQuery['listTodos']['items']>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [batchPage, setBatchPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [action, setAction] = useState<'due' | 'done' | 'delete' | null>(null);
  const [dueDate, setDueDate] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [hasNextPage, setHasNextPage] = useState(false);
  const [hasPreviousPage, setHasPreviousPage] = useState(false);
  const [statusFilter, setStatusFilter] = useState<'OPEN' | 'DONE'>('OPEN');
  const [searchQuery, setSearchQuery] = useState('');
  const [pageSize, setPageSize] = useState<number>(DEFAULT_PAGE_SIZE);
  const [dueAfter, setDueAfter] = useState('');
  const [dueBefore, setDueBefore] = useState('');
  const [sortBy, setSortBy] = useState<TodoSortBy>('dueDateAsc');
  const selectAllHeaderRef = useRef<HTMLInputElement | null>(null);
  const selectAllMobileRef = useRef<HTMLInputElement | null>(null);

	const isModernLayout = hideTrigger;
	const handleFilterDateRangeChange = (startDate: string, endDate: string) => {
		setDueAfter(startDate);
		setDueBefore(endDate);
	};

  useEffect(() => {
    setShow(open);
  }, [open]);

  useEffect(() => {
    if (open) {
      setShow(true);
      setSelected([]);
      setAction(null);
      setDueDate('');
      setBatchPage(1);
      setDueAfter('');
      setDueBefore('');
      setSortBy('dueDateAsc');
    } else {
      setShow(false);
    }
  }, [open]);

  const fetchTodos = async (pageToFetch = 1) => {
    setLoading(true);
    try {
      const effectiveSortBy =
        !searchQuery && (sortBy === 'similarityAsc' || sortBy === 'similarityDesc')
          ? 'dueDateAsc'
          : sortBy;

      const data = await gqlListTodos({
        status: statusFilter,
        page: pageToFetch,
        pageSize,
        query: searchQuery || undefined,
        dateRange: dueAfter && dueBefore ? { DueAfter: dueAfter, DueBefore: dueBefore } : undefined,
        sortBy: effectiveSortBy || undefined,
      });
      setHasNextPage(data.nextPage != null);
      setHasPreviousPage(data.previousPage != null);
      setTodos(data.items);
    } catch {
      setError('Failed to load todos');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (show) {
      setTodos([]);
      setSelected([]);
      setBatchPage(1);
      fetchTodos(1);
    }
    // eslint-disable-next-line
  }, [show, statusFilter, searchQuery, dueAfter, dueBefore, sortBy, pageSize]);

  useEffect(() => {
    setBatchPage(1);
  }, [pageSize]);

  useEffect(() => {
    if (batchPage > 0 && show) {
      fetchTodos(batchPage);
    }
  }, [batchPage, show]);

  useEffect(() => {
    if (!searchQuery && (sortBy === 'similarityAsc' || sortBy === 'similarityDesc')) {
      setSortBy('dueDateAsc' as TodoSortBy);
    }
  }, [searchQuery, sortBy]);

  useEffect(() => {
    if (dueAfter && dueBefore && dueBefore < dueAfter) {
      setDueBefore(dueAfter);
    }
  }, [dueAfter, dueBefore]);

  const allSelected = todos.length > 0 && selected.length === todos.length;
  const someSelected = selected.length > 0 && selected.length < todos.length;

  useEffect(() => {
    if (selectAllHeaderRef.current) {
      selectAllHeaderRef.current.indeterminate = someSelected;
    }
    if (selectAllMobileRef.current) {
      selectAllMobileRef.current.indeterminate = someSelected;
    }
  }, [someSelected, selected.length, todos.length]);

  const handleSelect = (id: string) => {
    setSelected((sel) => (sel.includes(id) ? sel.filter((sid) => sid !== id) : [...sel, id]));
  };

  const handleSelectAllToggle = () => {
    if (allSelected) {
      setSelected([]);
    } else {
      setSelected(todos.map((todo) => todo.id));
    }
  };

  const handleBatchAction = async () => {
    setLoading(true);
    setError(null);
    try {
      if (action === 'due' && dueDate) {
        await gqlBatchUpdateTodos(selected, { due_date: dueDate });
      } else if (action === 'done') {
        await gqlBatchUpdateTodos(selected, { status: 'DONE' as TodoStatus });
      } else if (action === 'delete') {
        await gqlBatchDeleteTodos(selected);
      }

      onBatchComplete();
      setSelected([]);
      setAction(null);
      setShow(false);
      onClose();
    } catch {
      setError('Batch operation failed');
    } finally {
      setLoading(false);
    }
  };

  const closeModal = () => {
    setShow(false);
    onClose();
  };

  const getMinDate = () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    yesterday.setHours(0, 0, 0, 0);
    const year = yesterday.getFullYear();
    const month = String(yesterday.getMonth() + 1).padStart(2, '0');
    const day = String(yesterday.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  };

  const commandPanel = (
    <>
      {!action ? (
        <div className="batch-modal-actions" style={isModernLayout ? undefined : { padding: '0.5rem 0' }}>
          <button
            className="btn-primary batch-action-btn"
            disabled={selected.length === 0 || statusFilter === 'DONE'}
            onClick={() => setAction('due')}
          >
            <span className="ui-batch-label-full">Change Due Date</span>
            <span className="ui-batch-label-short" aria-hidden="true">Due</span>
          </button>
          <button
            className="btn-primary batch-action-btn"
            disabled={selected.length === 0 || statusFilter === 'DONE'}
            onClick={() => setAction('done')}
          >
            <span className="ui-batch-label-full">Mark as Done</span>
            <span className="ui-batch-label-short" aria-hidden="true">Done</span>
          </button>
          <button className="btn-danger batch-action-btn" disabled={selected.length === 0} onClick={() => setAction('delete')}>
            <span className="ui-batch-label-full">Delete</span>
            <span className="ui-batch-label-short" aria-hidden="true">Del</span>
          </button>
        </div>
      ) : null}

      {action === 'due' ? (
        <div className="batch-modal-due" style={isModernLayout ? undefined : { padding: '0.5rem 0' }}>
          <label htmlFor="batch-due-date">New Due Date</label>
          <input
            id="batch-due-date"
            type="date"
            value={dueDate}
            min={getMinDate()}
            onChange={(event) => setDueDate(event.target.value)}
            className="batch-date-input"
          />
          <div className="batch-modal-due-actions">
            <button className="btn-secondary" onClick={() => setAction(null)}>
              Cancel
            </button>
            <button className="btn-primary" disabled={!dueDate} onClick={handleBatchAction}>
              Confirm
            </button>
          </div>
        </div>
      ) : null}

      {action === 'done' ? (
        <div className="batch-modal-confirm" style={isModernLayout ? undefined : { padding: '0.5rem 0' }}>
          <span>Mark selected todos as done?</span>
          <div className="modal-footer">
            <button className="btn-secondary" onClick={() => setAction(null)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleBatchAction}>
              Confirm
            </button>
          </div>
        </div>
      ) : null}

      {action === 'delete' ? (
        <div className="batch-modal-confirm" style={isModernLayout ? undefined : { padding: '0.5rem 0' }}>
          <span>Delete selected todos?</span>
          <div className="modal-footer">
            <button className="btn-secondary" onClick={() => setAction(null)}>
              Cancel
            </button>
            <button className="btn-danger" onClick={handleBatchAction}>
              Confirm
            </button>
          </div>
        </div>
      ) : null}
    </>
  );

  return (
    <>
      {!hideTrigger ? (
        <button className="toolbar-button" onClick={() => setShow(true)} title="Batch operations" style={{ marginTop: '8px' }}>
          🗂️
        </button>
      ) : null}

      {show ? (
        <div className={`modal-overlay active ${isModernLayout ? 'ui-batch-overlay' : ''}`} onClick={closeModal}>
          <div
            className={`modal-dialog batch-modal-dialog ${isModernLayout ? 'ui-batch-dialog' : ''}`}
            style={isModernLayout ? undefined : { maxWidth: '80vw' }}
            onClick={(event) => event.stopPropagation()}
          >
            <div className={`modal-header ${isModernLayout ? 'ui-batch-header' : ''}`} style={isModernLayout ? undefined : { padding: '0.75rem 1rem' }}>
              <h2 style={isModernLayout ? undefined : { margin: 0, fontSize: '1.25rem' }}>Batch Operations</h2>
            </div>

            <div
              className={`filter-bar ${isModernLayout ? 'ui-batch-filter-bar' : ''}`}
              style={isModernLayout ? undefined : { padding: '0.5rem 1rem', gap: '0.75rem', margin: '0 0 0.5rem 0' }}
            >
              <div className="filter-group">
                <label>Status:</label>
                <div className="filter-buttons">
                  {(['OPEN', 'DONE'] as const).map((status) => (
                    <button
                      key={status}
                      className={`filter-button ${statusFilter === status ? 'active' : ''}`}
                      onClick={() => setStatusFilter(status)}
                    >
                      {status}
                    </button>
                  ))}
                </div>
              </div>

              <div className="filter-group ui-batch-date-range-group">
                <label htmlFor="batch-date-range-picker">Due Range:</label>
                <DateRangePicker
                  id="batch-date-range-picker"
                  startDate={dueAfter}
                  endDate={dueBefore}
                  onChange={handleFilterDateRangeChange}
                  placeholder="Select date range"
                />
              </div>

              <div className="filter-group">
                <label>Sort By:</label>
                <select value={sortBy} onChange={(event) => setSortBy(event.target.value as TodoSortBy)} className="sort-select">
                  <option value="createdAtAsc">Created At</option>
                  <option value="createdAtDesc">Created At Desc</option>
                  <option value="dueDateAsc">Due Date Asc</option>
                  <option value="dueDateDesc">Due Date Desc</option>
                  {searchQuery ? (
                    <>
                      <option value="similarityAsc">Similarity Asc</option>
                      <option value="similarityDesc">Similarity Desc</option>
                    </>
                  ) : null}
                </select>
              </div>

              <div className="filter-group">
                <label htmlFor="batch-page-size-select">Page Size:</label>
                <select
                  id="batch-page-size-select"
                  value={pageSize}
                  onChange={(event) => setPageSize(Number(event.target.value))}
                  className="sort-select"
                >
                  <option value={25}>25</option>
                  <option value={50}>50</option>
                  <option value={100}>100</option>
                </select>
              </div>

              <div className="filter-group ui-batch-search-group">
                <label htmlFor="batch-search-input">Search:</label>
                <input
                  id="batch-search-input"
                  type="text"
                  placeholder="Search todos..."
                  value={searchQuery}
                  onChange={(event) => setSearchQuery(event.target.value)}
                  className="search-input"
                  style={isModernLayout ? undefined : { flex: 1 }}
                />
              </div>
            </div>

            <div className={`modal-content ${isModernLayout ? 'ui-batch-content' : ''}`} style={isModernLayout ? undefined : { padding: '0 1rem 0.1rem 1rem' }}>
              {error ? (
                <div className="batch-modal-error" style={isModernLayout ? undefined : { marginBottom: '0.5rem' }}>
                  {error}
                </div>
              ) : null}

              {isModernLayout ? (
                <div className="ui-batch-select-all">
                  <label htmlFor="batch-select-all-mobile">
                    <input
                      id="batch-select-all-mobile"
                      ref={selectAllMobileRef}
                      type="checkbox"
                      checked={allSelected}
                      onChange={handleSelectAllToggle}
                      disabled={!!action || todos.length === 0}
                    />
                    <span>Select all on page</span>
                  </label>
                  <span className="ui-batch-select-count">
                    {selected.length}/{todos.length}
                  </span>
                </div>
              ) : null}

              <div className="batch-modal-grid">
                <table className={isModernLayout ? 'ui-batch-table' : ''}>
                  <thead>
                    <tr>
                      <th>
                        <input
                          ref={selectAllHeaderRef}
                          type="checkbox"
                          checked={allSelected}
                          onChange={handleSelectAllToggle}
                          disabled={!!action}
                        />
                      </th>
                      <th>Title</th>
                      <th>Due Date</th>
                      <th>Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {todos.map((todo) => (
                      <tr key={todo.id} className={selected.includes(todo.id) ? 'selected' : ''}>
                        <td data-label="Select">
                          <input
                            type="checkbox"
                            checked={selected.includes(todo.id)}
                            onChange={() => handleSelect(todo.id)}
                            disabled={!!action}
                          />
                        </td>
                        <td data-label="Title">{todo.title}</td>
                        <td data-label="Due Date">{todo.due_date}</td>
                        <td data-label="Status">{todo.status}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {loading ? <div className="ui-batch-loading-row">Loading...</div> : null}
              </div>

              {!isModernLayout ? commandPanel : null}

            </div>

            {isModernLayout ? <div className="ui-batch-command-shell">{commandPanel}</div> : null}

            <div className={isModernLayout ? 'ui-batch-pagination-shell' : ''}>
              <div className={`pagination ${isModernLayout ? 'ui-batch-pagination' : ''}`} style={isModernLayout ? undefined : { padding: '0.5rem 0' }}>
                <div className="pagination-info">Page {batchPage || 1}</div>
                <div
                  className={`batch-modal-selection-summary ${isModernLayout ? 'ui-batch-selection-summary' : ''}`}
                  style={isModernLayout ? undefined : { textAlign: 'center', color: '#667eea', fontWeight: 500 }}
                >
                  {selected.length > 0
                    ? `${selected.length} todo${selected.length > 1 ? 's' : ''} selected`
                    : 'No todos selected'}
                </div>
                <div className="pagination-buttons">
                  {isModernLayout ? (
                    <>
                      <button className="btn-secondary" disabled={!hasPreviousPage} onClick={() => setBatchPage(batchPage - 1)}>
                        <span className="ui-batch-label-full">Previous</span>
                        <span className="ui-batch-label-short" aria-hidden="true">Prev</span>
                      </button>
                      <button className="btn-secondary" disabled={!hasNextPage} onClick={() => setBatchPage(batchPage + 1)}>
                        <span className="ui-batch-label-full">Next</span>
                        <span className="ui-batch-label-short" aria-hidden="true">Next</span>
                      </button>
                    </>
                  ) : (
                    <>
                      <button className="btn-primary" disabled={!hasPreviousPage} onClick={() => setBatchPage(batchPage - 1)}>
                        ←
                      </button>
                      <button className="btn-primary" disabled={!hasNextPage} onClick={() => setBatchPage(batchPage + 1)}>
                        →
                      </button>
                    </>
                  )}
                </div>
              </div>
            </div>

            <div className={`modal-footer ${isModernLayout ? 'ui-batch-footer' : ''}`} style={isModernLayout ? undefined : { padding: '0.75rem 1rem' }}>
              <button type="button" className="btn-secondary" onClick={closeModal}>
                Close
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
};

export default BatchModal;
