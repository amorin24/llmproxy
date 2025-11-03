import React, { useState } from 'react';
import { SingleModelResponse, MultiModelResponse } from '../../types';
import { Copy, Download, Check, Grid3x3, Columns, Layers } from 'lucide-react';
import { downloadAsText, downloadAsPDF, downloadAsDOCX } from '../../utils/download';
import { MODEL_ICONS } from '../../utils/constants';

interface ResponseDisplayProps {
  response: SingleModelResponse | MultiModelResponse | null;
  isMultiModel: boolean;
}

type ViewMode = 'grid' | 'side-by-side' | 'stacked';

const ResponseDisplay: React.FC<ResponseDisplayProps> = ({ response, isMultiModel }) => {
  const [copied, setCopied] = useState(false);
  const [viewMode, setViewMode] = useState<ViewMode>('grid');
  const [showDownloadMenu, setShowDownloadMenu] = useState(false);

  if (!response) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <h3 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Response</h3>
        <p className="text-gray-500 dark:text-gray-400">No response yet. Submit a query to see results.</p>
      </div>
    );
  }

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const handleDownload = async (format: 'txt' | 'pdf' | 'docx', text: string, title: string) => {
    const timestamp = new Date().toISOString().slice(0, 10);
    const filename = `${title.toLowerCase().replace(/\s+/g, '-')}-${timestamp}`;

    try {
      if (format === 'txt') {
        downloadAsText(text, `${filename}.txt`);
      } else if (format === 'pdf') {
        await downloadAsPDF(text, title, `${filename}.pdf`);
      } else if (format === 'docx') {
        await downloadAsDOCX(text, title, `${filename}.docx`);
      }
      setShowDownloadMenu(false);
    } catch (err) {
      console.error('Download failed:', err);
    }
  };

  if (isMultiModel && 'responses' in response) {
    const multiResponse = response as MultiModelResponse;
    
    return (
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-xl font-semibold text-gray-900 dark:text-white">Multi-Model Responses</h3>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setViewMode('grid')}
              className={`p-2 rounded ${viewMode === 'grid' ? 'bg-blue-600 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'}`}
              title="Grid View"
            >
              <Grid3x3 size={18} />
            </button>
            <button
              onClick={() => setViewMode('side-by-side')}
              className={`p-2 rounded ${viewMode === 'side-by-side' ? 'bg-blue-600 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'}`}
              title="Side by Side"
            >
              <Columns size={18} />
            </button>
            <button
              onClick={() => setViewMode('stacked')}
              className={`p-2 rounded ${viewMode === 'stacked' ? 'bg-blue-600 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'}`}
              title="Stacked View"
            >
              <Layers size={18} />
            </button>
          </div>
        </div>

        <div className="mb-4 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
          <div className="flex flex-wrap gap-4 text-sm">
            <div>
              <strong className="text-gray-700 dark:text-gray-300">Total Time:</strong>{' '}
              <span className="text-gray-900 dark:text-white">{multiResponse.elapsed_time_ms}ms</span>
            </div>
            <div>
              <strong className="text-gray-700 dark:text-gray-300">Request ID:</strong>{' '}
              <span className="text-gray-900 dark:text-white">{multiResponse.request_id.substring(0, 8)}...</span>
            </div>
          </div>
        </div>

        <div className={`
          ${viewMode === 'grid' ? 'grid grid-cols-1 md:grid-cols-2 gap-4' : ''}
          ${viewMode === 'side-by-side' ? 'grid grid-cols-1 lg:grid-cols-2 gap-4' : ''}
          ${viewMode === 'stacked' ? 'space-y-4' : ''}
        `}>
          {Object.entries(multiResponse.responses).map(([modelName, modelResponse]) => (
            <div
              key={modelName}
              className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 bg-gray-50 dark:bg-gray-900"
            >
              <div className="flex items-center gap-2 mb-3 pb-3 border-b border-gray-200 dark:border-gray-700">
                <span className="text-2xl">{MODEL_ICONS[modelName as keyof typeof MODEL_ICONS] || '🤖'}</span>
                <h4 className="text-lg font-semibold capitalize text-gray-900 dark:text-white">{modelName}</h4>
              </div>

              <div className="space-y-2 mb-3">
                <div className="flex flex-wrap gap-3 text-sm">
                  <div>
                    <strong className="text-gray-700 dark:text-gray-300">Time:</strong>{' '}
                    <span className="text-gray-900 dark:text-white">{modelResponse.response_time}ms</span>
                  </div>
                  {modelResponse.total_tokens && (
                    <div>
                      <strong className="text-gray-700 dark:text-gray-300">Tokens:</strong>{' '}
                      <span className="text-gray-900 dark:text-white">{modelResponse.total_tokens}</span>
                    </div>
                  )}
                  {modelResponse.num_retries && (
                    <div>
                      <strong className="text-gray-700 dark:text-gray-300">Retries:</strong>{' '}
                      <span className="text-gray-900 dark:text-white">{modelResponse.num_retries}</span>
                    </div>
                  )}
                </div>
              </div>

              {modelResponse.error ? (
                <div className="p-3 bg-red-100 dark:bg-red-900/30 border border-red-300 dark:border-red-700 rounded text-red-700 dark:text-red-300 text-sm">
                  <strong>Error:</strong> {modelResponse.error}
                </div>
              ) : (
                <>
                  <div className="bg-white dark:bg-gray-800 rounded p-3 mb-3 max-h-64 overflow-y-auto whitespace-pre-wrap text-sm text-gray-900 dark:text-white">
                    {modelResponse.response}
                  </div>

                  <div className="flex gap-2">
                    <button
                      onClick={() => handleCopy(modelResponse.response)}
                      className="flex items-center gap-2 px-3 py-1.5 text-sm bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
                    >
                      {copied ? <Check size={16} /> : <Copy size={16} />}
                      {copied ? 'Copied!' : 'Copy'}
                    </button>
                    <div className="relative">
                      <button
                        onClick={() => setShowDownloadMenu(!showDownloadMenu)}
                        className="flex items-center gap-2 px-3 py-1.5 text-sm bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
                      >
                        <Download size={16} />
                        Download
                      </button>
                      {showDownloadMenu && (
                        <div className="absolute top-full mt-1 right-0 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-10 min-w-[150px]">
                          <button
                            onClick={() => handleDownload('txt', modelResponse.response, `${modelName} Response`)}
                            className="w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-white"
                          >
                            Text (.txt)
                          </button>
                          <button
                            onClick={() => handleDownload('pdf', modelResponse.response, `${modelName} Response`)}
                            className="w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-white"
                          >
                            PDF (.pdf)
                          </button>
                          <button
                            onClick={() => handleDownload('docx', modelResponse.response, `${modelName} Response`)}
                            className="w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-white"
                          >
                            Word (.docx)
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </>
              )}
            </div>
          ))}
        </div>
      </div>
    );
  }

  const singleResponse = response as SingleModelResponse;
  
  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-xl font-semibold text-gray-900 dark:text-white">Response</h3>
        <div className="flex gap-2">
          <button
            onClick={() => handleCopy(singleResponse.response)}
            className="flex items-center gap-2 px-3 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
          >
            {copied ? <Check size={18} /> : <Copy size={18} />}
            {copied ? 'Copied!' : 'Copy'}
          </button>
          <div className="relative">
            <button
              onClick={() => setShowDownloadMenu(!showDownloadMenu)}
              className="flex items-center gap-2 px-3 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"
            >
              <Download size={18} />
              Download
            </button>
            {showDownloadMenu && (
              <div className="absolute top-full mt-1 right-0 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-10 min-w-[150px]">
                <button
                  onClick={() => handleDownload('txt', singleResponse.response, 'LLM Response')}
                  className="w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-white"
                >
                  Text (.txt)
                </button>
                <button
                  onClick={() => handleDownload('pdf', singleResponse.response, 'LLM Response')}
                  className="w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-white"
                >
                  PDF (.pdf)
                </button>
                <button
                  onClick={() => handleDownload('docx', singleResponse.response, 'LLM Response')}
                  className="w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-white"
                >
                  Word (.docx)
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="mb-4 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
        <div className="flex flex-wrap gap-4 text-sm">
          <div>
            <strong className="text-gray-700 dark:text-gray-300">Model:</strong>{' '}
            <span className="text-gray-900 dark:text-white capitalize">{singleResponse.model}</span>
          </div>
          <div>
            <strong className="text-gray-700 dark:text-gray-300">Response Time:</strong>{' '}
            <span className="text-gray-900 dark:text-white">{singleResponse.response_time_ms}ms</span>
          </div>
          <div>
            <strong className="text-gray-700 dark:text-gray-300">Cached:</strong>{' '}
            <span className="text-gray-900 dark:text-white">{singleResponse.cached ? 'Yes' : 'No'}</span>
          </div>
          {(singleResponse.total_tokens || singleResponse.num_tokens) && (
            <div>
              <strong className="text-gray-700 dark:text-gray-300">Tokens:</strong>{' '}
              <span className="text-gray-900 dark:text-white">
                {singleResponse.total_tokens || singleResponse.num_tokens}
              </span>
            </div>
          )}
          {singleResponse.input_tokens && singleResponse.output_tokens && (
            <div>
              <strong className="text-gray-700 dark:text-gray-300">Input/Output:</strong>{' '}
              <span className="text-gray-900 dark:text-white">
                {singleResponse.input_tokens}/{singleResponse.output_tokens}
              </span>
            </div>
          )}
          {singleResponse.num_retries && (
            <div>
              <strong className="text-gray-700 dark:text-gray-300">Retries:</strong>{' '}
              <span className="text-gray-900 dark:text-white">{singleResponse.num_retries}</span>
            </div>
          )}
          {singleResponse.original_model && (
            <div>
              <strong className="text-gray-700 dark:text-gray-300">Fallback from:</strong>{' '}
              <span className="text-gray-900 dark:text-white capitalize">{singleResponse.original_model}</span>
            </div>
          )}
          <div>
            <strong className="text-gray-700 dark:text-gray-300">Request ID:</strong>{' '}
            <span className="text-gray-900 dark:text-white">{singleResponse.request_id.substring(0, 8)}...</span>
          </div>
        </div>
      </div>

      <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4 whitespace-pre-wrap text-gray-900 dark:text-white">
        {singleResponse.response}
      </div>
    </div>
  );
};

export default ResponseDisplay;
