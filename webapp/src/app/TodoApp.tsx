import { useCallback, useState } from 'react';
import BatchModal from '../components/BatchModal';
import { Button } from '../components/ui/Button';
import { useMediaQuery } from '../hooks/useMediaQuery';
import { useTodos } from '../hooks/useTodos';
import type { TodoStatus } from '../types';
import { BoardSummaryCard } from '../features/board-summary/BoardSummaryCard';
import { ChatPanel } from '../features/chat/ChatPanel';
import { TodoControlsBar } from '../features/todos/TodoControlsBar';
import { TodoCreateDialog } from '../features/todos/TodoCreateDialog';
import { TodoListSection } from '../features/todos/TodoListSection';

const TodoApp = () => {
  const isMobile = useMediaQuery('(max-width: 767px)');
  const isTablet = useMediaQuery('(max-width: 1100px)');
  const showRail = !isTablet;

  const [chatOpen, setChatOpen] = useState(false);
  const [batchOpen, setBatchOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);

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
    pageSize,
    setPageSize,
    dueAfter,
    setDueAfter,
    dueBefore,
    setDueBefore,
    clearDateRange,
  } = useTodos();

  const handleUpdateTodo = useCallback((id: string, status?: TodoStatus, title?: string, due_date?: string) => {
    updateTodo(id, status, title, due_date);
  }, [updateTodo]);

  const handlePreviousPage = useCallback(() => {
    if (previousPage !== null) {
      goToPage(previousPage);
    }
  }, [goToPage, previousPage]);

  const handleNextPage = useCallback(() => {
    if (nextPage !== null) {
      goToPage(nextPage);
    }
  }, [goToPage, nextPage]);

  const chatButtonLabel = chatOpen ? 'Hide assistant' : 'Open assistant';
  const hasDesktopChatColumn = chatOpen && !isTablet;

  return (
    <div className="ui-app">
      <header className="ui-topbar">
        <div className="ui-topbar-brand">
          <img src="/symbiont-icon.png" alt="Todo App" />
          <div>
            <h1>Todo App</h1>
          </div>
        </div>
        {isTablet ? (
          <div className="ui-topbar-actions">
            <Button type="button" variant="secondary" onClick={() => setCreateOpen(true)}>
              New Todo
            </Button>
            <Button type="button" variant="secondary" onClick={() => setBatchOpen(true)}>
              Batch
            </Button>
          </div>
        ) : null}
      </header>

      <main className={`ui-layout ${hasDesktopChatColumn ? 'chat-open' : 'chat-closed'} ${showRail ? 'has-rail' : 'no-rail'}`}>
        {showRail ? (
          <aside className="ui-rail" aria-label="Workspace actions">
            <Button type="button" variant="primary" className="ui-rail-btn" onClick={() => setCreateOpen(true)}>
              + Todo
            </Button>
            <Button type="button" variant="secondary" className="ui-rail-btn" onClick={() => setBatchOpen(true)}>
              Batch
            </Button>
          </aside>
        ) : null}

        <section className="ui-main">
          {boardSummary ? <BoardSummaryCard data={boardSummary} /> : null}

          {error ? (
            <section className="ui-error" role="alert">
              <strong>Error:</strong> {error}
            </section>
          ) : null}

          <TodoControlsBar
            statusFilter={statusFilter}
            onStatusFilterChange={setStatusFilter}
            sortBy={sortBy}
            onSortByChange={setSortBy}
            pageSize={pageSize}
            onPageSizeChange={setPageSize}
            searchQuery={searchQuery}
            onSearchQueryChange={setSearchQuery}
            dueAfter={dueAfter}
            onDueAfterChange={setDueAfter}
            dueBefore={dueBefore}
            onDueBeforeChange={setDueBefore}
            onClearDateRange={clearDateRange}
          />

          <TodoListSection
            todos={todos}
            loading={loading}
            currentPage={page}
            previousPage={previousPage}
            nextPage={nextPage}
            onPreviousPage={handlePreviousPage}
            onNextPage={handleNextPage}
            onUpdate={handleUpdateTodo}
            onDelete={deleteTodo}
          />
        </section>

        <aside className={`ui-chat-column ${chatOpen ? 'open' : ''} ${isTablet ? 'tablet' : ''}`}>
          {!isMobile && chatOpen ? (
            <ChatPanel
              onChatDone={refetch}
              onClose={isTablet ? () => setChatOpen(false) : undefined}
            />
          ) : null}
        </aside>
      </main>

      <button
        type="button"
        className={`ui-chat-fab ${chatOpen ? 'active' : ''}`}
        onClick={() => setChatOpen((value) => !value)}
        aria-label={chatButtonLabel}
        title={chatButtonLabel}
      >
        {chatOpen ? '✕' : 'AI'}
      </button>

      {isMobile && chatOpen ? (
        <div className="ui-sheet-overlay" role="presentation">
          <ChatPanel onChatDone={refetch} mode="sheet" onClose={() => setChatOpen(false)} />
        </div>
      ) : null}

      <TodoCreateDialog open={createOpen} onClose={() => setCreateOpen(false)} onCreateTodo={createTodo} />

      <BatchModal
        open={batchOpen}
        hideTrigger
        onClose={() => setBatchOpen(false)}
        onBatchComplete={refetch}
      />
    </div>
  );
};

export default TodoApp;
