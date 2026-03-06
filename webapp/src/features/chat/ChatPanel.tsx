import { useEffect, useMemo, useRef, useState } from 'react';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import { useChat } from '../../hooks/useChat';
import { useMediaQuery } from '../../hooks/useMediaQuery';
import { fetchAvailableSkills } from '../../services/chatApi';
import type {
  AvailableSkill,
  AssistantTodoFilters,
  ChatMessage,
  ChatMessageActionDetail,
  SelectedSkill,
} from '../../types';

marked.setOptions({
  breaks: true,
  gfm: true,
});

interface ChatPanelProps {
  onChatDone?: () => void;
  onToolExecuted?: () => void;
  onApplyAssistantFilters?: (filters: AssistantTodoFilters) => void;
  mode?: 'panel' | 'sheet';
  onClose?: () => void;
}

const MINUTE_MS = 60 * 1000;
const SECOND_MS = 1000;
const HOUR_MINUTES = 60;
const DAY_MINUTES = 24 * HOUR_MINUTES;
const AUTO_SCROLL_THRESHOLD_PX = 72;

const formatConversationAge = (updatedAt: string): string => {
  const updatedTime = new Date(updatedAt).getTime();
  if (Number.isNaN(updatedTime)) {
    return '0m';
  }

  const diff = Math.max(0, Date.now() - updatedTime);
  const totalMinutes = Math.max(1, diff / MINUTE_MS);

  if (totalMinutes < HOUR_MINUTES) {
    return `${Math.max(1, Math.round(totalMinutes))}m`;
  }
  if (totalMinutes < DAY_MINUTES) {
    return `${Math.max(1, Math.round(totalMinutes / HOUR_MINUTES))}h`;
  }
  return `${Math.max(1, Math.round(totalMinutes / DAY_MINUTES))}d`;
};

const formatCountdown = (remainingMs: number): string => {
  const totalSeconds = Math.max(0, Math.ceil(remainingMs / SECOND_MS));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  const paddedMinutes = String(minutes).padStart(2, '0');
  const paddedSeconds = String(seconds).padStart(2, '0');
  if (hours > 0) {
    return `${hours}:${paddedMinutes}:${paddedSeconds}`;
  }
  return `${paddedMinutes}:${paddedSeconds}`;
};

const formatApprovalInput = (rawInput: string): string => {
  const trimmed = rawInput.trim();
  if (!trimmed) {
    return 'No arguments.';
  }

  try {
    const parsed = JSON.parse(trimmed);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return rawInput;
  }
};

const toPreviewValue = (value: unknown): string => {
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  return JSON.stringify(value);
};

const mergePreviewParts = (parts: string[]): string => {
  if (parts.length === 0) {
    return '';
  }
  if (parts.length === 1) {
    return parts[0];
  }
  return `${parts[0]} (${parts.slice(1).join(' | ')})`;
};

const extractPathValues = (root: unknown, path: string): unknown[] => {
  const trimmed = path.trim();
  if (!trimmed) {
    return [];
  }

  const segments = trimmed.split('.').filter((segment) => segment.length > 0);
  let current: unknown[] = [root];

  for (const segment of segments) {
    const next: unknown[] = [];
    const isArraySegment = segment.endsWith('[]');
    const key = isArraySegment ? segment.slice(0, -2) : segment;

    for (const item of current) {
      if (item === null || item === undefined) {
        continue;
      }

      let value: unknown;
      if (key === '') {
        value = item;
      } else if (typeof item === 'object' && !Array.isArray(item) && key in item) {
        value = (item as Record<string, unknown>)[key];
      } else {
        continue;
      }

      if (isArraySegment) {
        if (Array.isArray(value)) {
          next.push(...value);
        }
        continue;
      }

      next.push(value);
    }

    current = next;
    if (current.length === 0) {
      break;
    }
  }

  return current;
};

interface ArrayPreviewPath {
  arrayPath: string;
  itemPath: string;
}

const parseArrayPreviewPath = (path: string): ArrayPreviewPath | null => {
  const trimmed = path.trim();
  if (!trimmed) {
    return null;
  }

  const segments = trimmed.split('.').filter((segment) => segment.length > 0);
  if (segments.length === 0) {
    return null;
  }

  for (let i = 0; i < segments.length; i++) {
    const segment = segments[i];
    if (!segment.endsWith('[]')) {
      continue;
    }

    const arraySegment = segment.slice(0, -2);
    if (!arraySegment) {
      return null;
    }

    const prefix = [...segments.slice(0, i), arraySegment].join('.');
    const suffix = segments.slice(i + 1).join('.');
    return {
      arrayPath: prefix,
      itemPath: suffix,
    };
  }

  return null;
};

