import { useState, useCallback, useRef, useEffect } from 'react';
import {
  fetchChatMessages,
  deleteConversation,
  fetchAvailableModels,
  listConversations,
  streamChat,
  updateConversation,
} from '../services/chatApi';
import type { AssistantTodoFilters, ChatMessage, Conversation } from '../types';

interface UseChatReturn {
  messages: ChatMessage[];
  conversations: Conversation[];
  activeConversationId: string | null;
  models: string[];
  selectedModel: string;
  toolCallingStatus: string | null;
  toolCallingCount: number;
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

interface UseChatOptions {
  onChatDone?: () => void;
  onToolExecuted?: () => void;
  onApplyAssistantFilters?: (filters: AssistantTodoFilters) => void;
}

const CHAT_SELECTED_MODEL_STORAGE_KEY = 'todoapp.chat.selectedModel';
const CONVERSATIONS_PAGE_SIZE = 100;
const CHAT_MESSAGES_PAGE_SIZE = 200;
const AUTO_CONVERSATION_TITLE_SOURCE = 'auto';
const AUTO_TITLE_REFRESH_DELAY_MS = 1200;
const TODO_SORT_OPTIONS = new Set([
  'createdAtAsc',
  'createdAtDesc',
  'dueDateAsc',
  'dueDateDesc',
  'similarityAsc',
  'similarityDesc',
]);

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

const getEventNumber = (data: Record<string, unknown>, ...keys: string[]): number | undefined => {
  for (const key of keys) {
    const value = data[key];
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
  }
  return undefined;
};

const getEventObject = (data: Record<string, unknown>, ...keys: string[]): Record<string, unknown> | undefined => {
  for (const key of keys) {
    const value = data[key];
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      return value as Record<string, unknown>;
    }
  }
  return undefined;
};

const getArgumentsObject = (data: Record<string, unknown>): Record<string, unknown> | undefined => {
  const rawArguments = getEventString(data, 'arguments', 'Arguments');
  if (!rawArguments) {
    return undefined;
  }

  try {
    const parsed = JSON.parse(rawArguments);
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    return undefined;
  }

  return undefined;
};

const parseAssistantFilters = (data: Record<string, unknown>): AssistantTodoFilters | null => {
  const toolName = getEventString(data, 'function', 'Function');
  const shouldApplyUIFilters = getEventBoolean(data, 'apply_ui_filters', 'applyUiFilters');
  const shouldApplyFromTool = toolName === 'set_ui_filters';
  if (!shouldApplyFromTool && !shouldApplyUIFilters) {
    return null;
  }

  const filtersData = getEventObject(data, 'filters') ?? getArgumentsObject(data);
  if (!filtersData) {
    return {};
  }

  const filters: AssistantTodoFilters = {};
  const status = getEventString(filtersData, 'status');
  if (status === 'OPEN' || status === 'DONE') {
    filters.status = status;
  }

  const searchBySimilarity = getEventString(filtersData, 'search_by_similarity');
  const searchByTitle = getEventString(filtersData, 'search_by_title');
  if (typeof searchBySimilarity === 'string') {
    filters.searchQuery = searchBySimilarity;
    filters.searchType = 'SIMILARITY';
  } else if (typeof searchByTitle === 'string') {
    filters.searchQuery = searchByTitle;
    filters.searchType = 'TITLE';
  } else {
    const searchQuery = getEventString(filtersData, 'search_query', 'searchQuery');
    if (typeof searchQuery === 'string') {
      filters.searchQuery = searchQuery;
    }

    const searchType = getEventString(filtersData, 'search_type', 'searchType');
    if (searchType === 'TITLE' || searchType === 'SIMILARITY') {
      filters.searchType = searchType;
    }
  }

  const sortBy = getEventString(filtersData, 'sort_by', 'sortBy');
  if (sortBy && TODO_SORT_OPTIONS.has(sortBy)) {
    filters.sortBy = sortBy as AssistantTodoFilters['sortBy'];
  }

  const dueAfter = getEventString(filtersData, 'due_after', 'dueAfter');
  if (typeof dueAfter === 'string') {
    filters.dueAfter = dueAfter;
  }

  const dueBefore = getEventString(filtersData, 'due_before', 'dueBefore');
  if (typeof dueBefore === 'string') {
    filters.dueBefore = dueBefore;
  }

  const page = getEventNumber(filtersData, 'page');
  if (typeof page === 'number') {
    filters.page = page;
  }

  const pageSize = getEventNumber(filtersData, 'page_size', 'pageSize');
  if (typeof pageSize === 'number') {
    filters.pageSize = pageSize;
  }

  return filters;
};

