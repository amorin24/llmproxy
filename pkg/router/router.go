package router

import (
	"context"
	"errors"
	"math/rand"
	"time"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/logging"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/sirupsen/logrus"
)

type Router struct {
	availableModels map[models.ModelType]bool
	testMode        bool // Flag to indicate if we're in test mode
}

func NewRouter() *Router {
	return &Router{
		availableModels: make(map[models.ModelType]bool),
		testMode:        false,
	}
}

func (r *Router) SetTestMode(enabled bool) {
	r.testMode = enabled
}

func (r *Router) SetModelAvailability(model models.ModelType, available bool) {
	r.availableModels[model] = available
}

func (r *Router) UpdateAvailability() {
	if r.testMode {
		return
	}

	modelTypes := []models.ModelType{models.OpenAI, models.Gemini, models.Mistral, models.Claude}

	for _, modelType := range modelTypes {
		client, err := llm.Factory(modelType)
		if err != nil {
			r.availableModels[modelType] = false
			continue
		}

		r.availableModels[modelType] = client.CheckAvailability()
	}
}

func (r *Router) GetAvailability() models.StatusResponse {
	if !r.testMode {
		r.UpdateAvailability()
	}
	
	return models.StatusResponse{
		OpenAI:  r.availableModels[models.OpenAI],
		Gemini:  r.availableModels[models.Gemini],
		Mistral: r.availableModels[models.Mistral],
		Claude:  r.availableModels[models.Claude],
	}
}

func (r *Router) RouteRequest(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
	if req.Model != "" {
		if r.isModelAvailable(req.Model) {
			logging.LogRouterActivity(string(req.Model), string(req.Model), string(req.TaskType), "user_preference")
			return req.Model, nil
		}
		logrus.WithField("model", req.Model).Warn("Requested model not available, trying alternatives")
	}

	if req.TaskType != "" {
		model, err := r.routeByTaskType(req.TaskType)
		if err == nil {
			logging.LogRouterActivity("", string(model), string(req.TaskType), "task_type")
			return model, nil
		}
		logrus.WithError(err).Warn("Failed to route by task type")
	}

	model, err := r.getRandomAvailableModel()
	if err != nil {
		return "", myerrors.NewUnavailableError("all")
	}
	
	logging.LogRouterActivity(string(req.Model), string(model), string(req.TaskType), "fallback")
	return model, nil
}

func (r *Router) FallbackOnError(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error) {
	var modelErr *myerrors.ModelError
	if !errors.As(err, &modelErr) || !modelErr.Retryable {
		return "", err
	}

	availableModels := r.getAvailableModelsExcept(originalModel)
	if len(availableModels) == 0 {
		return "", myerrors.NewUnavailableError("all")
	}

	rand.Seed(time.Now().UnixNano())
	fallbackModel := availableModels[rand.Intn(len(availableModels))]
	
	logging.LogRouterActivity(string(originalModel), string(fallbackModel), string(req.TaskType), "error_fallback")
	
	return fallbackModel, nil
}

func (r *Router) isModelAvailable(model models.ModelType) bool {
	if !r.testMode {
		r.UpdateAvailability()
	}
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
	if !r.testMode {
		r.UpdateAvailability()
	}

	var availableModelTypes []models.ModelType
	modelTypes := []models.ModelType{models.OpenAI, models.Gemini, models.Mistral, models.Claude}

	for _, modelType := range modelTypes {
		if r.availableModels[modelType] {
			availableModelTypes = append(availableModelTypes, modelType)
		}
	}

	if len(availableModelTypes) == 0 {
		return "", myerrors.NewUnavailableError("all")
	}

	rand.Seed(time.Now().UnixNano())
	return availableModelTypes[rand.Intn(len(availableModelTypes))], nil
}

func (r *Router) getAvailableModelsExcept(excludeModel models.ModelType) []models.ModelType {
	if !r.testMode {
		r.UpdateAvailability()
	}

	var availableModelTypes []models.ModelType
	modelTypes := []models.ModelType{models.OpenAI, models.Gemini, models.Mistral, models.Claude}

	for _, modelType := range modelTypes {
		if modelType != excludeModel && r.availableModels[modelType] {
			availableModelTypes = append(availableModelTypes, modelType)
		}
	}

	return availableModelTypes
}
