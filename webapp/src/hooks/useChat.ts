import { useState, useCallback, useRef, useEffect } from 'react';
import {
  fetchChatMessages,
  deleteConversation,
  fetchAvailableModels,
  listConversations,
  submitActionApproval,
  streamChat,
  updateConversation,
  type ActionApprovalStatus,
} from '../services/chatApi';
import type {
  AssistantTodoFilters,
  ChatMessage,
  ChatMessageActionDetail,
  ChatMessageApprovalStatus,
  ChatMessageState,
  Conversation,
  ModelInfo,
  SelectedSkill,
} from '../types';

interface UseChatReturn {
  messages: ChatMessage[];
  conversations: Conversation[];
  activeConversationId: string | null;
  models: ModelInfo[];
  selectedModel: string;
  toolCallingStatus: string | null;
  toolCallingCount: number;
  pendingApproval: ActionApprovalRequest | null;
  approvalSubmitting: boolean;
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
  submitApproval: (status: ActionApprovalStatus, reason?: string) => Promise<void>;
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

interface StreamTurnStartedEventData {
  conversation_id?: string;
  user_message_id?: string;
  assistant_message_id?: string;
  turn_id?: string;
  selected_skills?: unknown;
  conversation_created?: boolean;
}

interface StreamMessageDeltaEventData {
  text?: string;
}

interface StreamActionStartedEventData {
  id?: string;
  name?: string;
  input?: string;
  text?: string;
}

interface StreamActionCompletedEventData {
  id?: string;
  name?: string;
  success?: boolean;
  error?: string;
  should_refetch?: boolean;
  approval_status?: ChatMessageApprovalStatus;
  action_executed?: boolean;
  output_preview?: string;
  output_truncated?: boolean;
}

interface StreamActionApprovalRequiredEventData {
  conversation_id?: string;
  turn_id?: string;
  action_call_id?: string;
  name?: string;
  input?: string;
  title?: string;
  description?: string;
  preview_fields?: unknown;
  timeout?: unknown;
}

interface StreamActionApprovalResolvedEventData {
  conversation_id?: string;
  turn_id?: string;
  action_call_id?: string;
  name?: string;
  status?: ActionApprovalStatus | 'EXPIRED' | 'AUTO_REJECTED' | string;
  reason?: string;
}

interface StreamTurnCompletedEventData {
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

export interface ActionApprovalRequest {
  conversationId: string;
  turnId: string;
  actionCallId: string;
  actionName: string;
  input: string;
  title: string;
  description: string;
  previewFields: string[];
  timeoutMs: number | null;
  expiresAt: number | null;
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

const resolveSelectedModelId = (availableModels: ModelInfo[], currentSelection: string): string => {
  if (!currentSelection) {
    return availableModels[0]?.id ?? '';
  }

  const matchingById = availableModels.find((model) => model.id === currentSelection);
  if (matchingById) {
    return matchingById.id;
  }

  const matchingByName = availableModels.find((model) => model.name === currentSelection);
  if (matchingByName) {
    return matchingByName.id;
  }

  return availableModels[0]?.id ?? '';
};

const parseSetUIFiltersArguments = (rawInput?: string): SetUIFiltersArguments => {
  if (!rawInput) {
    return {};
  }

  return JSON.parse(rawInput) as SetUIFiltersArguments;
};

const parseAssistantFilters = (data: StreamActionStartedEventData): AssistantTodoFilters | null => {
  if (data.name !== 'set_ui_filters') {
    return null;
  }

  const args = parseSetUIFiltersArguments(data.input);

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

const durationUnitToMs: Record<string, number> = {
  ns: 1 / 1e6,
  us: 1 / 1e3,
  'µs': 1 / 1e3,
  ms: 1,
  s: 1e3,
  m: 60 * 1e3,
  h: 60 * 60 * 1e3,
};

const parseDurationStringToMs = (raw: string): number | null => {
  const trimmed = raw.trim();
  if (!trimmed) {
    return null;
  }

  const re = /(-?\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)/g;
  let total = 0;
  let consumed = 0;
  for (const match of trimmed.matchAll(re)) {
    const value = Number.parseFloat(match[1]);
    const unit = match[2];
    if (!Number.isFinite(value) || !(unit in durationUnitToMs)) {
      return null;
    }
    total += value * durationUnitToMs[unit];
    consumed += match[0].length;
  }

  if (consumed !== trimmed.length) {
    return null;
  }
  return Math.max(0, Math.round(total));
};

const parseApprovalTimeoutMs = (raw: unknown): number | null => {
  if (typeof raw === 'number' && Number.isFinite(raw)) {
    // Go time.Duration JSON value is an integer of nanoseconds.
    if (raw <= 0) {
      return null;
    }
    return Math.max(0, Math.round(raw / 1e6));
  }

  if (typeof raw === 'string') {
    const ms = parseDurationStringToMs(raw);
    if (ms !== null && ms > 0) {
      return ms;
    }
  }

  return null;
};

const normalizePreviewFields = (raw: unknown): string[] => {
  if (!Array.isArray(raw)) {
    return [];
  }

  const result: string[] = [];
  for (const item of raw) {
    if (typeof item !== 'string') {
      continue;
    }
    const trimmed = item.trim();
    if (!trimmed) {
      continue;
    }
    if (!result.includes(trimmed)) {
      result.push(trimmed);
    }
  }
  return result;
};

const wait = (ms: number): Promise<void> =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const normalizeSkillTools = (value: unknown): string[] => {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter((item): item is string => typeof item === 'string' && item.trim().length > 0);
};

const normalizeSelectedSkills = (raw: unknown): SelectedSkill[] | undefined => {
  if (!Array.isArray(raw) || raw.length === 0) {
    return undefined;
  }

  const normalized: SelectedSkill[] = [];
  for (const item of raw) {
    if (!item || typeof item !== 'object') {
      continue;
    }

    const record = item as Record<string, unknown>;
    const name = typeof record.name === 'string' ? record.name : typeof record.Name === 'string' ? record.Name : '';
    const source =
      typeof record.source === 'string' ? record.source : typeof record.Source === 'string' ? record.Source : '';

    if (!name.trim()) {
      continue;
    }

    normalized.push({
      name,
      source,
      tools: normalizeSkillTools(record.tools ?? record.Tools),
    });
  }

  return normalized.length > 0 ? normalized : undefined;
};

const cloneSelectedSkills = (skills?: SelectedSkill[] | unknown): SelectedSkill[] | undefined => {
  const normalized = normalizeSelectedSkills(skills);
  if (!normalized) {
    return undefined;
  }

  return normalized.map((skill) => ({
    ...skill,
    tools: [...skill.tools],
  }));
};

const cloneActionDetails = (details?: ChatMessageActionDetail[]): ChatMessageActionDetail[] | undefined => {
  if (!details || details.length === 0) {
    return undefined;
  }

  return details.map((detail) => ({ ...detail }));
};

const normalizeMessage = (message: ChatMessage): ChatMessage => ({
  ...message,
  selected_skills: cloneSelectedSkills(message.selected_skills),
  action_details: cloneActionDetails(message.action_details),
});

const updateMessageById = (
  messages: ChatMessage[],
  messageId: string,
  updater: (message: ChatMessage) => ChatMessage,
): ChatMessage[] => {
  const index = messages.findIndex((message) => message.id === messageId);
  if (index === -1) {
    return messages;
  }

  const next = [...messages];
  next[index] = updater(next[index]);
  return next;
};

const renameMessageId = (messages: ChatMessage[], previousId: string, nextId: string): ChatMessage[] => {
  if (!previousId || previousId === nextId) {
    return messages;
  }

  return updateMessageById(messages, previousId, (message) => ({
    ...message,
    id: nextId,
  }));
};

const upsertActionDetail = (
  details: ChatMessageActionDetail[] | undefined,
  actionCallId: string,
  patch: Partial<ChatMessageActionDetail>,
): ChatMessageActionDetail[] => {
  const normalizedActionCallId = actionCallId.trim();
  if (!normalizedActionCallId) {
    return details ? [...details] : [];
  }

  const nextDetails = details ? details.map((detail) => ({ ...detail })) : [];
  const index = nextDetails.findIndex((detail) => detail.action_call_id === normalizedActionCallId);

  if (index === -1) {
    nextDetails.push({
      action_call_id: normalizedActionCallId,
      name: patch.name ?? normalizedActionCallId,
      ...patch,
    });
    return nextDetails;
  }

  nextDetails[index] = {
    ...nextDetails[index],
    ...patch,
    action_call_id: normalizedActionCallId,
    name: patch.name ?? nextDetails[index].name,
  };
  return nextDetails;
};

export const useChat = ({
  onChatDone,
  onToolExecuted,
  onApplyAssistantFilters,
}: UseChatOptions = {}): UseChatReturn => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConversationId, setActiveConversationId] = useState<string | null>(null);
  const [models, setModels] = useState<ModelInfo[]>([]);
  const [selectedModel, setSelectedModel] = useState(loadPersistedModel);
  const [toolCallingStatus, setToolCallingStatus] = useState<string | null>(null);
  const [toolCallingCount, setToolCallingCount] = useState(0);
  const [pendingApproval, setPendingApproval] = useState<ActionApprovalRequest | null>(null);
  const [approvalSubmitting, setApprovalSubmitting] = useState(false);
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

