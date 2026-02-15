import { useState, useCallback, useRef, useEffect } from 'react';
import {
  fetchChatMessages,
  deleteConversation,
  fetchAvailableModels,
  listConversations,
  streamChat,
  updateConversation,
} from '../services/chatApi';
import type { ChatMessage, Conversation } from '../types';

interface UseChatReturn {
  messages: ChatMessage[];
  conversations: Conversation[];
  activeConversationId: string | null;
  models: string[];
  selectedModel: string;
  toolStatus: string | null;
  toolStatusCount: number;
  loading: boolean;
  loadingModels: boolean;
  loadingConversations: boolean;
  loadingMessages: boolean;
  error: string | null;
  loadConversations: () => Promise<void>;
  loadMessages: () => Promise<void>;
  loadModels: () => Promise<void>;
  setSelectedModel: (model: string) => void;
  sendMessage: (message: string) => Promise<void>;
  clearChat: () => Promise<void>;
  stopStream: () => void;
  startNewConversation: () => void;
  selectConversation: (conversationId: string | null) => Promise<void>;
  renameConversation: (conversationId: string, title: string) => Promise<void>;
  removeConversation: (conversationId: string) => Promise<void>;
}

const CHAT_SELECTED_MODEL_STORAGE_KEY = 'todoapp.chat.selectedModel';
const CONVERSATIONS_PAGE_SIZE = 100;
const CHAT_MESSAGES_PAGE_SIZE = 200;

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

const getEventString = (data: Record<string, unknown>, ...keys: string[]): string | undefined => {
  for (const key of keys) {
    const value = data[key];
    if (typeof value === 'string' && value.trim() !== '') {
      return value;
    }
  }
  return undefined;
};

const getEventBoolean = (data: Record<string, unknown>, ...keys: string[]): boolean | undefined => {
  for (const key of keys) {
    const value = data[key];
    if (typeof value === 'boolean') {
      return value;
    }
  }
  return undefined;
};

