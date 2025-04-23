# UI Templates Documentation

## Overview

The `ui/templates/index.html` file implements the HTML structure for the LLM Proxy user interface. It provides a responsive layout with a collapsible sidebar, model selection and query input forms, and a response display area with copy and download functionality in multiple formats (TXT, PDF, DOCX). The template uses modern HTML5 elements and integrates with Font Awesome for icons and the Inter font family for typography.

## Structure

### Head Section

```html
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LLM Proxy UI</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/static/css/styles.css">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.2/css/all.min.css">
</head>
```

Key features:
- Responsive viewport configuration
- Font preconnection for performance
- Inter font family integration
- Custom CSS stylesheet
- Font Awesome icons

### Sidebar Navigation

```html
<div class="sidebar">
    <div class="sidebar-header">
        <h2 class="sidebar-title">LLM Proxy</h2>
    </div>
    
    <nav class="sidebar-nav">
        <ul>
            <li class="nav-item active">
                <a href="#" class="nav-link">
                    <i class="fas fa-home"></i>
                    <span>Dashboard</span>
                </a>
            </li>
            <!-- More navigation items -->
        </ul>
    </nav>
    
    <div class="sidebar-footer">
        <div class="theme-toggle">
            <button id="theme-toggle-btn" aria-label="Toggle dark/light mode">
                <i class="fas fa-moon"></i>
            </button>
            <span>Toggle Theme</span>
        </div>
    </div>
</div>
```

Sidebar features:
- Collapsible navigation menu
- Icon-based navigation items
- Theme toggle button in footer
- Mobile-friendly design with overlay

### Model Status Display

```html
<div class="status-container card">
    <h2>Model Status</h2>
    <div id="status" class="status-grid">Loading status...</div>
</div>
```

Status display features:
- Grid layout for model status cards
- Dynamic status updates via JavaScript
- Visual indicators for model availability

### Query Form

```html
<div class="query-container card">
    <h2>Query LLM</h2>
    
    <div id="error-message" class="error-message" style="display: none;"></div>
    
    <div class="form-group">
        <label for="model">Model:</label>
        <div class="select-wrapper">
            <select id="model" class="custom-select">
                <option value="">Auto (based on task)</option>
                <option value="openai">OpenAI</option>
                <option value="gemini">Gemini</option>
                <option value="mistral">Mistral</option>
                <option value="claude">Claude</option>
            </select>
            <i class="fas fa-chevron-down select-icon"></i>
        </div>
    </div>
    
    <!-- Task type selection -->
    <div class="form-group">
        <label for="task-type">Task Type:</label>
        <div class="select-wrapper">
            <select id="task-type" class="custom-select">
                <option value="">Auto</option>
                <option value="text_generation">Text Generation</option>
                <option value="summarization">Summarization</option>
                <option value="sentiment_analysis">Sentiment Analysis</option>
                <option value="question_answering">Question Answering</option>
            </select>
            <i class="fas fa-chevron-down select-icon"></i>
        </div>
    </div>
    
    <!-- Query input -->
    <div class="form-group">
        <label for="query">Query:</label>
        <textarea id="query" rows="4" class="custom-input" placeholder="Ask your question..."></textarea>
    </div>
    
    <button id="submit-btn" class="submit-button">
        <span class="button-text">Submit</span>
        <div class="button-loader" style="display: none;">
            <div class="spinner"></div>
        </div>
    </button>
</div>
```

Query form features:
- Model selection dropdown
- Task type selection
- Query input textarea
- Submit button with loading state
- Error message display
- Custom styled form elements

### Response Display

