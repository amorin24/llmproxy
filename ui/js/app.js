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
    const copyBtn = document.getElementById('copy-btn');
    const downloadBtn = document.getElementById('download-btn');
    const downloadOptions = document.querySelectorAll('.download-option');
    const themeToggleBtn = document.getElementById('theme-toggle-btn');
    const themeIcon = themeToggleBtn.querySelector('i');
    const sidebarToggleBtn = document.getElementById('sidebar-toggle-btn');
    const sidebar = document.querySelector('.sidebar');
    const sidebarOverlay = document.querySelector('.sidebar-overlay');
    const multiModelCheckbox = document.getElementById('multi-model-checkbox');
    const modelSelection = document.getElementById('model-selection');
    let tempErrorEl = null;
    
    if (multiModelCheckbox && modelSelection) {
        multiModelCheckbox.addEventListener('change', function() {
            if (this.checked) {
                modelSelection.style.display = 'block';
                modelEl.disabled = true;
            } else {
                modelSelection.style.display = 'none';
                modelEl.disabled = false;
            }
        });
    }
    
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
    
    function toggleSidebar() {
        sidebar.classList.toggle('active');
        sidebarOverlay.classList.toggle('active');
        document.body.classList.toggle('sidebar-open');
        
        addRippleEffect(sidebarToggleBtn);
    }
    
    function closeSidebar() {
        sidebar.classList.remove('active');
        sidebarOverlay.classList.remove('active');
        document.body.classList.remove('sidebar-open');
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
        
        const multiModelCheckbox = document.getElementById('multi-model-checkbox');
        const isMultiModel = multiModelCheckbox && multiModelCheckbox.checked;
        
        const requestData = {
            query: query,
            request_id: generateRequestId()
        };
        
        const selectedModel = modelEl.value;
        if (selectedModel && !isMultiModel) {
            requestData.model = selectedModel;
        }
        
        const selectedTaskType = taskTypeEl.value;
        if (selectedTaskType) {
            requestData.task_type = selectedTaskType;
        }
        
        if (isMultiModel) {
            const modelCheckboxes = document.querySelectorAll('.model-checkbox:checked');
            const selectedModels = Array.from(modelCheckboxes).map(cb => cb.value);
            
            if (selectedModels.length === 0) {
                if (selectedModel) {
                    requestData.models = [selectedModel];
                } else {
                    showError('Please select at least one model');
                    return;
                }
            } else {
                requestData.models = selectedModels;
            }
        }
        
        submitBtn.disabled = true;
        buttonText.style.opacity = '0';
        buttonLoader.style.display = 'block';
        responseInfoEl.textContent = 'Processing...';
        responseEl.textContent = '';
        clearError();
        
        const responseContainer = document.querySelector('.response-container');
        responseContainer.classList.add('loading-response');
        
        const endpoint = isMultiModel ? '/api/query-parallel' : '/api/query';
        
        fetch(endpoint, {
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
                if (isMultiModel) {
                    displayMultiModelResponse(data);
                } else {
                    displaySingleModelResponse(data);
                }
                
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
    
    function displaySingleModelResponse(data) {
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
    }
    
    function displayMultiModelResponse(data) {
        const responses = data.responses;
        const elapsedTime = data.elapsed_time;
        
        let responsesHtml = `
            <div class="multi-model-responses">
                <div class="multi-model-header">
                    <h3>Responses from Multiple Models</h3>
                    <div class="response-meta-item"><strong>Total Time:</strong> ${data.elapsed_time_ms}ms</div>
                    <div class="response-meta-item"><strong>Request ID:</strong> ${data.request_id.substring(0, 8)}...</div>
                </div>
                <div class="model-tabs">
        `;
        
        let tabsHtml = '';
        let contentHtml = '';
        let firstModel = true;
        
        for (const [modelName, response] of Object.entries(responses)) {
            const model = modelName.charAt(0).toUpperCase() + modelName.slice(1);
            const activeClass = firstModel ? 'active' : '';
            const modelIcon = getModelIcon(modelName);
            
            tabsHtml += `
                <div class="model-tab ${activeClass}" data-model="${modelName}">
                    <div class="model-icon">${modelIcon}</div>
                    <div>${model}</div>
                </div>
            `;
            
            contentHtml += `
                <div class="model-response ${activeClass}" id="response-${modelName}">
                    <div class="model-response-meta">
                        <div class="response-meta-item"><strong>Response Time:</strong> ${response.response_time}ms</div>
            `;
            
            if (response.total_tokens) {
                contentHtml += `<div class="response-meta-item"><strong>Tokens:</strong> ${response.total_tokens}</div>`;
                
                if (response.input_tokens && response.output_tokens) {
                    contentHtml += `<div class="response-meta-item"><strong>Input/Output:</strong> ${response.input_tokens}/${response.output_tokens}</div>`;
                }
            } else if (response.num_tokens) {
                contentHtml += `<div class="response-meta-item"><strong>Tokens:</strong> ${response.num_tokens}</div>`;
            }
            
            if (response.num_retries) {
                contentHtml += `<div class="response-meta-item"><strong>Retries:</strong> ${response.num_retries}</div>`;
            }
            
            if (response.error) {
                contentHtml += `
                        <div class="response-meta-item error"><strong>Error:</strong> ${response.error}</div>
                    </div>
                    <div class="model-response-content error">${response.response}</div>
                </div>
                `;
            } else {
                contentHtml += `
                        </div>
                        <div class="model-response-content">${response.response}</div>
                    </div>
                `;
            }
            
            firstModel = false;
        }
        
        responsesHtml += tabsHtml;
        responsesHtml += `</div><div class="model-responses-content">`;
        responsesHtml += contentHtml;
        responsesHtml += `</div></div>`;
        
        responseEl.innerHTML = responsesHtml;
        
        document.querySelectorAll('.model-tab').forEach(tab => {
            tab.addEventListener('click', () => {
                const model = tab.dataset.model;
                
                document.querySelectorAll('.model-tab').forEach(t => t.classList.remove('active'));
                document.querySelectorAll('.model-response').forEach(c => c.classList.remove('active'));
                
                tab.classList.add('active');
                document.getElementById(`response-${model}`).classList.add('active');
            });
        });
        
        responseInfoEl.innerHTML = '';
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
    
    function copyToClipboard() {
        let responseText = '';
        
        const multiModelResponses = document.querySelector('.multi-model-responses');
        if (multiModelResponses) {
            const activeTab = document.querySelector('.model-tab.active');
            if (activeTab) {
                const modelName = activeTab.dataset.model;
                const modelResponseContent = document.querySelector(`#response-${modelName} .model-response-content`);
                if (modelResponseContent) {
                    responseText = modelResponseContent.textContent;
                }
            } else {
                const allResponses = {};
                document.querySelectorAll('.model-response').forEach(response => {
                    const modelName = response.id.replace('response-', '');
                    const content = response.querySelector('.model-response-content');
                    if (content) {
                        allResponses[modelName] = content.textContent;
                    }
                });
                
                for (const [model, text] of Object.entries(allResponses)) {
                    responseText += `=== ${model.toUpperCase()} RESPONSE ===\n${text}\n\n`;
                }
            }
        } else {
            responseText = responseEl.textContent;
        }
        
        if (!responseText) {
            showError('No response to copy');
            return;
        }
        
        if (navigator.clipboard) {
            navigator.clipboard.writeText(responseText)
                .then(() => {
                    const originalText = copyBtn.innerHTML;
                    copyBtn.innerHTML = '<i class="fas fa-check"></i> Copied!';
                    addRippleEffect(copyBtn);
                    
                    setTimeout(() => {
                        copyBtn.innerHTML = originalText;
                    }, 2000);
                })
                .catch(err => {
                    console.error('Failed to copy text: ', err);
                    showError('Failed to copy to clipboard');
                });
        } else {
            const textarea = document.createElement('textarea');
            textarea.value = responseText;
            textarea.style.position = 'fixed';  // Avoid scrolling to bottom
            document.body.appendChild(textarea);
            textarea.focus();
            textarea.select();
            
            try {
                const successful = document.execCommand('copy');
                if (successful) {
                    const originalText = copyBtn.innerHTML;
                    copyBtn.innerHTML = '<i class="fas fa-check"></i> Copied!';
                    addRippleEffect(copyBtn);
                    
                    setTimeout(() => {
                        copyBtn.innerHTML = originalText;
                    }, 2000);
                } else {
                    showError('Failed to copy to clipboard');
                }
            } catch (err) {
                console.error('Failed to copy text: ', err);
                showError('Failed to copy to clipboard');
            }
            
            document.body.removeChild(textarea);
        }
    }
    
    function downloadAsTxt() {
        let responseText = '';
        
        const multiModelResponses = document.querySelector('.multi-model-responses');
        if (multiModelResponses) {
            const activeTab = document.querySelector('.model-tab.active');
            if (activeTab) {
                const modelName = activeTab.dataset.model;
                const modelResponseContent = document.querySelector(`#response-${modelName} .model-response-content`);
                if (modelResponseContent) {
                    responseText = `=== ${modelName.toUpperCase()} RESPONSE ===\n${modelResponseContent.textContent}`;
                }
            } else {
                const allResponses = {};
                document.querySelectorAll('.model-response').forEach(response => {
                    const modelName = response.id.replace('response-', '');
                    const content = response.querySelector('.model-response-content');
                    if (content) {
                        allResponses[modelName] = content.textContent;
                    }
                });
                
                responseText = "=== LLM PROXY MULTI-MODEL RESPONSES ===\n\n";
                for (const [model, text] of Object.entries(allResponses)) {
                    responseText += `=== ${model.toUpperCase()} RESPONSE ===\n${text}\n\n`;
                }
            }
        } else {
            responseText = responseEl.textContent;
        }
        
        if (!responseText) {
            showError('No response to download');
            return;
        }
        
        const blob = new Blob([responseText], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `llm-response-${new Date().toISOString().slice(0, 10)}.txt`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        addRippleEffect(downloadBtn);
    }
    
    function downloadAsPdf() {
        let responseText = '';
        let title = 'LLM Response';
        
        const multiModelResponses = document.querySelector('.multi-model-responses');
        if (multiModelResponses) {
            const activeTab = document.querySelector('.model-tab.active');
            if (activeTab) {
                const modelName = activeTab.dataset.model;
                const modelResponseContent = document.querySelector(`#response-${modelName} .model-response-content`);
                if (modelResponseContent) {
                    title = `${modelName.toUpperCase()} Response`;
                    responseText = modelResponseContent.textContent;
                }
            } else {
                const allResponses = {};
                document.querySelectorAll('.model-response').forEach(response => {
                    const modelName = response.id.replace('response-', '');
                    const content = response.querySelector('.model-response-content');
                    if (content) {
                        allResponses[modelName] = content.textContent;
                    }
                });
                
                title = "LLM Proxy Multi-Model Responses";
                responseText = "";
                for (const [model, text] of Object.entries(allResponses)) {
                    responseText += `=== ${model.toUpperCase()} RESPONSE ===\n${text}\n\n`;
                }
            }
        } else {
            responseText = responseEl.textContent;
        }
        
        if (!responseText) {
            showError('No response to download');
            return;
        }
        
        const script = document.createElement('script');
        script.src = 'https://cdnjs.cloudflare.com/ajax/libs/jspdf/2.5.1/jspdf.umd.min.js';
        script.onload = function() {
            const { jsPDF } = window.jspdf;
            const doc = new jsPDF();
            
            doc.setFontSize(16);
            doc.text(title, 20, 20);
            
            doc.setFontSize(10);
            doc.text(`Generated on: ${new Date().toLocaleString()}`, 20, 30);
            
            doc.setFontSize(12);
            const splitText = doc.splitTextToSize(responseText, 170);
            doc.text(splitText, 20, 40);
            
            doc.save(`llm-response-${new Date().toISOString().slice(0, 10)}.pdf`);
        };
        
        script.onerror = function() {
            showError('Failed to load PDF generation library');
        };
        
        document.head.appendChild(script);
        addRippleEffect(downloadBtn);
    }
    
    function downloadAsDocx() {
        let responseText = '';
        let title = 'LLM Response';
        
        const multiModelResponses = document.querySelector('.multi-model-responses');
        if (multiModelResponses) {
            const activeTab = document.querySelector('.model-tab.active');
            if (activeTab) {
                const modelName = activeTab.dataset.model;
                const modelResponseContent = document.querySelector(`#response-${modelName} .model-response-content`);
                if (modelResponseContent) {
                    title = `${modelName.toUpperCase()} Response`;
                    responseText = modelResponseContent.textContent;
                }
            } else {
                const allResponses = {};
                document.querySelectorAll('.model-response').forEach(response => {
                    const modelName = response.id.replace('response-', '');
                    const content = response.querySelector('.model-response-content');
                    if (content) {
                        allResponses[modelName] = content.textContent;
                    }
                });
                
                title = "LLM Proxy Multi-Model Responses";
                responseText = "";
                for (const [model, text] of Object.entries(allResponses)) {
                    responseText += `=== ${model.toUpperCase()} RESPONSE ===\n${text}\n\n`;
                }
            }
        } else {
            responseText = responseEl.textContent;
        }
        
        if (!responseText) {
            showError('No response to download');
            return;
        }
        
        const script = document.createElement('script');
        script.src = 'https://unpkg.com/docx@7.8.2/build/index.js';
        script.onload = function() {
            const { Document, Packer, Paragraph, TextRun } = window.docx;
            
            const doc = new Document({
                sections: [{
                    properties: {},
                    children: [
                        new Paragraph({
                            children: [
                                new TextRun({
                                    text: title,
                                    bold: true,
                                    size: 28
                                })
                            ]
                        }),
                        new Paragraph({
                            children: [
                                new TextRun({
                                    text: `Generated on: ${new Date().toLocaleString()}`,
                                    size: 20,
                                    italics: true
                                })
                            ]
                        }),
                        new Paragraph({
                            children: [
                                new TextRun({
                                    text: responseText,
                                    size: 24
                                })
                            ]
                        })
                    ]
                }]
            });
            
            Packer.toBlob(doc).then(blob => {
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `llm-response-${new Date().toISOString().slice(0, 10)}.docx`;
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(url);
            });
        };
        
        script.onerror = function() {
            showError('Failed to load DOCX generation library');
        };
        
        document.head.appendChild(script);
        addRippleEffect(downloadBtn);
    }
    
    themeToggleBtn.addEventListener('click', toggleTheme);
    submitBtn.addEventListener('click', submitQuery);
    sidebarToggleBtn.addEventListener('click', toggleSidebar);
    sidebarOverlay.addEventListener('click', closeSidebar);
    copyBtn.addEventListener('click', copyToClipboard);
    
    downloadBtn.addEventListener('click', function(e) {
        e.preventDefault();
        const downloadOptions = document.querySelector('.download-options');
        downloadOptions.classList.toggle('show');
        
        document.addEventListener('click', function closeDropdown(event) {
            if (!downloadBtn.contains(event.target) && !downloadOptions.contains(event.target)) {
                downloadOptions.classList.remove('show');
                document.removeEventListener('click', closeDropdown);
            }
        });
        
        addRippleEffect(downloadBtn);
    });
    
    document.querySelectorAll('.download-option').forEach(option => {
        option.addEventListener('click', function() {
            const format = this.dataset.format;
            
            switch(format) {
                case 'txt':
                    downloadAsTxt();
                    break;
                case 'pdf':
                    downloadAsPdf();
                    break;
                case 'docx':
                    downloadAsDocx();
                    break;
            }
            
            document.querySelector('.download-options').classList.remove('show');
        });
    });
    
    document.querySelectorAll('.nav-link').forEach(link => {
        link.addEventListener('click', function(e) {
            if (window.innerWidth <= 992) {
                closeSidebar();
            }
        });
    });
    
    initTheme();
    fetchStatus();
    
    setInterval(fetchStatus, 30000);
});