const buildGroupedArrayPreviewItems = (parsedInput: unknown, previewFields: string[]): string[] => {
  const groupedByArrayPath = new Map<string, string[]>();
  const arrayPathOrder: string[] = [];

  for (const rawField of previewFields) {
    const parsed = parseArrayPreviewPath(rawField);
    if (!parsed) {
      continue;
    }
    if (!groupedByArrayPath.has(parsed.arrayPath)) {
      groupedByArrayPath.set(parsed.arrayPath, []);
      arrayPathOrder.push(parsed.arrayPath);
    }

    const currentPaths = groupedByArrayPath.get(parsed.arrayPath) ?? [];
    if (!currentPaths.includes(parsed.itemPath)) {
      currentPaths.push(parsed.itemPath);
      groupedByArrayPath.set(parsed.arrayPath, currentPaths);
    }
  }

  if (groupedByArrayPath.size === 0) {
    return [];
  }

  const rows: string[] = [];
  for (const arrayPath of arrayPathOrder) {
    const itemPaths = groupedByArrayPath.get(arrayPath) ?? [];
    const containers = extractPathValues(parsedInput, arrayPath);

    for (const container of containers) {
      if (!Array.isArray(container)) {
        continue;
      }

      for (const item of container) {
        const parts: string[] = [];
        for (const itemPath of itemPaths) {
          const values = itemPath ? extractPathValues(item, itemPath) : [item];
          const first = values.find((value) => value !== null && value !== undefined);
          if (first === undefined) {
            continue;
          }

          const normalized = toPreviewValue(first).trim();
          if (!normalized) {
            continue;
          }
          parts.push(normalized);
        }

        const merged = mergePreviewParts(parts);
        if (merged && !rows.includes(merged)) {
          rows.push(merged);
        }
      }
    }
  }

  return rows;
};

const buildApprovalPreviewItems = (rawInput: string, previewFields: string[]): string[] => {
  if (previewFields.length === 0) {
    return [];
  }

  let parsedInput: unknown;
  try {
    parsedInput = JSON.parse(rawInput);
  } catch {
    return [];
  }

  const groupedRows = buildGroupedArrayPreviewItems(parsedInput, previewFields);
  if (groupedRows.length > 0) {
    return groupedRows;
  }

  const values: string[] = [];
  for (const fieldPath of previewFields) {
    const extracted = extractPathValues(parsedInput, fieldPath);
    for (const value of extracted) {
      if (value === null || value === undefined) {
        continue;
      }
      const normalized = toPreviewValue(value).trim();
      if (!normalized) {
        continue;
      }
      if (!values.includes(normalized)) {
        values.push(normalized);
      }
    }
  }
  return values;
};

const formatSkillSource = (source: string): string => {
  const trimmed = source.trim();
  if (!trimmed) {
    return 'Unknown source';
  }

  const segments = trimmed.split('/').filter(Boolean);
  return segments[segments.length - 1] ?? trimmed;
};

interface SkillDirectiveQuery {
  query: string;
  replaceStart: number;
  replaceEnd: number;
}

const isWhitespace = (value: string): boolean => /\s/.test(value);
const normalizeInlineWhitespace = (value: string): string => value.replace(/\s+/g, ' ').trim();
const normalizeSkillName = (value: string): string => normalizeInlineWhitespace(value).replace(/^\/+/, '');
const normalizeSkillAliases = (values: string[] | undefined): string[] => {
  if (!values || values.length === 0) {
    return [];
  }

  const next: string[] = [];
  const seen = new Set<string>();
  for (const raw of values) {
    const normalized = normalizeSkillName(raw).toLowerCase();
    if (!normalized || seen.has(normalized)) {
      continue;
    }
    seen.add(normalized);
    next.push(normalized);
  }
  return next;
};

const preferredSkillCommand = (skill: AvailableSkill): string => {
  const aliases = normalizeSkillAliases(skill.aliases);
  if (aliases.length > 0) {
    return aliases[0];
  }
  return normalizeSkillName(skill.name).toLowerCase();
};

const selectedSkillLabel = (canonicalName: string, skills: AvailableSkill[]): string => {
  const normalized = normalizeSkillName(canonicalName).toLowerCase();
  const matched = skills.find((skill) => normalizeSkillName(skill.name).toLowerCase() === normalized);
  if (!matched) {
    return canonicalName;
  }
  return preferredSkillCommand(matched);
};

const normalizeAvailableSkills = (skills: AvailableSkill[]): AvailableSkill[] => {
  const normalized: AvailableSkill[] = [];
  const seenNames = new Set<string>();

  for (const skill of skills) {
    const name = normalizeSkillName(skill.name).toLowerCase();
    if (!name || seenNames.has(name)) {
      continue;
    }

    seenNames.add(name);
    const aliases = normalizeSkillAliases(skill.aliases).filter((alias) => alias !== name);
    normalized.push({
      ...skill,
      name,
      display_name: normalizeInlineWhitespace(skill.display_name ?? name),
      aliases,
      description: normalizeInlineWhitespace(skill.description ?? ''),
    });
  }

  return normalized;
};

const parseSkillDirectiveQuery = (value: string, cursor: number): SkillDirectiveQuery | null => {
  if (!value) {
    return null;
  }

  const safeCursor = Math.max(0, Math.min(cursor, value.length));

  let tokenStart = safeCursor;
  while (tokenStart > 0 && !isWhitespace(value[tokenStart - 1])) {
    tokenStart--;
  }

  let tokenEnd = safeCursor;
  while (tokenEnd < value.length && !isWhitespace(value[tokenEnd])) {
    tokenEnd++;
  }

  const token = value.slice(tokenStart, tokenEnd);
  if (!token.startsWith('/')) {
    return null;
  }

  return {
    query: token.slice(1).toLowerCase(),
    replaceStart: tokenStart,
    replaceEnd: tokenEnd,
  };
};

const filterSkillOptions = (skills: AvailableSkill[], query: string): AvailableSkill[] => {
  if (!query.trim()) {
    return skills.slice(0, 8);
  }
  const normalized = query.toLowerCase();
  return skills
    .filter((skill) => {
      if (normalizeSkillName(skill.name).toLowerCase().includes(normalized)) {
        return true;
      }
      if (normalizeInlineWhitespace(skill.display_name ?? '').toLowerCase().includes(normalized)) {
        return true;
      }
      return normalizeSkillAliases(skill.aliases).some((alias) => alias.includes(normalized));
    })
    .slice(0, 8);
};

