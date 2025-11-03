package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	shutdownDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "llmproxy_shutdown_duration_seconds",
		Help:    "Duration of graceful shutdown",
		Buckets: []float64{1, 5, 10, 30, 60, 120},
	})

	inflightRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "llmproxy_inflight_requests",
		Help: "Number of in-flight requests during shutdown",
	})
)

type ShutdownHandler struct {
	hooks           []ShutdownHook
	timeout         time.Duration
	signalChan      chan os.Signal
	shutdownChan    chan struct{}
	inflightCounter *InflightCounter
	mu              sync.Mutex
	shuttingDown    bool
}

type ShutdownHook struct {
	Name     string
	Priority int // Lower priority runs first
	Func     func(ctx context.Context) error
	Timeout  time.Duration
}

type InflightCounter struct {
	count int64
	mu    sync.RWMutex
}

type ShutdownConfig struct {
	Timeout         time.Duration
	GracePeriod     time.Duration
	Signals         []os.Signal
}

func NewShutdownHandler(config ShutdownConfig) *ShutdownHandler {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if len(config.Signals) == 0 {
		config.Signals = []os.Signal{syscall.SIGTERM, syscall.SIGINT}
	}

	handler := &ShutdownHandler{
		hooks:           make([]ShutdownHook, 0),
		timeout:         config.Timeout,
		signalChan:      make(chan os.Signal, 1),
		shutdownChan:    make(chan struct{}),
		inflightCounter: &InflightCounter{},
	}

	signal.Notify(handler.signalChan, config.Signals...)

	return handler
}

func (sh *ShutdownHandler) RegisterHook(name string, priority int, fn func(ctx context.Context) error, timeout time.Duration) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	if timeout == 0 {
		timeout = 10 * time.Second
	}

	sh.hooks = append(sh.hooks, ShutdownHook{
		Name:     name,
		Priority: priority,
		Func:     fn,
		Timeout:  timeout,
	})

	logrus.WithFields(logrus.Fields{
		"name":     name,
		"priority": priority,
		"timeout":  timeout,
	}).Info("Registered shutdown hook")
}

func (sh *ShutdownHandler) Wait() {
	sig := <-sh.signalChan
	logrus.WithField("signal", sig).Info("Received shutdown signal")
	close(sh.shutdownChan)
}

func (sh *ShutdownHandler) Shutdown(ctx context.Context) error {
	sh.mu.Lock()
	if sh.shuttingDown {
		sh.mu.Unlock()
		return fmt.Errorf("shutdown already in progress")
	}
	sh.shuttingDown = true
	sh.mu.Unlock()

	startTime := time.Now()
	defer func() {
		shutdownDuration.Observe(time.Since(startTime).Seconds())
	}()

	logrus.Info("Starting graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(ctx, sh.timeout)
	defer cancel()

	if err := sh.waitForInflightRequests(shutdownCtx); err != nil {
		logrus.WithError(err).Warn("Timeout waiting for in-flight requests")
	}

	hooks := sh.getSortedHooks()

	for _, hook := range hooks {
		if err := sh.executeHook(shutdownCtx, hook); err != nil {
			logrus.WithError(err).WithField("hook", hook.Name).Error("Shutdown hook failed")
		}
	}

	logrus.WithField("duration", time.Since(startTime)).Info("Graceful shutdown completed")
	return nil
}

func (sh *ShutdownHandler) IsShuttingDown() bool {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	return sh.shuttingDown
}

func (sh *ShutdownHandler) GetShutdownChan() <-chan struct{} {
	return sh.shutdownChan
}

func (sh *ShutdownHandler) InflightInc() {
	sh.inflightCounter.Inc()
}

func (sh *ShutdownHandler) InflightDec() {
	sh.inflightCounter.Dec()
}

func (sh *ShutdownHandler) waitForInflightRequests(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		count := sh.inflightCounter.Count()
		inflightRequests.Set(float64(count))

		if count == 0 {
			logrus.Info("All in-flight requests completed")
			return nil
		}

		logrus.WithField("count", count).Debug("Waiting for in-flight requests")

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %d in-flight requests", count)
		case <-ticker.C:
			continue
		}
	}
}

func (sh *ShutdownHandler) executeHook(ctx context.Context, hook ShutdownHook) error {
	logrus.WithField("hook", hook.Name).Info("Executing shutdown hook")

	hookCtx, cancel := context.WithTimeout(ctx, hook.Timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- hook.Func(hookCtx)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("hook %s failed: %w", hook.Name, err)
		}
		logrus.WithField("hook", hook.Name).Info("Shutdown hook completed")
		return nil
	case <-hookCtx.Done():
		return fmt.Errorf("hook %s timed out after %v", hook.Name, hook.Timeout)
	}
}

func (sh *ShutdownHandler) getSortedHooks() []ShutdownHook {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	hooks := make([]ShutdownHook, len(sh.hooks))
	copy(hooks, sh.hooks)

	for i := 0; i < len(hooks); i++ {
		for j := i + 1; j < len(hooks); j++ {
			if hooks[j].Priority < hooks[i].Priority {
				hooks[i], hooks[j] = hooks[j], hooks[i]
			}
		}
	}

	return hooks
}

func (ic *InflightCounter) Inc() {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.count++
}

func (ic *InflightCounter) Dec() {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	if ic.count > 0 {
		ic.count--
	}
}

func (ic *InflightCounter) Count() int64 {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.count
}


func HTTPServerShutdownHook(name string, shutdownFunc func(ctx context.Context) error) ShutdownHook {
	return ShutdownHook{
		Name:     name,
		Priority: 10, // Shutdown HTTP server early
		Func:     shutdownFunc,
		Timeout:  30 * time.Second,
	}
}

func DatabaseShutdownHook(name string, closeFunc func() error) ShutdownHook {
	return ShutdownHook{
		Name:     name,
		Priority: 50, // Shutdown database after HTTP server
		Func: func(ctx context.Context) error {
			return closeFunc()
		},
		Timeout: 10 * time.Second,
	}
}

func CacheShutdownHook(name string, closeFunc func() error) ShutdownHook {
	return ShutdownHook{
		Name:     name,
		Priority: 40, // Shutdown cache after HTTP server
		Func: func(ctx context.Context) error {
			return closeFunc()
		},
		Timeout: 10 * time.Second,
	}
}

func WorkerShutdownHook(name string, stopFunc func(ctx context.Context) error) ShutdownHook {
	return ShutdownHook{
		Name:     name,
		Priority: 20, // Shutdown workers after HTTP server
		Func:     stopFunc,
		Timeout:  30 * time.Second,
	}
}
