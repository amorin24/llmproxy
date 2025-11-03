import React, { useState } from 'react';
import { Save, RefreshCw } from 'lucide-react';

const Settings: React.FC = () => {
  const [apiUrl, setApiUrl] = useState(import.meta.env.VITE_API_URL || 'http://localhost:8080');
  const [autoRefreshStatus, setAutoRefreshStatus] = useState(true);
  const [refreshInterval, setRefreshInterval] = useState(30);
  const [maxHistoryItems, setMaxHistoryItems] = useState(100);
  const [saved, setSaved] = useState(false);

  const handleSave = () => {
    localStorage.setItem('settings', JSON.stringify({
      apiUrl,
      autoRefreshStatus,
      refreshInterval,
      maxHistoryItems,
    }));
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  const handleReset = () => {
    setApiUrl('http://localhost:8080');
    setAutoRefreshStatus(true);
    setRefreshInterval(30);
    setMaxHistoryItems(100);
    localStorage.removeItem('settings');
  };

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
        Settings
      </h1>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <h2 className="text-xl font-semibold mb-6 text-gray-900 dark:text-white">API Configuration</h2>
        
        <div className="space-y-6">
          <div>
            <label htmlFor="api-url" className="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">
              API Base URL
            </label>
            <input
              id="api-url"
              type="text"
              value={apiUrl}
              onChange={(e) => setApiUrl(e.target.value)}
              placeholder="http://localhost:8080"
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              The base URL for the LLM Proxy API server
            </p>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <label htmlFor="auto-refresh" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                Auto-refresh Model Status
              </label>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Automatically refresh model availability status
              </p>
            </div>
            <input
              id="auto-refresh"
              type="checkbox"
              checked={autoRefreshStatus}
              onChange={(e) => setAutoRefreshStatus(e.target.checked)}
              className="w-5 h-5 text-blue-600 rounded focus:ring-blue-500"
            />
          </div>

          {autoRefreshStatus && (
            <div>
              <label htmlFor="refresh-interval" className="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">
                Refresh Interval (seconds)
              </label>
              <input
                id="refresh-interval"
                type="number"
                min="5"
                max="300"
                value={refreshInterval}
                onChange={(e) => setRefreshInterval(parseInt(e.target.value))}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
              />
            </div>
          )}

          <div>
            <label htmlFor="max-history" className="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">
              Maximum History Items
            </label>
            <input
              id="max-history"
              type="number"
              min="10"
              max="1000"
              value={maxHistoryItems}
              onChange={(e) => setMaxHistoryItems(parseInt(e.target.value))}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Maximum number of queries to keep in history
            </p>
          </div>
        </div>

        <div className="flex gap-3 mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
          <button
            onClick={handleSave}
            className="flex items-center gap-2 px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Save size={18} />
            {saved ? 'Saved!' : 'Save Settings'}
          </button>
          <button
            onClick={handleReset}
            className="flex items-center gap-2 px-6 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
          >
            <RefreshCw size={18} />
            Reset to Defaults
          </button>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Display Preferences</h2>
        
        <div className="space-y-4">
          <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
            <p className="text-sm text-gray-700 dark:text-gray-300">
              <strong>Theme:</strong> Use the theme toggle in the sidebar to switch between light and dark modes.
            </p>
          </div>
          
          <div className="p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
            <p className="text-sm text-gray-700 dark:text-gray-300">
              <strong>Response View:</strong> When viewing multi-model responses, you can switch between Grid, Side-by-Side, and Stacked views using the view mode buttons.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Settings;
