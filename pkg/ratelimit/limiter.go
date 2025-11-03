package ratelimit

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
	rateLimitExceeded = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llmproxy_rate_limit_exceeded_total",
		Help: "Total number of rate limit exceeded events",
	}, []string{"tenant", "limit_type"})

	quotaUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "llmproxy_quota_usage",
		Help: "Current quota usage",
	}, []string{"tenant", "quota_type", "period"})

	quotaLimit = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "llmproxy_quota_limit",
		Help: "Configured quota limits",
	}, []string{"tenant", "quota_type", "period"})
)

type LimitStrategy string

const (
	StrategyTokenBucket   LimitStrategy = "token_bucket"
	StrategySlidingWindow LimitStrategy = "sliding_window"
	StrategyFixedWindow   LimitStrategy = "fixed_window"
)

type RateLimiter struct {
	strategy LimitStrategy
	quotas   map[string]*Quota
	buckets  map[string]*TokenBucket
	windows  map[string]*SlidingWindow
	mu       sync.RWMutex
}

type Quota struct {
	Tenant          string
	RequestsPerHour int
	RequestsPerDay  int
	CostPerHour     float64
	CostPerDay      float64
	CostPerMonth    float64
	BurstAllowance  int
	Current         *QuotaUsage
	mu              sync.RWMutex
}

type QuotaUsage struct {
	RequestsThisHour  int
	RequestsThisDay   int
	CostThisHour      float64
	CostThisDay       float64
	CostThisMonth     float64
	LastResetHour     time.Time
	LastResetDay      time.Time
	LastResetMonth    time.Time
}

type TokenBucket struct {
	capacity       int
	tokens         float64
	refillRate     float64 // tokens per second
	lastRefillTime time.Time
	mu             sync.Mutex
}

type SlidingWindow struct {
	limit      int
	windowSize time.Duration
	requests   []time.Time
	mu         sync.Mutex
}

type RateLimiterConfig struct {
	Strategy           LimitStrategy
	DefaultQuota       *Quota
	BurstAllowance     int
	TokenRefillRate    float64
	SlidingWindowSize  time.Duration
}

func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.Strategy == "" {
		config.Strategy = StrategyTokenBucket
	}

	return &RateLimiter{
		strategy: config.Strategy,
		quotas:   make(map[string]*Quota),
		buckets:  make(map[string]*TokenBucket),
		windows:  make(map[string]*SlidingWindow),
	}
}

func (rl *RateLimiter) Allow(ctx context.Context, tenant string, estimatedCost float64) (bool, error) {
	rl.mu.RLock()
	quota, exists := rl.quotas[tenant]
	rl.mu.RUnlock()

	if !exists {
		return true, nil
	}

	if !rl.checkQuota(quota, estimatedCost) {
		rateLimitExceeded.WithLabelValues(tenant, "quota").Inc()
		return false, fmt.Errorf("quota exceeded for tenant %s", tenant)
	}

	allowed := false
	switch rl.strategy {
	case StrategyTokenBucket:
		allowed = rl.checkTokenBucket(tenant, quota)
	case StrategySlidingWindow:
		allowed = rl.checkSlidingWindow(tenant, quota)
	case StrategyFixedWindow:
		allowed = rl.checkFixedWindow(tenant, quota)
	default:
		allowed = true
	}

	if !allowed {
		rateLimitExceeded.WithLabelValues(tenant, "rate").Inc()
		return false, fmt.Errorf("rate limit exceeded for tenant %s", tenant)
	}

	rl.updateUsage(quota, estimatedCost)

	return true, nil
}

func (rl *RateLimiter) SetQuota(tenant string, quota *Quota) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	quota.Tenant = tenant
	if quota.Current == nil {
		quota.Current = &QuotaUsage{
			LastResetHour:  time.Now(),
			LastResetDay:   time.Now(),
			LastResetMonth: time.Now(),
		}
	}

	rl.quotas[tenant] = quota

	switch rl.strategy {
	case StrategyTokenBucket:
		rl.buckets[tenant] = NewTokenBucket(quota.RequestsPerHour, float64(quota.RequestsPerHour)/3600.0)
	case StrategySlidingWindow:
		rl.windows[tenant] = NewSlidingWindow(quota.RequestsPerHour, 1*time.Hour)
	}

	quotaLimit.WithLabelValues(tenant, "requests", "hour").Set(float64(quota.RequestsPerHour))
	quotaLimit.WithLabelValues(tenant, "requests", "day").Set(float64(quota.RequestsPerDay))
	quotaLimit.WithLabelValues(tenant, "cost", "hour").Set(quota.CostPerHour)
	quotaLimit.WithLabelValues(tenant, "cost", "day").Set(quota.CostPerDay)
	quotaLimit.WithLabelValues(tenant, "cost", "month").Set(quota.CostPerMonth)

	logrus.WithFields(logrus.Fields{
		"tenant":            tenant,
		"requests_per_hour": quota.RequestsPerHour,
		"cost_per_day":      quota.CostPerDay,
	}).Info("Set quota for tenant")
}

