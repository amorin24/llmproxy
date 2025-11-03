import React, { useState } from 'react';
import ModelStatus from '../components/Dashboard/ModelStatus';
import QueryForm from '../components/Dashboard/QueryForm';
import ResponseDisplay from '../components/Dashboard/ResponseDisplay';
import { SingleModelResponse, MultiModelResponse } from '../types';

const Dashboard: React.FC = () => {
  const [response, setResponse] = useState<SingleModelResponse | MultiModelResponse | null>(null);
  const [isMultiModel, setIsMultiModel] = useState(false);

  const handleResponse = (res: SingleModelResponse | MultiModelResponse, multi: boolean) => {
    setResponse(res);
    setIsMultiModel(multi);
  };

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
        LLM Proxy System
      </h1>
      
      <ModelStatus />
      <QueryForm onResponse={handleResponse} />
      <ResponseDisplay response={response} isMultiModel={isMultiModel} />
    </div>
  );
};

export default Dashboard;