const getActionStatusMeta = (
  detail: ChatMessageActionDetail,
): { label: string; tone: 'completed' | 'failed' | 'pending' | 'blocked' } => {
  if (detail.approval_status === 'REJECTED') {
    return { label: 'Rejected', tone: 'blocked' };
  }
  if (detail.approval_status === 'EXPIRED') {
    return { label: 'Expired', tone: 'blocked' };
  }
  if (detail.approval_status === 'AUTO_REJECTED') {
    return { label: 'Auto-rejected', tone: 'blocked' };
  }
  if (detail.approval_status === 'PENDING') {
    return { label: 'Awaiting approval', tone: 'pending' };
  }
  if (detail.message_state === 'COMPLETED' && detail.action_executed !== false) {
    return { label: 'Completed', tone: 'completed' };
  }
  if (detail.message_state === 'FAILED') {
    return { label: 'Failed', tone: 'failed' };
  }
  return { label: 'In progress', tone: 'pending' };
};

const buildDetailsSummary = (message: ChatMessage): { meta: string } | null => {
  const skillCount = message.selected_skills?.length ?? 0;
  const actionCount = message.action_details?.length ?? 0;

  if (skillCount === 0 && actionCount === 0) {
    return null;
  }

  const metaParts: string[] = [];
  if (skillCount > 0) {
    metaParts.push(`${skillCount} skill${skillCount === 1 ? '' : 's'}`);
  }
  if (actionCount > 0) {
    metaParts.push(`${actionCount} action${actionCount === 1 ? '' : 's'}`);
  }

  return { meta: metaParts.join(' · ') };
};

const hasTurnDetails = (message: ChatMessage): boolean =>
  (message.selected_skills?.length ?? 0) > 0 || (message.action_details?.length ?? 0) > 0;