```html
<div class="response-container card">
    <div class="response-header">
        <h3>Response</h3>
        <div class="response-actions">
            <button id="copy-btn" class="action-button" title="Copy to clipboard">
                <i class="fas fa-copy"></i> Copy
            </button>
            <div class="download-dropdown">
                <button id="download-btn" class="action-button" title="Download">
                    <i class="fas fa-download"></i> Download
                </button>
                <div class="download-options">
                    <button class="download-option" data-format="txt">
                        <i class="fas fa-file-alt"></i> Text (.txt)
                    </button>
                    <button class="download-option" data-format="pdf">
                        <i class="fas fa-file-pdf"></i> PDF (.pdf)
                    </button>
                    <button class="download-option" data-format="docx">
                        <i class="fas fa-file-word"></i> Word (.docx)
                    </button>
                </div>
            </div>
        </div>
    </div>
    <div id="response-info" class="response-info"></div>
    <div id="response" class="response-content"></div>
    <div id="token-usage" class="token-usage"></div>
    <div id="response-metadata" class="response-metadata"></div>
</div>
```

Response display features:
- Copy to clipboard functionality
- Download options (TXT, PDF, DOCX)
- Response metadata display
- Token usage information
- Response content area

### Mobile Responsiveness

```html
<!-- Mobile sidebar toggle button -->
<div class="sidebar-toggle">
    <button id="sidebar-toggle-btn" aria-label="Toggle sidebar">
        <i class="fas fa-bars"></i>
    </button>
</div>

<!-- Overlay for mobile sidebar -->
<div class="sidebar-overlay"></div>
```

Mobile features:
- Hamburger menu for sidebar toggle
- Overlay background for sidebar
- Responsive layout adjustments
- Touch-friendly interface elements

## Integration with Other Components

### CSS Integration

The template integrates with:
- Custom styles from `/static/css/styles.css`
- Font Awesome icons for visual elements
- Inter font family for typography

### JavaScript Integration

The template integrates with:
- Main application logic in `/static/js/app.js`
- Event handlers for user interactions
- Dynamic content updates
- Theme switching functionality

## Dependencies

External dependencies:
- Font Awesome 6.4.2 for icons
- Inter font family from Google Fonts
- Custom CSS stylesheet
- Custom JavaScript file

## Best Practices Implemented

The template demonstrates several HTML best practices:

1. **Semantic HTML**: Uses appropriate HTML5 elements
2. **Accessibility**: Includes ARIA labels and semantic structure
3. **Performance**: Implements font preconnection
4. **Responsive Design**: Mobile-first approach with responsive elements
5. **Progressive Enhancement**: Basic structure works without JavaScript
6. **Modularity**: Clear separation of components
7. **Error Handling**: Dedicated error message container

## Usage Examples

### Theme Switching

```html
<div class="theme-toggle">
    <button id="theme-toggle-btn" aria-label="Toggle dark/light mode">
        <i class="fas fa-moon"></i>
    </button>
    <span>Toggle Theme</span>
</div>
```

### Model Selection

```html
<select id="model" class="custom-select">
    <option value="">Auto (based on task)</option>
    <option value="openai">OpenAI</option>
    <option value="gemini">Gemini</option>
    <option value="mistral">Mistral</option>
    <option value="claude">Claude</option>
</select>
```

### Download Options

```html
<div class="download-options">
    <button class="download-option" data-format="txt">
        <i class="fas fa-file-alt"></i> Text (.txt)
    </button>
    <button class="download-option" data-format="pdf">
        <i class="fas fa-file-pdf"></i> PDF (.pdf)
    </button>
    <button class="download-option" data-format="docx">
        <i class="fas fa-file-word"></i> Word (.docx)
    </button>
</div>
```

## Integration with UI System

This HTML template works with the CSS in `ui/css/styles.css` and the JavaScript in `ui/js/app.js` to create a complete UI system for the LLM Proxy application. The three files together implement:

1. Responsive layout with sidebar navigation
2. Form for submitting queries to LLMs
3. Display area for viewing responses
4. Copy and download functionality for responses in multiple formats (TXT, PDF, DOCX)
5. Light/dark theme switching
6. Status indicators for model availability
7. Error handling and loading states

This comprehensive HTML structure provides the foundation for a modern, user-friendly interface for interacting with multiple LLM providers through the proxy system.
