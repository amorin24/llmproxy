package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestModelError(t *testing.T) {
	testCases := []struct {
		name           string
		model          string
		code           int
		err            error
		retryable      bool
		expectedMsg    string
		expectedUnwrap error
	}{
		{
			name:           "Rate limit error",
			model:          "openai",
			code:           429,
			err:            ErrRateLimit,
			retryable:      true,
			expectedMsg:    "model openai error: rate limit exceeded (code: 429)",
			expectedUnwrap: ErrRateLimit,
		},
		{
			name:           "Timeout error",
			model:          "gemini",
			code:           408,
			err:            ErrTimeout,
			retryable:      true,
			expectedMsg:    "model gemini error: request timed out (code: 408)",
			expectedUnwrap: ErrTimeout,
		},
		{
			name:           "Invalid response error",
			model:          "mistral",
			code:           500,
			err:            ErrInvalidResponse,
			retryable:      false,
			expectedMsg:    "model mistral error: invalid response (code: 500)",
			expectedUnwrap: ErrInvalidResponse,
		},
		{
			name:           "Custom error message",
			model:          "claude",
			code:           400,
			err:            fmt.Errorf("custom error message"),
			retryable:      false,
			expectedMsg:    "model claude error: custom error message (code: 400)",
			expectedUnwrap: fmt.Errorf("custom error message"),
		},
		{
			name:           "Nested error",
			model:          "openai",
			code:           500,
			err:            fmt.Errorf("outer error: %w", fmt.Errorf("inner error")),
			retryable:      true,
			expectedMsg:    "model openai error: outer error: inner error (code: 500)",
			expectedUnwrap: fmt.Errorf("outer error: %w", fmt.Errorf("inner error")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := NewModelError(tc.model, tc.code, tc.err, tc.retryable)
			
			if err.Model != tc.model {
				t.Errorf("Expected model to be '%s', got '%s'", tc.model, err.Model)
			}
			
			if err.Code != tc.code {
				t.Errorf("Expected code to be %d, got %d", tc.code, err.Code)
			}
			
			if err.Retryable != tc.retryable {
				t.Errorf("Expected retryable to be %v, got %v", tc.retryable, err.Retryable)
			}
			
			if err.Error() != tc.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tc.expectedMsg, err.Error())
			}
			
			unwrapped := err.Unwrap()
			if unwrapped.Error() != tc.expectedUnwrap.Error() {
				t.Errorf("Expected unwrapped error '%v', got '%v'", tc.expectedUnwrap, unwrapped)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	testCases := []struct {
		name           string
		createFunc     func() error
		expectedModel  string
		expectedCode   int
		expectedErr    error
		expectedRetry  bool
	}{
		{
			name:          "Timeout error",
			createFunc:    func() error { return NewTimeoutError("gemini") },
			expectedModel: "gemini",
			expectedCode:  408,
			expectedErr:   ErrTimeout,
			expectedRetry: true,
		},
		{
			name:          "Rate limit error",
			createFunc:    func() error { return NewRateLimitError("claude") },
			expectedModel: "claude",
			expectedCode:  429,
			expectedErr:   ErrRateLimit,
			expectedRetry: true,
		},
		{
			name:          "Invalid response error",
			createFunc:    func() error { return NewInvalidResponseError("mistral", errors.New("bad json")) },
			expectedModel: "mistral",
			expectedCode:  500,
			expectedErr:   ErrInvalidResponse,
			expectedRetry: false,
		},
		{
			name:          "Empty response error",
			createFunc:    func() error { return NewEmptyResponseError("openai") },
			expectedModel: "openai",
			expectedCode:  500,
			expectedErr:   ErrEmptyResponse,
			expectedRetry: false,
		},
		{
			name:          "Unavailable error",
			createFunc:    func() error { return NewUnavailableError("all") },
			expectedModel: "all",
			expectedCode:  503,
			expectedErr:   ErrUnavailable,
			expectedRetry: true,
		},
		{
			name:          "API key missing error",
			createFunc:    func() error { return NewModelError("gemini", 401, ErrAPIKeyMissing, false) },
			expectedModel: "gemini",
			expectedCode:  401,
			expectedErr:   ErrAPIKeyMissing,
			expectedRetry: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.createFunc()
			
			var modelErr *ModelError
			if !errors.As(err, &modelErr) {
				t.Fatalf("Expected error to be a ModelError, got %T", err)
			}
			
			if modelErr.Model != tc.expectedModel {
				t.Errorf("Expected model to be '%s', got '%s'", tc.expectedModel, modelErr.Model)
			}
			
			if modelErr.Code != tc.expectedCode {
				t.Errorf("Expected code to be %d, got %d", tc.expectedCode, modelErr.Code)
			}
			
			if !errors.Is(modelErr.Unwrap(), tc.expectedErr) {
				t.Errorf("Expected unwrapped error to be %v, got %v", tc.expectedErr, modelErr.Unwrap())
			}
			
			if modelErr.Retryable != tc.expectedRetry {
				t.Errorf("Expected retryable to be %v, got %v", tc.expectedRetry, modelErr.Retryable)
			}
		})
	}
}

func TestErrorsIs(t *testing.T) {
	testCases := []struct {
		name        string
		err         error
		target      error
		expectMatch bool
	}{
		{
			name:        "Timeout error matches ErrTimeout",
			err:         NewTimeoutError("openai"),
			target:      ErrTimeout,
			expectMatch: true,
		},
		{
			name:        "Rate limit error matches ErrRateLimit",
			err:         NewRateLimitError("gemini"),
			target:      ErrRateLimit,
			expectMatch: true,
		},
		{
			name:        "Timeout error does not match ErrRateLimit",
			err:         NewTimeoutError("openai"),
			target:      ErrRateLimit,
			expectMatch: false,
		},
		{
			name:        "Rate limit error does not match ErrTimeout",
			err:         NewRateLimitError("gemini"),
			target:      ErrTimeout,
			expectMatch: false,
		},
		{
			name:        "Invalid response error matches ErrInvalidResponse",
			err:         NewInvalidResponseError("mistral", errors.New("bad json")),
			target:      ErrInvalidResponse,
			expectMatch: true,
		},
		{
			name:        "Empty response error matches ErrEmptyResponse",
			err:         NewEmptyResponseError("claude"),
			target:      ErrEmptyResponse,
			expectMatch: true,
		},
		{
			name:        "Unavailable error matches ErrUnavailable",
			err:         NewUnavailableError("all"),
			target:      ErrUnavailable,
			expectMatch: true,
		},
		{
			name:        "API key missing error matches ErrAPIKeyMissing",
			err:         NewModelError("openai", 401, ErrAPIKeyMissing, false),
			target:      ErrAPIKeyMissing,
			expectMatch: true,
		},
		{
			name:        "Custom error does not match standard errors",
			err:         NewModelError("openai", 400, errors.New("custom error"), false),
			target:      ErrTimeout,
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if errors.Is(tc.err, tc.target) != tc.expectMatch {
				if tc.expectMatch {
					t.Errorf("Expected errors.Is to return true for %v and %v", tc.err, tc.target)
				} else {
					t.Errorf("Expected errors.Is to return false for %v and %v", tc.err, tc.target)
				}
			}
		})
	}
}

