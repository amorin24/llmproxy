package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
)

func TestCircuitBreaker_Call(t *testing.T) {
	t.Run("Successful calls keep circuit closed", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 1*time.Second)

		for i := 0; i < 5; i++ {
			err := cb.Call(func() error {
				return nil
			})
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if cb.GetState() != StateClosed {
				t.Errorf("Expected state closed, got %s", cb.GetState())
			}
		}
	})

	t.Run("Circuit opens after max failures", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 1*time.Second)
		testErr := errors.New("test error")

		for i := 0; i < 3; i++ {
			if err := cb.Call(func() error {
				return testErr
			}); err != testErr {
				t.Errorf("Expected test error, got %v", err)
			}
		}

		if cb.GetState() != StateOpen {
			t.Errorf("Expected state open, got %s", cb.GetState())
		}
	})

	t.Run("Circuit rejects calls when open", func(t *testing.T) {
		cb := NewCircuitBreaker(2, 1*time.Second)
		testErr := errors.New("test error")

		for i := 0; i < 2; i++ {
			if err := cb.Call(func() error {
				return testErr
			}); err != testErr {
				t.Errorf("Expected test error, got %v", err)
			}
		}

		err := cb.Call(func() error {
			return nil
		})

		if err != ErrCircuitOpen {
			t.Errorf("Expected ErrCircuitOpen, got %v", err)
		}
	})

	t.Run("Circuit transitions to half-open after timeout", func(t *testing.T) {
		cb := NewCircuitBreaker(2, 100*time.Millisecond)
		testErr := errors.New("test error")

		for i := 0; i < 2; i++ {
			if err := cb.Call(func() error {
				return testErr
			}); err != testErr {
				t.Errorf("Expected test error, got %v", err)
			}
		}

		if cb.GetState() != StateOpen {
			t.Errorf("Expected state open, got %s", cb.GetState())
		}

		time.Sleep(150 * time.Millisecond)

		if err := cb.Call(func() error {
			return nil
		}); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if cb.GetState() != StateClosed {
			t.Errorf("Expected state closed after successful half-open call, got %s", cb.GetState())
		}
	})

	t.Run("Successful call in half-open closes circuit", func(t *testing.T) {
		cb := NewCircuitBreaker(2, 100*time.Millisecond)
		testErr := errors.New("test error")

		for i := 0; i < 2; i++ {
			if err := cb.Call(func() error {
				return testErr
			}); err != testErr {
				t.Errorf("Expected test error, got %v", err)
			}
		}

		time.Sleep(150 * time.Millisecond)

		err := cb.Call(func() error {
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if cb.GetState() != StateClosed {
			t.Errorf("Expected state closed, got %s", cb.GetState())
		}
	})

	t.Run("Failed call in half-open reopens circuit", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 100*time.Millisecond)
		testErr := errors.New("test error")

		if err := cb.Call(func() error {
			return testErr
		}); err != testErr {
			t.Errorf("Expected test error, got %v", err)
		}

		time.Sleep(150 * time.Millisecond)

		if err := cb.Call(func() error {
			return testErr
		}); err != testErr {
			t.Errorf("Expected test error, got %v", err)
		}

		if cb.GetState() != StateOpen {
			t.Errorf("Expected state open, got %s", cb.GetState())
		}
	})
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(2, 1*time.Second)
	testErr := errors.New("test error")

	for i := 0; i < 2; i++ {
		if err := cb.Call(func() error {
			return testErr
		}); err != testErr {
			t.Errorf("Expected test error, got %v", err)
		}
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state open, got %s", cb.GetState())
	}

	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state closed after reset, got %s", cb.GetState())
	}

	err := cb.Call(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error after reset, got %v", err)
	}
}

func TestCircuitBreakerManager(t *testing.T) {
	t.Run("Manager creates breakers for all models", func(t *testing.T) {
		manager := NewCircuitBreakerManager(3, 1*time.Second)

		models := []models.ModelType{
			models.OpenAI,
			models.Gemini,
			models.Mistral,
			models.Claude,
			models.VertexAI,
			models.Bedrock,
		}

		for _, model := range models {
			breaker := manager.GetBreaker(model)
			if breaker == nil {
				t.Errorf("Expected breaker for model %s, got nil", model)
			}
			if breaker.GetState() != StateClosed {
				t.Errorf("Expected initial state closed for model %s, got %s", model, breaker.GetState())
			}
		}
	})

	t.Run("Manager isolates failures per model", func(t *testing.T) {
		manager := NewCircuitBreakerManager(2, 1*time.Second)
		testErr := errors.New("test error")

		for i := 0; i < 2; i++ {
			if err := manager.Call(models.OpenAI, func() error {
				return testErr
			}); err != testErr {
				t.Errorf("Expected test error, got %v", err)
			}
		}

		if manager.GetState(models.OpenAI) != StateOpen {
			t.Errorf("Expected OpenAI state open, got %s", manager.GetState(models.OpenAI))
		}

		if manager.GetState(models.Gemini) != StateClosed {
			t.Errorf("Expected Gemini state closed, got %s", manager.GetState(models.Gemini))
		}

		err := manager.Call(models.Gemini, func() error {
			return nil
		})

		if err != nil {
			t.Errorf("Expected Gemini call to succeed, got %v", err)
		}
	})

	t.Run("Manager can reset individual breakers", func(t *testing.T) {
		manager := NewCircuitBreakerManager(2, 1*time.Second)
		testErr := errors.New("test error")

		for i := 0; i < 2; i++ {
			if err := manager.Call(models.OpenAI, func() error {
				return testErr
			}); err != testErr {
				t.Errorf("Expected test error, got %v", err)
			}
		}

		if manager.GetState(models.OpenAI) != StateOpen {
			t.Errorf("Expected OpenAI state open, got %s", manager.GetState(models.OpenAI))
		}

		manager.Reset(models.OpenAI)

		if manager.GetState(models.OpenAI) != StateClosed {
			t.Errorf("Expected OpenAI state closed after reset, got %s", manager.GetState(models.OpenAI))
		}
	})

	t.Run("Manager can reset all breakers", func(t *testing.T) {
		manager := NewCircuitBreakerManager(2, 1*time.Second)
		testErr := errors.New("test error")

		for i := 0; i < 2; i++ {
			if err := manager.Call(models.OpenAI, func() error {
				return testErr
			}); err != testErr {
				t.Errorf("Expected test error, got %v", err)
			}
			if err := manager.Call(models.Gemini, func() error {
				return testErr
			}); err != testErr {
				t.Errorf("Expected test error, got %v", err)
			}
		}

		manager.ResetAll()

		models := []models.ModelType{
			models.OpenAI,
			models.Gemini,
			models.Mistral,
			models.Claude,
			models.VertexAI,
			models.Bedrock,
		}

		for _, model := range models {
			if manager.GetState(model) != StateClosed {
				t.Errorf("Expected state closed for model %s after ResetAll, got %s", model, manager.GetState(model))
			}
		}
	})

	t.Run("Manager returns default breaker for unknown model", func(t *testing.T) {
		manager := NewCircuitBreakerManager(3, 1*time.Second)

		unknownModel := models.ModelType("unknown")
		breaker := manager.GetBreaker(unknownModel)

		if breaker == nil {
			t.Errorf("Expected default breaker for unknown model, got nil")
		}

		if breaker.GetState() != StateClosed {
			t.Errorf("Expected default breaker state closed, got %s", breaker.GetState())
		}
	})
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(10, 1*time.Second)
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = cb.Call(func() error {
					if j%10 == 0 {
						return errors.New("test error")
					}
					return nil
				})
				cb.GetState()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

}
