import { apiClient, API_BASE_URL } from './httpClient';

export const streamChat = async (message: string, model: string, signal?: AbortSignal) => {
  const response = await fetch(`${API_BASE_URL}/api/v1/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message,
      model,
    }),
    signal,
  });

  if (!response.ok) {
    throw new Error('Failed to stream chat');
  }

  return response;
};

export const fetchChatMessages = async (page: number, pageSize: number) => {
  const response = await apiClient.get('/api/v1/chat/messages', {
    params: { page, pagesize: pageSize },
  });
  return response.data;
};

export const clearChatMessages = async () => {
  await apiClient.delete('/api/v1/chat/messages');
};

export const fetchAvailableModels = async (): Promise<string[]> => {
  const response = await apiClient.get('/api/v1/models');
  return response.data?.models ?? [];
};
