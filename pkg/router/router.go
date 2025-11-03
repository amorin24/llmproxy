package router

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/logging"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/sirupsen/logrus"
)

const defaultAvailabilityTTL = 300 // 5 minutes

type Router struct {
	availableModels     map[models.ModelType]bool
	testMode            bool // Flag to indicate if we're in test mode
	lastUpdated         time.Time
	availabilityTTL     time.Duration
	availabilityMutex   sync.RWMutex
	randomSource        *rand.Rand
	randomSourceMutex   sync.Mutex
}

func NewRouter() *Router {
	ttlStr := os.Getenv("AVAILABILITY_TTL")
	ttl := defaultAvailabilityTTL
	
	if ttlStr != "" {
		if parsedTTL, err := strconv.Atoi(ttlStr); err == nil && parsedTTL > 0 {
			ttl = parsedTTL
		}
	}
	
	source := rand.NewSource(time.Now().UnixNano())
	
	return &Router{
		availableModels:   make(map[models.ModelType]bool),
		testMode:          false,
		availabilityTTL:   time.Duration(ttl) * time.Second,
		randomSource:      rand.New(source),
	}
}

func (r *Router) SetTestMode(enabled bool) {
	r.testMode = enabled
}

func (r *Router) SetModelAvailability(model models.ModelType, available bool) {
	r.availabilityMutex.Lock()
	defer r.availabilityMutex.Unlock()
	
	r.availableModels[model] = available
}

func (r *Router) UpdateAvailability() {
	if r.testMode {
		return
	}
	
	r.availabilityMutex.Lock()
	defer r.availabilityMutex.Unlock()
	
	if !r.lastUpdated.IsZero() && time.Since(r.lastUpdated) < r.availabilityTTL {
		logrus.WithFields(logrus.Fields{
			"last_updated": r.lastUpdated,
			"ttl":          r.availabilityTTL,
			"elapsed":      time.Since(r.lastUpdated),
		}).Debug("Skipping availability update due to TTL")
		return
	}
	
	logrus.Debug("Updating model availability")
	modelTypes := []models.ModelType{models.OpenAI, models.Gemini, models.Mistral, models.Claude, models.VertexAI, models.Bedrock}
	
	for _, modelType := range modelTypes {
		client, err := llm.Factory(modelType)
		if err != nil {
			r.availableModels[modelType] = false
			continue
		}
		
		r.availableModels[modelType] = client.CheckAvailability()
	}
	
	r.lastUpdated = time.Now()
}

func (r *Router) ensureAvailabilityUpdated() {
	if r.testMode {
		return
	}
	
	r.availabilityMutex.RLock()
	needsUpdate := r.lastUpdated.IsZero() || time.Since(r.lastUpdated) >= r.availabilityTTL
	r.availabilityMutex.RUnlock()
	
	if needsUpdate {
		r.UpdateAvailability()
	}
}

func (r *Router) GetAvailability() models.StatusResponse {
	r.ensureAvailabilityUpdated()
	
	r.availabilityMutex.RLock()
	defer r.availabilityMutex.RUnlock()
	
	return models.StatusResponse{
		OpenAI:   r.availableModels[models.OpenAI],
		Gemini:   r.availableModels[models.Gemini],
		Mistral:  r.availableModels[models.Mistral],
		Claude:   r.availableModels[models.Claude],
		VertexAI: r.availableModels[models.VertexAI],
		Bedrock:  r.availableModels[models.Bedrock],
	}
}

func (r *Router) RouteRequest(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	
	if req.Model != "" {
		if r.isModelAvailable(req.Model) {
			logging.LogRouterActivity(string(req.Model), string(req.Model), string(req.TaskType), "user_preference")
			return req.Model, nil
		}
		logrus.WithField("model", req.Model).Warn("Requested model not available, trying alternatives")
	}

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if req.TaskType != "" {
		model, err := r.routeByTaskType(req.TaskType)
		if err == nil {
			logging.LogRouterActivity("", string(model), string(req.TaskType), "task_type")
			return model, nil
		}
		logrus.WithError(err).Warn("Failed to route by task type")
	}

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	model, err := r.getRandomAvailableModel()
	if err != nil {
		return "", myerrors.NewUnavailableError("all")
	}
	
	logging.LogRouterActivity(string(req.Model), string(model), string(req.TaskType), "fallback")
	return model, nil
}