const wait = (ms: number): Promise<void> =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export const useChat = ({
  onChatDone,
  onToolExecuted,
  onApplyAssistantFilters,
}: UseChatOptions = {}): UseChatReturn => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConversationId, setActiveConversationId] = useState<string | null>(null);
  const [models, setModels] = useState<string[]>([]);
  const [selectedModel, setSelectedModel] = useState(loadPersistedModel);
  const [toolCallingStatus, setToolCallingStatus] = useState<string | null>(null);
  const [toolCallingCount, setToolCallingCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingModels, setLoadingModels] = useState(false);
  const [loadingConversations, setLoadingConversations] = useState(false);
  const [loadingMessages, setLoadingMessages] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const readerRef = useRef<ReadableStreamReader<Uint8Array> | null>(null);
  const toolCallingKeyRef = useRef<string | null>(null);
  const toolCallingCountRef = useRef(0);
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

  const clearToolCallingStatus = useCallback(() => {
    toolCallingKeyRef.current = null;
    toolCallingCountRef.current = 0;
    setToolCallingStatus(null);
    setToolCallingCount(0);
  }, []);

  const resetToolActivity = useCallback(() => {
    clearToolCallingStatus();
  }, [clearToolCallingStatus]);

  const updateToolCallingStatus = useCallback((text: string, fnName?: string) => {
    const normalizedText = text.trim();
    if (!normalizedText) {
      return;
    }

    const normalizedFnName = typeof fnName === 'string' ? fnName.trim().toLowerCase() : '';
    const key = normalizedFnName || normalizedText.toLowerCase();

    if (toolCallingKeyRef.current === key) {
      toolCallingCountRef.current += 1;
    } else {
      toolCallingKeyRef.current = key;
      toolCallingCountRef.current = 1;
    }

    setToolCallingStatus(normalizedText);
    setToolCallingCount(toolCallingCountRef.current);
  }, []);

  const loadMessagesForConversation = useCallback(
    async (conversationId: string) => {
      try {
        setLoadingMessages(true);
        const response = await fetchChatMessages(conversationId, 1, CHAT_MESSAGES_PAGE_SIZE);
        setMessages(response.messages || []);
        resetToolActivity();
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load messages');
      } finally {
        setLoadingMessages(false);
      }
    },
    [resetToolActivity],
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

  const refreshConversationTitleIfAuto = useCallback(
    async (conversationId: string | null) => {
      if (!conversationId) {
        return;
      }

      const conversation = conversationsRef.current.find((item) => item.id === conversationId);
      if (!conversation || conversation.title_source !== AUTO_CONVERSATION_TITLE_SOURCE) {
        return;
      }

      await wait(AUTO_TITLE_REFRESH_DELAY_MS);
      await loadConversations();
    },
    [loadConversations],
  );

  const loadMessages = useCallback(async () => {
    const conversationId = activeConversationRef.current;
    if (!conversationId) {
      setMessages([]);
      resetToolActivity();
      return;
    }
    await loadMessagesForConversation(conversationId);
  }, [loadMessagesForConversation, resetToolActivity]);

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

    resetToolActivity();
    setLoading(false);
    setError('Stream stopped by user');
  }, [resetToolActivity]);

  const startNewConversation = useCallback(() => {
    composingNewConversationRef.current = true;
    activeConversationRef.current = null;
    setActiveConversationId(null);
    setMessages([]);
    resetToolActivity();
    setError(null);
  }, [resetToolActivity]);

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
      resetToolActivity();
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear chat');
    }
  }, [removeConversation, resetToolActivity, startNewConversation]);

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
        resetToolActivity();
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
        let streamConversationId: string | null = activeConversationRef.current;
        let assistantCompleted = false;
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
                streamConversationId = eventConversationId;
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
              clearToolCallingStatus();
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

            if (eventType === 'tool_call_started') {
              const toolStartedText = getEventString(data, 'text', 'Text');
              const toolStartedName = getEventString(data, 'function', 'Function');
              if (toolStartedText) {
                updateToolCallingStatus(toolStartedText, toolStartedName);
              }
              const assistantFilters = parseAssistantFilters(data);
              if (assistantFilters !== null) {
                onApplyAssistantFilters?.(assistantFilters);
              }
              return;
            }

            if (eventType === 'tool_call_finished') {
              const shouldRefetch = getEventBoolean(data, 'should_refetch', 'shouldRefetch');
              if (shouldRefetch === true) {
                onToolExecuted?.();
              }
              return;
            }

            if (eventType === 'done') {
              assistantCompleted = true;
              const eventAssistantMessageId = getEventString(data, 'assistant_message_id', 'AssistantMessageID', 'assistantMessageId');
              if (eventAssistantMessageId) {
                assistantMessageId = eventAssistantMessageId;
              }
              resetToolActivity();
              setLoading(false);
              readerRef.current = null;
              abortControllerRef.current = null;
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
              resetToolActivity();
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
            resetToolActivity();
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
        await loadConversations();
        if (assistantCompleted) {
          await refreshConversationTitleIfAuto(streamConversationId);
          onChatDone?.();
        }
        resetToolActivity();
        readerRef.current = null;
      } catch (err) {
        if (err instanceof Error && err.name === 'AbortError') {
          setError('Stream stopped by user');
        } else {
          setError(err instanceof Error ? err.message : 'Failed to send message');
        }
        resetToolActivity();
        setLoading(false);
        readerRef.current = null;
        abortControllerRef.current = null;
      }
    },
    [
      clearToolCallingStatus,
      loadConversations,
      onApplyAssistantFilters,
      onChatDone,
      onToolExecuted,
      refreshConversationTitleIfAuto,
      resetToolActivity,
      selectedModel,
      updateToolCallingStatus,
    ],
  );

  return {
    messages,
    conversations,
    activeConversationId,
    models,
    selectedModel,
    toolCallingStatus,
    toolCallingCount,
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