func (rl *RateLimiter) GetUsage(tenant string) (*QuotaUsage, error) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	quota, exists := rl.quotas[tenant]
	if !exists {
		return nil, fmt.Errorf("no quota found for tenant %s", tenant)
	}

	quota.mu.RLock()
	defer quota.mu.RUnlock()

	usage := *quota.Current
	return &usage, nil
}

func (rl *RateLimiter) checkQuota(quota *Quota, estimatedCost float64) bool {
	quota.mu.Lock()
	defer quota.mu.Unlock()

	now := time.Now()

	if now.Sub(quota.Current.LastResetHour) >= 1*time.Hour {
		quota.Current.RequestsThisHour = 0
		quota.Current.CostThisHour = 0
		quota.Current.LastResetHour = now
	}
	if now.Sub(quota.Current.LastResetDay) >= 24*time.Hour {
		quota.Current.RequestsThisDay = 0
		quota.Current.CostThisDay = 0
		quota.Current.LastResetDay = now
	}
	if now.Sub(quota.Current.LastResetMonth) >= 30*24*time.Hour {
		quota.Current.CostThisMonth = 0
		quota.Current.LastResetMonth = now
	}

	if quota.RequestsPerHour > 0 && quota.Current.RequestsThisHour >= quota.RequestsPerHour {
		return false
	}
	if quota.RequestsPerDay > 0 && quota.Current.RequestsThisDay >= quota.RequestsPerDay {
		return false
	}
	if quota.CostPerHour > 0 && quota.Current.CostThisHour+estimatedCost > quota.CostPerHour {
		return false
	}
	if quota.CostPerDay > 0 && quota.Current.CostThisDay+estimatedCost > quota.CostPerDay {
		return false
	}
	if quota.CostPerMonth > 0 && quota.Current.CostThisMonth+estimatedCost > quota.CostPerMonth {
		return false
	}

	return true
}

func (rl *RateLimiter) updateUsage(quota *Quota, cost float64) {
	quota.mu.Lock()
	defer quota.mu.Unlock()

	quota.Current.RequestsThisHour++
	quota.Current.RequestsThisDay++
	quota.Current.CostThisHour += cost
	quota.Current.CostThisDay += cost
	quota.Current.CostThisMonth += cost

	quotaUsage.WithLabelValues(quota.Tenant, "requests", "hour").Set(float64(quota.Current.RequestsThisHour))
	quotaUsage.WithLabelValues(quota.Tenant, "requests", "day").Set(float64(quota.Current.RequestsThisDay))
	quotaUsage.WithLabelValues(quota.Tenant, "cost", "hour").Set(quota.Current.CostThisHour)
	quotaUsage.WithLabelValues(quota.Tenant, "cost", "day").Set(quota.Current.CostThisDay)
	quotaUsage.WithLabelValues(quota.Tenant, "cost", "month").Set(quota.Current.CostThisMonth)
}

func (rl *RateLimiter) checkTokenBucket(tenant string, quota *Quota) bool {
	rl.mu.RLock()
	bucket, exists := rl.buckets[tenant]
	rl.mu.RUnlock()

	if !exists {
		return true
	}

	return bucket.Allow()
}

func (rl *RateLimiter) checkSlidingWindow(tenant string, quota *Quota) bool {
	rl.mu.RLock()
	window, exists := rl.windows[tenant]
	rl.mu.RUnlock()

	if !exists {
		return true
	}

	return window.Allow()
}

func (rl *RateLimiter) checkFixedWindow(tenant string, quota *Quota) bool {
	return true
}

func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:       capacity,
		tokens:         float64(capacity),
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()
	tb.tokens = min(float64(tb.capacity), tb.tokens+elapsed*tb.refillRate)
	tb.lastRefillTime = now

	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

func NewSlidingWindow(limit int, windowSize time.Duration) *SlidingWindow {
	return &SlidingWindow{
		limit:      limit,
		windowSize: windowSize,
		requests:   make([]time.Time, 0),
	}
}

func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.windowSize)

	validRequests := make([]time.Time, 0)
	for _, reqTime := range sw.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	sw.requests = validRequests

	if len(sw.requests) >= sw.limit {
		return false
	}

	sw.requests = append(sw.requests, now)
	return true
}

func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := map[string]interface{}{
		"strategy":     rl.strategy,
		"tenant_count": len(rl.quotas),
		"quotas":       make(map[string]interface{}),
	}

	for tenant, quota := range rl.quotas {
		quota.mu.RLock()
		stats["quotas"].(map[string]interface{})[tenant] = map[string]interface{}{
			"requests_this_hour": quota.Current.RequestsThisHour,
			"requests_this_day":  quota.Current.RequestsThisDay,
			"cost_this_hour":     quota.Current.CostThisHour,
			"cost_this_day":      quota.Current.CostThisDay,
			"cost_this_month":    quota.Current.CostThisMonth,
		}
		quota.mu.RUnlock()
	}

	return stats
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
