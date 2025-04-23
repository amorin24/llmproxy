# UI JavaScript Documentation

## Overview

The `ui/js/app.js` file implements the client-side functionality for the LLM Proxy user interface. It provides a complete set of interactive features including theme switching, responsive sidebar navigation, model status display, query submission, and response handling with copy and download capabilities in multiple formats (TXT, PDF, DOCX). The file uses modern JavaScript with DOM manipulation, fetch API for backend communication, and dynamic content rendering.

## Key Components

### DOM Element References

The file begins by capturing references to all required DOM elements:

```javascript
const statusEl = document.getElementById('status');
const modelEl = document.getElementById('model');
const taskTypeEl = document.getElementById('task-type');
const queryEl = document.getElementById('query');
const submitBtn = document.getElementById('submit-btn');
// ... more element references ...
```

These references are used throughout the file to manipulate the UI without repeatedly querying the DOM, improving performance.

### Theme Management

The file implements a complete theme system with light and dark modes:

```javascript
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
```

Key theme features:
- Theme persistence using localStorage
- Smooth toggling between light and dark themes
- Visual feedback with icon changes and ripple effects
- Initialization on page load

### Responsive Sidebar

The file manages a responsive sidebar for navigation:

```javascript
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
```

Sidebar features:
- Toggle functionality for showing/hiding on mobile
- Overlay background for mobile view
- Automatic closing when clicking outside or on navigation links
- Visual feedback with ripple effects

### Model Status Display

The file fetches and displays the status of available LLM models:

```javascript
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
            
            // Add click handlers to select available models
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
```

Status display features:
- Periodic fetching of model availability (every 30 seconds)
- Visual indicators for available/unavailable models
- Custom icons for each model type
- Click-to-select functionality for available models
- Error handling for API failures

### Query Submission

The file handles query submission to the backend:

```javascript
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
    
    // Add optional parameters if selected
    const selectedModel = modelEl.value;
    if (selectedModel) {
        requestData.model = selectedModel;
    }
    
    const selectedTaskType = taskTypeEl.value;
    if (selectedTaskType) {
        requestData.task_type = selectedTaskType;
    }
    
    // Update UI to show loading state
    submitBtn.disabled = true;
    buttonText.style.opacity = '0';
    buttonLoader.style.display = 'block';
    responseInfoEl.textContent = 'Processing...';
    responseEl.textContent = '';
    clearError();
    
    const responseContainer = document.querySelector('.response-container');
    responseContainer.classList.add('loading-response');
    
    // ... API call and response handling ...
}
```

Query submission features:
- Input validation with error messages
- Dynamic request building based on user selections
- Loading state indicators
- Unique request ID generation
- Error handling and recovery

### Response Display

The file processes and displays LLM responses:

```javascript
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
```

Response display features:
- Typewriter effect for gradual text reveal
- Metadata display (model, response time, token usage)
- Token usage breakdown
- Visual styling based on current theme

### Copy to Clipboard

The file implements clipboard functionality:

```javascript
function copyToClipboard() {
    const responseText = responseEl.textContent;
    
    if (!responseText) {
        showError('No response to copy');
        return;
    }
    
    // Modern clipboard API with fallback
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
        // Fallback for older browsers
        const textarea = document.createElement('textarea');
        textarea.value = responseText;
        textarea.style.position = 'fixed';
        document.body.appendChild(textarea);
        textarea.focus();
        textarea.select();
        
        try {
            const successful = document.execCommand('copy');
            // ... success/error handling ...
        } catch (err) {
            // ... error handling ...
        }
        
        document.body.removeChild(textarea);
    }
}
```

Copy features:
- Modern Clipboard API with fallback for older browsers
- Visual feedback on successful copy
- Error handling and user notification
- Ripple effect for button interaction

### Download Functionality

The file implements download capabilities in multiple formats:

```javascript
function downloadAsTxt() {
    const responseText = responseEl.textContent;
    
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
    // ... PDF generation using jsPDF library ...
}

function downloadAsDocx() {
    // ... DOCX generation using docx.js library ...
}
```

Download features:
- Multiple format support (TXT, PDF, DOCX)
- Dynamic filename generation with date
- On-demand library loading for PDF and DOCX
- Error handling and user notification
- Visual feedback with ripple effects

### Error Handling

The file implements comprehensive error handling:

```javascript
function showError(message) {
    if (!tempErrorEl) {
        tempErrorEl = document.createElement('div');
        tempErrorEl.id = 'error-message';
        tempErrorEl.className = 'error-message';
        
        // ... insert error element into DOM ...
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
```

Error handling features:
- Dynamic error message creation
- Visual shake animation for attention
- Contextual placement based on error type
- Error clearing on new actions

