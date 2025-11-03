export interface ModelStatus {
  [key: string]: boolean;
}

export interface QueryRequest {
  query: string;
  model?: string;
  models?: string[];
  task_type?: string;
  request_id: string;
  model_version?: string;
  model_versions?: { [key: string]: string };
}

export interface SingleModelResponse {
  model: string;
  response: string;
  response_time_ms: number;
  cached: boolean;
  total_tokens?: number;
  input_tokens?: number;
  output_tokens?: number;
  num_tokens?: number;
  num_retries?: number;
  original_model?: string;
  request_id: string;
}

export interface MultiModelResponse {
  responses: {
    [key: string]: {
      response: string;
      response_time: number;
      total_tokens?: number;
      input_tokens?: number;
      output_tokens?: number;
      num_tokens?: number;
      num_retries?: number;
      error?: string;
    };
  };
  elapsed_time_ms: number;
  request_id: string;
}

export type ModelType = 'openai' | 'gemini' | 'mistral' | 'claude';

export interface ModelVersion {
  [key: string]: string[];
}
