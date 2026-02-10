import { useState, useCallback, useRef, useEffect } from 'react';
import { fetchChatMessages, clearChatMessages, fetchAvailableModels, streamChat } from '../services/chatApi';
import type { ChatMessage } from '../types';

interface UseChatReturn {
  messages: ChatMessage[];
  models: string[];
  selectedModel: string;
  toolStatus: string | null;
  toolStatusCount: number;
  loading: boolean;
  loadingModels: boolean;
  error: string | null;
  loadMessages: () => Promise<void>;
  loadModels: () => Promise<void>;
  setSelectedModel: (model: string) => void;
  sendMessage: (message: string) => Promise<void>;
  clearChat: () => Promise<void>;
  stopStream: () => void;
}

const CHAT_SELECTED_MODEL_STORAGE_KEY = 'todoapp.chat.selectedModel';

const loadPersistedModel = (): string => {
  if (typeof window === 'undefined') {
    return '';
  }

  try {
    return window.localStorage.getItem(CHAT_SELECTED_MODEL_STORAGE_KEY) ?? '';
  } catch {
    return '';
  }
};

const persistModel = (model: string): void => {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    if (model) {
      window.localStorage.setItem(CHAT_SELECTED_MODEL_STORAGE_KEY, model);
    } else {
      window.localStorage.removeItem(CHAT_SELECTED_MODEL_STORAGE_KEY);
    }
  } catch {
    // Ignore storage failures (e.g., private mode restrictions).
  }
};

export const useChat = (onChatDone?: () => void): UseChatReturn => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [models, setModels] = useState<string[]>([]);
  const [selectedModel, setSelectedModel] = useState(loadPersistedModel);
  const [toolStatus, setToolStatus] = useState<string | null>(null);
  const [toolStatusCount, setToolStatusCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingModels, setLoadingModels] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const readerRef = useRef<ReadableStreamReader<Uint8Array> | null>(null);
  const toolStatusKeyRef = useRef<string | null>(null);
  const toolStatusCountRef = useRef(0);

  const clearToolStatus = useCallback(() => {
    toolStatusKeyRef.current = null;
    toolStatusCountRef.current = 0;
    setToolStatus(null);
    setToolStatusCount(0);
  }, []);

  const updateToolStatus = useCallback((text: string, fnName?: string) => {
    const normalizedText = text.trim();
    if (!normalizedText) {
      return;
    }

    const normalizedFnName = typeof fnName === 'string' ? fnName.trim().toLowerCase() : '';
    const key = normalizedFnName || normalizedText.toLowerCase();

    if (toolStatusKeyRef.current === key) {
      toolStatusCountRef.current += 1;
    } else {
      toolStatusKeyRef.current = key;
      toolStatusCountRef.current = 1;
    }

    setToolStatus(normalizedText);
    setToolStatusCount(toolStatusCountRef.current);
  }, []);

  const loadMessages = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetchChatMessages(1, 200);
      setMessages(response.messages || []);
      clearToolStatus();
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load messages');
    } finally {
      setLoading(false);
    }
  }, [clearToolStatus]);

  const loadModels = useCallback(async () => {
    try {
      setLoadingModels(true);
      const availableModels = await fetchAvailableModels();
      setModels(availableModels);
      setSelectedModel((current) => {
        if (current && availableModels.includes(current)) {
          return current;
        }
        return availableModels[0] ?? '';
      });
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load models');
    } finally {
      setLoadingModels(false);
    }
  }, []);

  useEffect(() => {
    persistModel(selectedModel);
  }, [selectedModel]);

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

    clearToolStatus();
    setLoading(false);
    setError('Stream stopped by user');
  }, [clearToolStatus]);

  const sendMessage = useCallback(async (message: string) => {
    if (!message.trim()) return;

    try {
      setLoading(true);
      clearToolStatus();
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

      const response = await streamChat(message, selectedModel, abortControllerRef.current.signal);

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
            clearToolStatus();
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

          if (eventType === 'tool_call') {
            const functionCallText = data.Text ?? data.text;
            const functionCallName = data.Function ?? data.function;
            if (typeof functionCallText === 'string' && functionCallText.trim() !== '') {
              updateToolStatus(functionCallText, typeof functionCallName === 'string' ? functionCallName : undefined);
            }
            return;
          }

          if (eventType === 'done') {
            if (data.AssistantMessageID) {
              assistantMessageId = data.AssistantMessageID;
            }
            clearToolStatus();
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
            clearToolStatus();
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
          clearToolStatus();
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
      clearToolStatus();
      readerRef.current = null;
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        setError('Stream stopped by user');
      } else {
        setError(err instanceof Error ? err.message : 'Failed to send message');
      }
      clearToolStatus();
      setLoading(false);
      readerRef.current = null;
      abortControllerRef.current = null;
    }
  }, [clearToolStatus, onChatDone, selectedModel, updateToolStatus]);

  const clearChat = useCallback(async () => {
    try {
      await clearChatMessages();
      setMessages([]);
      clearToolStatus();
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear chat');
    }
  }, [clearToolStatus]);

  return {
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
  };
};
