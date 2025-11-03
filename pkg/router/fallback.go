package router

import (
	"context"
	"fmt"
	"time"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	fallbackAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llmproxy_fallback_attempts_total",
		Help: "Total number of fallback attempts",
	}, []string{"primary_provider", "fallback_provider", "success"})

	fallbackLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "llmproxy_fallback_latency_seconds",
		Help:    "Latency of fallback operations",
		Buckets: prometheus.DefBuckets,
	})
)

type FallbackStrategy struct {
	Primary     models.ModelType
	Secondary   []models.ModelType
	MaxRetries  int
	BackoffBase time.Duration
	CostAware   bool
	router      *Router
}

type FallbackConfig struct {
	Primary     models.ModelType
	Secondary   []models.ModelType
	MaxRetries  int
	BackoffBase time.Duration
	CostAware   bool
	Router      *Router
}

func NewFallbackStrategy(config FallbackConfig) *FallbackStrategy {
	if config.MaxRetries == 0 {
		config.MaxRetries = 2
	}
	if config.BackoffBase == 0 {
		config.BackoffBase = 100 * time.Millisecond
	}

	return &FallbackStrategy{
		Primary:     config.Primary,
		Secondary:   config.Secondary,
		MaxRetries:  config.MaxRetries,
		BackoffBase: config.BackoffBase,
		CostAware:   config.CostAware,
		router:      config.Router,
	}
}

func (s *FallbackStrategy) Execute(ctx context.Context, query string, modelVersion string) (*llm.QueryResult, models.ModelType, error) {
	startTime := time.Now()
	defer func() {
		fallbackLatency.Observe(time.Since(startTime).Seconds())
	}()

	result, err := s.tryProvider(ctx, s.Primary, query, modelVersion, 0)
	if err == nil {
		logrus.WithFields(logrus.Fields{
			"provider": s.Primary,
			"duration": time.Since(startTime),
		}).Debug("Primary provider succeeded")
		return result, s.Primary, nil
	}

	logrus.WithFields(logrus.Fields{
		"provider": s.Primary,
		"error":    err.Error(),
	}).Warn("Primary provider failed, attempting fallback")

	for i, secondary := range s.Secondary {
		if !s.isProviderAvailable(secondary) {
			logrus.WithField("provider", secondary).Debug("Skipping unavailable fallback provider")
			continue
		}

		for retry := 0; retry <= s.MaxRetries; retry++ {
			if retry > 0 {
				backoff := s.BackoffBase * time.Duration(1<<uint(retry-1))
				logrus.WithFields(logrus.Fields{
					"provider": secondary,
					"retry":    retry,
					"backoff":  backoff,
				}).Debug("Retrying with backoff")

				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return nil, "", ctx.Err()
				}
			}

			result, err := s.tryProvider(ctx, secondary, query, modelVersion, retry)
			if err == nil {
				fallbackAttempts.WithLabelValues(string(s.Primary), string(secondary), "true").Inc()

				logrus.WithFields(logrus.Fields{
					"primary_provider":  s.Primary,
					"fallback_provider": secondary,
					"fallback_index":    i,
					"retry":             retry,
					"duration":          time.Since(startTime),
				}).Info("Fallback succeeded")

				return result, secondary, nil
			}

			logrus.WithFields(logrus.Fields{
				"provider": secondary,
				"retry":    retry,
				"error":    err.Error(),
			}).Warn("Fallback provider attempt failed")
		}

		fallbackAttempts.WithLabelValues(string(s.Primary), string(secondary), "false").Inc()
	}

	return nil, "", fmt.Errorf("all providers failed: primary=%s, secondary=%v", s.Primary, s.Secondary)
}

func (s *FallbackStrategy) tryProvider(ctx context.Context, provider models.ModelType, query string, modelVersion string, retryCount int) (*llm.QueryResult, error) {
	client, err := llm.Factory(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for %s: %w", provider, err)
	}

	validatedVersion := llm.ValidateModelVersion(provider, modelVersion)

	result, err := client.Query(ctx, query, validatedVersion)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *FallbackStrategy) isProviderAvailable(provider models.ModelType) bool {
	if s.router == nil {
		return true // Assume available if no router
	}

	availability := s.router.GetAvailability()

	switch provider {
	case models.OpenAI:
		return availability.OpenAI
	case models.Gemini:
		return availability.Gemini
	case models.Mistral:
		return availability.Mistral
	case models.Claude:
		return availability.Claude
	case models.VertexAI:
		return availability.VertexAI
	case models.Bedrock:
		return availability.Bedrock
	default:
		return false
	}
}

func (s *FallbackStrategy) GetFallbackChain() []models.ModelType {
	chain := []models.ModelType{s.Primary}
	chain = append(chain, s.Secondary...)
	return chain
}

func (s *FallbackStrategy) UpdateSecondary(secondary []models.ModelType) {
	s.Secondary = secondary
	logrus.WithField("secondary", secondary).Debug("Updated fallback secondary providers")
}

func CostAwareFallbackStrategy(primary models.ModelType, router *Router) *FallbackStrategy {
	costOrder := []models.ModelType{
		models.Gemini,   // Typically cheapest
		models.Mistral,  // Mid-range
		models.OpenAI,   // Mid-range
		models.Claude,   // Higher cost
		models.VertexAI, // Variable
		models.Bedrock,  // Variable
	}

	secondary := []models.ModelType{}
	for _, provider := range costOrder {
		if provider != primary {
			secondary = append(secondary, provider)
		}
	}

	return NewFallbackStrategy(FallbackConfig{
		Primary:     primary,
		Secondary:   secondary,
		MaxRetries:  2,
		BackoffBase: 100 * time.Millisecond,
		CostAware:   true,
		Router:      router,
	})
}

func QualityAwareFallbackStrategy(primary models.ModelType, router *Router) *FallbackStrategy {
	qualityOrder := []models.ModelType{
		models.Claude,   // Typically highest quality
		models.OpenAI,   // High quality
		models.Bedrock,  // High quality (Claude on AWS)
		models.VertexAI, // High quality (Gemini on GCP)
		models.Gemini,   // Good quality
		models.Mistral,  // Good quality
	}

	secondary := []models.ModelType{}
	for _, provider := range qualityOrder {
		if provider != primary {
			secondary = append(secondary, provider)
		}
	}

	return NewFallbackStrategy(FallbackConfig{
		Primary:     primary,
		Secondary:   secondary,
		MaxRetries:  2,
		BackoffBase: 100 * time.Millisecond,
		CostAware:   false,
		Router:      router,
	})
}
