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

// allKnownModelTypes is a package-level slice containing all model types
// the router is aware of. This avoids re-allocating this slice in multiple functions.
var allKnownModelTypes = []models.ModelType{models.OpenAI, models.Gemini, models.Mistral, models.Claude}

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
	
	// Create a new random source seeded with the current time for generating random numbers.
	// Using a mutex-protected rand.Rand instance is important for concurrent use.
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
	
	// This check ensures that if UpdateAvailability is called directly or by multiple goroutines
	// after passing ensureAvailabilityUpdated's check, it still respects TTL and avoids redundant work.
	if !r.lastUpdated.IsZero() && time.Since(r.lastUpdated) < r.availabilityTTL {
		logrus.WithFields(logrus.Fields{
			"last_updated": r.lastUpdated,
			"ttl":          r.availabilityTTL,
			"elapsed":      time.Since(r.lastUpdated),
		}).Debug("Skipping availability update due to TTL (within UpdateAvailability)")
		return
	}
	
	logrus.Debug("Updating model availability")
	// Iterate over the package-level slice of all known model types.
	for _, modelType := range allKnownModelTypes {
		client, err := llm.Factory(modelType)
		if err != nil {
			// If factory fails (e.g. misconfiguration), consider model unavailable.
			r.availableModels[modelType] = false
			logrus.WithError(err).WithField("model_type", modelType).Error("Failed to create LLM client for availability check")
			continue
		}
		
		r.availableModels[modelType] = client.CheckAvailability()
	}
	
	r.lastUpdated = time.Now()
}

// ensureAvailabilityUpdated checks if the model availability status is stale and updates it if necessary.
// It uses a read lock to check the timestamp and only triggers a full update if needed.
func (r *Router) ensureAvailabilityUpdated() {
	if r.testMode {
		return
	}
	
	r.availabilityMutex.RLock()
	// Check if an update is needed: either never updated or TTL expired.
	needsUpdate := r.lastUpdated.IsZero() || time.Since(r.lastUpdated) >= r.availabilityTTL
	r.availabilityMutex.RUnlock()
	
	if needsUpdate {
		// Call UpdateAvailability, which will handle the full lock and the actual update logic,
		// including its own TTL check to prevent redundant updates if multiple goroutines call this.
		r.UpdateAvailability()
	}
}

func (r *Router) GetAvailability() models.StatusResponse {
	// Ensure the availability information is reasonably up-to-date before returning.
	r.ensureAvailabilityUpdated()
	
	r.availabilityMutex.RLock()
	defer r.availabilityMutex.RUnlock()
	
	return models.StatusResponse{
		OpenAI:  r.availableModels[models.OpenAI],
		Gemini:  r.availableModels[models.Gemini],
		Mistral: r.availableModels[models.Mistral],
		Claude:  r.availableModels[models.Claude],
	}
}

func (r *Router) RouteRequest(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
	// Ensure availability is checked/updated once at the beginning of the request routing.
	// This provides a consistent view of availability for the duration of this function call.
	r.ensureAvailabilityUpdated()

	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	
	// 1. User-specified model
	if req.Model != "" {
		// isModelAvailable itself calls ensureAvailabilityUpdated, but after the initial call above,
		// it will likely be a quick check due to the TTL.
		if r.isModelAvailable(req.Model) {
			logging.LogRouterActivity(string(req.Model), string(req.Model), string(req.TaskType), "user_preference")
			return req.Model, nil
		}
		logrus.WithField("model", req.Model).Warn("Requested model not available, trying alternatives")
	}

	// Context check after potentially time-consuming operations or before next step.
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 2. Task-specific model
	if req.TaskType != "" {
		model, err := r.routeByTaskType(req.TaskType) // This also uses isModelAvailable internally.
		if err == nil {
			logging.LogRouterActivity("", string(model), string(req.TaskType), "task_type")
			return model, nil
		}
		// Log warning if task-based routing fails, but proceed to fallback.
		logrus.WithError(err).WithField("task_type", req.TaskType).Warn("Failed to route by task type, trying general fallback")
	}

	// Context check
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 3. Fallback to a random available model
	model, err := r.getRandomAvailableModel() // This also uses ensureAvailabilityUpdated internally.
	if err != nil {
		// If no models are available at all.
		return "", myerrors.NewUnavailableError("all models")
	}
	
	logging.LogRouterActivity(string(req.Model), string(model), string(req.TaskType), "fallback")
	return model, nil
}