### Visual Effects

The file implements various visual effects:

```javascript
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
```

Visual effect features:
- Ripple effect for button clicks
- Typewriter effect for text display
- Shake animation for errors
- Loading animations for pending operations

### Event Listeners

The file sets up event listeners for user interactions:

```javascript
themeToggleBtn.addEventListener('click', toggleTheme);
submitBtn.addEventListener('click', submitQuery);
sidebarToggleBtn.addEventListener('click', toggleSidebar);
sidebarOverlay.addEventListener('click', closeSidebar);
copyBtn.addEventListener('click', copyToClipboard);

// ... more event listeners ...

document.querySelectorAll('.nav-link').forEach(link => {
    link.addEventListener('click', function(e) {
        if (window.innerWidth <= 992) {
            closeSidebar();
        }
    });
});
```

Event listener features:
- Button click handlers
- Responsive behavior based on screen size
- Dropdown menu toggling
- Format selection for downloads

### Initialization

The file initializes the application on page load:

```javascript
initTheme();
fetchStatus();

setInterval(fetchStatus, 30000);
```

Initialization features:
- Theme setup based on saved preference
- Initial status fetch
- Periodic status updates every 30 seconds

## Testing Mode

The file includes a testing mode for development:

```javascript
// TESTING MODE - Always use test mode for now
setTimeout(() => {
    // ... simulate response data ...
    
    // ... update UI with simulated data ...
    
    typeWriterEffect(responseEl, data.response);
    
    responseContainer.classList.remove('loading-response');
}, 1500);

return; // Skip actual API call
```

Testing mode features:
- Simulated API responses
- Configurable response timing
- Sample data for UI testing
- Bypassing actual API calls

## Integration with Backend

The file communicates with the backend API:

```javascript
fetch('/api/query', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json'
    },
    body: JSON.stringify(requestData)
})
    .then(response => {
        // ... response handling ...
    })
    .then(data => {
        // ... data processing ...
    })
    .catch(error => {
        // ... error handling ...
    })
    .finally(() => {
        // ... cleanup ...
    });
```

Backend integration features:
- RESTful API calls using fetch
- JSON request/response handling
- Error handling with status codes
- Response processing and display

## Dependencies

The file dynamically loads external libraries as needed:

- **jsPDF**: For PDF generation (`https://cdnjs.cloudflare.com/ajax/libs/jspdf/2.5.1/jspdf.umd.min.js`)
- **docx.js**: For DOCX generation (`https://unpkg.com/docx@7.8.2/build/index.js`)

The file also expects:
- Font Awesome for icons
- CSS styles defined in `ui/css/styles.css`
- HTML structure defined in `ui/templates/index.html`

## Browser Compatibility

The file implements fallbacks for older browsers:

- Modern Clipboard API with execCommand fallback
- Feature detection for API availability
- Graceful degradation for unsupported features

## Best Practices Implemented

The file demonstrates several JavaScript best practices:

1. **Event Delegation**: Uses event delegation for dynamically created elements
2. **Error Handling**: Comprehensive error handling for API calls and user interactions
3. **Feature Detection**: Checks for API availability before using modern features
4. **Performance Optimization**: Minimizes DOM queries by storing element references
5. **Visual Feedback**: Provides immediate visual feedback for user actions
6. **Responsive Design**: Adapts behavior based on screen size
7. **Code Organization**: Modular functions with clear responsibilities
8. **State Management**: Manages UI state consistently across interactions

## Usage Examples

### Theme Switching

```javascript
// Initialize theme from saved preference
initTheme();

// Toggle between light and dark themes
themeToggleBtn.addEventListener('click', toggleTheme);
```

### Query Submission

```javascript
// Submit a query to the selected model
submitBtn.addEventListener('click', submitQuery);
```

### Response Handling

```javascript
// Display response with typewriter effect
typeWriterEffect(responseEl, data.response);

// Copy response to clipboard
copyBtn.addEventListener('click', copyToClipboard);

// Download response in different formats
downloadAsTxt();
downloadAsPdf();
downloadAsDocx();
```

## Integration with UI Components

This JavaScript file works with the HTML structure in `ui/templates/index.html` and the CSS in `ui/css/styles.css` to create a complete UI system for the LLM Proxy application. The three files together implement:

1. Responsive layout with sidebar navigation
2. Form for submitting queries to LLMs
3. Display area for viewing responses
4. Copy and download functionality for responses in multiple formats (TXT, PDF, DOCX)
5. Light/dark theme switching
6. Status indicators for model availability
7. Error handling and loading states

This comprehensive JavaScript implementation ensures a dynamic, interactive, and user-friendly interface for interacting with multiple LLM providers through the proxy system.
