import React, { useState, useEffect } from 'react';
import { gqlListTodos, gqlBatchUpdateTodos, gqlBatchDeleteTodos } from '../services/graphql';
import type { ListTodosQuery, TodoStatus, TodoSortBy } from '../types/graphql';

interface BatchModalProps {
  open: boolean;
  onClose: () => void;
  onBatchComplete: () => void;
}

const PAGE_SIZE = 50;

const BatchModal: React.FC<BatchModalProps> = ({ open, onClose, onBatchComplete }) => {
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
  
  // New state variables for date range and sort (fixed property names)
  const [dueAfter, setDueAfter] = useState('');
  const [dueBefore, setDueBefore] = useState('');
  const [sortBy, setSortBy] = useState<TodoSortBy>('createdAtDesc');

  // Open modal when parent prop changes
  useEffect(() => {
    setShow(open);
  }, [open]);

  useEffect(() => {
    if (open) {
      setShow(true);
      setSelected([]);    // Reset selections
      setAction(null);    // Reset action
      setDueDate('');     // Reset due date
      setBatchPage(1);    // Reset to first page
      setDueAfter('');    // Reset dueAfter
      setDueBefore('');   // Reset dueBefore
      setSortBy('createdAtAsc'); // Reset sortBy
    } else {
      setShow(false);
    }
  }, [open]);

  // Fetch todos (append if loading more)
  const fetchTodos = async (pageToFetch = 1) => {
    setLoading(true);
    try {
      const effectiveSortBy =
        !searchQuery && (sortBy === 'similarityAsc' || sortBy === 'similarityDesc')
          ? 'createdAtAsc'
          : sortBy;

      const data = await gqlListTodos({ 
        status: statusFilter, 
        page: pageToFetch, 
        pageSize: PAGE_SIZE,
        query: searchQuery || undefined,
        // Only send dateRange if BOTH dueAfter AND dueBefore have values
        dateRange: dueAfter && dueBefore ? { 
          DueAfter: dueAfter, 
          DueBefore: dueBefore 
        } : undefined, 
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

  // Initial and reset fetch
  useEffect(() => {
    if (show) {
      setTodos([]);
      setSelected([]);
      setBatchPage(1);
      fetchTodos(1);
    }
    // eslint-disable-next-line
  }, [show, statusFilter, searchQuery, dueAfter, dueBefore, sortBy]);

  //Fetch next page when page changes (but not on first render)
  useEffect(() => {
    if (batchPage > 0 && show) {
      fetchTodos(batchPage);
    }
  }, [batchPage, show]);

  useEffect(() => {
    if (!searchQuery && (sortBy === 'similarityAsc' || sortBy === 'similarityDesc')) {
      setSortBy('createdAtDesc' as TodoSortBy);
    }
  }, [searchQuery, sortBy]);

  const allSelected = todos.length > 0 && selected.length === todos.length;
  const someSelected = selected.length > 0 && selected.length < todos.length;

  const handleSelect = (id: string) => {
    setSelected(sel => sel.includes(id) ? sel.filter(sid => sid !== id) : [...sel, id]);
  };

  const handleSelectAllToggle = () => {
    if (allSelected) {
      setSelected([]);
    } else {
      setSelected(todos.map(t => t.id));
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
    } catch (e) {
      setError('Batch operation failed');
    } finally {
      setLoading(false);
    }
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

  return (
    <>
      <button
        className="toolbar-button"
        onClick={() => setShow(true)}
        title="Batch operations"
        style={{ marginTop: '8px' }}
      >
        🗂️
      </button>
      {show && (
        <div className={`modal-overlay active`} onClick={() => { setShow(false); onClose(); }}>
          <div className="modal-dialog batch-modal-dialog" style={{ maxWidth: '80vw' }} onClick={e => e.stopPropagation()}>
            <div className="modal-header" style={{ padding: '0.75rem 1rem' }}>
              <h2 style={{ margin: 0, fontSize: '1.25rem' }}>Batch Operations</h2>
            </div>
            
            {/* First row: Status, Date Range, Sort */}
            <div className="filter-bar" style={{ padding: '0.5rem 1rem', gap: '0.75rem', margin: '0 0 0.5rem 0' }}>
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
              
              {/* Date Range Filters */}
              <div className="filter-group">
                <label>Due After:</label>
                <input
                  type="date"
                  value={dueAfter}
                  onChange={(e) => setDueAfter(e.target.value)}
                  className="date-input"
                  style={{ marginRight: '0.5rem' }}
                />
              </div>
              
              <div className="filter-group">
                <label>Due Before:</label>
                <input
                  type="date"
                  value={dueBefore}
                  onChange={(e) => setDueBefore(e.target.value)}
                  className="date-input"
                />
              </div>
              
              {/* Sort Filter */}
              <div className="filter-group">
                <label>Sort By:</label>
                <select 
                  value={sortBy} 
                  onChange={(e) => setSortBy(e.target.value as TodoSortBy)}
                  className="sort-select"
                >
                  <option value="createdAtAsc">Created At</option>
                  <option value="createdAtDesc">Created At Desc</option>
                  <option value="dueDateAsc">Due Date Asc</option>
                  <option value="dueDateDesc">Due Date Desc</option>
                  {searchQuery && (
                    <>
                      <option value="similarityAsc">Similarity Asc</option>
                      <option value="similarityDesc">Similarity Desc</option>
                    </>
                  )}
                </select>
              </div>
              <div className="filter-group">
                  <input
                    type="text"
                    placeholder="Search todos..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="search-input"
                    style={{ flex: 1}}
                  />
              </div>
            </div>
            
            <div className="modal-content" style={{ padding: '0 1rem 0.1rem 1rem' }}>
              {error && <div className="batch-modal-error" style={{ marginBottom: '0.5rem' }}>{error}</div>}
              <div className="batch-modal-grid">
                <table>
                  <thead>
                    <tr>
                      <th>
                        <input
                          type="checkbox"
                          checked={allSelected}
                          ref={el => {
                            if (el) el.indeterminate = someSelected;
                          }}
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
                    {todos.map(todo => (
                      <tr key={todo.id} className={selected.includes(todo.id) ? 'selected' : ''}>
                        <td>
                          <input
                            type="checkbox"
                            checked={selected.includes(todo.id)}
                            onChange={() => handleSelect(todo.id)}
                            disabled={!!action}
                          />
                        </td>
                        <td>{todo.title}</td>
                        <td>{todo.due_date}</td>
                        <td>{todo.status}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {loading && <div style={{ textAlign: 'center', padding: '0.5rem' }}>Loading...</div>}
              </div>
              
              <div className="pagination" style={{ padding: '0.5rem 0' }}>
                <div className="pagination-info">
                  Page {batchPage || 1}
                </div>
                <div className="batch-modal-selection-summary" style={{ textAlign: 'center', color: '#667eea', fontWeight: 500 }}>
                  {selected.length > 0
                  ? `${selected.length} todo${selected.length > 1 ? 's' : ''} selected`
                  : 'No todos selected'}
                </div>
                <div className="pagination-buttons">
                  <button className="btn-primary" disabled={!hasPreviousPage} onClick={() => setBatchPage(batchPage - 1)}>←</button>
                  <button className="btn-primary" disabled={!hasNextPage} onClick={() => setBatchPage(batchPage + 1)}>→</button>
                </div>
              </div>
              
              {!action && (
                <div className="batch-modal-actions" style={{ padding: '0.5rem 0' }}>
                  <button
                    className="btn-primary batch-action-btn"
                    disabled={selected.length === 0 || statusFilter === 'DONE'}
                    onClick={() => setAction('due')}
                  >
                    Change Due Date
                  </button>
                  <button
                    className="btn-primary batch-action-btn"
                    disabled={selected.length === 0 || statusFilter === 'DONE'}
                    onClick={() => setAction('done')}
                  >
                    Mark as Done
                  </button>
                  <button
                    className="btn-danger batch-action-btn"
                    disabled={selected.length === 0}
                    onClick={() => setAction('delete')}
                  >
                    Delete
                  </button>
                </div>
              )}
              {action === 'due' && (
                <div className="batch-modal-due" style={{ padding: '0.5rem 0' }}>
                  <label htmlFor="batch-due-date">New Due Date</label>
                  <input
                    id="batch-due-date"
                    type="date"
                    value={dueDate}
                    min={getMinDate()}
                    onChange={e => setDueDate(e.target.value)}
                    className="batch-date-input"
                  />
                  <div className="batch-modal-due-actions">
                    <button className="btn-secondary" onClick={() => setAction(null)}>Cancel</button>
                    <button className="btn-primary" disabled={!dueDate} onClick={handleBatchAction}>Confirm</button>
                  </div>
                </div>
              )}
              {action === 'done' && (
                <div className="batch-modal-confirm" style={{ padding: '0.5rem 0' }}>
                  <span>Mark selected todos as done?</span>
                  <div className="modal-footer">
                    <button className="btn-secondary" onClick={() => setAction(null)}>Cancel</button>
                    <button className="btn-primary" onClick={handleBatchAction}>Confirm</button>
                  </div>
                </div>
              )}
              {action === 'delete' && (
                <div className="batch-modal-confirm" style={{ padding: '0.5rem 0' }}>
                  <span>Delete selected todos?</span>
                  <div className="modal-footer">
                    <button className="btn-secondary" onClick={() => setAction(null)}>Cancel</button>
                    <button className="btn-danger" onClick={handleBatchAction}>Confirm</button>
                  </div>
                </div>
              )}
            </div>
            <div className="modal-footer" style={{ padding: '0.75rem 1rem' }}>
              <button 
                type="button" 
                className="btn-secondary"
                onClick={() => { setShow(false); onClose(); }}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
};

export default BatchModal;