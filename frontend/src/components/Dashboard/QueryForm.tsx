import React, { useState } from 'react';
import { api, generateRequestId } from '../../utils/api';
import { SingleModelResponse, MultiModelResponse } from '../../types';
import { MODEL_VERSIONS, TASK_TYPES } from '../../utils/constants';
import { Loader2 } from 'lucide-react';
import { useHistory } from '../../contexts/HistoryContext';

interface QueryFormProps {
  onResponse: (response: SingleModelResponse | MultiModelResponse, isMultiModel: boolean) => void;
}

const QueryForm: React.FC<QueryFormProps> = ({ onResponse }) => {
  const [query, setQuery] = useState('');
  const [taskType, setTaskType] = useState('');
  const [selectedModels, setSelectedModels] = useState<{ [key: string]: boolean }>({
    openai: true,
    gemini: true,
    mistral: true,
    claude: true,
  });
  const [modelVersions, setModelVersions] = useState<{ [key: string]: string }>({
    openai: 'gpt-3.5-turbo',
    gemini: 'gemini-pro',
    mistral: 'mistral-small',
    claude: 'claude-3-sonnet-20240229',
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { addToHistory } = useHistory();

  const selectedModelsList = Object.entries(selectedModels)
    .filter(([, selected]) => selected)
    .map(([model]) => model);

  const handleModelToggle = (model: string) => {
    setSelectedModels(prev => ({ ...prev, [model]: !prev[model] }));
  };

  const handleVersionChange = (model: string, version: string) => {
    setModelVersions(prev => ({ ...prev, [model]: version }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!query.trim()) {
      setError('Please enter a query');
      return;
    }

    if (selectedModelsList.length === 0) {
      setError('Please select at least one model');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const isMultiModel = selectedModelsList.length > 1;
      const requestId = generateRequestId();

      if (isMultiModel) {
        const selectedVersions: { [key: string]: string } = {};
        selectedModelsList.forEach(model => {
          selectedVersions[model] = modelVersions[model];
        });

        const request = {
          query,
          models: selectedModelsList,
          task_type: taskType || undefined,
          request_id: requestId,
          model_versions: selectedVersions,
        };

        const response = await api.submitParallelQuery(request);
        onResponse(response, true);
        
        addToHistory({
          query,
          models: selectedModelsList,
          taskType: taskType || undefined,
          response: JSON.stringify(response.responses),
          responseTime: response.elapsed_time_ms,
          cached: false,
        });
      } else {
        const model = selectedModelsList[0];
        const request = {
          query,
          model,
          task_type: taskType || undefined,
          request_id: requestId,
          model_version: modelVersions[model],
        };

        const response = await api.submitQuery(request);
        onResponse(response, false);
        
        addToHistory({
          query,
          model,
          taskType: taskType || undefined,
          response: response.response,
          responseTime: response.response_time_ms,
          cached: response.cached,
          tokens: response.total_tokens || response.num_tokens,
        });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit query');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
      <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Query LLM</h2>
      
      {error && (
        <div className="mb-4 p-4 bg-red-100 dark:bg-red-900 border border-red-300 dark:border-red-700 rounded-lg text-red-700 dark:text-red-300">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Model Selection */}
        <div>
          <label className="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">
            Select Models
          </label>
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
              {Object.keys(selectedModels).map((model) => (
                <div key={model} className="flex items-center gap-3 p-2 rounded hover:bg-blue-100 dark:hover:bg-blue-900/30">
                  <input
                    type="checkbox"
                    id={`model-${model}`}
                    checked={selectedModels[model]}
                    onChange={() => handleModelToggle(model)}
                    className="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
                  />
                  <label htmlFor={`model-${model}`} className="flex-1 capitalize text-gray-900 dark:text-white">
                    {model}
                  </label>
                  {selectedModels[model] && (
                    <select
                      value={modelVersions[model]}
                      onChange={(e) => handleVersionChange(model, e.target.value)}
                      className="text-sm px-2 py-1 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                    >
                      {MODEL_VERSIONS[model as keyof typeof MODEL_VERSIONS].map((version) => (
                        <option key={version.value} value={version.value}>
                          {version.label}
                        </option>
                      ))}
                    </select>
                  )}
                </div>
              ))}
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className={`px-3 py-1 rounded-full font-medium ${
                selectedModelsList.length > 1
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'
              }`}>
                {selectedModelsList.length} selected
              </span>
              <span className="text-gray-600 dark:text-gray-400">
                Select multiple models to compare responses
              </span>
            </div>
          </div>
        </div>

        {/* Task Type */}
        <div>
          <label htmlFor="task-type" className="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">
            Task Type
          </label>
          <select
            id="task-type"
            value={taskType}
            onChange={(e) => setTaskType(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
          >
            {TASK_TYPES.map((type) => (
              <option key={type.value} value={type.value}>
                {type.label}
              </option>
            ))}
          </select>
        </div>

        {/* Query Input */}
        <div>
          <label htmlFor="query" className="block text-sm font-medium mb-2 text-gray-700 dark:text-gray-300">
            Query
          </label>
          <textarea
            id="query"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            rows={4}
            placeholder="Ask your question..."
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 resize-none"
          />
        </div>

        {/* Submit Button */}
        <button
          type="submit"
          disabled={loading}
          className="w-full md:w-auto px-6 py-3 bg-gradient-to-r from-blue-600 to-purple-600 text-white font-medium rounded-lg hover:from-blue-700 hover:to-purple-700 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          {loading ? (
            <>
              <Loader2 className="animate-spin" size={20} />
              <span>Processing...</span>
            </>
          ) : (
            <span>Submit</span>
          )}
        </button>
      </form>
    </div>
  );
};

export default QueryForm;