func (r *Router) FallbackOnError(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	var modelErr *myerrors.ModelError
	if !errors.As(err, &modelErr) || !modelErr.Retryable {
		return "", err
	}

	availableModels := r.getAvailableModelsExcept(originalModel)
	if len(availableModels) == 0 {
		return "", myerrors.NewUnavailableError("all")
	}

	if req.Model != "" && req.Model != originalModel {
		for _, model := range availableModels {
			if model == req.Model && r.isModelAvailable(model) {
				logging.LogRouterActivity(string(originalModel), string(model), string(req.TaskType), "error_fallback")
				return model, nil
			}
		}
	}

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	r.randomSourceMutex.Lock()
	fallbackIndex := r.randomSource.Intn(len(availableModels))
	r.randomSourceMutex.Unlock()
	
	fallbackModel := availableModels[fallbackIndex]
	
	logging.LogRouterActivity(string(originalModel), string(fallbackModel), string(req.TaskType), "error_fallback")
	
	return fallbackModel, nil
}

func (r *Router) isModelAvailable(model models.ModelType) bool {
	r.ensureAvailabilityUpdated()
	
	r.availabilityMutex.RLock()
	defer r.availabilityMutex.RUnlock()
	
	return r.availableModels[model]
}

func (r *Router) routeByTaskType(taskType models.TaskType) (models.ModelType, error) {
	switch taskType {
	case models.TextGeneration:
		if r.isModelAvailable(models.OpenAI) {
			return models.OpenAI, nil
		}
	case models.Summarization:
		if r.isModelAvailable(models.Claude) {
			return models.Claude, nil
		}
	case models.SentimentAnalysis:
		if r.isModelAvailable(models.Gemini) {
			return models.Gemini, nil
		}
	case models.QuestionAnswering:
		if r.isModelAvailable(models.Mistral) {
			return models.Mistral, nil
		}
	}

	return r.getRandomAvailableModel()
}

func (r *Router) getRandomAvailableModel() (models.ModelType, error) {
	r.ensureAvailabilityUpdated()
	
	r.availabilityMutex.RLock()
	defer r.availabilityMutex.RUnlock()
	
	var availableModelTypes []models.ModelType
	modelTypes := []models.ModelType{models.OpenAI, models.Gemini, models.Mistral, models.Claude, models.VertexAI, models.Bedrock}

	for _, modelType := range modelTypes {
		if r.availableModels[modelType] {
			availableModelTypes = append(availableModelTypes, modelType)
		}
	}

	if len(availableModelTypes) == 0 {
		return "", myerrors.NewUnavailableError("all")
	}

	r.randomSourceMutex.Lock()
	randomIndex := r.randomSource.Intn(len(availableModelTypes))
	r.randomSourceMutex.Unlock()
	
	return availableModelTypes[randomIndex], nil
}

func (r *Router) getAvailableModelsExcept(excludeModel models.ModelType) []models.ModelType {
	r.ensureAvailabilityUpdated()
	
	r.availabilityMutex.RLock()
	defer r.availabilityMutex.RUnlock()
	
	var availableModelTypes []models.ModelType
	modelTypes := []models.ModelType{models.OpenAI, models.Gemini, models.Mistral, models.Claude, models.VertexAI, models.Bedrock}

	for _, modelType := range modelTypes {
		if modelType != excludeModel && r.availableModels[modelType] {
			availableModelTypes = append(availableModelTypes, modelType)
		}
	}

	return availableModelTypes
}