func (r *Router) FallbackOnError(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error) {
	// Ensure availability is checked/updated once at the beginning of the fallback logic.
	r.ensureAvailabilityUpdated()

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	var modelErr *myerrors.ModelError
	if !errors.As(err, &modelErr) || !modelErr.Retryable {
		// If the error is not a retryable ModelError, don't attempt fallback.
		return "", err
	}

	// Get available models, excluding the one that just failed.
	// getAvailableModelsExcept also uses ensureAvailabilityUpdated internally (quick check due to TTL).
	availableModels := r.getAvailableModelsExcept(originalModel)
	if len(availableModels) == 0 {
		// No other models available to fallback to.
		return "", myerrors.NewUnavailableError("all other models for fallback")
	}

	// Check if user's original preference (if any, and different from failing one) is available
	if req.Model != "" && req.Model != originalModel {
		for _, model := range availableModels {
			// isModelAvailable also uses ensureAvailabilityUpdated internally (quick check due to TTL).
			if model == req.Model && r.isModelAvailable(model) {
				logging.LogRouterActivity(string(originalModel), string(model), string(req.TaskType), "error_fallback_to_user_preference")
				return model, nil
			}
		}
	}

	// Context check before random selection.
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Select a random model from the remaining available ones.
	r.randomSourceMutex.Lock()
	fallbackIndex := r.randomSource.Intn(len(availableModels))
	r.randomSourceMutex.Unlock()
	
	fallbackModel := availableModels[fallbackIndex]
	
	logging.LogRouterActivity(string(originalModel), string(fallbackModel), string(req.TaskType), "error_fallback_random")
	
	return fallbackModel, nil
}

// isModelAvailable checks if a specific model is marked as available.
// It ensures that the availability data is reasonably fresh before checking.
func (r *Router) isModelAvailable(model models.ModelType) bool {
	r.ensureAvailabilityUpdated() // Ensures data is fresh, respects TTL.
	
	r.availabilityMutex.RLock()
	defer r.availabilityMutex.RUnlock()
	
	// Returns true if the model is in the map and its value is true.
	// If the model is not in the map (should not happen if UpdateAvailability is comprehensive),
	// it defaults to false.
	return r.availableModels[model]
}

// routeByTaskType attempts to select a model based on the task type.
// Falls back to a random available model if the preferred model for the task is not available.
func (r *Router) routeByTaskType(taskType models.TaskType) (models.ModelType, error) {
	// Note: isModelAvailable calls ensureAvailabilityUpdated, so data freshness is handled.
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

	// If no task-specific model is available, try to get any random available model.
	return r.getRandomAvailableModel()
}

// getRandomAvailableModel selects a random model from the list of currently available models.
func (r *Router) getRandomAvailableModel() (models.ModelType, error) {
	r.ensureAvailabilityUpdated() // Ensures data is fresh, respects TTL.
	
	r.availabilityMutex.RLock()
	// Create a slice to hold available model types. Pre-allocate capacity
	// using the length of allKnownModelTypes to potentially reduce re-allocations.
	availableModelTypes := make([]models.ModelType, 0, len(allKnownModelTypes))
	for _, modelType := range allKnownModelTypes {
		if r.availableModels[modelType] { // Check against the current availability map.
			availableModelTypes = append(availableModelTypes, modelType)
		}
	}
	r.availabilityMutex.RUnlock() // Release RLock as soon as map access is done.

	if len(availableModelTypes) == 0 {
		return "", myerrors.NewUnavailableError("all models (checked in getRandomAvailableModel)")
	}

	r.randomSourceMutex.Lock()
	randomIndex := r.randomSource.Intn(len(availableModelTypes))
	r.randomSourceMutex.Unlock()
	
	return availableModelTypes[randomIndex], nil
}

// getAvailableModelsExcept returns a slice of available models, excluding a specified model.
func (r *Router) getAvailableModelsExcept(excludeModel models.ModelType) []models.ModelType {
	r.ensureAvailabilityUpdated() // Ensures data is fresh, respects TTL.
	
	r.availabilityMutex.RLock()
	// Pre-allocate with capacity using the length of allKnownModelTypes.
	availableModelTypes := make([]models.ModelType, 0, len(allKnownModelTypes))
	for _, modelType := range allKnownModelTypes {
		// Check if the model is not the one to exclude and is available.
		if modelType != excludeModel && r.availableModels[modelType] {
			availableModelTypes = append(availableModelTypes, modelType)
		}
	}
	r.availabilityMutex.RUnlock() // Release RLock as soon as map access is done.

	return availableModelTypes
}