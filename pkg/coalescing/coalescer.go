package coalescing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	coalescedRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llmproxy_coalesced_requests_total",
		Help: "Total number of requests that were coalesced with in-flight requests",
	})

	coalescingWaitTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "llmproxy_coalescing_wait_time_seconds",
		Help:    "Time spent waiting for coalesced request results",
		Buckets: prometheus.DefBuckets,
	})

	inflightRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "llmproxy_inflight_coalesced_requests",
		Help: "Number of in-flight requests being coalesced",
	})
)

type Coalescer struct {
	inflight map[string]*inflightRequest
	mu       sync.Mutex
	timeout  time.Duration
}

type inflightRequest struct {
	wg        sync.WaitGroup
	result    *llm.QueryResult
	err       error
	startTime time.Time
	waiters   int
}

type RequestKey struct {
	Query        string
	Model        models.ModelType
	ModelVersion string
	TaskType     string
}

func NewCoalescer(timeout time.Duration) *Coalescer {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Coalescer{
		inflight: make(map[string]*inflightRequest),
		timeout:  timeout,
	}
}

func (c *Coalescer) Do(ctx context.Context, key RequestKey, fn func() (*llm.QueryResult, error)) (*llm.QueryResult, bool, error) {
	keyStr := c.generateKey(key)

	c.mu.Lock()
	if req, exists := c.inflight[keyStr]; exists {
		req.waiters++
		c.mu.Unlock()

		logrus.WithFields(logrus.Fields{
			"key":     keyStr,
			"waiters": req.waiters,
		}).Debug("Coalescing request with in-flight request")

		coalescedRequestsTotal.Inc()
		startWait := time.Now()

		done := make(chan struct{})
		go func() {
			req.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			coalescingWaitTime.Observe(time.Since(startWait).Seconds())
			return req.result, true, req.err
		case <-ctx.Done():
			return nil, true, ctx.Err()
		case <-time.After(c.timeout):
			return nil, true, fmt.Errorf("coalescing timeout after %v", c.timeout)
		}
	}

	req := &inflightRequest{
		startTime: time.Now(),
		waiters:   0,
	}
	req.wg.Add(1)
	c.inflight[keyStr] = req
	inflightRequests.Inc()
	c.mu.Unlock()

	result, err := fn()

	c.mu.Lock()
	req.result = result
	req.err = err
	delete(c.inflight, keyStr)
	inflightRequests.Dec()
	c.mu.Unlock()

	req.wg.Done()

	logrus.WithFields(logrus.Fields{
		"key":      keyStr,
		"waiters":  req.waiters,
		"duration": time.Since(req.startTime),
	}).Debug("Completed coalesced request")

	return result, false, err
}

func (c *Coalescer) generateKey(key RequestKey) string {
	data := map[string]interface{}{
		"query":         key.Query,
		"model":         string(key.Model),
		"model_version": key.ModelVersion,
		"task_type":     key.TaskType,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

func (c *Coalescer) GetStats() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	return map[string]interface{}{
		"inflight_requests": len(c.inflight),
		"timeout_seconds":   c.timeout.Seconds(),
	}
}

func (c *Coalescer) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.inflight = make(map[string]*inflightRequest)
}
