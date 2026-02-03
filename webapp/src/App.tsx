import React, { useState } from 'react';
import CreateTodoForm from './components/CreateTodoForm';
import TodoList from './components/TodoList';
import { useTodos } from './hooks/useTodos';
import Chat from './components/Chat';
import BatchModal from './components/BatchModal';

const App: React.FC = () => {
  const [isChatOpen, setIsChatOpen] = useState(false); // start hidden by default
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
  } = useTodos();

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
            onBatchComplete={refetch} // <-- This will reload todos after batch ops
          />
        </div>
        <div className="content-area">
          {error && (
            <div className="error" style={{ marginBottom: '1.5rem' }}>
              <strong>Error:</strong> {error}
            </div>
          )}
          {loading ? (
            <div className="loading">Loading...</div>
          ) : (
            <TodoList 
              todos={todos} 
              boardSummary={boardSummary}
              onUpdate={updateTodo}
              onDelete={deleteTodo}
              statusFilter={statusFilter}
              onStatusFilterChange={setStatusFilter}
              currentPage={page}
              previousPage={previousPage}
              nextPage={nextPage}
              onPreviousPage={() => previousPage !== null && goToPage(previousPage)}
              onNextPage={() => nextPage !== null && goToPage(nextPage)}
              loading={loading}
              error={null}
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