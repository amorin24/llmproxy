import { ModelStatus, QueryRequest, SingleModelResponse, MultiModelResponse } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const api = {
  async fetchStatus(): Promise<ModelStatus> {
    const response = await fetch(`${API_BASE_URL}/api/status`);
    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }
    return response.json();
  },

  async submitQuery(request: QueryRequest): Promise<SingleModelResponse> {
    const response = await fetch(`${API_BASE_URL}/api/query`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });
    
    if (!response.ok) {
      const text = await response.text();
      try {
        const errorData = JSON.parse(text);
        throw new Error(errorData.error || `HTTP error! Status: ${response.status}`);
      } catch {
        throw new Error(text || `HTTP error! Status: ${response.status}`);
      }
    }
    
    return response.json();
  },

  async submitParallelQuery(request: QueryRequest): Promise<MultiModelResponse> {
    const response = await fetch(`${API_BASE_URL}/api/parallel`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });
    
    if (!response.ok) {
      const text = await response.text();
      try {
        const errorData = JSON.parse(text);
        throw new Error(errorData.error || `HTTP error! Status: ${response.status}`);
      } catch {
        throw new Error(text || `HTTP error! Status: ${response.status}`);
      }
    }
    
    return response.json();
  },
};

export const generateRequestId = (): string => {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
};
