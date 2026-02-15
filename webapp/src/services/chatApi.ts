import { apiClient, API_BASE_URL } from './httpClient';
import type { Conversation, ConversationListResp } from '../types';

export const streamChat = async (
  message: string,
  model: string,
  conversationId?: string | null,
  signal?: AbortSignal,
) => {
  const response = await fetch(`${API_BASE_URL}/api/v1/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message,
      model,
      ...(conversationId ? { conversation_id: conversationId } : {}),
    }),
    signal,
  });

  if (!response.ok) {
    throw new Error('Failed to stream chat');
  }

  return response;
};

export const fetchChatMessages = async (conversationId: string, page: number, pageSize: number) => {
  const response = await apiClient.get('/api/v1/chat/messages', {
    params: { conversation_id: conversationId, page, pageSize },
  });
  return response.data;
};

export const listConversations = async (page: number, pageSize: number): Promise<ConversationListResp> => {
  const response = await apiClient.get('/api/conversations', {
    params: { page, pageSize },
  });
  return response.data;
};

export const updateConversation = async (conversationId: string, title: string): Promise<Conversation> => {
  const response = await apiClient.patch(`/api/conversations/${conversationId}`, { title });
  return response.data;
};

export const deleteConversation = async (conversationId: string): Promise<void> => {
  await apiClient.delete(`/api/conversations/${conversationId}`);
};

export const clearChatMessages = async (conversationId: string): Promise<void> => {
  await deleteConversation(conversationId);
};

export const fetchAvailableModels = async (): Promise<string[]> => {
  const response = await apiClient.get('/api/v1/models');
  return response.data?.models ?? [];
};
