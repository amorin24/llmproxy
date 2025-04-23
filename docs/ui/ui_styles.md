# UI Styles Documentation

## Overview

The `ui/css/styles.css` file implements a comprehensive styling system for the LLM Proxy user interface. It provides a modern, responsive design with both light and dark themes, a sidebar navigation system, and specialized components for displaying LLM responses with copy and download functionality. The stylesheet uses CSS variables for consistent theming, implements smooth transitions and animations, and ensures responsive behavior across different device sizes.

## Key Components

### Theme System

The stylesheet implements a complete theming system with light and dark mode support:

```css
:root {
    /* Light Theme Variables */
    --light-bg: #f8f9fc;
    --light-card-bg: #ffffff;
    /* ... more light theme variables ... */
    
    /* Dark Theme Variables */
    --dark-bg: #131c2e;
    --dark-card-bg: #1a263c;
    /* ... more dark theme variables ... */
    
    /* Common Variables */
    --transition-speed: 0.3s;
    --transition-func: cubic-bezier(0.4, 0, 0.2, 1);
    /* ... more common variables ... */
    
    /* Neon Accents */
    --neon-cyan: #00f0ff;
    --neon-purple: #7a18f7;
    --neon-green: #00ff9d;
}
```

The theme system uses:
- CSS variables for all colors, spacing, and sizing
- Separate variable sets for light and dark themes
- Common variables for consistent spacing and transitions
- Neon accent colors for visual highlights

Theme switching is implemented through body classes:

```css
body.light-theme {
    background-color: var(--light-bg);
    color: var(--light-text);
}

body.dark-theme {
    background-color: var(--dark-bg);
    color: var(--dark-text);
}
```

### Sidebar Navigation

The sidebar implements a collapsible navigation panel:

```css
.sidebar {
    width: var(--sidebar-width);
    height: 100vh;
    position: fixed;
    /* ... more sidebar styles ... */
}
```

Key sidebar features:
- Fixed positioning for always-visible navigation
- Smooth transitions for collapsing/expanding
- Styled navigation items with hover and active states
- Responsive behavior that transforms into a mobile menu on smaller screens
- Overlay background for mobile view

### Form Elements

The stylesheet provides consistent styling for form elements:

```css
.custom-select {
    width: 100%;
    padding: 12px 16px;
    border-radius: var(--border-radius);
    /* ... more select styles ... */
}

.custom-input {
    width: 100%;
    padding: 12px 16px;
    border-radius: var(--border-radius);
    /* ... more input styles ... */
}

.submit-button {
    display: flex;
    align-items: center;
    justify-content: center;
    /* ... more button styles ... */
}
```

Form styling includes:
- Custom select dropdowns with icons
- Textarea inputs with resizing
- Gradient buttons with hover effects
- Focus states with custom outlines
- Loading indicators for form submission

### Response Display

The response area includes specialized styling for displaying LLM outputs:

```css
.response-container {
    margin-top: var(--spacing-xl);
}

.response-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    /* ... more header styles ... */
}

.response-content {
    white-space: pre-wrap;
    line-height: 1.6;
    /* ... more content styles ... */
}
```

Response display features:
- Header with model information and action buttons
- Pre-formatted content area for preserving whitespace
- Metadata display for response timing and token usage
- Copy and download functionality

### Download Options

The stylesheet implements a dropdown menu for downloading responses in different formats:

```css
.download-dropdown {
    position: relative;
}

.download-options {
    position: absolute;
    top: 100%;
    right: 0;
    width: 180px;
    /* ... more dropdown styles ... */
}

.download-option {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    /* ... more option styles ... */
}
```

Download functionality includes:
- Dropdown menu with format options (TXT, PDF, DOCX)
- Hover states for menu items
- Positioning to avoid layout disruption

### Token Usage Display

A specialized component for displaying token usage information:

```css
.token-usage {
    padding: var(--spacing-md);
    margin: var(--spacing-md) 0;
    border-radius: var(--border-radius);
    /* ... more token usage styles ... */
}

.token-breakdown {
    display: flex;
    justify-content: space-between;
    flex-wrap: wrap;
}
```

Token usage features:
- Highlighted section with accent border
- Breakdown of token usage by category
- Responsive layout for different screen sizes

### Error Messages

Styled error messages for API failures and validation errors:

```css
.error-message {
    padding: var(--spacing-md);
    margin-bottom: var(--spacing-lg);
    border-radius: var(--border-radius);
    /* ... more error styles ... */
}
```

Error message features:
- Distinct background and border colors
- Fade-in animation for visibility
- Responsive sizing

### Animations and Effects