  const clearPendingApproval = useCallback(() => {
    setPendingApproval(null);
    setApprovalSubmitting(false);
  }, []);

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
        clearPendingApproval();
        const response = await fetchChatMessages(conversationId, 1, CHAT_MESSAGES_PAGE_SIZE);
        setMessages((response.messages || []).map(normalizeMessage));
        resetToolActivity();
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load messages');
      } finally {
        setLoadingMessages(false);
      }
    },
    [clearPendingApproval, resetToolActivity],
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
      setSelectedModel((current) => resolveSelectedModelId(availableModels, current));
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
    clearPendingApproval();
    setLoading(false);
    setError('Stream stopped by user');
  }, [clearPendingApproval, resetToolActivity]);

  const startNewConversation = useCallback(() => {
    composingNewConversationRef.current = true;
    activeConversationRef.current = null;
    setActiveConversationId(null);
    setMessages([]);
    resetToolActivity();
    clearPendingApproval();
    setError(null);
  }, [clearPendingApproval, resetToolActivity]);

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

  const submitApprovalDecision = useCallback(
    async (status: ActionApprovalStatus, reason?: string) => {
      const approval = pendingApproval;
      if (!approval) {
        return;
      }

      try {
        setApprovalSubmitting(true);
        await submitActionApproval({
          conversation_id: approval.conversationId,
          turn_id: approval.turnId,
          action_call_id: approval.actionCallId,
          action_name: approval.actionName || undefined,
          status,
          reason: reason?.trim() ? reason.trim() : undefined,
        });
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to submit action approval');
      } finally {
        setApprovalSubmitting(false);
      }
    },
    [pendingApproval],
  );

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
        clearPendingApproval();
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
        let currentAssistantMessageId = '';
        let streamConversationId: string | null = activeConversationRef.current;
        let assistantCompleted = false;
        let buffer = '';

        const tempAssistantMsg: ChatMessage = {
          id: `tmp-assistant-${Date.now()}`,
          role: 'assistant',
          content: '',
          created_at: new Date().toISOString(),
        };
        currentAssistantMessageId = tempAssistantMsg.id;
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

            if (eventType === 'turn_started') {
              const data = rawData as StreamTurnStartedEventData;

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
                const previousAssistantMessageId = currentAssistantMessageId;
                assistantMessageId = data.assistant_message_id;
                currentAssistantMessageId = data.assistant_message_id;
                setMessages((prev) =>
                  renameMessageId(prev, previousAssistantMessageId, data.assistant_message_id as string),
                );
              }
              setMessages((prev) =>
                updateMessageById(prev, currentAssistantMessageId, (message) => ({
                  ...message,
                  id: currentAssistantMessageId,
                  turn_id: typeof data.turn_id === 'string' ? data.turn_id : message.turn_id,
                  selected_skills:
                    cloneSelectedSkills(data.selected_skills) ?? cloneSelectedSkills(message.selected_skills),
                })),
              );
              if (data.conversation_created === true) {
                composingNewConversationRef.current = false;
              }
              return;
            }

            if (eventType === 'message_delta') {
              const data = rawData as StreamMessageDeltaEventData;
              if (!data.text) {
                return;
              }
              clearToolCallingStatus();
              assistantContent += data.text;

              setMessages((prev) => {
                const nextAssistantMessageId = assistantMessageId || currentAssistantMessageId;
                currentAssistantMessageId = nextAssistantMessageId;
                return updateMessageById(prev, nextAssistantMessageId, (message) => ({
                  ...message,
                  id: nextAssistantMessageId,
                  content: assistantContent,
                }));
              });
              return;
            }

            if (eventType === 'action_started') {
              const data = rawData as StreamActionStartedEventData;
              if (data.text) {
                updateToolCallingStatus(data.text, data.name);
              }
              if (data.id) {
                setMessages((prev) =>
                  updateMessageById(prev, currentAssistantMessageId, (message) => ({
                    ...message,
                    action_details: upsertActionDetail(message.action_details, data.id as string, {
                      name: typeof data.name === 'string' ? data.name : undefined,
                      input: typeof data.input === 'string' ? data.input : undefined,
                      text: typeof data.text === 'string' ? data.text : undefined,
                    }),
                  })),
                );
              }
              const assistantFilters = parseAssistantFilters(data);
              if (assistantFilters !== null) {
                onApplyAssistantFilters?.(assistantFilters);
              }
              return;
            }

            if (eventType === 'action_completed') {
              const data = rawData as StreamActionCompletedEventData;
              if (data.id) {
                const nextMessageState: ChatMessageState | undefined =
                  data.success === true ? 'COMPLETED' : data.success === false ? 'FAILED' : undefined;
                setMessages((prev) =>
                  updateMessageById(prev, currentAssistantMessageId, (message) => ({
                    ...message,
                    action_details: upsertActionDetail(message.action_details, data.id as string, {
                      name: typeof data.name === 'string' ? data.name : undefined,
                      message_state: nextMessageState,
                      error_message: typeof data.error === 'string' ? data.error : undefined,
                      approval_status: data.approval_status,
                      action_executed:
                        typeof data.action_executed === 'boolean' ? data.action_executed : undefined,
                      output: typeof data.output_preview === 'string' ? data.output_preview : undefined,
                      output_truncated:
                        typeof data.output_truncated === 'boolean' ? data.output_truncated : undefined,
                    }),
                  })),
                );
              }
              if (data.should_refetch === true) {
                onToolExecuted?.();
              }
              return;
            }

            if (eventType === 'action_approval_required') {
              const data = rawData as StreamActionApprovalRequiredEventData;
              if (!data.conversation_id || !data.turn_id || !data.action_call_id || !data.name) {
                return;
              }

              const timeoutMs = parseApprovalTimeoutMs(data.timeout);
              setPendingApproval({
                conversationId: data.conversation_id,
                turnId: data.turn_id,
                actionCallId: data.action_call_id,
                actionName: data.name,
                input: typeof data.input === 'string' ? data.input : '',
                title: typeof data.title === 'string' && data.title.trim() ? data.title.trim() : 'Approval required',
                description:
                  typeof data.description === 'string' && data.description.trim()
                    ? data.description.trim()
                    : `Approve action '${data.name}' execution.`,
                previewFields: normalizePreviewFields(data.preview_fields),
                timeoutMs,
                expiresAt: timeoutMs ? Date.now() + timeoutMs : null,
              });
              setMessages((prev) =>
                updateMessageById(prev, currentAssistantMessageId, (message) => ({
                  ...message,
                  turn_id: data.turn_id,
                  action_details: upsertActionDetail(message.action_details, data.action_call_id as string, {
                    name: data.name,
                    input: typeof data.input === 'string' ? data.input : undefined,
                    approval_status: 'PENDING',
                  }),
                })),
              );
              return;
            }

            if (eventType === 'action_approval_resolved') {
              const data = rawData as StreamActionApprovalResolvedEventData;
              const resolvedActionCallID =
                typeof data.action_call_id === 'string' && data.action_call_id.trim()
                  ? data.action_call_id.trim()
                  : '';
              if (!resolvedActionCallID) {
                clearPendingApproval();
                return;
              }

              setPendingApproval((current) => {
                if (!current || current.actionCallId === resolvedActionCallID) {
                  return null;
                }
                return current;
              });
              setMessages((prev) =>
                updateMessageById(prev, currentAssistantMessageId, (message) => ({
                  ...message,
                  action_details: upsertActionDetail(message.action_details, resolvedActionCallID, {
                    name: typeof data.name === 'string' ? data.name : undefined,
                    approval_status:
                      typeof data.status === 'string'
                        ? (data.status as ChatMessageApprovalStatus)
                        : undefined,
                    approval_decision_reason: typeof data.reason === 'string' ? data.reason : undefined,
                    approval_decided_at: new Date().toISOString(),
                    action_executed:
                      data.status === 'APPROVED'
                        ? undefined
                        : data.status === 'REJECTED' ||
                            data.status === 'EXPIRED' ||
                            data.status === 'AUTO_REJECTED'
                          ? false
                          : undefined,
                  }),
                })),
              );
              return;
            }

            if (eventType === 'turn_completed') {
              const data = rawData as StreamTurnCompletedEventData;
              assistantCompleted = true;
              if (data.assistant_message_id) {
                const previousAssistantMessageId = currentAssistantMessageId;
                assistantMessageId = data.assistant_message_id;
                currentAssistantMessageId = data.assistant_message_id;
                setMessages((prev) =>
                  renameMessageId(prev, previousAssistantMessageId, data.assistant_message_id as string),
                );
              }
              clearPendingApproval();
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
              clearPendingApproval();
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
            clearPendingApproval();
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
            const previousAssistantMessageId = currentAssistantMessageId;
            currentAssistantMessageId = assistantMessageId;
            return renameMessageId(prev, previousAssistantMessageId, assistantMessageId);
          });
        }

        setLoading(false);
        await loadConversations();
        if (assistantCompleted) {
          await refreshConversationTitleIfAuto(streamConversationId);
          onChatDone?.();
        }
        clearPendingApproval();
        resetToolActivity();
        readerRef.current = null;
      } catch (err) {
        if (err instanceof Error && err.name === 'AbortError') {
          setError('Stream stopped by user');
        } else {
          setError(err instanceof Error ? err.message : 'Failed to send message');
        }
        clearPendingApproval();
        resetToolActivity();
        setLoading(false);
        readerRef.current = null;
        abortControllerRef.current = null;
      }
    },
    [
      clearToolCallingStatus,
      clearPendingApproval,
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
    pendingApproval,
    approvalSubmitting,
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
    submitApproval: submitApprovalDecision,
    startNewConversation,
    selectConversation,
    renameConversation,
    removeConversation,
  };
};
