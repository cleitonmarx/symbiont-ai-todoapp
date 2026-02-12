import axios from 'axios';
import { apiClient } from './httpClient';

export interface BoardSummary {
  counts: {
    OPEN: number;
    DONE: number;
  };
  next_up: Array<{
    title: string;
    reason: string;
  }>;
  overdue: string[];
  near_deadline: string[];
  summary: string;
}

export const getBoardSummary = async (): Promise<BoardSummary | null> => {
  try {
    const response = await apiClient.get<BoardSummary>('/api/v1/board/summary');
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.status === 404) {
      return null;
    }
    return null;
  }
};
