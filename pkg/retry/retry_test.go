package retry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
)

func TestRetrySuccess(t *testing.T) {
	testCases := []struct {
		name           string
		operation      func() (interface{}, error)
		config         Config
		expectedResult interface{}
		expectedError  bool
		expectedAttempts int
	}{
		{
			name: "Immediate success",
			operation: func() (interface{}, error) {
				return "success", nil
			},
			config:          DefaultConfig,
			expectedResult:  "success",
			expectedError:   false,
			expectedAttempts: 1,
		},
		{
			name: "Success after retries",
			operation: func() (interface{}, error) {
				attempts := 0
				return func() (interface{}, error) {
					attempts++
					if attempts <= 2 {
						return nil, myerrors.NewRateLimitError("test")
					}
					return "success after retry", nil
				}, nil
			},
			config:          DefaultConfig,
			expectedResult:  "success after retry",
			expectedError:   false,
			expectedAttempts: 3,
		},
		{
			name: "Non-retryable error",
			operation: func() (interface{}, error) {
				return nil, errors.New("non-retryable error")
			},
			config:          DefaultConfig,
			expectedResult:  nil,
			expectedError:   true,
			expectedAttempts: 1,
		},
		{
			name: "Max retries exceeded",
			operation: func() (interface{}, error) {
				return nil, myerrors.NewRateLimitError("test")
			},
			config: Config{
				MaxRetries:     2,
				InitialBackoff: 10 * time.Millisecond,
				MaxBackoff:     100 * time.Millisecond,
				BackoffFactor:  2.0,
				Jitter:         0.0,
			},
			expectedResult:  nil,
			expectedError:   true,
			expectedAttempts: 3, // Initial attempt + 2 retries
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attempts := 0
			var operation func() (interface{}, error)
			
			if tc.name == "Success after retries" {
				opFunc, _ := tc.operation()
				operation = opFunc.(func() (interface{}, error))
			} else {
				operation = func() (interface{}, error) {
					attempts++
					return tc.operation()
				}
			}
			
			result, err := Do(context.Background(), operation, tc.config)
			
			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if result != tc.expectedResult {
					t.Errorf("Expected result '%v', got '%v'", tc.expectedResult, result)
				}
			}
			
			if tc.name != "Success after retries" && attempts != tc.expectedAttempts {
				t.Errorf("Expected %d attempts, got %d", tc.expectedAttempts, attempts)
			}
		})
	}
}

func TestRetryWithDifferentErrorTypes(t *testing.T) {
	testCases := []struct {
		name           string
		errorFunc      func(attempt int) error
		expectedAttempts int
	}{
		{
			name: "Rate limit error",
			errorFunc: func(attempt int) error {
				if attempt < 3 {
					return myerrors.NewRateLimitError("test")
				}
				return nil
			},
			expectedAttempts: 3,
		},
		{
			name: "Timeout error",
			errorFunc: func(attempt int) error {
				if attempt < 2 {
					return myerrors.NewTimeoutError("test")
				}
				return nil
			},
			expectedAttempts: 2,
		},
		{
			name: "Unavailable error",
			errorFunc: func(attempt int) error {
				if attempt < 4 {
					return myerrors.NewUnavailableError("test")
				}
				return nil
			},
			expectedAttempts: 4,
		},
		{
			name: "Non-retryable error",
			errorFunc: func(attempt int) error {
				if attempt < 5 {
					return myerrors.NewInvalidResponseError("test", errors.New("bad json"))
				}
				return nil
			},
			expectedAttempts: 1, // Should not retry
		},
		{
			name: "Mixed error types",
			errorFunc: func(attempt int) error {
				switch attempt {
				case 1:
					return myerrors.NewRateLimitError("test")
				case 2:
					return myerrors.NewTimeoutError("test")
				case 3:
					return myerrors.NewUnavailableError("test")
				default:
					return nil
				}
			},
			expectedAttempts: 4, // Initial + 3 retries to handle all error types
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attempts := 0
			
			operation := func() (interface{}, error) {
				attempts++
				err := tc.errorFunc(attempts)
				if err != nil {
					return nil, err
				}
				return "success", nil
			}
			
			config := Config{
				MaxRetries:     10, // High enough to not interfere with test
				InitialBackoff: 1 * time.Millisecond,
				MaxBackoff:     10 * time.Millisecond,
				BackoffFactor:  1.5,
				Jitter:         0.0,
			}
			
			result, err := Do(context.Background(), operation, config)
			
			if tc.name != "Non-retryable error" {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if result != "success" {
					t.Errorf("Expected result 'success', got '%v'", result)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for non-retryable error, got nil")
				}
			}
			
			if attempts != tc.expectedAttempts {
				t.Errorf("Expected %d attempts, got %d", tc.expectedAttempts, attempts)
			}
		})
	}
}

