document.addEventListener('DOMContentLoaded', function() {
    const statusEl = document.getElementById('status');
    const modelEl = document.getElementById('model');
    const taskTypeEl = document.getElementById('task-type');
    const queryEl = document.getElementById('query');
    const submitBtn = document.getElementById('submit-btn');
    const responseInfoEl = document.getElementById('response-info');
    const responseEl = document.getElementById('response');
    let tempErrorEl = null;

    function fetchStatus() {
        statusEl.innerHTML = 'Checking status...';
        
        fetch('/api/status')
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! Status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                let statusHtml = '';
                
                for (const [model, available] of Object.entries(data)) {
                    const statusClass = available ? 'status-available' : 'status-unavailable';
                    const statusText = available ? 'Available' : 'Unavailable';
                    
                    statusHtml += `
                        <div class="status-item ${statusClass}">
                            <div>${model.charAt(0).toUpperCase() + model.slice(1)}</div>
                            <div>${statusText}</div>
                        </div>
                    `;
                }
                
                statusEl.innerHTML = statusHtml;
            })
            .catch(error => {
                console.error('Error fetching status:', error);
                statusEl.innerHTML = 'Error fetching status';
            });
    }

    function submitQuery() {
        const query = queryEl.value.trim();
        
        if (!query) {
            showError('Please enter a query');
            return;
        }
        
        const requestData = {
            query: query,
            request_id: generateRequestId()
        };
        
        const selectedModel = modelEl.value;
        if (selectedModel) {
            requestData.model = selectedModel;
        }
        
        const selectedTaskType = taskTypeEl.value;
        if (selectedTaskType) {
            requestData.task_type = selectedTaskType;
        }
        
        submitBtn.disabled = true;
        responseInfoEl.textContent = 'Loading...';
        responseEl.textContent = '';
        clearError();
        
        fetch('/api/query', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        })
            .then(response => {
                if (!response.ok) {
                    return response.text().then(text => {
                        try {
                            const errorData = JSON.parse(text);
                            throw new Error(errorData.error || `HTTP error! Status: ${response.status}`);
                        } catch (e) {
                            throw new Error(text || `HTTP error! Status: ${response.status}`);
                        }
                    });
                }
                return response.json();
            })
            .then(data => {
                const model = data.model.charAt(0).toUpperCase() + data.model.slice(1);
                const cached = data.cached ? 'Yes' : 'No';
                const time = data.response_time_ms;
                
                let infoHtml = `
                    <strong>Model:</strong> ${model} | 
                    <strong>Response Time:</strong> ${time}ms | 
                    <strong>Cached:</strong> ${cached}
                `;
                
                if (data.num_tokens) {
                    infoHtml += ` | <strong>Tokens:</strong> ${data.num_tokens}`;
                }
                
                if (data.num_retries) {
                    infoHtml += ` | <strong>Retries:</strong> ${data.num_retries}`;
                }
                
                if (data.original_model) {
                    const originalModel = data.original_model.charAt(0).toUpperCase() + data.original_model.slice(1);
                    infoHtml += ` | <strong>Fallback from:</strong> ${originalModel}`;
                }
                
                if (data.request_id) {
                    infoHtml += ` | <strong>Request ID:</strong> ${data.request_id.substring(0, 8)}...`;
                }
                
                responseInfoEl.innerHTML = infoHtml;
                responseEl.textContent = data.response;
            })
            .catch(error => {
                console.error('Error submitting query:', error);
                responseInfoEl.textContent = '';
                responseEl.textContent = '';
                showError(`Error: ${error.message}`);
            })
            .finally(() => {
                submitBtn.disabled = false;
            });
    }

    function showError(message) {
        if (!tempErrorEl) {
            tempErrorEl = document.createElement('div');
            tempErrorEl.id = 'error-message';
            tempErrorEl.className = 'error-message';
            tempErrorEl.style.backgroundColor = '#f8d7da';
            tempErrorEl.style.color = '#721c24';
            tempErrorEl.style.padding = '10px';
            tempErrorEl.style.marginBottom = '15px';
            tempErrorEl.style.border = '1px solid #f5c6cb';
            tempErrorEl.style.borderRadius = '4px';
            
            const queryContainer = document.querySelector('.query-container');
            if (queryContainer) {
                const firstFormGroup = queryContainer.querySelector('.form-group');
                if (firstFormGroup) {
                    queryContainer.insertBefore(tempErrorEl, firstFormGroup);
                } else {
                    queryContainer.insertBefore(tempErrorEl, submitBtn);
                }
            } else {
                const responseContainer = document.querySelector('.response-container');
                if (responseContainer) {
                    responseContainer.parentNode.insertBefore(tempErrorEl, responseContainer);
                }
            }
        }
        
        tempErrorEl.textContent = message;
        tempErrorEl.style.display = 'block';
    }

    function clearError() {
        if (tempErrorEl) {
            tempErrorEl.style.display = 'none';
            tempErrorEl.textContent = '';
        }
    }

    function generateRequestId() {
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
            const r = Math.random() * 16 | 0;
            const v = c === 'x' ? r : (r & 0x3 | 0x8);
            return v.toString(16);
        });
    }

    submitBtn.addEventListener('click', submitQuery);
    
    fetchStatus();
    
    setInterval(fetchStatus, 30000);
});
