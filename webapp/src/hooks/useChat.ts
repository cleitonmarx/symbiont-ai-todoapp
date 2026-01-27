import { useState, useCallback } from 'react';
import { fetchChatMessages, clearChatMessages, streamChat } from '../services/api';
import type { ChatMessage } from '../types';

interface UseChatReturn {
  messages: ChatMessage[];
  loading: boolean;
  error: string | null;
  loadMessages: () => Promise<void>;
  sendMessage: (message: string) => Promise<void>;
  clearChat: () => Promise<void>;
}

export const useChat = (): UseChatReturn => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadMessages = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetchChatMessages(1, 200);
      setMessages(response.messages || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load messages');
    } finally {
      setLoading(false);
    }
  }, []);

  const sendMessage = useCallback(async (message: string) => {
    if (!message.trim()) return;

    try {
      setLoading(true);
      setError(null);

      const userMessage: ChatMessage = {
        id: Date.now().toString(),
        role: 'user',
        content: message,
        created_at: new Date().toISOString(),
      };
      setMessages((prev) => [...prev, userMessage]);

      const response = await streamChat(message);

      if (!response.body) {
        throw new Error('No response body');
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let assistantContent = '';
      let assistantMessageId = '';
      let buffer = '';

      // Add initial empty assistant message
      const tempAssistantMsg: ChatMessage = {
        id: 'temp-' + Date.now(),
        role: 'assistant',
        content: '',
        created_at: new Date().toISOString(),
      };
      setMessages((prev) => [...prev, tempAssistantMsg]);

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (!line.trim()) continue;

          if (line.startsWith('event: meta')) {
            continue;
          } else if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));

              if (data.assistant_message_id) {
                assistantMessageId = data.assistant_message_id;
              }

              if (data.text) {
                assistantContent += data.text;
                // Update the assistant message in real-time
                setMessages((prev) => {
                  const updated = [...prev];
                  const lastMsg = updated[updated.length - 1];
                  if (lastMsg && lastMsg.role === 'assistant') {
                    lastMsg.content = assistantContent;
                    lastMsg.id = assistantMessageId || lastMsg.id;
                  }
                  return updated;
                });
              }
            } catch (e) {
              console.error('Failed to parse SSE data:', e);
            }
          } else if (line.startsWith('event: done')) {
            setLoading(false);
          } else if (line.startsWith('event: error')) {
            setError('Failed to get response from assistant');
            setLoading(false);
          }
        }
      }

      setLoading(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send message');
      setLoading(false);
    }
  }, []);

  const clearChat = useCallback(async () => {
    try {
      await clearChatMessages();
      setMessages([]);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear chat');
    }
  }, []);

  return {
    messages,
    loading,
    error,
    loadMessages,
    sendMessage,
    clearChat,
  };
};