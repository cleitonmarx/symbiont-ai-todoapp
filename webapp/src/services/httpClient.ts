import axios from 'axios';

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

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
