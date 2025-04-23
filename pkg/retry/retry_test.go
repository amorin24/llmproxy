package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
)

func TestRetry(t *testing.T) {
	attempts := 0
	successFunc := func() (interface{}, error) {
		attempts++
		return "success", nil
	}
	
	result, err := Do(context.Background(), successFunc, DefaultConfig)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if result != "success" {
		t.Errorf("Expected result 'success', got '%v'", result)
	}
	
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
	
	attempts = 0
	retryFunc := func() (interface{}, error) {
		attempts++
		if attempts <= 2 {
			return nil, myerrors.NewRateLimitError("test")
		}
		return "success after retry", nil
	}
	
	result, err = Do(context.Background(), retryFunc, DefaultConfig)
	if err != nil {
		t.Errorf("Expected no error after retries, got %v", err)
	}
	
	if result != "success after retry" {
		t.Errorf("Expected result 'success after retry', got '%v'", result)
	}
	
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
	
	attempts = 0
	nonRetryFunc := func() (interface{}, error) {
		attempts++
		return nil, errors.New("non-retryable error")
	}
	
	result, err = Do(context.Background(), nonRetryFunc, DefaultConfig)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
	
	attempts = 0
	maxRetriesFunc := func() (interface{}, error) {
		attempts++
		return nil, myerrors.NewRateLimitError("test")
	}
	
	testConfig := DefaultConfig
	testConfig.MaxRetries = 2
	
	result, err = Do(context.Background(), maxRetriesFunc, testConfig)
	if err == nil {
		t.Errorf("Expected error after max retries, got nil")
	}
	
	if attempts != 3 { // Initial attempt + 2 retries
		t.Errorf("Expected 3 attempts for max retries, got %d", attempts)
	}
	
	attempts = 0
	ctx, cancel := context.WithCancel(context.Background())
	
	cancelFunc := func() (interface{}, error) {
		attempts++
		cancel()
		return nil, myerrors.NewRateLimitError("test")
	}
	
	result, err = Do(ctx, cancelFunc, DefaultConfig)
	if err == nil {
		t.Errorf("Expected error after context cancellation, got nil")
	}
	
	if attempts != 1 {
		t.Errorf("Expected 1 attempt before context cancellation, got %d", attempts)
	}
}

func TestCalculateBackoff(t *testing.T) {
	cfg := Config{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         0.0, // No jitter for deterministic testing
	}
	
	backoff := calculateBackoff(0, cfg)
	if backoff != 1*time.Second {
		t.Errorf("Expected initial backoff of 1s, got %v", backoff)
	}
	
	backoff = calculateBackoff(1, cfg)
	if backoff != 2*time.Second {
		t.Errorf("Expected backoff of 2s, got %v", backoff)
	}
	
	backoff = calculateBackoff(2, cfg)
	if backoff != 4*time.Second {
		t.Errorf("Expected backoff of 4s, got %v", backoff)
	}
	
	backoff = calculateBackoff(5, cfg)
	if backoff != 10*time.Second {
		t.Errorf("Expected max backoff of 10s, got %v", backoff)
	}
	
	jitterCfg := Config{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         0.5, // 50% jitter
	}
	
	backoff = calculateBackoff(0, jitterCfg)
	if backoff < 500*time.Millisecond || backoff > 1500*time.Millisecond {
		t.Errorf("Expected backoff with jitter to be between 0.5s and 1.5s, got %v", backoff)
	}
}