func TestErrorsAs(t *testing.T) {
	testCases := []struct {
		name        string
		err         error
		checkFields func(*ModelError) bool
	}{
		{
			name: "Timeout error",
			err:  NewTimeoutError("openai"),
			checkFields: func(modelErr *ModelError) bool {
				return modelErr.Model == "openai" && 
				       modelErr.Code == 408 && 
				       errors.Is(modelErr.Unwrap(), ErrTimeout) &&
				       modelErr.Retryable
			},
		},
		{
			name: "Rate limit error",
			err:  NewRateLimitError("gemini"),
			checkFields: func(modelErr *ModelError) bool {
				return modelErr.Model == "gemini" && 
				       modelErr.Code == 429 && 
				       errors.Is(modelErr.Unwrap(), ErrRateLimit) &&
				       modelErr.Retryable
			},
		},
		{
			name: "Invalid response error",
			err:  NewInvalidResponseError("mistral", errors.New("bad json")),
			checkFields: func(modelErr *ModelError) bool {
				return modelErr.Model == "mistral" && 
				       modelErr.Code == 500 && 
				       errors.Is(modelErr.Unwrap(), ErrInvalidResponse) &&
				       !modelErr.Retryable
			},
		},
		{
			name: "Empty response error",
			err:  NewEmptyResponseError("claude"),
			checkFields: func(modelErr *ModelError) bool {
				return modelErr.Model == "claude" && 
				       modelErr.Code == 500 && 
				       errors.Is(modelErr.Unwrap(), ErrEmptyResponse) &&
				       !modelErr.Retryable
			},
		},
		{
			name: "Unavailable error",
			err:  NewUnavailableError("all"),
			checkFields: func(modelErr *ModelError) bool {
				return modelErr.Model == "all" && 
				       modelErr.Code == 503 && 
				       errors.Is(modelErr.Unwrap(), ErrUnavailable) &&
				       modelErr.Retryable
			},
		},
		{
			name: "API key missing error",
			err:  NewModelError("openai", 401, ErrAPIKeyMissing, false),
			checkFields: func(modelErr *ModelError) bool {
				return modelErr.Model == "openai" && 
				       modelErr.Code == 401 && 
				       errors.Is(modelErr.Unwrap(), ErrAPIKeyMissing) &&
				       !modelErr.Retryable
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var modelErr *ModelError
			if !errors.As(tc.err, &modelErr) {
				t.Fatalf("errors.As failed for %v", tc.err)
			}
			
			if !tc.checkFields(modelErr) {
				t.Errorf("ModelError fields not correctly extracted with errors.As: %+v", modelErr)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := fmt.Errorf("wrapped: %w", baseErr)
	modelErr := NewModelError("openai", 500, wrappedErr, false)
	
	if !errors.Is(modelErr, baseErr) {
		t.Errorf("errors.Is should find base error in chain")
	}
	
	if !errors.Is(modelErr, wrappedErr) {
		t.Errorf("errors.Is should find wrapped error in chain")
	}
	
	expectedMsg := "model openai error: wrapped: base error (code: 500)"
	if modelErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, modelErr.Error())
	}
	
	// Test multiple levels of ModelError
	outerErr := NewModelError("gemini", 400, modelErr, true)
	
	var extractedOuter *ModelError
	if !errors.As(outerErr, &extractedOuter) {
		t.Fatalf("errors.As failed for outer ModelError")
	}
	
	if extractedOuter.Model != "gemini" || extractedOuter.Code != 400 {
		t.Errorf("Outer ModelError not correctly extracted")
	}
	
	var extractedInner *ModelError
	unwrappedOnce := extractedOuter.Unwrap()
	if !errors.As(unwrappedOnce, &extractedInner) {
		t.Fatalf("errors.As failed for inner ModelError")
	}
	
	if extractedInner.Model != "openai" || extractedInner.Code != 500 {
		t.Errorf("Inner ModelError not correctly extracted")
	}
}

func TestErrorMessageFormatting(t *testing.T) {
	testCases := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "Simple error",
			err:         NewModelError("openai", 400, errors.New("simple error"), false),
			expectedMsg: "model openai error: simple error (code: 400)",
		},
		{
			name:        "Multiline error",
			err:         NewModelError("gemini", 500, errors.New("line 1\nline 2\nline 3"), false),
			expectedMsg: "model gemini error: line 1\nline 2\nline 3 (code: 500)",
		},
		{
			name:        "Error with special characters",
			err:         NewModelError("claude", 429, errors.New("error with: <special> & \"characters\""), true),
			expectedMsg: "model claude error: error with: <special> & \"characters\" (code: 429)",
		},
		{
			name:        "Empty error message",
			err:         NewModelError("mistral", 408, errors.New(""), true),
			expectedMsg: "model mistral error: unknown error (code: 408)",
		},
		{
			name:        "Nil error",
			err:         NewModelError("openai", 500, nil, false),
			expectedMsg: "model openai error: unknown error (code: 500)",
		},
		{
			name:        "Long error message",
			err:         NewModelError("gemini", 400, errors.New(strings.Repeat("a", 100)), false),
			expectedMsg: fmt.Sprintf("model gemini error: %s (code: 400)", strings.Repeat("a", 100)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Error() != tc.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tc.expectedMsg, tc.err.Error())
			}
		})
	}
}

func TestContextCancellationErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately to simulate context cancellation
	
	// Test wrapping context.Canceled in ModelError
	modelErr := NewModelError("openai", 499, ctx.Err(), true)
	
	if !errors.Is(modelErr, context.Canceled) {
		t.Errorf("Expected ModelError to be context.Canceled, got %v", modelErr)
	}
	
	if modelErr.Code != 499 {
		t.Errorf("Expected code to be 499, got %d", modelErr.Code)
	}
	
	if !modelErr.Retryable {
		t.Errorf("Expected retryable to be true, got false")
	}
	
	expectedMsg := "model openai error: context canceled (code: 499)"
	if modelErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, modelErr.Error())
	}
	
	// Test wrapping context.DeadlineExceeded in ModelError
	ctxWithTimeout, cancelTimeout := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancelTimeout()
	
	<-ctxWithTimeout.Done()
	
	timeoutErr := NewModelError("gemini", 408, ctxWithTimeout.Err(), true)
	
	if !errors.Is(timeoutErr, context.DeadlineExceeded) {
		t.Errorf("Expected ModelError to be context.DeadlineExceeded, got %v", timeoutErr)
	}
	
	if timeoutErr.Code != 408 {
		t.Errorf("Expected code to be 408, got %d", timeoutErr.Code)
	}
	
	expectedTimeoutMsg := "model gemini error: context deadline exceeded (code: 408)"
	if timeoutErr.Error() != expectedTimeoutMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedTimeoutMsg, timeoutErr.Error())
	}
}

