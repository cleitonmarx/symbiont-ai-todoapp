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

interface StreamMetaEventData {
  conversation_id?: string;
  user_message_id?: string;
  assistant_message_id?: string;
  conversation_created?: boolean;
}

interface StreamDeltaEventData {
  text?: string;
}

interface StreamToolCallStartedEventData {
  id?: string;
  function?: string;
  arguments?: string;
  text?: string;
}

interface StreamToolCallFinishedEventData {
  id?: string;
  function?: string;
  success?: boolean;
  error?: string;
  should_refetch?: boolean;
}

interface StreamDoneEventData {
  assistant_message_id?: string;
  completed_at?: string;
}

interface StreamErrorEventData {
  error?: string;
}

interface SetUIFiltersArguments {
  status?: string;
  search_by_similarity?: string;
  search_by_title?: string;
  sort_by?: string;
  due_after?: string;
  due_before?: string;
  page?: number;
  page_size?: number;
}

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

const parseSetUIFiltersArguments = (rawArguments?: string): SetUIFiltersArguments => {
  if (!rawArguments) {
    return {};
  }

  return JSON.parse(rawArguments) as SetUIFiltersArguments;
};

const parseAssistantFilters = (data: StreamToolCallStartedEventData): AssistantTodoFilters | null => {
  if (data.function !== 'set_ui_filters') {
    return null;
  }

  const args = parseSetUIFiltersArguments(data.arguments);

  const filters: AssistantTodoFilters = {};
  if (args.status === 'OPEN' || args.status === 'DONE') {
    filters.status = args.status;
  }

  if (args.search_by_similarity) {
    filters.searchQuery = args.search_by_similarity;
    filters.searchType = 'SIMILARITY';
  } else if (args.search_by_title) {
    filters.searchQuery = args.search_by_title;
    filters.searchType = 'TITLE';
  }

  if (args.sort_by && TODO_SORT_OPTIONS.has(args.sort_by)) {
    filters.sortBy = args.sort_by as AssistantTodoFilters['sortBy'];
  }

  if (args.due_after) {
    filters.dueAfter = args.due_after;
  }

  if (args.due_before) {
    filters.dueBefore = args.due_before;
  }

  if (typeof args.page === 'number') {
    filters.page = args.page;
  }

  if (typeof args.page_size === 'number') {
    filters.pageSize = args.page_size;
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
            const rawData = JSON.parse(dataStr);

            if (eventType === 'meta') {
              const data = rawData as StreamMetaEventData;

              if (data.conversation_id) {
                streamConversationId = data.conversation_id;
                activeConversationRef.current = data.conversation_id;
                setActiveConversationId(data.conversation_id);
              }
              if (data.user_message_id) {
                const userMessageId = data.user_message_id;
                setMessages((prev) => {
                  const next = [...prev];
                  for (let i = next.length - 1; i >= 0; i--) {
                    if (next[i].role === 'user') {
                      next[i] = { ...next[i], id: userMessageId };
                      break;
                    }
                  }
                  return next;
                });
              }
              if (data.assistant_message_id) {
                assistantMessageId = data.assistant_message_id;
              }
              if (data.conversation_created === true) {
                composingNewConversationRef.current = false;
              }
              return;
            }

            if (eventType === 'delta') {
              const data = rawData as StreamDeltaEventData;
              if (!data.text) {
                return;
              }
              clearToolCallingStatus();
              assistantContent += data.text;

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
              const data = rawData as StreamToolCallStartedEventData;
              if (data.text) {
                updateToolCallingStatus(data.text, data.function);
              }
              const assistantFilters = parseAssistantFilters(data);
              if (assistantFilters !== null) {
                onApplyAssistantFilters?.(assistantFilters);
              }
              return;
            }

            if (eventType === 'tool_call_finished') {
              const data = rawData as StreamToolCallFinishedEventData;
              if (data.should_refetch === true) {
                onToolExecuted?.();
              }
              return;
            }

            if (eventType === 'done') {
              const data = rawData as StreamDoneEventData;
              assistantCompleted = true;
              if (data.assistant_message_id) {
                assistantMessageId = data.assistant_message_id;
              }
              resetToolActivity();
              setLoading(false);
              readerRef.current = null;
              abortControllerRef.current = null;
              return;
            }

            if (eventType === 'error') {
              const data = rawData as StreamErrorEventData;
              const errorCode = data.error;
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
