import { useEffect, useMemo, useRef, useState } from 'react';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import { useChat } from '../../hooks/useChat';
import { useMediaQuery } from '../../hooks/useMediaQuery';

marked.setOptions({
  breaks: true,
  gfm: true,
});

interface ChatPanelProps {
  onChatDone?: () => void;
  mode?: 'panel' | 'sheet';
  onClose?: () => void;
}

const MINUTE_MS = 60 * 1000;
const DAY_MS = 24 * 60 * MINUTE_MS;

const formatConversationAge = (updatedAt: string): string => {
  const updatedTime = new Date(updatedAt).getTime();
  if (Number.isNaN(updatedTime)) {
    return '0m';
  }

  const diff = Math.max(0, Date.now() - updatedTime);
  if (diff < DAY_MS) {
    const minutes = Math.max(1, Math.floor(diff / MINUTE_MS));
    return `${minutes}m`;
  }

  const days = Math.max(1, Math.floor(diff / DAY_MS));
  return `${days}d`;
};

export const ChatPanel = ({ onChatDone, mode = 'panel', onClose }: ChatPanelProps) => {
  const isViewportCompact = useMediaQuery('(max-width: 960px)');
  const isCompact = mode === 'panel' ? true : isViewportCompact;
  const [activeTab, setActiveTab] = useState<'chat' | 'sessions'>('chat');
  const {
    messages,
    conversations,
    activeConversationId,
    models,
    selectedModel,
    toolStatus,
    toolStatusCount,
    loading,
    loadingModels,
    loadingConversations,
    loadingMessages,
    error,
    loadConversations,
    loadModels,
    setSelectedModel,
    sendMessage,
    stopStream,
    startNewConversation,
    selectConversation,
    renameConversation,
    removeConversation,
  } = useChat(onChatDone);
  const [input, setInput] = useState('');
  const [renderedMessages, setRenderedMessages] = useState<Record<string, string>>({});
  const [renameConversationId, setRenameConversationId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [conversationError, setConversationError] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const isInitialLoad = useRef(true);

  const activeConversation = useMemo(
    () => conversations.find((conversation) => conversation.id === activeConversationId) ?? null,
    [activeConversationId, conversations],
  );

  useEffect(() => {
    if (!isCompact) {
      setActiveTab('chat');
    }
  }, [isCompact]);

  useEffect(() => {
    void loadConversations();
    void loadModels();
  }, [loadConversations, loadModels]);

  useEffect(() => {
    if (messages.length > 0 && messagesContainerRef.current) {
      if (isInitialLoad.current) {
        messagesContainerRef.current.scrollTop = messagesContainerRef.current.scrollHeight;
        isInitialLoad.current = false;
      } else {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
    }
  }, [messages]);

  useEffect(() => {
    let isActive = true;

    const renderMessages = async () => {
      const entries = await Promise.all(
        messages
          .filter((message) => message.role === 'assistant')
          .map(async (message) => {
            const parsed = await Promise.resolve(marked.parse(message.content));
            return [String(message.id), DOMPurify.sanitize(parsed)] as const;
          }),
      );

      if (!isActive) {
        return;
      }

      setRenderedMessages((current) => {
        const next = { ...current };
        for (const [id, html] of entries) {
          next[id] = html;
        }
        return next;
      });
    };

    if (messages.length > 0) {
      void renderMessages();
    }

    return () => {
      isActive = false;
    };
  }, [messages]);

  const handleSend = async () => {
    const message = input.trim();
    if (!message) {
      return;
    }
    setInput('');
    await sendMessage(message);
  };

  const handleRenameSubmit = async () => {
    if (!renameConversationId) {
      return;
    }
    try {
      setConversationError(null);
      await renameConversation(renameConversationId, renameValue);
      setRenameConversationId(null);
      setRenameValue('');
    } catch (err) {
      setConversationError(err instanceof Error ? err.message : 'Failed to rename conversation');
    }
  };

  const handleDeleteConversation = async (conversationId: string) => {
    if (!window.confirm('Delete this conversation and all messages?')) {
      return;
    }
    try {
      setConversationError(null);
      await removeConversation(conversationId);
    } catch (err) {
      setConversationError(err instanceof Error ? err.message : 'Failed to delete conversation');
    }
  };

  const showSessionsPane = !isCompact || activeTab === 'sessions';
  const showChatPane = !isCompact || activeTab === 'chat';
  const sessionsLocked = loading || loadingMessages;
  const chatGuideText = activeConversationId
    ? 'Break down work, search by title or similarity, sort tasks, or run batch updates.'
    : "Tell me what you need to do and I'll turn it into todos, then help you find, sort, and batch-update them.";

  return (
    <section className={`ui-chat ${mode}`}>
      <header className="ui-chat-topbar">
        <div className="ui-chat-tabs" role="tablist" aria-label="Chat view">
          <button
            type="button"
            role="tab"
            className={`ui-chat-tab ${!isCompact || activeTab === 'chat' ? 'active' : ''}`}
            aria-selected={!isCompact || activeTab === 'chat'}
            onClick={() => setActiveTab('chat')}
          >
            Chat
          </button>
          <button
            type="button"
            role="tab"
            className={`ui-chat-tab ${isCompact && activeTab === 'sessions' ? 'active' : ''}`}
            aria-selected={isCompact && activeTab === 'sessions'}
            onClick={() => setActiveTab('sessions')}
          >
            History
          </button>
        </div>
        <div className="ui-chat-topbar-actions">
          <button
            type="button"
            className="ui-icon-btn"
            title="New conversation"
            aria-label="New conversation"
            onClick={() => {
              setConversationError(null);
              setRenameConversationId(null);
              setRenameValue('');
              startNewConversation();
              if (isCompact) {
                setActiveTab('chat');
              }
            }}
            disabled={sessionsLocked}
          >
            ＋
          </button>
          {showSessionsPane ? (
            <button
              type="button"
              className="ui-icon-btn"
              title="Refresh history"
              aria-label="Refresh history"
              onClick={() => void loadConversations()}
              disabled={sessionsLocked || loadingConversations}
            >
              ↻
            </button>
          ) : null}
          {onClose ? (
            <button type="button" className="ui-icon-btn" onClick={onClose} title="Close chat" aria-label="Close chat">
              ✕
            </button>
          ) : null}
        </div>
      </header>

      <div className={`ui-chat-shell ${isCompact ? 'compact' : 'desktop'}`}>
        {showSessionsPane ? (
          <aside className="ui-chat-conversations" aria-label="Conversations">
            <div className="ui-chat-conversations-header">
              <h3>History</h3>
            </div>

            <div className="ui-chat-conversation-list">
              <button
                type="button"
                className={`ui-chat-conversation-item ${activeConversationId === null ? 'active' : ''}`}
                onClick={() => {
                  setConversationError(null);
                  setRenameConversationId(null);
                  setRenameValue('');
                  startNewConversation();
                  if (isCompact) {
                    setActiveTab('chat');
                  }
                }}
                disabled={sessionsLocked}
              >
                <span className="ui-chat-session-dot" aria-hidden>
                  •
                </span>
                <div className="ui-chat-conversation-body">
                  <div className="ui-chat-conversation-title">New conversation</div>
                  <div className="ui-chat-conversation-date">Starts on first message</div>
                </div>
              </button>

              {conversations.map((conversation) => {
                const isActive = activeConversationId === conversation.id;
                const isEditing = renameConversationId === conversation.id;
                return (
                  <article
                    key={conversation.id}
                    className={`ui-chat-conversation-item-wrap ${isActive ? 'active' : ''}`}
                  >
                    <div className={`ui-chat-conversation-item ui-chat-conversation-item-with-actions ${isActive ? 'active' : ''}`}>
                      <button
                        type="button"
                        className="ui-chat-conversation-main"
                        onClick={() => {
                          setConversationError(null);
                          setRenameConversationId(null);
                          setRenameValue('');
                          void selectConversation(conversation.id);
                          if (isCompact) {
                            setActiveTab('chat');
                          }
                        }}
                        disabled={sessionsLocked}
                      >
                        <span className="ui-chat-session-dot" aria-hidden>
                          •
                        </span>
                        <div className="ui-chat-conversation-body">
                          <div className="ui-chat-conversation-title">{conversation.title}</div>
                          <div className="ui-chat-conversation-date">
                            {formatConversationAge(conversation.updated_at)}
                          </div>
                        </div>
                      </button>

                      <div className="ui-chat-conversation-actions">
                        <button
                          type="button"
                          className="ui-icon-btn"
                          title="Rename conversation"
                          aria-label="Rename conversation"
                          onClick={() => {
                            setConversationError(null);
                            setRenameConversationId(conversation.id);
                            setRenameValue(conversation.title);
                          }}
                          disabled={sessionsLocked}
                        >
                          ✎
                        </button>
                        <button
                          type="button"
                          className="ui-icon-btn danger"
                          title="Delete conversation"
                          aria-label="Delete conversation"
                          onClick={() => void handleDeleteConversation(conversation.id)}
                          disabled={sessionsLocked}
                        >
                          🗑
                        </button>
                      </div>
                    </div>

                    {isEditing ? (
                      <div className="ui-chat-rename-box">
                        <input
                          className="ui-input"
                          value={renameValue}
                          onChange={(event) => setRenameValue(event.target.value)}
                          onKeyDown={(event) => {
                            if (event.key === 'Enter') {
                              event.preventDefault();
                              void handleRenameSubmit();
                            }
                            if (event.key === 'Escape') {
                              setRenameConversationId(null);
                              setRenameValue('');
                            }
                          }}
                          autoFocus
                        />
                        <div className="ui-chat-rename-actions">
                          <button type="button" className="ui-btn ui-btn-primary" onClick={() => void handleRenameSubmit()}>
                            Save
                          </button>
                          <button
                            type="button"
                            className="ui-btn ui-btn-secondary"
                            onClick={() => {
                              setRenameConversationId(null);
                              setRenameValue('');
                            }}
                          >
                            Cancel
                          </button>
                        </div>
                      </div>
                    ) : null}
                  </article>
                );
              })}
            </div>

            {loadingConversations ? <p className="ui-chat-conversation-meta">Loading history...</p> : null}
            {conversationError ? <p className="ui-chat-conversation-error">{conversationError}</p> : null}
          </aside>
        ) : null}

        {showChatPane ? (
          <div className="ui-chat-thread">
            <header className="ui-chat-header">
              <div>
                <h2>{activeConversation?.title ?? 'New conversation'}</h2>
                <p>{chatGuideText}</p>
              </div>
            </header>

            <div className="ui-chat-messages" ref={messagesContainerRef}>
            {error ? <div className="ui-chat-error">{error}</div> : null}

            {!loadingMessages && messages.length === 0 ? (
              <div className="ui-chat-empty-state">
                <strong>{activeConversationId ? 'No messages yet.' : 'Start a new conversation.'}</strong>
                <span>Send a message to begin.</span>
              </div>
            ) : null}

              {messages.map((message) => (
                <article key={message.id} className={`ui-chat-message ${message.role}`}>
                  <div className="ui-chat-message-content">
                    {message.role === 'assistant' ? (
                      <div dangerouslySetInnerHTML={{ __html: renderedMessages[String(message.id)] ?? '' }} />
                    ) : (
                      message.content
                    )}
                  </div>
                  <time>{new Date(message.created_at).toLocaleTimeString()}</time>
                </article>
              ))}

              {loading || loadingMessages ? (
                <article className="ui-chat-message assistant">
                  <div className="ui-chat-typing">
                    <span />
                    <span />
                    <span />
                  </div>
                </article>
              ) : null}

              <div ref={messagesEndRef} />
            </div>

            {toolStatus ? (
              <div className="ui-chat-tool-status">
                <span>{toolStatus}</span>
                <span className="ui-chat-tool-count">x{toolStatusCount}</span>
              </div>
            ) : null}

            <footer className="ui-chat-composer">
              <div className="ui-chat-input-shell">
                <textarea
                  className="ui-chat-input"
                  value={input}
                  disabled={loading}
                  placeholder={activeConversationId ? 'Ask for follow-up changes' : 'Start a new conversation'}
                  onChange={(event) => setInput(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' && !event.shiftKey) {
                      event.preventDefault();
                      void handleSend();
                    }
                  }}
                />

                <div className="ui-chat-input-controls">
                  <div className="ui-chat-input-meta">
                    <select
                      className="ui-chat-model-select"
                      aria-label="AI model"
                      value={selectedModel}
                      disabled={loading || loadingModels || models.length === 0}
                      onChange={(event) => setSelectedModel(event.target.value)}
                    >
                      {models.length === 0 ? (
                        <option value="">{loadingModels ? 'Loading models...' : 'Default model'}</option>
                      ) : (
                        models.map((model) => (
                          <option key={model} value={model}>
                            {model}
                          </option>
                        ))
                      )}
                    </select>
                  </div>

                  {loading ? (
                    <button type="button" className="ui-chat-send-btn stop" onClick={stopStream} aria-label="Stop stream">
                      ■
                    </button>
                  ) : (
                    <button
                      type="button"
                      className="ui-chat-send-btn"
                      onClick={() => void handleSend()}
                      disabled={!input.trim()}
                      aria-label="Send message"
                    >
                      ↑
                    </button>
                  )}
                </div>
              </div>
            </footer>
          </div>
        ) : null}
      </div>
    </section>
  );
};
