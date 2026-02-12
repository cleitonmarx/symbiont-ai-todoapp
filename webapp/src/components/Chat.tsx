import React, { useEffect, useRef } from 'react';
import { useChat } from '../hooks/useChat';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import '../styles/Chat.css';

// Configure marked for tables and line breaks
marked.setOptions({
  breaks: true,
  gfm: true,
});

interface ChatProps {
  onChatDone?: () => void;
}

const Chat: React.FC<ChatProps> = ({ onChatDone }) => {
  const {
    messages,
    models,
    selectedModel,
    toolStatus,
    toolStatusCount,
    loading,
    loadingModels,
    error,
    loadMessages,
    loadModels,
    setSelectedModel,
    sendMessage,
    clearChat,
    stopStream,
  } = useChat(onChatDone);
  const [input, setInput] = React.useState('');
  const [renderedMessages, setRenderedMessages] = React.useState<Record<string, string>>({});
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const isInitialLoad = useRef(true);

  useEffect(() => {
    loadMessages();
    loadModels();
  }, [loadMessages, loadModels]);

  useEffect(() => {
    if (messages.length > 0 && messagesContainerRef.current) {
      if (isInitialLoad.current) {
        messagesContainerRef.current.scrollTop = messagesContainerRef.current.scrollHeight;
        isInitialLoad.current = false;
      } else {
        messagesEndRef.current?.scrollIntoView({ 
          behavior: 'smooth',
          block: 'nearest',
          inline: 'nearest'
        });
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
          })
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
    if (!message) return;
    setInput('');
    await sendMessage(message);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="chat-panel">
      <div className="chat-header">
        <h2>AI Assistant</h2>
        <button className="chat-clear-btn" onClick={clearChat} title="Clear chat history">
          🗑️
        </button>
      </div>

      <div className="chat-messages" ref={messagesContainerRef}>
        {error && <div className="chat-error">{error}</div>}
        {messages.map((msg) => (
          <div key={msg.id} className={`chat-message chat-message-${msg.role}`}>
            <div className="chat-message-content">
              {msg.role === 'assistant' ? (
                <div dangerouslySetInnerHTML={{ __html: renderedMessages[String(msg.id)] ?? '' }} />
              ) : (
                msg.content
              )}
            </div>
            <div className="chat-message-time">{new Date(msg.created_at).toLocaleTimeString()}</div>
          </div>
        ))}
        {loading && (
          <div className="chat-message chat-message-assistant">
            <div className="chat-typing-indicator">
              <span></span>
              <span></span>
              <span></span>
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      {toolStatus ? (
        <div className="chat-tool-status">
          <span>{toolStatus}</span>
          <span className="chat-tool-count">x{toolStatusCount}</span>
        </div>
      ) : null}

      <div className="chat-composer">
        <div className="chat-input-shell">
          <textarea
            className="chat-input"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask for follow-up changes"
            disabled={loading}
          />
          <div className="chat-input-controls">
            <div className="chat-input-meta">
              <select
                className="chat-model-select"
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
            {!loading ? (
              <button
                className="chat-send-btn"
                onClick={handleSend}
                disabled={!input.trim()}
                title="Send message"
                aria-label="Send message"
              >
                ↑
              </button>
            ) : (
              <button
                className="chat-send-btn chat-send-btn-stop"
                onClick={stopStream}
                title="Stop stream"
                aria-label="Stop stream"
              >
                ■
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Chat;
