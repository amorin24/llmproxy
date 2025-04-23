package errors

import (
	"errors"
	"testing"
)

func TestModelError(t *testing.T) {
	err := NewModelError("openai", 429, ErrRateLimit, true)
	
	if err.Model != "openai" {
		t.Errorf("Expected model to be 'openai', got '%s'", err.Model)
	}
	
	if err.Code != 429 {
		t.Errorf("Expected code to be 429, got %d", err.Code)
	}
	
	if !err.Retryable {
		t.Errorf("Expected error to be retryable")
	}
	
	expected := "model openai error: rate limit exceeded (code: 429)"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
	
	unwrapped := err.Unwrap()
	if !errors.Is(unwrapped, ErrRateLimit) {
		t.Errorf("Expected unwrapped error to be ErrRateLimit")
	}
}

func TestHelperFunctions(t *testing.T) {
	timeoutErr := NewTimeoutError("gemini")
	if timeoutErr.Model != "gemini" || timeoutErr.Code != 408 || !errors.Is(timeoutErr.Unwrap(), ErrTimeout) {
		t.Errorf("Timeout error not created correctly")
	}
	
	rateLimitErr := NewRateLimitError("claude")
	if rateLimitErr.Model != "claude" || rateLimitErr.Code != 429 || !errors.Is(rateLimitErr.Unwrap(), ErrRateLimit) {
		t.Errorf("Rate limit error not created correctly")
	}
	
	invalidRespErr := NewInvalidResponseError("mistral", errors.New("bad json"))
	if invalidRespErr.Model != "mistral" || invalidRespErr.Code != 500 || !errors.Is(invalidRespErr.Unwrap(), ErrInvalidResponse) {
		t.Errorf("Invalid response error not created correctly")
	}
	
	emptyRespErr := NewEmptyResponseError("openai")
	if emptyRespErr.Model != "openai" || emptyRespErr.Code != 500 || !errors.Is(emptyRespErr.Unwrap(), ErrEmptyResponse) {
		t.Errorf("Empty response error not created correctly")
	}
	
	unavailableErr := NewUnavailableError("all")
	if unavailableErr.Model != "all" || unavailableErr.Code != 503 || !errors.Is(unavailableErr.Unwrap(), ErrUnavailable) {
		t.Errorf("Unavailable error not created correctly")
	}
}

func TestErrorsIs(t *testing.T) {
	timeoutErr := NewTimeoutError("openai")
	if !errors.Is(timeoutErr, ErrTimeout) {
		t.Errorf("errors.Is failed for timeout error")
	}
	
	rateLimitErr := NewRateLimitError("gemini")
	if !errors.Is(rateLimitErr, ErrRateLimit) {
		t.Errorf("errors.Is failed for rate limit error")
	}
	
	if errors.Is(timeoutErr, ErrRateLimit) {
		t.Errorf("errors.Is incorrectly returned true for different error types")
	}
}

func TestErrorsAs(t *testing.T) {
	var modelErr *ModelError
	err := NewTimeoutError("openai")
	
	if !errors.As(err, &modelErr) {
		t.Errorf("errors.As failed for ModelError")
	}
	
	if modelErr.Model != "openai" || modelErr.Code != 408 || !errors.Is(modelErr.Unwrap(), ErrTimeout) {
		t.Errorf("ModelError not correctly extracted with errors.As")
	}
}
