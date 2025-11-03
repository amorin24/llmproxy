import React, { useState } from 'react';
import { useHistory } from '../contexts/HistoryContext';
import { Search, Trash2, Download, Calendar, Clock } from 'lucide-react';
import { downloadAsText } from '../utils/download';

const History: React.FC = () => {
  const { history, clearHistory } = useHistory();
  const [searchTerm, setSearchTerm] = useState('');
  const [filterModel, setFilterModel] = useState('all');

  const filteredHistory = history.filter(item => {
    const matchesSearch = item.query.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         item.response.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesModel = filterModel === 'all' || 
                        item.model === filterModel ||
                        (item.models && item.models.includes(filterModel));
    return matchesSearch && matchesModel;
  });

  const handleExportHistory = () => {
    const historyText = filteredHistory.map(item => {
      return `
=== Query (${item.timestamp.toLocaleString()}) ===
Model: ${item.model || item.models?.join(', ') || 'N/A'}
Task Type: ${item.taskType || 'Auto'}
Query: ${item.query}

Response:
${item.response}

Metadata:
- Response Time: ${item.responseTime}ms
- Cached: ${item.cached ? 'Yes' : 'No'}
- Tokens: ${item.tokens || 'N/A'}

---
`;
    }).join('\n');

    const timestamp = new Date().toISOString().slice(0, 10);
    downloadAsText(historyText, `llm-proxy-history-${timestamp}.txt`);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
          Query History
        </h1>
        <div className="flex gap-2">
          <button
            onClick={handleExportHistory}
            disabled={filteredHistory.length === 0}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Download size={18} />
            Export
          </button>
          <button
            onClick={clearHistory}
            disabled={history.length === 0}
            className="flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Trash2 size={18} />
            Clear All
          </button>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <div className="flex flex-col md:flex-row gap-4 mb-6">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" size={20} />
            <input
              type="text"
              placeholder="Search queries and responses..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <select
            value={filterModel}
            onChange={(e) => setFilterModel(e.target.value)}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">All Models</option>
            <option value="openai">OpenAI</option>
            <option value="gemini">Gemini</option>
            <option value="mistral">Mistral</option>
            <option value="claude">Claude</option>
          </select>
        </div>

        {filteredHistory.length === 0 ? (
          <div className="text-center py-12 text-gray-500 dark:text-gray-400">
            {history.length === 0 ? (
              <p>No query history yet. Submit a query to see it here.</p>
            ) : (
              <p>No results found for your search.</p>
            )}
          </div>
        ) : (
          <div className="space-y-4">
            {filteredHistory.map((item) => (
              <div
                key={item.id}
                className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors"
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <span className="px-3 py-1 bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 rounded-full text-sm font-medium capitalize">
                        {item.model || item.models?.join(', ') || 'Multiple'}
                      </span>
                      {item.taskType && (
                        <span className="px-3 py-1 bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300 rounded-full text-sm">
                          {item.taskType}
                        </span>
                      )}
                      {item.cached && (
                        <span className="px-3 py-1 bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 rounded-full text-sm">
                          Cached
                        </span>
                      )}
                    </div>
                    <p className="text-gray-900 dark:text-white font-medium mb-2">{item.query}</p>
                    <div className="flex items-center gap-4 text-sm text-gray-600 dark:text-gray-400">
                      <div className="flex items-center gap-1">
                        <Calendar size={14} />
                        {item.timestamp.toLocaleDateString()}
                      </div>
                      <div className="flex items-center gap-1">
                        <Clock size={14} />
                        {item.timestamp.toLocaleTimeString()}
                      </div>
                      <div>
                        {item.responseTime}ms
                      </div>
                      {item.tokens && (
                        <div>
                          {item.tokens} tokens
                        </div>
                      )}
                    </div>
                  </div>
                </div>
                <div className="bg-gray-50 dark:bg-gray-900 rounded p-3 text-sm text-gray-700 dark:text-gray-300 max-h-32 overflow-y-auto">
                  {item.response.length > 200 ? `${item.response.substring(0, 200)}...` : item.response}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default History;
