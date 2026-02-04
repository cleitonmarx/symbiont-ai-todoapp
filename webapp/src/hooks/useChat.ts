import { useState, useCallback, useRef } from 'react';
import { fetchChatMessages, clearChatMessages, streamChat } from '../services/api';
import type { ChatMessage } from '../types';

interface UseChatReturn {
  messages: ChatMessage[];
  loading: boolean;
  error: string | null;
  loadMessages: () => Promise<void>;
  sendMessage: (message: string) => Promise<void>;
  clearChat: () => Promise<void>;
  stopStream: () => void;
}

export const useChat = (onChatDone?: () => void): UseChatReturn => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const readerRef = useRef<ReadableStreamReader<Uint8Array> | null>(null);

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

  const stopStream = useCallback(() => {
    // Cancel the fetch request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    // Close the reader if it exists
    if (readerRef.current) {
      readerRef.current.cancel();
      readerRef.current = null;
    }

    setLoading(false);
    setError('Stream stopped by user');
  }, []);

  const sendMessage = useCallback(async (message: string) => {
    if (!message.trim()) return;

    try {
      setLoading(true);
      setError(null);

      // Create new abort controller for this request
      abortControllerRef.current = new AbortController();

      const userMessage: ChatMessage = {
        id: Date.now().toString(),
        role: 'user',
        content: message,
        created_at: new Date().toISOString(),
      };
      setMessages((prev) => [...prev, userMessage]);

      const response = await streamChat(message, abortControllerRef.current.signal);

      if (!response.body) {
        throw new Error('No response body');
      }

      readerRef.current = response.body.getReader();
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
            readerRef.current = null;
            abortControllerRef.current = null;
            onChatDone?.();
            return;
          }

          if (eventType === 'error') {
            const errorMsg = data.error === 'stream_cancelled' 
              ? 'Stream stopped by user'
              : data.error === 'client_closed'
              ? 'Connection closed'
              : 'Failed to get response from assistant';
            setError(errorMsg);
            setLoading(false);
            readerRef.current = null;
            abortControllerRef.current = null;
          }
        } catch (e) {
          console.error('Failed to parse SSE data:', e, dataStr);
        }
      };

      while (true) {
        // Check if abort was called
        if (abortControllerRef.current?.signal.aborted) {
          if (readerRef.current) {
            readerRef.current.cancel();
            readerRef.current = null;
          }
          setLoading(false);
          setError('Stream stopped by user');
          abortControllerRef.current = null;
          break;
        }

        // Add null check before reading
        if (!readerRef.current) {
          setLoading(false);
          break;
        }

        const { done, value } = await readerRef.current.read();
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
      readerRef.current = null;
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        setError('Stream stopped by user');
      } else {
        setError(err instanceof Error ? err.message : 'Failed to send message');
      }
      setLoading(false);
      readerRef.current = null;
      abortControllerRef.current = null;
    }
  }, [onChatDone]);

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
    stopStream,
  };
};