export const ChatPanel = ({
  onChatDone,
  onToolExecuted,
  onApplyAssistantFilters,
  mode = 'panel',
  onClose,
}: ChatPanelProps) => {
  const isViewportCompact = useMediaQuery('(max-width: 960px)');
  const isCompact = mode === 'panel' ? true : isViewportCompact;
  const [activeTab, setActiveTab] = useState<'chat' | 'sessions'>('chat');
  const {
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
    loadModels,
    setSelectedModel,
    sendMessage,
    stopStream,
    submitApproval,
    startNewConversation,
    selectConversation,
    renameConversation,
    removeConversation,
  } = useChat({
    onChatDone,
    onToolExecuted,
    onApplyAssistantFilters,
  });
  const [input, setInput] = useState('');
  const [availableSkills, setAvailableSkills] = useState<AvailableSkill[]>([]);
  const [selectedSkillNames, setSelectedSkillNames] = useState<string[]>([]);
  const [loadingAvailableSkills, setLoadingAvailableSkills] = useState(false);
  const [skillCursorIndex, setSkillCursorIndex] = useState(0);
  const [activeSkillSuggestionIndex, setActiveSkillSuggestionIndex] = useState(0);
  const [renderedMessages, setRenderedMessages] = useState<Record<string, string>>({});
  const [renameConversationId, setRenameConversationId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [pendingDeleteConversationId, setPendingDeleteConversationId] = useState<string | null>(null);
  const [conversationError, setConversationError] = useState<string | null>(null);
  const [approvalReason, setApprovalReason] = useState('');
  const [approvalNow, setApprovalNow] = useState(() => Date.now());
  const [showScrollToLatest, setShowScrollToLatest] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const composerInputRef = useRef<HTMLTextAreaElement>(null);
  const skillDropdownRef = useRef<HTMLDivElement>(null);
  const skillOptionRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const isInitialLoad = useRef(true);
  const shouldAutoScrollRef = useRef(true);

  const activeConversation = useMemo(
    () => conversations.find((conversation) => conversation.id === activeConversationId) ?? null,
    [activeConversationId, conversations],
  );

  useEffect(() => {
    if (!isCompact) {
      setActiveTab('chat');
    }
  }, [isCompact]);

  useEffect(() => {
    if (!pendingApproval?.expiresAt) {
      return;
    }

    setApprovalNow(Date.now());
    const timer = window.setInterval(() => {
      setApprovalNow(Date.now());
    }, SECOND_MS);

    return () => {
      window.clearInterval(timer);
    };
  }, [pendingApproval?.expiresAt]);

  useEffect(() => {
    setApprovalReason('');
  }, [pendingApproval?.actionCallId]);

  useEffect(() => {
    void loadConversations();
    void loadModels();
  }, [loadConversations, loadModels]);

  useEffect(() => {
    let active = true;

    const loadSkills = async () => {
      setLoadingAvailableSkills(true);
      try {
        const skills = await fetchAvailableSkills();
        if (!active) {
          return;
        }
        setAvailableSkills(normalizeAvailableSkills(skills));
      } catch {
        if (!active) {
          return;
        }
        setAvailableSkills([]);
      } finally {
        if (active) {
          setLoadingAvailableSkills(false);
        }
      }
    };

    void loadSkills();
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    const container = messagesContainerRef.current;
    if (messages.length > 0 && container) {
      if (isInitialLoad.current) {
        container.scrollTop = container.scrollHeight;
        isInitialLoad.current = false;
        shouldAutoScrollRef.current = true;
        setShowScrollToLatest(false);
      } else if (shouldAutoScrollRef.current) {
        container.scrollTop = container.scrollHeight;
        setShowScrollToLatest(false);
      } else {
        setShowScrollToLatest(true);
      }
    }
  }, [messages]);

  useEffect(() => {
    isInitialLoad.current = true;
    shouldAutoScrollRef.current = true;
    setShowScrollToLatest(false);
  }, [activeConversationId]);

  useEffect(() => {
    if (loading && !shouldAutoScrollRef.current) {
      setShowScrollToLatest(true);
    }
  }, [loading]);

  const handleMessagesScroll = () => {
    const container = messagesContainerRef.current;
    if (!container) {
      return;
    }

    const distanceFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
    shouldAutoScrollRef.current = distanceFromBottom <= AUTO_SCROLL_THRESHOLD_PX;
    setShowScrollToLatest(!shouldAutoScrollRef.current);
  };

  const scrollToLatestMessage = () => {
    const container = messagesContainerRef.current;
    if (!container) {
      return;
    }

    shouldAutoScrollRef.current = true;
    setShowScrollToLatest(false);
    container.scrollTo({ top: container.scrollHeight, behavior: 'smooth' });
  };

  useEffect(() => {
    let isActive = true;

    const renderMessages = async () => {
      const entries = await Promise.all(
        messages
          .filter((message) => message.role === 'assistant')
          .map(async (message) => {
            const parsed = await Promise.resolve(marked.parse(message.content));
            return [String(message.id), DOMPurify.sanitize(parsed)] as const;
          }),
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
    const messageBody = input.trim();
    const selectedSkillDirectives = selectedSkillNames.map((name) => `/${name}`).join(' ');
    const message = [selectedSkillDirectives, messageBody].filter(Boolean).join(' ').trim();
    if (!message) {
      return;
    }
    setInput('');
    setSelectedSkillNames([]);
    setSkillCursorIndex(0);
    await sendMessage(message);
  };

  const handleRenameSubmit = async () => {
    if (!renameConversationId) {
      return;
    }
    try {
      setConversationError(null);
      await renameConversation(renameConversationId, renameValue);
      setRenameConversationId(null);
      setRenameValue('');
    } catch (err) {
      setConversationError(err instanceof Error ? err.message : 'Failed to rename conversation');
    }
  };

  const handleDeleteConversation = async (conversationId: string) => {
    try {
      setConversationError(null);
      await removeConversation(conversationId);
      setPendingDeleteConversationId((current) => (current === conversationId ? null : current));
    } catch (err) {
      setConversationError(err instanceof Error ? err.message : 'Failed to delete conversation');
    }
  };

  const showSessionsPane = !isCompact || activeTab === 'sessions';
  const showChatPane = !isCompact || activeTab === 'chat';
  const sessionsLocked = loading || loadingMessages;
  const approvalRemainingMs =
    pendingApproval?.expiresAt != null ? Math.max(0, pendingApproval.expiresAt - approvalNow) : null;
  const approvalExpired = approvalRemainingMs === 0 && pendingApproval?.expiresAt != null;
  const approvalInputPreview = pendingApproval ? formatApprovalInput(pendingApproval.input) : '';
  const approvalPreviewItems = pendingApproval
    ? buildApprovalPreviewItems(pendingApproval.input, pendingApproval.previewFields)
    : [];
  const chatGuideText = activeConversationId
    ? 'Break down work and I can apply filters, sort tasks, or run batch updates.'
    : "Tell me what you need to do and I'll turn it into todos, then help you find, sort, and batch-update them.";
  const composerPlaceholder = activeConversationId
    ? 'Ask for follow-up changes. Type / to choose a skill.'
    : 'Start a new conversation. Type / to choose a skill.';
  const skillHelpText =
    selectedSkillNames.length > 0
      ? 'Selected skills apply to the next message. Click a skill pill to remove it.'
      : 'Tip: type / to open skills, then pick one from the list.';
  const skillDirectiveQuery = useMemo(
    () => parseSkillDirectiveQuery(input, skillCursorIndex),
    [input, skillCursorIndex],
  );
  const skillSuggestions = useMemo(() => {
    if (!skillDirectiveQuery) {
      return [];
    }
    const unselectedSkills = availableSkills.filter((skill) => !selectedSkillNames.includes(skill.name));
    return filterSkillOptions(unselectedSkills, skillDirectiveQuery.query);
  }, [availableSkills, skillDirectiveQuery, selectedSkillNames]);
  const showSkillDropdown = skillDirectiveQuery !== null;
  const hasSkillSuggestions = skillSuggestions.length > 0;

  useEffect(() => {
    setActiveSkillSuggestionIndex(0);
  }, [input, skillDirectiveQuery?.query, hasSkillSuggestions]);

  useEffect(() => {
    skillOptionRefs.current = skillOptionRefs.current.slice(0, skillSuggestions.length);
  }, [skillSuggestions.length]);

  useEffect(() => {
    if (!showSkillDropdown || !hasSkillSuggestions) {
      return;
    }
    const container = skillDropdownRef.current;
    const option = skillOptionRefs.current[activeSkillSuggestionIndex];
    if (!container || !option) {
      return;
    }

    const containerTop = container.scrollTop;
    const containerBottom = containerTop + container.clientHeight;
    const optionTop = option.offsetTop;
    const optionBottom = optionTop + option.offsetHeight;

    if (optionTop < containerTop) {
      container.scrollTop = optionTop;
      return;
    }
    if (optionBottom > containerBottom) {
      container.scrollTop = optionBottom - container.clientHeight;
    }
  }, [activeSkillSuggestionIndex, hasSkillSuggestions, showSkillDropdown, skillSuggestions]);

  const handleSkillSuggestionSelect = (skill: AvailableSkill) => {
    if (!skillDirectiveQuery) {
      return;
    }
    const normalizedName = normalizeSkillName(skill.name);
    if (!normalizedName) {
      return;
    }

    const before = input.slice(0, skillDirectiveQuery.replaceStart);
    const after = input.slice(skillDirectiveQuery.replaceEnd);
    const mergedInput = `${before}${after}`.replace(/\s{2,}/g, ' ');
    const normalizedInput = mergedInput.trimStart();
    const trimmedPrefixChars = mergedInput.length - normalizedInput.length;
    const nextCursor = Math.max(0, before.length - trimmedPrefixChars);

    setSelectedSkillNames((current) => {
      if (current.includes(normalizedName)) {
        return current;
      }
      return [...current, normalizedName];
    });
    setInput(normalizedInput);
    setSkillCursorIndex(nextCursor);

    requestAnimationFrame(() => {
      const composer = composerInputRef.current;
      if (!composer) {
        return;
      }
      composer.focus();
      composer.setSelectionRange(nextCursor, nextCursor);
    });
  };

  const handleRemoveSelectedSkill = (skillName: string) => {
    setSelectedSkillNames((current) => current.filter((name) => name !== skillName));
    requestAnimationFrame(() => {
      composerInputRef.current?.focus();
    });
  };

  const handleApprovalSubmit = async (status: 'APPROVED' | 'REJECTED') => {
    if (!pendingApproval || approvalExpired) {
      return;
    }
    await submitApproval(status, approvalReason);
  };

  return (
    <section className={`ui-chat ${mode}`}>
      <header className="ui-chat-topbar">
        <div className="ui-chat-tabs" role="tablist" aria-label="Chat view">
          <button
            type="button"
            role="tab"
            className={`ui-chat-tab ${!isCompact || activeTab === 'chat' ? 'active' : ''}`}
            aria-selected={!isCompact || activeTab === 'chat'}
            onClick={() => setActiveTab('chat')}
          >
            Chat
          </button>
          <button
            type="button"
            role="tab"
            className={`ui-chat-tab ${isCompact && activeTab === 'sessions' ? 'active' : ''}`}
            aria-selected={isCompact && activeTab === 'sessions'}
            onClick={() => setActiveTab('sessions')}
          >
            History
          </button>
        </div>
        <div className="ui-chat-topbar-actions">
          <button
            type="button"
            className="ui-icon-btn"
            title="New conversation"
            aria-label="New conversation"
            onClick={() => {
              setConversationError(null);
              setRenameConversationId(null);
              setRenameValue('');
              startNewConversation();
              if (isCompact) {
                setActiveTab('chat');
              }
            }}
            disabled={sessionsLocked}
          >
            ＋
          </button>
          {showSessionsPane ? (
            <button
              type="button"
              className="ui-icon-btn"
              title="Refresh history"
              aria-label="Refresh history"
              onClick={() => void loadConversations()}
              disabled={sessionsLocked || loadingConversations}
            >
              ↻
            </button>
          ) : null}
          {onClose ? (
            <button type="button" className="ui-icon-btn" onClick={onClose} title="Close chat" aria-label="Close chat">
              ✕
            </button>
          ) : null}
        </div>
      </header>

      <div className={`ui-chat-shell ${isCompact ? 'compact' : 'desktop'}`}>
        {showSessionsPane ? (
          <aside className="ui-chat-conversations" aria-label="Conversations">
            <div className="ui-chat-conversations-header">
              <h3>History</h3>
            </div>

            <div className="ui-chat-conversation-list">
              {conversations.map((conversation) => {
                const isActive = activeConversationId === conversation.id;
                const isEditing = renameConversationId === conversation.id;
                return (
                  <article
                    key={conversation.id}
                    className={`ui-chat-conversation-item-wrap ${isActive ? 'active' : ''}`}
                  >
                    <div className={`ui-chat-conversation-item ui-chat-conversation-item-with-actions ${isActive ? 'active' : ''}`}>
                      <button
                        type="button"
                        className="ui-chat-conversation-main"
                        onClick={() => {
                          setConversationError(null);
                          setPendingDeleteConversationId(null);
                          setRenameConversationId(null);
                          setRenameValue('');
                          void selectConversation(conversation.id);
                          if (isCompact) {
                            setActiveTab('chat');
                          }
                        }}
                        disabled={sessionsLocked}
                      >
                        <span className="ui-chat-session-dot" aria-hidden>
                          •
                        </span>
                        <div className="ui-chat-conversation-body">
                          <div className="ui-chat-conversation-title">{conversation.title}</div>
                          <div className="ui-chat-conversation-date">
                            {formatConversationAge(conversation.updated_at)}
                          </div>
                        </div>
                      </button>

                      <div className="ui-chat-conversation-actions">
                        {pendingDeleteConversationId === conversation.id ? (
                          <>
                            <button
                              type="button"
                              className="ui-btn ui-btn-secondary ui-chat-conversation-confirm-btn"
                              onClick={() => setPendingDeleteConversationId(null)}
                              disabled={sessionsLocked}
                            >
                              Cancel
                            </button>
                            <button
                              type="button"
                              className="ui-btn ui-btn-danger ui-chat-conversation-confirm-btn"
                              onClick={() => void handleDeleteConversation(conversation.id)}
                              disabled={sessionsLocked}
                            >
                              Delete
                            </button>
                          </>
                        ) : (
                          <>
                            <button
                              type="button"
                              className="ui-icon-btn"
                              title="Rename conversation"
                              aria-label="Rename conversation"
                              onClick={() => {
                                setConversationError(null);
                                setPendingDeleteConversationId(null);
                                setRenameConversationId(conversation.id);
                                setRenameValue(conversation.title);
                              }}
                              disabled={sessionsLocked}
                            >
                              ✎
                            </button>
                            <button
                              type="button"
                              className="ui-icon-btn danger"
                              title="Delete conversation"
                              aria-label="Delete conversation"
                              onClick={() => {
                                setConversationError(null);
                                setRenameConversationId(null);
                                setRenameValue('');
                                setPendingDeleteConversationId(conversation.id);
                              }}
                              disabled={sessionsLocked}
                            >
                              🗑
                            </button>
                          </>
                        )}
                      </div>
                    </div>

                    {isEditing ? (
                      <div className="ui-chat-rename-box">
                        <input
                          className="ui-input"
                          value={renameValue}
                          onChange={(event) => setRenameValue(event.target.value)}
                          onKeyDown={(event) => {
                            if (event.key === 'Enter') {
                              event.preventDefault();
                              void handleRenameSubmit();
                            }
                            if (event.key === 'Escape') {
                              setRenameConversationId(null);
                              setRenameValue('');
                            }
                          }}
                          autoFocus
                        />
                        <div className="ui-chat-rename-actions">
                          <button type="button" className="ui-btn ui-btn-primary" onClick={() => void handleRenameSubmit()}>
                            Save
                          </button>
                          <button
                            type="button"
                            className="ui-btn ui-btn-secondary"
                            onClick={() => {
                              setRenameConversationId(null);
                              setRenameValue('');
                            }}
                          >
                            Cancel
                          </button>
                        </div>
                      </div>
                    ) : null}
                  </article>
                );
              })}
            </div>

            {loadingConversations ? <p className="ui-chat-conversation-meta">Loading history...</p> : null}
            {conversationError ? <p className="ui-chat-conversation-error">{conversationError}</p> : null}
          </aside>
        ) : null}

        {showChatPane ? (
          <div className="ui-chat-thread">
            <header className="ui-chat-header">
              <div>
                <h2>{activeConversation?.title ?? 'New conversation'}</h2>
                <p>{chatGuideText}</p>
              </div>
            </header>

            <div className="ui-chat-messages" ref={messagesContainerRef} onScroll={handleMessagesScroll}>
            {error ? <div className="ui-chat-error">{error}</div> : null}

            {!loadingMessages && messages.length === 0 ? (
              <div className="ui-chat-empty-state">
                <strong>{activeConversationId ? 'No messages yet.' : 'Start a new conversation.'}</strong>
                <span>Send a message to begin.</span>
              </div>
            ) : null}

              {messages.map((message) => (
                <article key={message.id} className={`ui-chat-message ${message.role}`}>
                  {(() => {
                    const detailsSummary = buildDetailsSummary(message);

                    return (
                      <>
                        {message.role === 'assistant' && hasTurnDetails(message) ? (
                          <details className="ui-chat-turn-details">
                            <summary className="ui-chat-turn-summary">
                              <span className="ui-chat-turn-summary-line" aria-hidden="true" />
                              <span className="ui-chat-turn-summary-center">
                                <span className="ui-chat-turn-summary-controls">
                                  <span className="ui-chat-turn-summary-meta">{detailsSummary?.meta ?? ''}</span>
                                </span>
                                <span className="ui-chat-turn-summary-icon" aria-hidden="true" />
                              </span>
                              <span className="ui-chat-turn-summary-line" aria-hidden="true" />
                            </summary>

                            {(message.selected_skills?.length ?? 0) > 0 ? (
                              <section className="ui-chat-turn-section">
                                <h3>Skills</h3>
                                <div className="ui-chat-skill-list">
                                  {(message.selected_skills ?? []).map((skill: SelectedSkill) => (
                                    <article
                                      key={`${message.id}-${skill.name}-${skill.source}`}
                                      className="ui-chat-skill-card"
                                    >
                                      <div className="ui-chat-skill-header">
                                        <strong>{skill.name}</strong>
                                        <span>{formatSkillSource(skill.source)}</span>
                                      </div>
                                      {skill.tools.length > 0 ? (
                                        <div className="ui-chat-skill-tools">
                                          {skill.tools.map((tool) => (
                                            <span key={`${skill.name}-${tool}`} className="ui-chat-chip">
                                              {tool}
                                            </span>
                                          ))}
                                        </div>
                                      ) : null}
                                    </article>
                                  ))}
                                </div>
                              </section>
                            ) : null}

                            {(message.action_details?.length ?? 0) > 0 ? (
                              <section className="ui-chat-turn-section">
                                <h3>Actions</h3>
                                <div className="ui-chat-action-list">
                                  {(message.action_details ?? []).map((detail: ChatMessageActionDetail) => {
                                    const status = getActionStatusMeta(detail);
                                    const statusText = detail.text?.trim() || detail.name || 'Action';
                                    const hasRawDetails = Boolean(detail.input?.trim() || detail.output?.trim());

                                    return (
                                      <article
                                        key={`${message.id}-${detail.action_call_id}`}
                                        className="ui-chat-action-card"
                                      >
                                        <div className="ui-chat-action-header">
                                          <div>
                                            <strong>{statusText}</strong>
                                            {detail.name && detail.text !== detail.name ? (
                                              <p className="ui-chat-action-name">{detail.name}</p>
                                            ) : null}
                                          </div>
                                          <span className={`ui-chat-action-badge ${status.tone}`}>{status.label}</span>
                                        </div>

                                        {detail.error_message ? (
                                          <p className="ui-chat-action-meta ui-chat-action-meta-error">
                                            {detail.error_message}
                                          </p>
                                        ) : null}
                                        {detail.approval_decision_reason ? (
                                          <p className="ui-chat-action-meta">
                                            Decision: {detail.approval_decision_reason}
                                          </p>
                                        ) : null}
                                        {detail.output_truncated ? (
                                          <p className="ui-chat-action-meta">Result preview truncated for chat history.</p>
                                        ) : null}

                                        {hasRawDetails ? (
                                          <details className="ui-chat-action-raw">
                                            <summary>Raw input and result</summary>
                                            {detail.input?.trim() ? (
                                              <div className="ui-chat-action-raw-block">
                                                <span>Input</span>
                                                <pre>{formatApprovalInput(detail.input)}</pre>
                                              </div>
                                            ) : null}
                                            {detail.output?.trim() ? (
                                              <div className="ui-chat-action-raw-block">
                                                <span>Result</span>
                                                <pre>{detail.output}</pre>
                                              </div>
                                            ) : null}
                                          </details>
                                        ) : null}
                                      </article>
                                    );
                                  })}
                                </div>
                              </section>
                            ) : null}
                          </details>
                        ) : null}
                        {message.role === 'assistant' ? (
                          renderedMessages[String(message.id)]?.trim() ? (
                            <div className="ui-chat-message-content">
                              <div dangerouslySetInnerHTML={{ __html: renderedMessages[String(message.id)] ?? '' }} />
                            </div>
                          ) : null
                        ) : (
                          <div className="ui-chat-message-content">{message.content}</div>
                        )}
                        <time>{new Date(message.created_at).toLocaleTimeString()}</time>
                      </>
                    );
                  })()}
                </article>
              ))}

              {loading || loadingMessages ? (
                <article className="ui-chat-message assistant">
                  <div className="ui-chat-typing">
                    <span />
                    <span />
                    <span />
                  </div>
                </article>
              ) : null}

              {showScrollToLatest ? (
                <button
                  type="button"
                  className="ui-chat-scroll-latest"
                  onClick={scrollToLatestMessage}
                  aria-label="Scroll to latest message"
                  title="Scroll to latest message"
                >
                  ↓
                </button>
              ) : null}

              <div ref={messagesEndRef} />
            </div>

            {toolCallingStatus ? (
              <div className="ui-chat-tool-status">
                <span>{toolCallingStatus}</span>
                <span className="ui-chat-tool-count">x{toolCallingCount}</span>
              </div>
            ) : null}

            {pendingApproval ? (
              <section className={`ui-chat-approval ${approvalExpired ? 'expired' : ''}`}>
                <header className="ui-chat-approval-header">
                  <h3>{pendingApproval.title}</h3>
                  <span className={`ui-chat-approval-time ${approvalExpired ? 'expired' : ''}`}>
                    {approvalRemainingMs !== null
                      ? approvalExpired
                        ? 'Expired'
                        : `Expires in ${formatCountdown(approvalRemainingMs)}`
                      : 'No timeout'}
                  </span>
                </header>

                <p className="ui-chat-approval-description">{pendingApproval.description}</p>
                <p className="ui-chat-approval-action">
                  <strong>Action:</strong> {pendingApproval.actionName}
                </p>

                {approvalPreviewItems.length > 0 ? (
                  <div className="ui-chat-approval-preview">
                    <strong>Items awaiting approval ({approvalPreviewItems.length})</strong>
                    <ul>
                      {approvalPreviewItems.map((item) => (
                        <li key={item}>{item}</li>
                      ))}
                    </ul>
                  </div>
                ) : (
                  <pre className="ui-chat-approval-input">{approvalInputPreview}</pre>
                )}

                <label className="ui-chat-approval-reason" htmlFor="approval-reason">
                  Reason (optional)
                </label>
                <input
                  id="approval-reason"
                  className="ui-input"
                  value={approvalReason}
                  onChange={(event) => setApprovalReason(event.target.value)}
                  placeholder="Add context for this decision"
                  disabled={approvalSubmitting || approvalExpired}
                />

                <div className="ui-chat-approval-actions">
                  <button
                    type="button"
                    className="ui-btn ui-btn-primary"
                    onClick={() => void handleApprovalSubmit('APPROVED')}
                    disabled={approvalSubmitting || approvalExpired}
                  >
                    {approvalSubmitting ? 'Submitting...' : 'Approve'}
                  </button>
                  <button
                    type="button"
                    className="ui-btn ui-btn-danger"
                    onClick={() => void handleApprovalSubmit('REJECTED')}
                    disabled={approvalSubmitting || approvalExpired}
                  >
                    Reject
                  </button>
                </div>
              </section>
            ) : null}

            <footer className="ui-chat-composer">
              <div className="ui-chat-input-shell">
                <div className="ui-chat-input-wrapper">
                  {selectedSkillNames.length > 0 ? (
                    <div className="ui-chat-selected-skills" aria-label="Selected skills">
                      {selectedSkillNames.map((skillName) => (
                        <button
                          key={skillName}
                          type="button"
                          className="ui-chat-skill-pill"
                          onClick={() => handleRemoveSelectedSkill(skillName)}
                          title={`Remove /${selectedSkillLabel(skillName, availableSkills)}`}
                          aria-label={`Remove /${selectedSkillLabel(skillName, availableSkills)}`}
                        >
                          <span className="ui-chat-skill-pill-label">/{selectedSkillLabel(skillName, availableSkills)}</span>
                          <span className="ui-chat-skill-pill-remove" aria-hidden="true">
                            ×
                          </span>
                        </button>
                      ))}
                    </div>
                  ) : null}

                  <textarea
                    ref={composerInputRef}
                    className="ui-chat-input"
                    value={input}
                    disabled={loading}
                    placeholder={composerPlaceholder}
                    onChange={(event) => {
                      setInput(event.target.value);
                      setSkillCursorIndex(event.target.selectionStart ?? event.target.value.length);
                    }}
                    onClick={(event) => setSkillCursorIndex(event.currentTarget.selectionStart ?? event.currentTarget.value.length)}
                    onSelect={(event) =>
                      setSkillCursorIndex(event.currentTarget.selectionStart ?? event.currentTarget.value.length)
                    }
                    onKeyUp={(event) => setSkillCursorIndex(event.currentTarget.selectionStart ?? event.currentTarget.value.length)}
                    onKeyDown={(event) => {
                      if (event.key === 'Backspace' && input.trim() === '' && selectedSkillNames.length > 0) {
                        event.preventDefault();
                        const lastSkill = selectedSkillNames[selectedSkillNames.length - 1];
                        if (lastSkill) {
                          handleRemoveSelectedSkill(lastSkill);
                        }
                        return;
                      }

                      if (showSkillDropdown && hasSkillSuggestions) {
                        if (event.key === 'ArrowDown') {
                          event.preventDefault();
                          setActiveSkillSuggestionIndex((current) => (current + 1) % skillSuggestions.length);
                          return;
                        }
                        if (event.key === 'ArrowUp') {
                          event.preventDefault();
                          setActiveSkillSuggestionIndex((current) =>
                            (current - 1 + skillSuggestions.length) % skillSuggestions.length,
                          );
                          return;
                        }
                        if ((event.key === 'Enter' && !event.shiftKey) || event.key === 'Tab') {
                          event.preventDefault();
                          const selected =
                            skillSuggestions[Math.min(activeSkillSuggestionIndex, Math.max(0, skillSuggestions.length - 1))];
                          if (selected) {
                            handleSkillSuggestionSelect(selected);
                          }
                          return;
                        }
                      }

                      if (event.key === 'Enter' && !event.shiftKey) {
                        event.preventDefault();
                        void handleSend();
                      }
                    }}
                  />

                  {showSkillDropdown ? (
                    <div ref={skillDropdownRef} className="ui-chat-skill-dropdown" role="listbox" aria-label="Available skills">
                      {loadingAvailableSkills ? (
                        <div className="ui-chat-skill-empty">Loading skills...</div>
                      ) : hasSkillSuggestions ? (
                        skillSuggestions.map((skill, index) => {
                          const normalizedName = normalizeSkillName(skill.name);
                          const command = preferredSkillCommand(skill);
                          const displayName = normalizeInlineWhitespace(skill.display_name ?? '');
                          const description = normalizeInlineWhitespace(skill.description ?? '');
                          return (
                            <button
                              key={normalizedName}
                              ref={(element) => {
                                skillOptionRefs.current[index] = element;
                              }}
                              type="button"
                              className={`ui-chat-skill-option ${index === activeSkillSuggestionIndex ? 'active' : ''}`}
                              role="option"
                              aria-selected={index === activeSkillSuggestionIndex}
                              onMouseDown={(event) => event.preventDefault()}
                              onClick={() => handleSkillSuggestionSelect(skill)}
                            >
                              <span className="ui-chat-skill-option-name">/{command}</span>
                              {description ? (
                                <span className="ui-chat-skill-option-hint">{description}</span>
                              ) : displayName ? (
                                <span className="ui-chat-skill-option-hint">{displayName}</span>
                              ) : null}
                            </button>
                          );
                        })
                      ) : (
                        <div className="ui-chat-skill-empty">No matching skills.</div>
                      )}
                    </div>
                  ) : null}

                  <p className="ui-chat-skill-help">{skillHelpText}</p>
                </div>

                <div className="ui-chat-input-controls">
                  <div className="ui-chat-input-meta">
                    <select
                      className="ui-chat-model-select"
                      aria-label="AI model"
                      value={selectedModel}
                      disabled={loading || loadingModels || models.length === 0}
                      onChange={(event) => setSelectedModel(event.target.value)}
                    >
                      {models.length === 0 ? (
                        <option value="">{loadingModels ? 'Loading models...' : 'Default model'}</option>
                      ) : (
                        models.map((model) => (
                          <option key={model.id} value={model.id}>
                            {model.name}
                          </option>
                        ))
                      )}
                    </select>
                  </div>

                  {loading ? (
                    <button type="button" className="ui-chat-send-btn stop" onClick={stopStream} aria-label="Stop stream">
                      ■
                    </button>
                  ) : (
                    <button
                      type="button"
                      className="ui-chat-send-btn"
                      onClick={() => void handleSend()}
                      disabled={!input.trim() && selectedSkillNames.length === 0}
                      aria-label="Send message"
                    >
                      ↑
                    </button>
                  )}
                </div>
              </div>
            </footer>
          </div>
        ) : null}
      </div>
    </section>
  );
};
