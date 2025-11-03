package circuitbreaker

import (
	"errors"
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/sirupsen/logrus"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half_open"
)

type CircuitBreaker struct {
	maxFailures  int
	timeout      time.Duration
	failures     int
	lastFailTime time.Time
	state        State
	mu           sync.RWMutex
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       StateClosed,
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.state = StateHalfOpen
			logrus.Debug("Circuit breaker transitioning to half-open")
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}

	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
			logrus.WithField("failures", cb.failures).Warn("Circuit breaker opened")
		}

		return err
	}

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failures = 0
		logrus.Info("Circuit breaker closed")
	} else if cb.state == StateClosed {
		cb.failures = 0
	}

	return nil
}

func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	logrus.Info("Circuit breaker manually reset")
}

type CircuitBreakerManager struct {
	breakers map[models.ModelType]*CircuitBreaker
	mu       sync.RWMutex
}

func NewCircuitBreakerManager(maxFailures int, timeout time.Duration) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: map[models.ModelType]*CircuitBreaker{
			models.OpenAI:   NewCircuitBreaker(maxFailures, timeout),
			models.Gemini:   NewCircuitBreaker(maxFailures, timeout),
			models.Mistral:  NewCircuitBreaker(maxFailures, timeout),
			models.Claude:   NewCircuitBreaker(maxFailures, timeout),
			models.VertexAI: NewCircuitBreaker(maxFailures, timeout),
			models.Bedrock:  NewCircuitBreaker(maxFailures, timeout),
		},
	}
}

func (m *CircuitBreakerManager) GetBreaker(model models.ModelType) *CircuitBreaker {
	m.mu.RLock()
	defer m.mu.RUnlock()

	breaker, exists := m.breakers[model]
	if !exists {
		return NewCircuitBreaker(5, 60*time.Second)
	}

	return breaker
}

func (m *CircuitBreakerManager) Call(model models.ModelType, fn func() error) error {
	breaker := m.GetBreaker(model)
	return breaker.Call(fn)
}

func (m *CircuitBreakerManager) GetState(model models.ModelType) State {
	breaker := m.GetBreaker(model)
	return breaker.GetState()
}

func (m *CircuitBreakerManager) Reset(model models.ModelType) {
	breaker := m.GetBreaker(model)
	breaker.Reset()
}

func (m *CircuitBreakerManager) ResetAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, breaker := range m.breakers {
		breaker.Reset()
	}

	logrus.Info("All circuit breakers reset")
}