func TestRetryWithContextTimeout(t *testing.T) {
	testCases := []struct {
		name           string
		contextTimeout time.Duration
		operationDelay time.Duration
		expectedError  bool
		expectedAttempts int
	}{
		{
			name:            "Context timeout before first retry",
			contextTimeout:  50 * time.Millisecond,
			operationDelay:  100 * time.Millisecond,
			expectedError:   true,
			expectedAttempts: 1,
		},
		{
			name:            "Context timeout during retries",
			contextTimeout:  150 * time.Millisecond,
			operationDelay:  60 * time.Millisecond,
			expectedError:   true,
			expectedAttempts: 2, // Initial + 1 retry before timeout
		},
		{
			name:            "Operation completes before context timeout",
			contextTimeout:  500 * time.Millisecond,
			operationDelay:  20 * time.Millisecond,
			expectedError:   false,
			expectedAttempts: 3, // Initial + 2 retries to succeed
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attempts := 0
			
			ctx, cancel := context.WithTimeout(context.Background(), tc.contextTimeout)
			defer cancel()
			
			operation := func() (interface{}, error) {
				attempts++
				
				select {
				case <-time.After(tc.operationDelay):
					if attempts < 3 {
						return nil, myerrors.NewRateLimitError("test")
					}
					return "success", nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			
			config := Config{
				MaxRetries:     5,
				InitialBackoff: 10 * time.Millisecond,
				MaxBackoff:     100 * time.Millisecond,
				BackoffFactor:  2.0,
				Jitter:         0.0,
			}
			
			result, err := Do(ctx, operation, config)
			
			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error due to context timeout, got nil")
				}
				if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
					t.Errorf("Expected context timeout error, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if result != "success" {
					t.Errorf("Expected result 'success', got '%v'", result)
				}
			}
			
			if attempts != tc.expectedAttempts {
				t.Errorf("Expected %d attempts, got %d", tc.expectedAttempts, attempts)
			}
		})
	}
}

func TestRetryWithCustomConfig(t *testing.T) {
	testCases := []struct {
		name           string
		config         Config
		expectedAttempts int
		expectedMinDuration time.Duration
		expectedMaxDuration time.Duration
	}{
		{
			name: "Fast retries",
			config: Config{
				MaxRetries:     3,
				InitialBackoff: 5 * time.Millisecond,
				MaxBackoff:     20 * time.Millisecond,
				BackoffFactor:  1.5,
				Jitter:         0.0,
			},
			expectedAttempts: 4, // Initial + 3 retries
			expectedMinDuration: 15 * time.Millisecond, // 5 + 7.5 + 11.25 = ~24ms
			expectedMaxDuration: 50 * time.Millisecond, // With some buffer
		},
		{
			name: "Slow retries",
			config: Config{
				MaxRetries:     2,
				InitialBackoff: 50 * time.Millisecond,
				MaxBackoff:     200 * time.Millisecond,
				BackoffFactor:  2.0,
				Jitter:         0.0,
			},
			expectedAttempts: 3, // Initial + 2 retries
			expectedMinDuration: 150 * time.Millisecond, // 50 + 100 = 150ms
			expectedMaxDuration: 250 * time.Millisecond, // With some buffer
		},
		{
			name: "High jitter",
			config: Config{
				MaxRetries:     2,
				InitialBackoff: 20 * time.Millisecond,
				MaxBackoff:     100 * time.Millisecond,
				BackoffFactor:  2.0,
				Jitter:         0.5, // 50% jitter
			},
			expectedAttempts: 3, // Initial + 2 retries
			expectedMinDuration: 30 * time.Millisecond, // Minimum with jitter
			expectedMaxDuration: 150 * time.Millisecond, // Maximum with jitter
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attempts := 0
			
			operation := func() (interface{}, error) {
				attempts++
				if attempts <= tc.expectedAttempts-1 {
					return nil, myerrors.NewRateLimitError("test")
				}
				return "success", nil
			}
			
			start := time.Now()
			result, err := Do(context.Background(), operation, tc.config)
			duration := time.Since(start)
			
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			
			if result != "success" {
				t.Errorf("Expected result 'success', got '%v'", result)
			}
			
			if attempts != tc.expectedAttempts {
				t.Errorf("Expected %d attempts, got %d", tc.expectedAttempts, attempts)
			}
			
			if duration < tc.expectedMinDuration {
				t.Errorf("Expected duration >= %v, got %v", tc.expectedMinDuration, duration)
			}
			
			if duration > tc.expectedMaxDuration {
				t.Errorf("Expected duration <= %v, got %v", tc.expectedMaxDuration, duration)
			}
		})
	}
}

