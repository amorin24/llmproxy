import React, { createContext, useContext, useState } from 'react';

export interface QueryHistoryItem {
  id: string;
  query: string;
  model?: string;
  models?: string[];
  taskType?: string;
  response: string;
  timestamp: Date;
  responseTime: number;
  cached: boolean;
  tokens?: number;
}

interface HistoryContextType {
  history: QueryHistoryItem[];
  addToHistory: (item: Omit<QueryHistoryItem, 'id' | 'timestamp'>) => void;
  clearHistory: () => void;
}

const HistoryContext = createContext<HistoryContextType | undefined>(undefined);

export const HistoryProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [history, setHistory] = useState<QueryHistoryItem[]>(() => {
    const savedHistory = localStorage.getItem('queryHistory');
    if (savedHistory) {
      try {
        const parsed = JSON.parse(savedHistory);
        return parsed.map((item: QueryHistoryItem & { timestamp: string }) => ({
          ...item,
          timestamp: new Date(item.timestamp)
        }));
      } catch {
        return [];
      }
    }
    return [];
  });

  const addToHistory = (item: Omit<QueryHistoryItem, 'id' | 'timestamp'>) => {
    const newItem: QueryHistoryItem = {
      ...item,
      id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      timestamp: new Date()
    };
    
    const updatedHistory = [newItem, ...history].slice(0, 100);
    setHistory(updatedHistory);
    localStorage.setItem('queryHistory', JSON.stringify(updatedHistory));
  };

  const clearHistory = () => {
    setHistory([]);
    localStorage.removeItem('queryHistory');
  };

  return (
    <HistoryContext.Provider value={{ history, addToHistory, clearHistory }}>
      {children}
    </HistoryContext.Provider>
  );
};

export const useHistory = () => {
  const context = useContext(HistoryContext);
  if (!context) {
    throw new Error('useHistory must be used within a HistoryProvider');
  }
  return context;
};
