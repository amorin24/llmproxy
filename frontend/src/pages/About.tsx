import React from 'react';
import { Github, Globe, Zap, Shield, TrendingUp, Users } from 'lucide-react';

const About: React.FC = () => {
  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
        About LLM Proxy
      </h1>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <h2 className="text-2xl font-semibold mb-4 text-gray-900 dark:text-white">Overview</h2>
        <p className="text-gray-700 dark:text-gray-300 leading-relaxed mb-4">
          LLM Proxy is a unified interface for accessing multiple Large Language Models (LLMs) including OpenAI, 
          Google Gemini, Mistral AI, and Anthropic Claude. The system provides a consistent API for querying 
          different models, comparing responses side-by-side, and optimizing performance through intelligent 
          caching and routing.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-3 bg-blue-100 dark:bg-blue-900 rounded-lg">
              <Zap className="text-blue-600 dark:text-blue-400" size={24} />
            </div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Performance</h3>
          </div>
          <p className="text-gray-700 dark:text-gray-300">
            Intelligent caching and request routing ensure optimal response times and reduced API costs.
          </p>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-3 bg-green-100 dark:bg-green-900 rounded-lg">
              <Shield className="text-green-600 dark:text-green-400" size={24} />
            </div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Reliability</h3>
          </div>
          <p className="text-gray-700 dark:text-gray-300">
            Automatic fallback mechanisms and retry logic ensure high availability and fault tolerance.
          </p>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-3 bg-purple-100 dark:bg-purple-900 rounded-lg">
              <TrendingUp className="text-purple-600 dark:text-purple-400" size={24} />
            </div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Monitoring</h3>
          </div>
          <p className="text-gray-700 dark:text-gray-300">
            Comprehensive metrics and logging with Prometheus and Grafana integration for observability.
          </p>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-3 bg-orange-100 dark:bg-orange-900 rounded-lg">
              <Users className="text-orange-600 dark:text-orange-400" size={24} />
            </div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Multi-Model</h3>
          </div>
          <p className="text-gray-700 dark:text-gray-300">
            Compare responses from multiple models simultaneously to find the best answer for your needs.
          </p>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <h2 className="text-2xl font-semibold mb-4 text-gray-900 dark:text-white">Supported Models</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
            <h4 className="font-semibold text-gray-900 dark:text-white mb-2">🤖 OpenAI</h4>
            <ul className="text-sm text-gray-700 dark:text-gray-300 space-y-1">
              <li>• GPT-3.5 Turbo</li>
              <li>• GPT-4</li>
              <li>• GPT-4 Turbo</li>
              <li>• GPT-4o</li>
            </ul>
          </div>
          <div className="p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
            <h4 className="font-semibold text-gray-900 dark:text-white mb-2">💎 Google Gemini</h4>
            <ul className="text-sm text-gray-700 dark:text-gray-300 space-y-1">
              <li>• Gemini Pro</li>
              <li>• Gemini 1.5 Pro</li>
              <li>• Gemini 2.0 Flash</li>
              <li>• Gemini 1.5 Flash</li>
            </ul>
          </div>
          <div className="p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
            <h4 className="font-semibold text-gray-900 dark:text-white mb-2">🌬️ Mistral AI</h4>
            <ul className="text-sm text-gray-700 dark:text-gray-300 space-y-1">
              <li>• Mistral Small</li>
              <li>• Mistral Medium</li>
              <li>• Mistral Large</li>
            </ul>
          </div>
          <div className="p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
            <h4 className="font-semibold text-gray-900 dark:text-white mb-2">💬 Anthropic Claude</h4>
            <ul className="text-sm text-gray-700 dark:text-gray-300 space-y-1">
              <li>• Claude 3 Sonnet</li>
              <li>• Claude 3 Opus</li>
              <li>• Claude 3 Haiku</li>
              <li>• Claude 3.5 Sonnet</li>
            </ul>
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <h2 className="text-2xl font-semibold mb-4 text-gray-900 dark:text-white">Features</h2>
        <ul className="space-y-2 text-gray-700 dark:text-gray-300">
          <li className="flex items-start gap-2">
            <span className="text-blue-600 dark:text-blue-400 mt-1">✓</span>
            <span>Unified API for multiple LLM providers</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-600 dark:text-blue-400 mt-1">✓</span>
            <span>Side-by-side comparison of model responses</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-600 dark:text-blue-400 mt-1">✓</span>
            <span>Intelligent caching for improved performance</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-600 dark:text-blue-400 mt-1">✓</span>
            <span>Automatic fallback and retry mechanisms</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-600 dark:text-blue-400 mt-1">✓</span>
            <span>Task-based model routing</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-600 dark:text-blue-400 mt-1">✓</span>
            <span>Query history and export functionality</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-600 dark:text-blue-400 mt-1">✓</span>
            <span>Download responses in multiple formats (TXT, PDF, DOCX)</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-600 dark:text-blue-400 mt-1">✓</span>
            <span>Dark mode support</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-600 dark:text-blue-400 mt-1">✓</span>
            <span>Responsive design for mobile and desktop</span>
          </li>
        </ul>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-lg p-6 border border-gray-200 dark:border-gray-700">
        <h2 className="text-2xl font-semibold mb-4 text-gray-900 dark:text-white">Links</h2>
        <div className="flex flex-wrap gap-4">
          <a
            href="https://github.com/amorin24/llmproxy"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 px-4 py-2 bg-gray-900 dark:bg-gray-700 text-white rounded-lg hover:bg-gray-800 dark:hover:bg-gray-600 transition-colors"
          >
            <Github size={20} />
            GitHub Repository
          </a>
          <a
            href="/api/metrics"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Globe size={20} />
            API Metrics
          </a>
        </div>
      </div>

      <div className="bg-gradient-to-r from-blue-600 to-purple-600 rounded-xl shadow-lg p-6 text-white">
        <p className="text-center">
          <strong>Version:</strong> 1.0.0 | <strong>Built with:</strong> Go, React, TypeScript, Tailwind CSS
        </p>
      </div>
    </div>
  );
};

export default About;