func TestConcurrentRetries(t *testing.T) {
	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	results := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			
			attempts := 0
			operation := func() (interface{}, error) {
				attempts++
				if attempts <= 2 {
					return nil, myerrors.NewRateLimitError(fmt.Sprintf("test-%d", index))
				}
				return fmt.Sprintf("success-%d", index), nil
			}
			
			config := Config{
				MaxRetries:     5,
				InitialBackoff: 5 * time.Millisecond,
				MaxBackoff:     50 * time.Millisecond,
				BackoffFactor:  2.0,
				Jitter:         0.1,
			}
			
			result, err := Do(context.Background(), operation, config)
			
			if err != nil {
				errors[index] = err
			} else if resultStr, ok := result.(string); ok {
				results[index] = resultStr
			}
		}(i)
	}
	
	wg.Wait()
	
	for i := 0; i < numGoroutines; i++ {
		if errors[i] != nil {
			t.Errorf("Goroutine %d returned error: %v", i, errors[i])
		}
		
		expectedResult := fmt.Sprintf("success-%d", i)
		if results[i] != expectedResult {
			t.Errorf("Goroutine %d expected result '%s', got '%s'", i, expectedResult, results[i])
		}
	}
}

func TestCalculateBackoff(t *testing.T) {
	testCases := []struct {
		name          string
		attempt       int
		config        Config
		expectedMin   time.Duration
		expectedMax   time.Duration
		exactExpected time.Duration // Only used when jitter is 0
	}{
		{
			name:    "Initial attempt no jitter",
			attempt: 0,
			config: Config{
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     10 * time.Second,
				BackoffFactor:  2.0,
				Jitter:         0.0,
			},
			exactExpected: 1 * time.Second,
		},
		{
			name:    "Second attempt no jitter",
			attempt: 1,
			config: Config{
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     10 * time.Second,
				BackoffFactor:  2.0,
				Jitter:         0.0,
			},
			exactExpected: 2 * time.Second,
		},
		{
			name:    "Third attempt no jitter",
			attempt: 2,
			config: Config{
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     10 * time.Second,
				BackoffFactor:  2.0,
				Jitter:         0.0,
			},
			exactExpected: 4 * time.Second,
		},
		{
			name:    "Max backoff reached",
			attempt: 5,
			config: Config{
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     10 * time.Second,
				BackoffFactor:  2.0,
				Jitter:         0.0,
			},
			exactExpected: 10 * time.Second,
		},
		{
			name:    "With jitter",
			attempt: 0,
			config: Config{
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     10 * time.Second,
				BackoffFactor:  2.0,
				Jitter:         0.5, // 50% jitter
			},
			expectedMin: 500 * time.Millisecond,
			expectedMax: 1500 * time.Millisecond,
		},
		{
			name:    "Different backoff factor",
			attempt: 2,
			config: Config{
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     10 * time.Second,
				BackoffFactor:  1.5,
				Jitter:         0.0,
			},
			exactExpected: 2250 * time.Millisecond, // 1 * 1.5^2 = 2.25
		},
		{
			name:    "Very small initial backoff",
			attempt: 3,
			config: Config{
				InitialBackoff: 1 * time.Millisecond,
				MaxBackoff:     1 * time.Second,
				BackoffFactor:  2.0,
				Jitter:         0.0,
			},
			exactExpected: 8 * time.Millisecond, // 1 * 2^3 = 8
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			backoff := calculateBackoff(tc.attempt, tc.config)
			
			if tc.config.Jitter == 0 {
				if backoff != tc.exactExpected {
					t.Errorf("Expected backoff of %v, got %v", tc.exactExpected, backoff)
				}
			} else {
				if backoff < tc.expectedMin || backoff > tc.expectedMax {
					t.Errorf("Expected backoff between %v and %v, got %v", tc.expectedMin, tc.expectedMax, backoff)
				}
			}
		})
	}
}

func TestRetryWithPanic(t *testing.T) {
	attempts := 0
	
	operation := func() (interface{}, error) {
		attempts++
		if attempts <= 2 {
			panic(fmt.Sprintf("Panic on attempt %d", attempts))
		}
		return "success", nil
	}
	
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic, but none occurred")
		} else {
			if attempts != 1 {
				t.Errorf("Expected 1 attempt before panic, got %d", attempts)
			}
		}
	}()
	
	_, _ = Do(context.Background(), operation, DefaultConfig)
	
	t.Errorf("Expected panic, but function returned normally")
}
