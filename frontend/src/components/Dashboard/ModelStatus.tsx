import React, { useEffect, useState } from 'react';
import { api } from '../../utils/api';
import { ModelStatus as ModelStatusType } from '../../types';
import { MODEL_ICONS } from '../../utils/constants';
import { Loader2 } from 'lucide-react';

const ModelStatus: React.FC = () => {
  const [status, setStatus] = useState<ModelStatusType | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStatus = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.fetchStatus();
      setStatus(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch status');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading && !status) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Model Status</h2>
        <div className="flex items-center justify-center py-8">
          <Loader2 className="animate-spin text-blue-600" size={32} />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Model Status</h2>
        <div className="text-red-600 dark:text-red-400">{error}</div>
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
      <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Model Status</h2>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {status && Object.entries(status).map(([model, available]) => (
          <div
            key={model}
            className={`p-4 rounded-lg text-center transition-transform hover:scale-105 cursor-pointer ${
              available
                ? 'bg-gradient-to-br from-green-500 to-teal-500 text-white'
                : 'bg-gradient-to-br from-red-500 to-pink-500 text-white'
            }`}
          >
            <div className="text-3xl mb-2">
              {MODEL_ICONS[model as keyof typeof MODEL_ICONS] || '🤖'}
            </div>
            <div className="font-medium capitalize">{model}</div>
            <div className="text-sm">{available ? 'Available' : 'Unavailable'}</div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default ModelStatus;
