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

export const useChat = (onChatDone?: () => void): UseChatReturn => {
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

      const processEvent = (rawEvent: string) => {
        const lines = rawEvent.split(/\r?\n/).filter(Boolean);
        let eventType = 'message';
        const dataLines: string[] = [];

        for (const line of lines) {
          if (line.startsWith('event:')) {
            eventType = line.replace('event:', '').trim();
          } else if (line.startsWith('data:')) {
            dataLines.push(line.replace('data:', '').trimStart());
          }
        }

        if (dataLines.length === 0) return;

        const dataStr = dataLines.join('\n');
        try {
          const data = JSON.parse(dataStr);

          if (eventType === 'meta' && data.AssistantMessageID) {
            assistantMessageId = data.AssistantMessageID;
            return;
          }

          if (eventType === 'delta' && data.Text) {
            assistantContent += data.Text;

            setMessages((prev) => {
              const updated = [...prev];
              const lastIndex = updated.length - 1;
              const lastMsg = updated[lastIndex];
              if (lastMsg && lastMsg.role === 'assistant') {
                updated[lastIndex] = {
                  ...lastMsg,
                  id: assistantMessageId || lastMsg.id,
                  content: assistantContent,
                };
              }
              return updated;
            });
            return;
          }

          if (eventType === 'done') {
            if (data.AssistantMessageID) {
              assistantMessageId = data.AssistantMessageID;
            }
            setLoading(false);
            // Call the callback when done
            onChatDone?.();
            return;
          }

          if (eventType === 'error') {
            setError('Failed to get response from assistant');
            setLoading(false);
          }
        } catch (e) {
          console.error('Failed to parse SSE data:', e, dataStr);
        }
      };

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const events = buffer.split(/\r?\n\r?\n/);
        buffer = events.pop() || '';

        for (const evt of events) {
          if (!evt.trim()) continue;
          processEvent(evt);
        }
      }

      setLoading(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send message');
      setLoading(false);
    }
  }, [onChatDone]); // Add onChatDone to dependency array

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