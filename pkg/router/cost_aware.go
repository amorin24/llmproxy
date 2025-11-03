package router

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/pricing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	routingCostSavings = promauto.NewCounter(prometheus.CounterOpts{
		Name: "llmproxy_routing_cost_savings_usd_total",
		Help: "Total cost savings from cost-aware routing",
	})

	routingStrategySelections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "llmproxy_routing_strategy_selections_total",
		Help: "Total number of provider selections by routing strategy",
	}, []string{"strategy", "provider"})

	providerQualityScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "llmproxy_provider_quality_score",
		Help: "Quality score for each provider (0.0 to 1.0)",
	}, []string{"provider"})

	providerLatencyP95 = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "llmproxy_provider_latency_p95_seconds",
		Help: "P95 latency for each provider",
	}, []string{"provider"})
)

type RoutingStrategy string

const (
	StrategyCostOptimized    RoutingStrategy = "cost_optimized"
	StrategyBalanced         RoutingStrategy = "balanced"
	StrategyQualityFirst     RoutingStrategy = "quality_first"
	StrategyLatencyOptimized RoutingStrategy = "latency_optimized"
)

type CostAwareRouter struct {
	router         *Router
	catalogLoader  *pricing.CatalogLoader
	qualityScores  map[string]float64 // provider -> score (0.0 to 1.0)
	latencyP95     map[string]float64 // provider -> p95 latency in seconds
	strategy       RoutingStrategy
	costWeight     float64
	latencyWeight  float64
	qualityWeight  float64
	mu             sync.RWMutex
}

type ProviderScore struct {
	Provider      string
	Model         models.ModelType
	CostScore     float64
	LatencyScore  float64
	QualityScore  float64
	TotalScore    float64
	EstimatedCost float64
}

type CostAwareRouterConfig struct {
	Router        *Router
	CatalogLoader *pricing.CatalogLoader
	Strategy      RoutingStrategy
	CostWeight    float64
	LatencyWeight float64
	QualityWeight float64
}

func NewCostAwareRouter(config CostAwareRouterConfig) *CostAwareRouter {
	if config.Strategy == "" {
		config.Strategy = StrategyBalanced
	}

	costWeight, latencyWeight, qualityWeight := getStrategyWeights(config.Strategy)
	
	if config.CostWeight > 0 {
		costWeight = config.CostWeight
	}
	if config.LatencyWeight > 0 {
		latencyWeight = config.LatencyWeight
	}
	if config.QualityWeight > 0 {
		qualityWeight = config.QualityWeight
	}

	total := costWeight + latencyWeight + qualityWeight
	if total > 0 {
		costWeight /= total
		latencyWeight /= total
		qualityWeight /= total
	}

	router := &CostAwareRouter{
		router:        config.Router,
		catalogLoader: config.CatalogLoader,
		qualityScores: make(map[string]float64),
		latencyP95:    make(map[string]float64),
		strategy:      config.Strategy,
		costWeight:    costWeight,
		latencyWeight: latencyWeight,
		qualityWeight: qualityWeight,
	}

	router.initializeDefaults()

	return router
}

func (r *CostAwareRouter) SelectProvider(ctx context.Context, req models.QueryRequest, inputTokens int, expectedOutputTokens int) (models.ModelType, *ProviderScore, error) {
	availability := r.router.GetAvailability()
	availableProviders := []models.ModelType{}

	for provider, available := range availability {
		if available {
			modelType := stringToModelType(provider)
			if modelType != "" {
				availableProviders = append(availableProviders, modelType)
			}
		}
	}

	if len(availableProviders) == 0 {
		return "", nil, fmt.Errorf("no providers available")
	}

	if req.Model != "" {
		for _, provider := range availableProviders {
			if provider == req.Model {
				score := r.calculateProviderScore(provider, inputTokens, expectedOutputTokens)
				routingStrategySelections.WithLabelValues(string(r.strategy), string(provider)).Inc()
				return provider, score, nil
			}
		}
		return "", nil, fmt.Errorf("requested provider %s not available", req.Model)
	}

	scores := make([]*ProviderScore, 0, len(availableProviders))
	for _, provider := range availableProviders {
		score := r.calculateProviderScore(provider, inputTokens, expectedOutputTokens)
		scores = append(scores, score)
	}

	var bestScore *ProviderScore
	for _, score := range scores {
		if bestScore == nil || score.TotalScore > bestScore.TotalScore {
			bestScore = score
		}
	}

	if bestScore == nil {
		return "", nil, fmt.Errorf("failed to calculate provider scores")
	}

	var maxCost float64
	for _, score := range scores {
		if score.EstimatedCost > maxCost {
			maxCost = score.EstimatedCost
		}
	}
	savings := maxCost - bestScore.EstimatedCost
	if savings > 0 {
		routingCostSavings.Add(savings)
	}

	routingStrategySelections.WithLabelValues(string(r.strategy), bestScore.Provider).Inc()

	logrus.WithFields(logrus.Fields{
		"selected_provider": bestScore.Provider,
		"strategy":          r.strategy,
		"total_score":       bestScore.TotalScore,
		"cost_score":        bestScore.CostScore,
		"latency_score":     bestScore.LatencyScore,
		"quality_score":     bestScore.QualityScore,
		"estimated_cost":    bestScore.EstimatedCost,
		"cost_savings":      savings,
	}).Debug("Cost-aware routing selection")

	return bestScore.Model, bestScore, nil
}