The stylesheet includes various animations and visual effects:

```css
@keyframes fadeIn {
    from { opacity: 0; transform: translateY(-10px); }
    to { opacity: 1; transform: translateY(0); }
}

@keyframes shake {
    10%, 90% { transform: translate3d(-1px, 0, 0); }
    /* ... more keyframes ... */
}

@keyframes shimmer {
    0% { background-position: -200% 0; }
    100% { background-position: 200% 0; }
}

@keyframes ripple {
    to {
        transform: scale(2);
        opacity: 0;
    }
}
```

Animation features:
- Fade-in for new elements
- Shake animation for validation errors
- Shimmer effect for loading states
- Ripple effect for button clicks

### Responsive Design

The stylesheet implements a comprehensive responsive design system:

```css
@media (max-width: 992px) {
    .sidebar {
        transform: translateX(-100%);
    }
    /* ... more tablet styles ... */
}

@media (max-width: 768px) {
    .container {
        width: 100%;
        padding: 0;
    }
    /* ... more mobile styles ... */
}

@media (max-width: 480px) {
    .status-grid {
        grid-template-columns: 1fr;
    }
    /* ... more small mobile styles ... */
}
```

Responsive features:
- Collapsible sidebar on tablet and mobile
- Adjusted spacing and sizing for smaller screens
- Single-column layouts on mobile
- Touch-friendly sizing for interactive elements

## Integration with JavaScript

The CSS is designed to work with JavaScript for dynamic features:

- Theme toggling via the `.light-theme` and `.dark-theme` classes
- Sidebar toggling via the `.active` class
- Download options display via the `.show` class
- Loading states via the `.loading-response` class
- Error display via the `.error-message` class with display property
- Animation triggers via the `.shake` and `.ripple-effect` classes

## Best Practices Implemented

The stylesheet demonstrates several CSS best practices:

1. **Variable-Based Theming**: Uses CSS variables for consistent theming and easy updates
2. **Mobile-First Approach**: Ensures the UI works well on all device sizes
3. **Performance Optimization**: Uses hardware-accelerated properties for animations
4. **Accessibility Considerations**: Provides sufficient contrast and focus states
5. **Modular Organization**: Organizes styles by component for maintainability
6. **Smooth Transitions**: Implements smooth transitions for state changes
7. **Visual Feedback**: Provides visual feedback for user interactions

## Usage Examples

### Theme Switching

```javascript
// Toggle between light and dark themes
document.body.classList.toggle('light-theme');
document.body.classList.toggle('dark-theme');
```

### Sidebar Toggle

```javascript
// Toggle sidebar visibility on mobile
document.querySelector('.sidebar').classList.toggle('active');
document.querySelector('.sidebar-overlay').classList.toggle('active');
```

### Download Options

```javascript
// Show download options dropdown
document.querySelector('.download-options').classList.toggle('show');
```

### Loading State

```javascript
// Show loading state for response
document.querySelector('.response-content').classList.add('loading-response');
// Remove loading state when response is received
document.querySelector('.response-content').classList.remove('loading-response');
```

### Error Display

```javascript
// Show error message
const errorElement = document.querySelector('.error-message');
errorElement.textContent = 'An error occurred while processing your request.';
errorElement.style.display = 'block';
```

## Dependencies

The stylesheet is designed to work with:

- Font Awesome or similar icon library for icons
- Inter font family (or fallback to sans-serif)
- Modern browsers that support CSS variables and flexbox/grid

## Browser Compatibility

The stylesheet uses modern CSS features that are compatible with:

- Chrome 49+
- Firefox 44+
- Safari 9.1+
- Edge 15+
- iOS Safari 9.3+
- Android Browser 4.4+

Older browsers may require polyfills or fallbacks for:
- CSS Variables
- Grid Layout
- Flexbox
- CSS Animations

## Customization

The stylesheet can be customized by:

1. Modifying the CSS variables in the `:root` selector
2. Adjusting the media query breakpoints for different device sizes
3. Changing the animation timing and easing functions
4. Updating the color schemes for light and dark themes

## Integration with UI Framework

This CSS is designed to work with the HTML structure in `ui/templates/index.html` and the JavaScript in `ui/js/app.js`. The three files together implement a complete UI system for the LLM Proxy application, providing:

1. Responsive layout with sidebar navigation
2. Form for submitting queries to LLMs
3. Display area for viewing responses
4. Copy and download functionality for responses
5. Light/dark theme switching
6. Status indicators for model availability
7. Error handling and loading states

This comprehensive styling system ensures a consistent, modern, and user-friendly interface for interacting with multiple LLM providers through the proxy system.