export const useChat = (onChatDone?: () => void): UseChatReturn => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConversationId, setActiveConversationId] = useState<string | null>(null);
  const [models, setModels] = useState<string[]>([]);
  const [selectedModel, setSelectedModel] = useState(loadPersistedModel);
  const [toolStatus, setToolStatus] = useState<string | null>(null);
  const [toolStatusCount, setToolStatusCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingModels, setLoadingModels] = useState(false);
  const [loadingConversations, setLoadingConversations] = useState(false);
  const [loadingMessages, setLoadingMessages] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const readerRef = useRef<ReadableStreamReader<Uint8Array> | null>(null);
  const toolStatusKeyRef = useRef<string | null>(null);
  const toolStatusCountRef = useRef(0);
  const activeConversationRef = useRef<string | null>(null);
  const conversationsRef = useRef<Conversation[]>([]);
  const hasLoadedConversationsRef = useRef(false);
  const composingNewConversationRef = useRef(false);

  useEffect(() => {
    activeConversationRef.current = activeConversationId;
  }, [activeConversationId]);

  useEffect(() => {
    conversationsRef.current = conversations;
  }, [conversations]);

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

  const loadMessagesForConversation = useCallback(
    async (conversationId: string) => {
      try {
        setLoadingMessages(true);
        const response = await fetchChatMessages(conversationId, 1, CHAT_MESSAGES_PAGE_SIZE);
        setMessages(response.messages || []);
        clearToolStatus();
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load messages');
      } finally {
        setLoadingMessages(false);
      }
    },
    [clearToolStatus],
  );

  const loadConversations = useCallback(async () => {
    try {
      setLoadingConversations(true);
      const response = await listConversations(1, CONVERSATIONS_PAGE_SIZE);
      const items = response.conversations ?? [];
      setConversations(items);

      const currentConversationId = activeConversationRef.current;
      const stillExists = currentConversationId ? items.some((item) => item.id === currentConversationId) : false;

      if (currentConversationId && !stillExists) {
        activeConversationRef.current = null;
        setActiveConversationId(null);
        setMessages([]);
      } else if (!hasLoadedConversationsRef.current && !currentConversationId && items.length > 0 && !composingNewConversationRef.current) {
        const latestConversationId = items[0].id;
        activeConversationRef.current = latestConversationId;
        setActiveConversationId(latestConversationId);
        await loadMessagesForConversation(latestConversationId);
      }

      hasLoadedConversationsRef.current = true;
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load conversations');
    } finally {
      setLoadingConversations(false);
    }
  }, [loadMessagesForConversation]);

  const loadMessages = useCallback(async () => {
    const conversationId = activeConversationRef.current;
    if (!conversationId) {
      setMessages([]);
      clearToolStatus();
      return;
    }
    await loadMessagesForConversation(conversationId);
  }, [clearToolStatus, loadMessagesForConversation]);

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
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    if (readerRef.current) {
      void readerRef.current.cancel();
      readerRef.current = null;
    }

    clearToolStatus();
    setLoading(false);
    setError('Stream stopped by user');
  }, [clearToolStatus]);

  const startNewConversation = useCallback(() => {
    composingNewConversationRef.current = true;
    activeConversationRef.current = null;
    setActiveConversationId(null);
    setMessages([]);
    clearToolStatus();
    setError(null);
  }, [clearToolStatus]);

  const selectConversation = useCallback(
    async (conversationId: string | null) => {
      if (conversationId === null) {
        startNewConversation();
        return;
      }
      composingNewConversationRef.current = false;
      activeConversationRef.current = conversationId;
      setActiveConversationId(conversationId);
      setError(null);
      await loadMessagesForConversation(conversationId);
    },
    [loadMessagesForConversation, startNewConversation],
  );

  const renameConversation = useCallback(async (conversationId: string, title: string) => {
    const nextTitle = title.trim();
    if (!nextTitle) {
      throw new Error('Conversation title cannot be empty');
    }

    const updated = await updateConversation(conversationId, nextTitle);
    setConversations((prev) => prev.map((conversation) => (conversation.id === updated.id ? updated : conversation)));
  }, []);

  const removeConversation = useCallback(async (conversationId: string) => {
    await deleteConversation(conversationId);

    setConversations((prev) => {
      const filtered = prev.filter((conversation) => conversation.id !== conversationId);
      return filtered;
    });

    if (activeConversationRef.current === conversationId) {
      const remaining = conversationsRef.current.filter((conversation) => conversation.id !== conversationId);
      const nextConversationId = remaining[0]?.id ?? null;
      if (nextConversationId) {
        composingNewConversationRef.current = false;
        activeConversationRef.current = nextConversationId;
        setActiveConversationId(nextConversationId);
        await loadMessagesForConversation(nextConversationId);
      } else {
        startNewConversation();
      }
    }
  }, [loadMessagesForConversation, startNewConversation]);

  const clearChat = useCallback(async () => {
    const conversationId = activeConversationRef.current;
    if (!conversationId) {
      startNewConversation();
      return;
    }

    try {
      await removeConversation(conversationId);
      clearToolStatus();
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear chat');
    }
  }, [clearToolStatus, removeConversation, startNewConversation]);

  const sendMessage = useCallback(
    async (message: string) => {
      if (!message.trim()) {
        return;
      }
      if (!selectedModel.trim()) {
        setError('Please select a model');
        return;
      }

      try {
        setLoading(true);
        clearToolStatus();
        setError(null);
        abortControllerRef.current = new AbortController();

        const userMessage: ChatMessage = {
          id: `tmp-user-${Date.now()}`,
          role: 'user',
          content: message,
          created_at: new Date().toISOString(),
        };
        setMessages((prev) => [...prev, userMessage]);

        const response = await streamChat(
          message,
          selectedModel,
          activeConversationRef.current,
          abortControllerRef.current.signal,
        );

        if (!response.body) {
          throw new Error('No response body');
        }

        readerRef.current = response.body.getReader();
        const decoder = new TextDecoder();
        let assistantContent = '';
        let assistantMessageId = '';
        let buffer = '';

        const tempAssistantMsg: ChatMessage = {
          id: `tmp-assistant-${Date.now()}`,
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

          if (dataLines.length === 0) {
            return;
          }

          const dataStr = dataLines.join('\n');
          try {
            const data = JSON.parse(dataStr) as Record<string, unknown>;

            if (eventType === 'meta') {
              const eventConversationId = getEventString(data, 'conversation_id', 'ConversationID', 'conversationId');
              const eventUserMessageId = getEventString(data, 'user_message_id', 'UserMessageID', 'userMessageId');
              const eventAssistantMessageId = getEventString(data, 'assistant_message_id', 'AssistantMessageID', 'assistantMessageId');
              const eventConversationCreated = getEventBoolean(data, 'conversation_created', 'ConversationCreated', 'conversationCreated');

              if (eventConversationId) {
                activeConversationRef.current = eventConversationId;
                setActiveConversationId(eventConversationId);
              }
              if (eventUserMessageId) {
                setMessages((prev) => {
                  const next = [...prev];
                  for (let i = next.length - 1; i >= 0; i--) {
                    if (next[i].role === 'user') {
                      next[i] = { ...next[i], id: eventUserMessageId };
                      break;
                    }
                  }
                  return next;
                });
              }
              if (eventAssistantMessageId) {
                assistantMessageId = eventAssistantMessageId;
              }
              if (eventConversationCreated === true) {
                composingNewConversationRef.current = false;
              }
              return;
            }

            if (eventType === 'delta') {
              const deltaText = getEventString(data, 'text', 'Text');
              if (!deltaText) {
                return;
              }
              clearToolStatus();
              assistantContent += deltaText;

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
              const functionCallText = getEventString(data, 'text', 'Text');
              const functionCallName = getEventString(data, 'function', 'Function');
              if (typeof functionCallText === 'string' && functionCallText.trim() !== '') {
                updateToolStatus(functionCallText, functionCallName);
              }
              return;
            }

            if (eventType === 'done') {
              const eventAssistantMessageId = getEventString(data, 'assistant_message_id', 'AssistantMessageID', 'assistantMessageId');
              if (eventAssistantMessageId) {
                assistantMessageId = eventAssistantMessageId;
              }
              clearToolStatus();
              setLoading(false);
              readerRef.current = null;
              abortControllerRef.current = null;
              void loadConversations();
              onChatDone?.();
              return;
            }

            if (eventType === 'error') {
              const errorCode = getEventString(data, 'error', 'Error');
              const errorMsg =
                errorCode === 'stream_cancelled'
                  ? 'Stream stopped by user'
                  : errorCode === 'client_closed'
                    ? 'Connection closed'
                    : 'Failed to get response from assistant';
              clearToolStatus();
              setError(errorMsg);
              setLoading(false);
              readerRef.current = null;
              abortControllerRef.current = null;
            }
          } catch (parseErr) {
            console.error('Failed to parse SSE data:', parseErr, dataStr);
          }
        };

        while (true) {
          if (abortControllerRef.current?.signal.aborted) {
            if (readerRef.current) {
              void readerRef.current.cancel();
              readerRef.current = null;
            }
            clearToolStatus();
            setLoading(false);
            setError('Stream stopped by user');
            abortControllerRef.current = null;
            break;
          }

          if (!readerRef.current) {
            setLoading(false);
            break;
          }

          const { done, value } = await readerRef.current.read();
          if (done) {
            break;
          }

          buffer += decoder.decode(value, { stream: true });
          const events = buffer.split(/\r?\n\r?\n/);
          buffer = events.pop() || '';

          for (const evt of events) {
            if (!evt.trim()) {
              continue;
            }
            processEvent(evt);
          }
        }

        if (assistantMessageId) {
          setMessages((prev) => {
            const updated = [...prev];
            const lastIndex = updated.length - 1;
            if (updated[lastIndex]?.role === 'assistant') {
              updated[lastIndex] = { ...updated[lastIndex], id: assistantMessageId };
            }
            return updated;
          });
        }

        setLoading(false);
        void loadConversations();
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
    },
    [clearToolStatus, loadConversations, onChatDone, selectedModel, updateToolStatus],
  );

  return {
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
    loadMessages,
    loadModels,
    setSelectedModel,
    sendMessage,
    clearChat,
    stopStream,
    startNewConversation,
    selectConversation,
    renameConversation,
    removeConversation,
  };
};
