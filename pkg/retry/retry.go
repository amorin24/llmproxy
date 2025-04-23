package retry

import (
    "context"
    "errors"
    "math"
    "math/rand"
    "time"

    myerrors "github.com/amorin24/llmproxy/pkg/errors"
    "github.com/sirupsen/logrus"
)

type Config struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    BackoffFactor  float64
    Jitter         float64
}

var DefaultConfig = Config{
    MaxRetries:     3,
    InitialBackoff: 1 * time.Second,
    MaxBackoff:     30 * time.Second,
    BackoffFactor:  2.0,
    Jitter:         0.1,
}

func Do(ctx context.Context, f func() (interface{}, error), cfg Config) (interface{}, error) {
    var err error
    var result interface{}
    
    for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
        result, err = f()
        
        if err == nil {
            return result, nil
        }
        
        var modelErr *myerrors.ModelError
        if !errors.As(err, &modelErr) || !modelErr.Retryable {
            return nil, err
        }
        
        if attempt == cfg.MaxRetries {
            return nil, err
        }
        
        backoff := calculateBackoff(attempt, cfg)
        
        logrus.WithFields(logrus.Fields{
            "attempt":      attempt + 1,
            "max_attempts": cfg.MaxRetries + 1,
            "backoff_ms":   backoff.Milliseconds(),
            "error":        err.Error(),
        }).Warn("Retrying request after error")
        
        timer := time.NewTimer(backoff)
        
        select {
        case <-ctx.Done():
            timer.Stop()
            return nil, ctx.Err()
        case <-timer.C:
        }
    }
    
    return nil, err
}

func calculateBackoff(attempt int, cfg Config) time.Duration {
    backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffFactor, float64(attempt))
    
    if backoff > float64(cfg.MaxBackoff) {
        backoff = float64(cfg.MaxBackoff)
    }
    
    jitterAmount := backoff * cfg.Jitter
    backoff = backoff + (rand.Float64()*jitterAmount*2 - jitterAmount)
    
    return time.Duration(math.Round(backoff))
}
