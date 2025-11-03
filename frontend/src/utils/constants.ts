export const MODEL_VERSIONS = {
  openai: [
    { value: 'gpt-3.5-turbo', label: 'GPT-3.5 Turbo' },
    { value: 'gpt-4', label: 'GPT-4' },
    { value: 'gpt-4-turbo', label: 'GPT-4 Turbo' },
    { value: 'gpt-4o', label: 'GPT-4o' },
  ],
  gemini: [
    { value: 'gemini-pro', label: 'Gemini Pro' },
    { value: 'gemini-1.5-pro', label: 'Gemini 1.5 Pro' },
    { value: 'gemini-2.0-flash', label: 'Gemini 2.0 Flash' },
    { value: 'gemini-1.5-flash', label: 'Gemini 1.5 Flash' },
  ],
  mistral: [
    { value: 'mistral-small', label: 'Mistral Small' },
    { value: 'mistral-medium', label: 'Mistral Medium' },
    { value: 'mistral-large', label: 'Mistral Large' },
  ],
  claude: [
    { value: 'claude-3-sonnet-20240229', label: 'Claude 3 Sonnet' },
    { value: 'claude-3-opus-20240229', label: 'Claude 3 Opus' },
    { value: 'claude-3-haiku-20240307', label: 'Claude 3 Haiku' },
    { value: 'claude-3-5-sonnet-20241022', label: 'Claude 3.5 Sonnet' },
  ],
};

export const TASK_TYPES = [
  { value: '', label: 'Auto' },
  { value: 'text_generation', label: 'Text Generation' },
  { value: 'summarization', label: 'Summarization' },
  { value: 'sentiment_analysis', label: 'Sentiment Analysis' },
  { value: 'question_answering', label: 'Question Answering' },
];

export const MODEL_ICONS = {
  openai: '🤖',
  gemini: '💎',
  mistral: '🌬️',
  claude: '💬',
};
