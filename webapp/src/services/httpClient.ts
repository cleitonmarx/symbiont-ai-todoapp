import axios from 'axios';

// Default to same-origin so ingress/domain deployments work without build args.
export const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? '').trim();

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      const errorData = error.response.data?.error;
      const message = errorData?.message || error.response.statusText || 'An error occurred';
      const status = error.response.status;
      throw new Error(`[${status}] ${message}`);
    }

    if (error.request) {
      throw new Error('No response from server');
    }

    throw new Error(error.message);
  }
);
