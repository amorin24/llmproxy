package errors

import (
    "errors"
    "fmt"
)

var (
    ErrTimeout        = errors.New("request timed out")
    ErrRateLimit      = errors.New("rate limit exceeded")
    ErrInvalidResponse = errors.New("invalid response")
    ErrEmptyResponse  = errors.New("empty response from LLM")
    ErrAPIKeyMissing  = errors.New("API key not configured")
    ErrUnavailable    = errors.New("service unavailable")
)

type ModelError struct {
    Model     string
    Code      int
    Err       error
    Retryable bool
}

func (e *ModelError) Error() string {
    var errMsg string
    if e.Err == nil || e.Err.Error() == "" {
        errMsg = "unknown error"
    } else {
        errMsg = e.Err.Error()
    }
    return fmt.Sprintf("model %s error: %s (code: %d)", e.Model, errMsg, e.Code)
}

func (e *ModelError) Unwrap() error {
    return e.Err
}

func NewModelError(model string, code int, err error, retryable bool) *ModelError {
    return &ModelError{
        Model:     model,
        Code:      code,
        Err:       err,
        Retryable: retryable,
    }
}

func NewTimeoutError(model string) *ModelError {
    return NewModelError(model, 408, ErrTimeout, true)
}

func NewRateLimitError(model string) *ModelError {
    return NewModelError(model, 429, ErrRateLimit, true)
}

func NewInvalidResponseError(model string, err error) *ModelError {
    return NewModelError(model, 500, fmt.Errorf("%w: %v", ErrInvalidResponse, err), false)
}

func NewEmptyResponseError(model string) *ModelError {
    return NewModelError(model, 500, ErrEmptyResponse, false)
}

func NewUnavailableError(model string) *ModelError {
    return NewModelError(model, 503, ErrUnavailable, true)
}
