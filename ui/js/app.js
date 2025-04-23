document.addEventListener('DOMContentLoaded', function() {
    const statusEl = document.getElementById('status');
    const modelEl = document.getElementById('model');
    const taskTypeEl = document.getElementById('task-type');
    const queryEl = document.getElementById('query');
    const submitBtn = document.getElementById('submit-btn');
    const responseInfoEl = document.getElementById('response-info');
    const responseEl = document.getElementById('response');

    function fetchStatus() {
        statusEl.innerHTML = 'Checking status...';
        
        fetch('/api/status')
            .then(response => response.json())
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
            alert('Please enter a query');
            return;
        }
        
        const requestData = {
            query: query
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
        
        fetch('/api/query', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! Status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                const model = data.model.charAt(0).toUpperCase() + data.model.slice(1);
                const cached = data.cached ? 'Yes' : 'No';
                const time = data.response_time_ms;
                
                responseInfoEl.innerHTML = `
                    <strong>Model:</strong> ${model} | 
                    <strong>Response Time:</strong> ${time}ms | 
                    <strong>Cached:</strong> ${cached}
                `;
                
                responseEl.textContent = data.response;
            })
            .catch(error => {
                console.error('Error submitting query:', error);
                responseInfoEl.textContent = '';
                responseEl.textContent = `Error: ${error.message}`;
            })
            .finally(() => {
                submitBtn.disabled = false;
            });
    }

    submitBtn.addEventListener('click', submitQuery);
    
    fetchStatus();
    
    setInterval(fetchStatus, 30000);
});
