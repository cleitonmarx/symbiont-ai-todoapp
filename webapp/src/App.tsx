import React, { useState, useCallback, useEffect } from 'react';
import CreateTodoForm from './components/CreateTodoForm';
import TodoList from './components/TodoList';
import { useTodos } from './hooks/useTodos';
import Chat from './components/Chat';
import BatchModal from './components/BatchModal';
import { BoardSummary } from './components/BoardSummary';
import type { TodoStatus } from './types';

const App: React.FC = () => {
  const [isChatOpen, setIsChatOpen] = useState(false);
  const [batchOpen, setBatchOpen] = useState(false);
  const { 
    todos, 
    boardSummary, 
    loading, 
    error, 
    createTodo, 
    updateTodo, 
    deleteTodo, 
    statusFilter, 
    setStatusFilter, 
    page, 
    previousPage, 
    nextPage, 
    goToPage, 
    refetch,
    searchQuery,
    setSearchQuery,
    sortBy,
    setSortBy,
  } = useTodos();

  // Reset sortBy to createdAtDesc when search is cleared
  useEffect(() => {
    if (!searchQuery && (sortBy === 'similarityAsc' || sortBy === 'similarityDesc')) {
      setSortBy('createdAtDesc');
    }
  }, [searchQuery, sortBy]);

  const handleUpdateTodo = useCallback((id: string, status?: TodoStatus, title?: string, due_date?: string) => {
    updateTodo(id, status, title, due_date);
  }, [updateTodo]);

  const handleDeleteTodo = useCallback((id: string) => {
    deleteTodo(id);
  }, [deleteTodo]);

  const handleStatusFilterChange = useCallback((status: TodoStatus | 'ALL') => {
    setStatusFilter(status);
  }, [setStatusFilter]);

  const handleSearchChange = useCallback((query: string) => {
    setSearchQuery(query);
  }, [setSearchQuery]);

  const handlePreviousPage = useCallback(() => {
    if (previousPage !== null) {
      goToPage(previousPage);
    }
  }, [previousPage, goToPage]);

  const handleNextPage = useCallback(() => {
    if (nextPage !== null) {
      goToPage(nextPage);
    }
  }, [nextPage, goToPage]);

  return (
    <div className="app">
      <header className="app-header">
        <div className="header-content">
          <img src="/symbiont-icon.png" alt="Todo App Logo" className="header-logo" />
          <h1>Todo App</h1>
        </div>
      </header>
      <div className="app-main">
        <div className="sidebar-toolbar">
          <CreateTodoForm onCreateTodo={createTodo} />
          <BatchModal
            open={batchOpen}
            onClose={() => setBatchOpen(false)}
            onBatchComplete={refetch}
          />
        </div>
        <div className="content-area">
          {boardSummary && <BoardSummary data={boardSummary} />}
          
          {error && (
            <div className="error" style={{ marginBottom: '1.5rem' }}>
              <strong>Error:</strong> {error}
            </div>
          )}

          {/* Filter Bar */}
          <div className="filter-bar">
            <div className="filter-group">
              <label>Status:</label>
              <div className="filter-buttons">
                {(['ALL', 'OPEN', 'DONE'] as const).map((status) => (
                  <button
                    key={status}
                    className={`filter-button ${statusFilter === status ? 'active' : ''}`}
                    onClick={() => handleStatusFilterChange(status)}
                  >
                    {status}
                  </button>
                ))}
              </div>
            </div>
            <div className="filter-group">
              <label>Sort:</label>
              <select
                className="sort-select"
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as any)}
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
            <input
              type="text"
              placeholder="Search todos..."
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="search-input"
            />
          </div>
          
          {loading ? (
            <div className="loading">Loading...</div>
          ) : (
            <TodoList
              todos={todos}
              onUpdate={handleUpdateTodo}
              onDelete={handleDeleteTodo}
              currentPage={page}
              previousPage={previousPage}
              nextPage={nextPage}
              onPreviousPage={handlePreviousPage}
              onNextPage={handleNextPage}
            />
          )}
        </div>
        {isChatOpen && <Chat onChatDone={refetch} />}
      </div>
      <button
        className="chat-toggle-btn"
        onClick={() => setIsChatOpen(!isChatOpen)}
        title={isChatOpen ? 'Hide chat' : 'Show chat'}
      >
        {isChatOpen ? '✕' : '💬'}
      </button>
    </div>
  );
};

export default App;