package testutil

type MockRateLimiter struct {
	allowClientFunc func(clientID string) bool
}

func NewMockRateLimiter() *MockRateLimiter {
	return &MockRateLimiter{
		allowClientFunc: func(clientID string) bool {
			return true // Default to allowing all requests
		},
	}
}

func (m *MockRateLimiter) AllowClient(clientID string) bool {
	return m.allowClientFunc(clientID)
}

func (m *MockRateLimiter) SetAllowClientFunc(fn func(clientID string) bool) {
	m.allowClientFunc = fn
}
