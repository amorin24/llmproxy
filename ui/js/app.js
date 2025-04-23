document.addEventListener('DOMContentLoaded', function() {
    const statusEl = document.getElementById('status');
    const modelEl = document.getElementById('model');
    const taskTypeEl = document.getElementById('task-type');
    const queryEl = document.getElementById('query');
    const submitBtn = document.getElementById('submit-btn');
    const buttonText = submitBtn.querySelector('.button-text');
    const buttonLoader = submitBtn.querySelector('.button-loader');
    const responseInfoEl = document.getElementById('response-info');
    const responseEl = document.getElementById('response');
    const themeToggleBtn = document.getElementById('theme-toggle-btn');
    const themeIcon = themeToggleBtn.querySelector('i');
    let tempErrorEl = null;
    
    function initTheme() {
        const savedTheme = localStorage.getItem('theme') || 'light';
        document.body.classList.add(`${savedTheme}-theme`);
        updateThemeIcon(savedTheme);
    }
    
    function toggleTheme() {
        const isDarkTheme = document.body.classList.contains('dark-theme');
        const newTheme = isDarkTheme ? 'light' : 'dark';
        
        document.body.classList.remove(isDarkTheme ? 'dark-theme' : 'light-theme');
        document.body.classList.add(`${newTheme}-theme`);
        
        localStorage.setItem('theme', newTheme);
        updateThemeIcon(newTheme);
        
        addRippleEffect(themeToggleBtn);
    }
    
    function updateThemeIcon(theme) {
        themeIcon.className = theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon';
    }
    
    function addRippleEffect(element) {
        const ripple = document.createElement('span');
        ripple.classList.add('ripple-effect');
        
        const rect = element.getBoundingClientRect();
        const size = Math.max(rect.width, rect.height);
        
        ripple.style.width = ripple.style.height = `${size * 2}px`;
        ripple.style.left = `${-size / 2}px`;
        ripple.style.top = `${-size / 2}px`;
        
        element.appendChild(ripple);
        
        setTimeout(() => {
            ripple.remove();
        }, 600);
    }
    
    function fetchStatus() {
        statusEl.innerHTML = '<div class="loading-status">Checking status...</div>';
        
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
                    const modelIcon = getModelIcon(model);
                    
                    statusHtml += `
                        <div class="status-item ${statusClass}" data-model="${model}">
                            <div class="model-icon">${modelIcon}</div>
                            <div>${model.charAt(0).toUpperCase() + model.slice(1)}</div>
                            <div>${statusText}</div>
                        </div>
                    `;
                }
                
                statusEl.innerHTML = statusHtml;
                
                document.querySelectorAll('.status-item').forEach(item => {
                    item.addEventListener('click', () => {
                        const model = item.dataset.model;
                        if (item.classList.contains('status-available')) {
                            modelEl.value = model;
                        }
                    });
                });
            })
            .catch(error => {
                console.error('Error fetching status:', error);
                statusEl.innerHTML = '<div class="error">Error fetching status</div>';
            });
    }
    
    function getModelIcon(model) {
        switch(model) {
            case 'openai':
                return '<i class="fas fa-robot"></i>';
            case 'gemini':
                return '<i class="fas fa-gem"></i>';
            case 'mistral':
                return '<i class="fas fa-wind"></i>';
            case 'claude':
                return '<i class="fas fa-comment-dots"></i>';
            default:
                return '<i class="fas fa-robot"></i>';
        }
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
        buttonText.style.opacity = '0';
        buttonLoader.style.display = 'block';
        responseInfoEl.textContent = 'Processing...';
        responseEl.textContent = '';
        clearError();
        
        const responseContainer = document.querySelector('.response-container');
        responseContainer.classList.add('loading-response');
        
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
                    <div class="response-meta-item"><strong>Model:</strong> ${model}</div>
                    <div class="response-meta-item"><strong>Response Time:</strong> ${time}ms</div>
                    <div class="response-meta-item"><strong>Cached:</strong> ${cached}</div>
                `;
                
                if (data.total_tokens) {
                    infoHtml += `<div class="response-meta-item"><strong>Tokens:</strong> ${data.total_tokens}</div>`;
                    
                    if (data.input_tokens && data.output_tokens) {
                        infoHtml += `<div class="response-meta-item"><strong>Input/Output:</strong> ${data.input_tokens}/${data.output_tokens}</div>`;
                    }
                } else if (data.num_tokens) {
                    infoHtml += `<div class="response-meta-item"><strong>Tokens:</strong> ${data.num_tokens}</div>`;
                }
                
                if (data.num_retries) {
                    infoHtml += `<div class="response-meta-item"><strong>Retries:</strong> ${data.num_retries}</div>`;
                }
                
                if (data.original_model) {
                    const originalModel = data.original_model.charAt(0).toUpperCase() + data.original_model.slice(1);
                    infoHtml += `<div class="response-meta-item"><strong>Fallback from:</strong> ${originalModel}</div>`;
                }
                
                if (data.request_id) {
                    infoHtml += `<div class="response-meta-item"><strong>Request ID:</strong> ${data.request_id.substring(0, 8)}...</div>`;
                }
                
                const tokenUsageEl = document.getElementById('token-usage');
                if (data.total_tokens || data.input_tokens || data.output_tokens) {
                    let tokenHtml = `<h4>Token Usage</h4>`;
                    tokenHtml += `<div class="token-breakdown">`;
                    
                    if (data.total_tokens) {
                        tokenHtml += `<div><strong>Total:</strong> ${data.total_tokens}</div>`;
                    } else if (data.num_tokens) {
                        tokenHtml += `<div><strong>Total:</strong> ${data.num_tokens}</div>`;
                    }
                    
                    if (data.input_tokens) {
                        tokenHtml += `<div><strong>Input:</strong> ${data.input_tokens}</div>`;
                    }
                    
                    if (data.output_tokens) {
                        tokenHtml += `<div><strong>Output:</strong> ${data.output_tokens}</div>`;
                    }
                    
                    tokenHtml += `</div>`;
                    tokenUsageEl.innerHTML = tokenHtml;
                    tokenUsageEl.style.display = 'block';
                } else {
                    tokenUsageEl.style.display = 'none';
                }
                
                responseInfoEl.innerHTML = infoHtml;
                
                typeWriterEffect(responseEl, data.response);
                
                responseContainer.classList.remove('loading-response');
            })
            .catch(error => {
                console.error('Error submitting query:', error);
                responseInfoEl.textContent = '';
                responseEl.textContent = '';
                showError(`Error: ${error.message}`);
                
                responseContainer.classList.remove('loading-response');
            })
            .finally(() => {
                submitBtn.disabled = false;
                buttonText.style.opacity = '1';
                buttonLoader.style.display = 'none';
            });
    }
    
    function typeWriterEffect(element, text, speed = 10) {
        let i = 0;
        element.textContent = '';
        
        function type() {
            if (i < text.length) {
                element.textContent += text.charAt(i);
                i++;
                setTimeout(type, speed);
            }
        }
        
        type();
    }
    
    function showError(message) {
        if (!tempErrorEl) {
            tempErrorEl = document.createElement('div');
            tempErrorEl.id = 'error-message';
            tempErrorEl.className = 'error-message';
            
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
        
        tempErrorEl.classList.add('shake');
        setTimeout(() => {
            tempErrorEl.classList.remove('shake');
        }, 500);
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
    
    themeToggleBtn.addEventListener('click', toggleTheme);
    submitBtn.addEventListener('click', submitQuery);
    
    initTheme();
    fetchStatus();
    
    setInterval(fetchStatus, 30000);
});
