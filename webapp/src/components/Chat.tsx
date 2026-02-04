import React, { useEffect, useRef } from 'react';
import { useChat } from '../hooks/useChat';
import { marked } from 'marked';
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
  const { messages, loading, error, loadMessages, sendMessage, clearChat, stopStream } = useChat(onChatDone);
  const [input, setInput] = React.useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const isInitialLoad = useRef(true);

  useEffect(() => {
    loadMessages();
  }, [loadMessages]);

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

  const handleSend = async () => {
    if (!input.trim()) return;
    await sendMessage(input);
    setInput('');
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
                <div dangerouslySetInnerHTML={{ __html: marked(msg.content) }} />
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

      <div className="chat-input-area">
        <textarea
          className="chat-input"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Type your message... (Enter to send, Shift+Enter for new line)"
          disabled={loading}
        />
        {!loading ? (
          <button
            className="chat-send-btn"
            onClick={handleSend}
            disabled={loading || !input.trim()}
            title="Send message"
          >
            ✈️
          </button>
        ) : (
          <button
            className="chat-stop-btn"
            onClick={stopStream}
            title="Stop stream"
          >
            ⏹️
          </button>
        )}
      </div>
    </div>
  );
};

export default Chat;