package anomaly

import (
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type CostBaseline struct {
	Provider       models.ModelType
	HourlyBaseline float64
	DailyBaseline  float64
	LastUpdated    time.Time
	SampleCount    int
}

type CostAnomaly struct {
	Provider    models.ModelType
	Timestamp   time.Time
	ActualCost  float64
	Baseline    float64
	Multiplier  float64 // How many times over baseline
	Description string
}

type ModelUsagePattern struct {
	Provider     models.ModelType
	ModelVersion string
	RequestCount int64
	TotalCost    float64
	LastSeen     time.Time
}

type Detector struct {
	baselines      map[models.ModelType]*CostBaseline
	anomalies      []CostAnomaly
	usagePatterns  map[string]*ModelUsagePattern // key: provider:model_version
	mu             sync.RWMutex
	spikeThreshold float64 // Alert when cost > baseline * threshold (default: 2.0)

	costBaseline *prometheus.GaugeVec
	costActual   *prometheus.GaugeVec
	costAnomaly  *prometheus.CounterVec
	modelUsage   *prometheus.CounterVec
}

func NewDetector(spikeThreshold float64) *Detector {
	if spikeThreshold <= 1.0 {
		spikeThreshold = 2.0 // Default to 2x baseline
	}

	detector := &Detector{
		baselines:      make(map[models.ModelType]*CostBaseline),
		anomalies:      make([]CostAnomaly, 0),
		usagePatterns:  make(map[string]*ModelUsagePattern),
		spikeThreshold: spikeThreshold,

		costBaseline: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llmproxy_cost_baseline_usd_per_hour",
				Help: "Baseline cost per hour by provider",
			},
			[]string{"provider"},
		),
		costActual: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llmproxy_cost_actual_usd_per_hour",
				Help: "Actual cost per hour by provider",
			},
			[]string{"provider"},
		),
		costAnomaly: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llmproxy_cost_anomalies_total",
				Help: "Total cost anomalies detected by provider",
			},
			[]string{"provider"},
		),
		modelUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llmproxy_model_usage_requests_total",
				Help: "Total requests by provider and model version",
			},
			[]string{"provider", "model_version"},
		),
	}

	providers := []models.ModelType{
		models.OpenAI, models.Gemini, models.Mistral,
		models.Claude, models.VertexAI, models.Bedrock,
	}

	for _, provider := range providers {
		detector.baselines[provider] = &CostBaseline{
			Provider:    provider,
			LastUpdated: time.Now(),
		}
	}

	prometheus.MustRegister(detector.costBaseline)
	prometheus.MustRegister(detector.costActual)
	prometheus.MustRegister(detector.costAnomaly)
	prometheus.MustRegister(detector.modelUsage)

	return detector
}

func (d *Detector) RecordCost(provider models.ModelType, cost float64, modelVersion string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := string(provider) + ":" + modelVersion
	pattern, exists := d.usagePatterns[key]
	if !exists {
		pattern = &ModelUsagePattern{
			Provider:     provider,
			ModelVersion: modelVersion,
		}
		d.usagePatterns[key] = pattern
	}

	pattern.RequestCount++
	pattern.TotalCost += cost
	pattern.LastSeen = time.Now()

	d.modelUsage.WithLabelValues(string(provider), modelVersion).Inc()

	baseline := d.baselines[provider]
	if baseline.SampleCount == 0 {
		baseline.HourlyBaseline = cost
	} else {
		alpha := 0.1
		baseline.HourlyBaseline = alpha*cost + (1-alpha)*baseline.HourlyBaseline
	}
	baseline.SampleCount++
	baseline.LastUpdated = time.Now()

	baseline.DailyBaseline = baseline.HourlyBaseline * 24

	d.costBaseline.WithLabelValues(string(provider)).Set(baseline.HourlyBaseline)
	d.costActual.WithLabelValues(string(provider)).Set(cost)

	if baseline.SampleCount > 10 && cost > baseline.HourlyBaseline*d.spikeThreshold {
		multiplier := cost / baseline.HourlyBaseline
		anomaly := CostAnomaly{
			Provider:    provider,
			Timestamp:   time.Now(),
			ActualCost:  cost,
			Baseline:    baseline.HourlyBaseline,
			Multiplier:  multiplier,
			Description: "Cost spike detected",
		}

		d.anomalies = append(d.anomalies, anomaly)
		d.costAnomaly.WithLabelValues(string(provider)).Inc()

		logrus.WithFields(logrus.Fields{
			"provider":   provider,
			"actual":     cost,
			"baseline":   baseline.HourlyBaseline,
			"multiplier": multiplier,
		}).Warn("Cost anomaly detected: spike above baseline")
	}
}

func (d *Detector) CheckUnusualUsage() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	alerts := make([]string, 0)

	providerPatterns := make(map[models.ModelType][]*ModelUsagePattern)
	for _, pattern := range d.usagePatterns {
		providerPatterns[pattern.Provider] = append(providerPatterns[pattern.Provider], pattern)
	}

	for provider, patterns := range providerPatterns {
		if len(patterns) < 2 {
			continue
		}

		totalRequests := int64(0)
		for _, p := range patterns {
			totalRequests += p.RequestCount
		}

		for _, p := range patterns {
			percentage := float64(p.RequestCount) / float64(totalRequests)
			if percentage > 0.9 {
				alert := "Unusual usage pattern: single model dominates"
				logrus.WithFields(logrus.Fields{
					"provider":      provider,
					"model_version": p.ModelVersion,
					"percentage":    percentage * 100,
				}).Warn(alert)

				alerts = append(alerts, alert)
			}
		}

		for _, p := range patterns {
			if time.Since(p.LastSeen) > 24*time.Hour {
				alert := "Unusual usage pattern: model not seen recently"
				logrus.WithFields(logrus.Fields{
					"provider":      provider,
					"model_version": p.ModelVersion,
					"last_seen":     p.LastSeen,
				}).Warn(alert)

				alerts = append(alerts, alert)
			}
		}
	}

	return alerts
}

func (d *Detector) GetBaseline(provider models.ModelType) *CostBaseline {
	d.mu.RLock()
	defer d.mu.RUnlock()

	baseline, exists := d.baselines[provider]
	if !exists {
		return nil
	}

	baselineCopy := *baseline
	return &baselineCopy
}

func (d *Detector) GetAnomalies(since time.Time) []CostAnomaly {
	d.mu.RLock()
	defer d.mu.RUnlock()

	anomalies := make([]CostAnomaly, 0)
	for _, a := range d.anomalies {
		if a.Timestamp.After(since) {
			anomalies = append(anomalies, a)
		}
	}

	return anomalies
}

func (d *Detector) GetUsagePatterns() map[string]*ModelUsagePattern {
	d.mu.RLock()
	defer d.mu.RUnlock()

	patterns := make(map[string]*ModelUsagePattern)
	for k, v := range d.usagePatterns {
		patternCopy := *v
		patterns[k] = &patternCopy
	}

	return patterns
}

func (d *Detector) ResetBaselines() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for provider := range d.baselines {
		d.baselines[provider] = &CostBaseline{
			Provider:    provider,
			LastUpdated: time.Now(),
		}
	}

	logrus.Info("Cost baselines reset")
}
