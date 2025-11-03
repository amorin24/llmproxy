package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	healthCheckStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "llmproxy_health_check_status",
		Help: "Health check status (1 = healthy, 0 = unhealthy)",
	}, []string{"check_name", "check_type"})

	healthCheckDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "llmproxy_health_check_duration_seconds",
		Help:    "Duration of health checks",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
	}, []string{"check_name"})
)

type CheckType string

const (
	CheckTypeLiveness  CheckType = "liveness"
	CheckTypeReadiness CheckType = "readiness"
	CheckTypeStartup   CheckType = "startup"
)

type HealthChecker struct {
	checks map[string]*HealthCheck
	mu     sync.RWMutex
}

type HealthCheck struct {
	Name       string
	Type       CheckType
	CheckFunc  func(ctx context.Context) error
	Timeout    time.Duration
	LastCheck  time.Time
	LastStatus bool
	LastError  error
	mu         sync.RWMutex
}

type HealthStatus struct {
	Healthy bool
	Checks  map[string]CheckStatus
}

type CheckStatus struct {
	Name      string
	Type      CheckType
	Healthy   bool
	Error     string
	LastCheck time.Time
	Duration  time.Duration
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]*HealthCheck),
	}
}

func (hc *HealthChecker) RegisterCheck(name string, checkType CheckType, checkFunc func(ctx context.Context) error, timeout time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if timeout == 0 {
		timeout = 5 * time.Second
	}

	hc.checks[name] = &HealthCheck{
		Name:      name,
		Type:      checkType,
		CheckFunc: checkFunc,
		Timeout:   timeout,
	}

	logrus.WithFields(logrus.Fields{
		"name":    name,
		"type":    checkType,
		"timeout": timeout,
	}).Info("Registered health check")
}

func (hc *HealthChecker) CheckLiveness(ctx context.Context) *HealthStatus {
	return hc.runChecks(ctx, CheckTypeLiveness)
}

func (hc *HealthChecker) CheckReadiness(ctx context.Context) *HealthStatus {
	return hc.runChecks(ctx, CheckTypeReadiness)
}

func (hc *HealthChecker) CheckStartup(ctx context.Context) *HealthStatus {
	return hc.runChecks(ctx, CheckTypeStartup)
}

func (hc *HealthChecker) CheckAll(ctx context.Context) *HealthStatus {
	hc.mu.RLock()
	checks := make([]*HealthCheck, 0, len(hc.checks))
	for _, check := range hc.checks {
		checks = append(checks, check)
	}
	hc.mu.RUnlock()

	status := &HealthStatus{
		Healthy: true,
		Checks:  make(map[string]CheckStatus),
	}

	for _, check := range checks {
		checkStatus := hc.runCheck(ctx, check)
		status.Checks[check.Name] = checkStatus
		if !checkStatus.Healthy {
			status.Healthy = false
		}
	}

	return status
}

func (hc *HealthChecker) runChecks(ctx context.Context, checkType CheckType) *HealthStatus {
	hc.mu.RLock()
	checks := make([]*HealthCheck, 0)
	for _, check := range hc.checks {
		if check.Type == checkType {
			checks = append(checks, check)
		}
	}
	hc.mu.RUnlock()

	status := &HealthStatus{
		Healthy: true,
		Checks:  make(map[string]CheckStatus),
	}

	for _, check := range checks {
		checkStatus := hc.runCheck(ctx, check)
		status.Checks[check.Name] = checkStatus
		if !checkStatus.Healthy {
			status.Healthy = false
		}
	}

	return status
}

func (hc *HealthChecker) runCheck(ctx context.Context, check *HealthCheck) CheckStatus {
	startTime := time.Now()

	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	err := check.CheckFunc(checkCtx)
	duration := time.Since(startTime)

	check.mu.Lock()
	check.LastCheck = startTime
	check.LastStatus = err == nil
	check.LastError = err
	check.mu.Unlock()

	if err == nil {
		healthCheckStatus.WithLabelValues(check.Name, string(check.Type)).Set(1)
	} else {
		healthCheckStatus.WithLabelValues(check.Name, string(check.Type)).Set(0)
	}
	healthCheckDuration.WithLabelValues(check.Name).Observe(duration.Seconds())

	status := CheckStatus{
		Name:      check.Name,
		Type:      check.Type,
		Healthy:   err == nil,
		LastCheck: startTime,
		Duration:  duration,
	}
	if err != nil {
		status.Error = err.Error()
	}

	return status
}

func (hc *HealthChecker) GetStatus() map[string]CheckStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	status := make(map[string]CheckStatus)
	for name, check := range hc.checks {
		check.mu.RLock()
		checkStatus := CheckStatus{
			Name:      check.Name,
			Type:      check.Type,
			Healthy:   check.LastStatus,
			LastCheck: check.LastCheck,
		}
		if check.LastError != nil {
			checkStatus.Error = check.LastError.Error()
		}
		status[name] = checkStatus
		check.mu.RUnlock()
	}

	return status
}

func DatabaseCheck(pingFunc func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := pingFunc(ctx); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}
		return nil
	}
}

func CacheCheck(pingFunc func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := pingFunc(ctx); err != nil {
			return fmt.Errorf("cache ping failed: %w", err)
		}
		return nil
	}
}

func ProviderCheck(provider string, checkFunc func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := checkFunc(ctx); err != nil {
			return fmt.Errorf("provider %s check failed: %w", provider, err)
		}
		return nil
	}
}

func DiskSpaceCheck(path string, minFreeBytes uint64) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return nil
	}
}

func MemoryCheck(maxMemoryBytes uint64) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return nil
	}
}