func (r *CostAwareRouter) calculateProviderScore(modelType models.ModelType, inputTokens int, expectedOutputTokens int) *ProviderScore {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider := pricing.MapModelTypeToProvider(modelType)
	
	score := &ProviderScore{
		Provider: provider,
		Model:    modelType,
	}

	estimator := pricing.NewCostEstimator(r.catalogLoader)
	modelVersion := pricing.GetDefaultModelVersion(modelType)
	costEstimate, err := estimator.EstimatePreCall(provider, modelVersion, inputTokens, expectedOutputTokens)
	
	if err == nil {
		score.EstimatedCost = costEstimate.EstimatedCostUSD
		score.CostScore = 1.0 - min(costEstimate.EstimatedCostUSD, 1.0)
	} else {
		score.CostScore = 0.5 // Default if cost unknown
	}

	if latency, exists := r.latencyP95[provider]; exists {
		score.LatencyScore = 1.0 - min(latency/10.0, 1.0)
	} else {
		score.LatencyScore = 0.5 // Default if latency unknown
	}

	if quality, exists := r.qualityScores[provider]; exists {
		score.QualityScore = quality
	} else {
		score.QualityScore = 0.5 // Default if quality unknown
	}

	score.TotalScore = (score.CostScore * r.costWeight) +
		(score.LatencyScore * r.latencyWeight) +
		(score.QualityScore * r.qualityWeight)

	return score
}

func (r *CostAwareRouter) UpdateQualityScore(provider string, score float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	score = max(0.0, min(1.0, score))
	
	r.qualityScores[provider] = score
	providerQualityScore.WithLabelValues(provider).Set(score)

	logrus.WithFields(logrus.Fields{
		"provider": provider,
		"score":    score,
	}).Debug("Updated provider quality score")
}

func (r *CostAwareRouter) UpdateLatencyP95(provider string, latencySeconds float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.latencyP95[provider] = latencySeconds
	providerLatencyP95.WithLabelValues(provider).Set(latencySeconds)

	logrus.WithFields(logrus.Fields{
		"provider":        provider,
		"latency_seconds": latencySeconds,
	}).Debug("Updated provider P95 latency")
}

func (r *CostAwareRouter) SetStrategy(strategy RoutingStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.strategy = strategy
	r.costWeight, r.latencyWeight, r.qualityWeight = getStrategyWeights(strategy)

	logrus.WithFields(logrus.Fields{
		"strategy":       strategy,
		"cost_weight":    r.costWeight,
		"latency_weight": r.latencyWeight,
		"quality_weight": r.qualityWeight,
	}).Info("Changed routing strategy")
}

func (r *CostAwareRouter) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"strategy":        r.strategy,
		"cost_weight":     r.costWeight,
		"latency_weight":  r.latencyWeight,
		"quality_weight":  r.qualityWeight,
		"quality_scores":  r.qualityScores,
		"latency_p95":     r.latencyP95,
	}
}

func (r *CostAwareRouter) initializeDefaults() {
	defaults := map[string]struct {
		quality float64
		latency float64
	}{
		"openai":     {quality: 0.90, latency: 1.5},
		"gemini":     {quality: 0.85, latency: 1.2},
		"mistral":    {quality: 0.80, latency: 1.8},
		"claude":     {quality: 0.92, latency: 2.0},
		"vertex_ai":  {quality: 0.85, latency: 1.3},
		"bedrock":    {quality: 0.88, latency: 1.7},
	}

	for provider, values := range defaults {
		r.qualityScores[provider] = values.quality
		r.latencyP95[provider] = values.latency
		providerQualityScore.WithLabelValues(provider).Set(values.quality)
		providerLatencyP95.WithLabelValues(provider).Set(values.latency)
	}
}

func getStrategyWeights(strategy RoutingStrategy) (cost, latency, quality float64) {
	switch strategy {
	case StrategyCostOptimized:
		return 0.7, 0.2, 0.1
	case StrategyBalanced:
		return 0.33, 0.33, 0.34
	case StrategyQualityFirst:
		return 0.1, 0.2, 0.7
	case StrategyLatencyOptimized:
		return 0.1, 0.7, 0.2
	default:
		return 0.33, 0.33, 0.34
	}
}

func stringToModelType(provider string) models.ModelType {
	switch provider {
	case "openai":
		return models.OpenAI
	case "gemini":
		return models.Gemini
	case "mistral":
		return models.Mistral
	case "claude":
		return models.Claude
	case "vertex_ai":
		return models.VertexAI
	case "bedrock":
		return models.Bedrock
	default:
		return ""
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