func TestConcurrentErrorHandling(t *testing.T) {
	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	errors := make([]*ModelError, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			
			var err *ModelError
			switch index % 5 {
			case 0:
				err = NewTimeoutError(fmt.Sprintf("model-%d", index))
			case 1:
				err = NewRateLimitError(fmt.Sprintf("model-%d", index))
			case 2:
				err = NewInvalidResponseError(fmt.Sprintf("model-%d", index), fmt.Errorf("invalid json"))
			case 3:
				err = NewEmptyResponseError(fmt.Sprintf("model-%d", index))
			case 4:
				err = NewUnavailableError(fmt.Sprintf("model-%d", index))
			}
			
			errors[index] = err
		}(i)
	}
	
	wg.Wait()
	
	for i, err := range errors {
		if err == nil {
			t.Errorf("Expected error at index %d, got nil", i)
			continue
		}
		
		expectedModel := fmt.Sprintf("model-%d", i)
		if err.Model != expectedModel {
			t.Errorf("Expected model '%s' at index %d, got '%s'", expectedModel, i, err.Model)
		}
		
		switch i % 5 {
		case 0:
			if err.Err != ErrTimeout {
				t.Errorf("Expected ErrTimeout at index %d, got %v", i, err.Err)
			}
			if err.Code != 408 {
				t.Errorf("Expected code 408 at index %d, got %d", i, err.Code)
			}
		case 1:
			if err.Err != ErrRateLimit {
				t.Errorf("Expected ErrRateLimit at index %d, got %v", i, err.Err)
			}
			if err.Code != 429 {
				t.Errorf("Expected code 429 at index %d, got %d", i, err.Code)
			}
		case 2:
			if !strings.Contains(err.Err.Error(), ErrInvalidResponse.Error()) {
				t.Errorf("Expected error containing %q at index %d, got %v", ErrInvalidResponse.Error(), i, err.Err)
			}
			if err.Code != 500 {
				t.Errorf("Expected code 500 at index %d, got %d", i, err.Code)
			}
		case 3:
			if err.Err != ErrEmptyResponse {
				t.Errorf("Expected ErrEmptyResponse at index %d, got %v", i, err.Err)
			}
			if err.Code != 500 {
				t.Errorf("Expected code 500 at index %d, got %d", i, err.Code)
			}
		case 4:
			if err.Err != ErrUnavailable {
				t.Errorf("Expected ErrUnavailable at index %d, got %v", i, err.Err)
			}
			if err.Code != 503 {
				t.Errorf("Expected code 503 at index %d, got %d", i, err.Code)
			}
		}
	}
}
