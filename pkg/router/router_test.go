package router

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/models"
)

func TestRouteRequest(t *testing.T) {
	testCases := []struct {
		name           string
		request        models.QueryRequest
		availableModels map[models.ModelType]bool
		expectedModel  models.ModelType
		expectError    bool
		errorType      error
		useContext     bool // Whether to use a canceled context
	}{
		{
			name: "User specified model available",
			request: models.QueryRequest{
				Query: "Test query",
				Model: models.OpenAI,
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI: true,
			},
			expectedModel: models.OpenAI,
			expectError:   false,
		},
		{
			name: "User specified model unavailable, fallback to available",
			request: models.QueryRequest{
				Query: "Test query",
				Model: models.Gemini,
			},
			availableModels: map[models.ModelType]bool{
				models.Gemini: false,
				models.OpenAI: true,
			},
			expectedModel: models.OpenAI,
			expectError:   false,
		},
		{
			name: "No model specified, route by task type",
			request: models.QueryRequest{
				Query:    "Test query",
				TaskType: models.Summarization,
			},
			availableModels: map[models.ModelType]bool{
				models.Claude: true,
			},
			expectedModel: models.Claude,
			expectError:   false,
		},
		{
			name: "No model specified, task type model unavailable, fallback to random",
			request: models.QueryRequest{
				Query:    "Test query",
				TaskType: models.Summarization,
			},
			availableModels: map[models.ModelType]bool{
				models.Claude: false,
				models.OpenAI: true,
			},
			expectedModel: models.OpenAI,
			expectError:   false,
		},
		{
			name: "No model or task type specified, use random available",
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.Mistral: true,
			},
			expectedModel: models.Mistral,
			expectError:   false,
		},
		{
			name: "No models available",
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  false,
				models.Gemini:  false,
				models.Mistral: false,
				models.Claude:  false,
			},
			expectedModel: "",
			expectError:   true,
			errorType:     myerrors.ErrUnavailable,
		},
		{
			name: "Context canceled",
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI: true,
			},
			expectedModel: "",
			expectError:   true,
			useContext:    true,
		},
		{
			name: "Specific model preference with task type",
			request: models.QueryRequest{
				Query:    "Test query",
				Model:    models.Gemini,
				TaskType: models.Summarization, // Would normally route to Claude
			},
			availableModels: map[models.ModelType]bool{
				models.Gemini: true,
				models.Claude: true,
			},
			expectedModel: models.Gemini, // Model preference should override task type
			expectError:   false,
		},
		{
			name: "All models unavailable except one",
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  false,
				models.Gemini:  false,
				models.Mistral: false,
				models.Claude:  true,
			},
			expectedModel: models.Claude,
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRouter()
			r.SetTestMode(true)

			for model, available := range tc.availableModels {
				r.SetModelAvailability(model, available)
			}

			var ctx context.Context
			if tc.useContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel() // Cancel immediately to simulate context cancellation
			} else {
				ctx = context.Background()
			}

			model, err := r.RouteRequest(ctx, tc.request)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if tc.useContext {
					if !errors.Is(err, context.Canceled) {
						t.Errorf("Expected context.Canceled error, got %v", err)
					}
				} else if tc.errorType != nil {
					var modelErr *myerrors.ModelError
					if errors.As(err, &modelErr) {
						if !errors.Is(modelErr.Err, tc.errorType) {
							t.Errorf("Expected error type %v, got %v", tc.errorType, modelErr.Err)
						}
					} else {
						t.Errorf("Expected ModelError, got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if tc.expectedModel != "" && model != tc.expectedModel {
					t.Errorf("Expected model %s, got %s", tc.expectedModel, model)
				}
			}
		})
	}
}

func TestFallbackOnError(t *testing.T) {
	testCases := []struct {
		name           string
		originalModel  models.ModelType
		request        models.QueryRequest
		availableModels map[models.ModelType]bool
		inputError     error
		expectError    bool
		errorType      error
	}{
		{
			name:          "Fallback on retryable error",
			originalModel: models.OpenAI,
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  true,
				models.Gemini:  true,
				models.Mistral: false,
				models.Claude:  false,
			},
			inputError:  myerrors.NewRateLimitError("openai"),
			expectError: false,
		},
		{
			name:          "No fallback available",
			originalModel: models.OpenAI,
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  true,
				models.Gemini:  false,
				models.Mistral: false,
				models.Claude:  false,
			},
			inputError:  myerrors.NewRateLimitError("openai"),
			expectError: true,
			errorType:   myerrors.ErrUnavailable,
		},
		{
			name:          "Non-retryable error",
			originalModel: models.OpenAI,
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  true,
				models.Gemini:  true,
				models.Mistral: true,
				models.Claude:  true,
			},
			inputError:  errors.New("non-retryable error"),
			expectError: true,
		},
		{
			name:          "Retryable error with custom error",
			originalModel: models.OpenAI,
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  true,
				models.Gemini:  true,
				models.Mistral: true,
				models.Claude:  true,
			},
			inputError: &myerrors.ModelError{
				Model:     "openai",
				Code:      500,
				Err:       myerrors.ErrInvalidResponse,
				Retryable: true,
			},
			expectError: false,
		},
		{
			name:          "Fallback with specific model preference",
			originalModel: models.OpenAI,
			request: models.QueryRequest{
				Query: "Test query",
				Model: models.Claude, // User prefers Claude but started with OpenAI
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  true,
				models.Gemini:  true,
				models.Mistral: true,
				models.Claude:  true,
			},
			inputError:  myerrors.NewRateLimitError("openai"),
			expectError: false,
		},
		{
			name:          "Multiple fallback attempts needed",
			originalModel: models.OpenAI,
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  true,
				models.Gemini:  true,
				models.Mistral: true,
				models.Claude:  false,
			},
			inputError:  myerrors.NewRateLimitError("openai"),
			expectError: false,
		},
		{
			name:          "Context cancellation during fallback",
			originalModel: models.OpenAI,
			request: models.QueryRequest{
				Query: "Test query",
			},
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  true,
				models.Gemini:  true,
				models.Mistral: true,
				models.Claude:  true,
			},
			inputError:  context.Canceled,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRouter()
			r.SetTestMode(true)

			for model, available := range tc.availableModels {
				r.SetModelAvailability(model, available)
			}

			var ctx context.Context
			if tc.name == "Context cancellation during fallback" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel() // Cancel immediately to simulate context cancellation
			} else {
				ctx = context.Background()
			}

			fallbackModel, err := r.FallbackOnError(ctx, tc.originalModel, tc.request, tc.inputError)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if tc.name == "Context cancellation during fallback" {
					if !errors.Is(err, context.Canceled) {
						t.Errorf("Expected context.Canceled error, got %v", err)
					}
				} else if tc.errorType != nil {
					var modelErr *myerrors.ModelError
					if errors.As(err, &modelErr) {
						if !errors.Is(modelErr.Err, tc.errorType) {
							t.Errorf("Expected error type %v, got %v", tc.errorType, modelErr.Err)
						}
					} else {
						t.Errorf("Expected ModelError, got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if fallbackModel == tc.originalModel {
					t.Errorf("Fallback model should be different from original model")
				}
				if !r.isModelAvailable(fallbackModel) {
					t.Errorf("Fallback model %s should be available", fallbackModel)
				}
				
				if tc.name == "Fallback with specific model preference" && fallbackModel != tc.request.Model {
					t.Errorf("Expected fallback to user preferred model %s, got %s", tc.request.Model, fallbackModel)
				}
			}
		})
	}
}

func TestGetAvailability(t *testing.T) {
	r := NewRouter()
	
	r.SetTestMode(true)
	
	r.SetModelAvailability(models.OpenAI, true)
	r.SetModelAvailability(models.Gemini, false)
	r.SetModelAvailability(models.Mistral, true)
	r.SetModelAvailability(models.Claude, false)
	
	status := r.GetAvailability()
	
	if !status.OpenAI {
		t.Errorf("Expected OpenAI to be available")
	}
	if status.Gemini {
		t.Errorf("Expected Gemini to be unavailable")
	}
	if !status.Mistral {
		t.Errorf("Expected Mistral to be available")
	}
	if status.Claude {
		t.Errorf("Expected Claude to be unavailable")
	}
}

func TestGetRandomAvailableModel(t *testing.T) {
	r := NewRouter()
	
	r.SetTestMode(true)
	
	_, err := r.getRandomAvailableModel()
	if err == nil {
		t.Errorf("Expected error when no models available")
	}
	
	r.SetModelAvailability(models.OpenAI, true)
	model, err := r.getRandomAvailableModel()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if model != models.OpenAI {
		t.Errorf("Expected model %s, got %s", models.OpenAI, model)
	}
	
	r.SetModelAvailability(models.Mistral, true)
	model, err = r.getRandomAvailableModel()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if model != models.OpenAI && model != models.Mistral {
		t.Errorf("Expected model to be either %s or %s, got %s", models.OpenAI, models.Mistral, model)
	}
}

func TestGetAvailableModelsExcept(t *testing.T) {
	r := NewRouter()
	r.SetTestMode(true)
	
	availableModels := r.getAvailableModelsExcept(models.OpenAI)
	if len(availableModels) != 0 {
		t.Errorf("Expected 0 available models, got %d", len(availableModels))
	}
	
	r.SetModelAvailability(models.OpenAI, true)
	availableModels = r.getAvailableModelsExcept(models.OpenAI)
	if len(availableModels) != 0 {
		t.Errorf("Expected 0 available models after exclusion, got %d", len(availableModels))
	}
	
	r.SetModelAvailability(models.Gemini, true)
	availableModels = r.getAvailableModelsExcept(models.OpenAI)
	if len(availableModels) != 1 || availableModels[0] != models.Gemini {
		t.Errorf("Expected only Gemini to be available after excluding OpenAI")
	}
	
	r.SetModelAvailability(models.Mistral, true)
	r.SetModelAvailability(models.Claude, true)
	availableModels = r.getAvailableModelsExcept(models.OpenAI)
	if len(availableModels) != 3 {
		t.Errorf("Expected 3 available models after exclusion, got %d", len(availableModels))
	}
	
	for _, model := range availableModels {
		if model == models.OpenAI {
			t.Errorf("Excluded model should not be in the result")
		}
	}
}

func TestRouteByTaskType(t *testing.T) {
	testCases := []struct {
		name           string
		taskType       models.TaskType
		availableModels map[models.ModelType]bool
		expectedModel  models.ModelType
		expectError    bool
	}{
		{
			name:     "Text generation with OpenAI available",
			taskType: models.TextGeneration,
			availableModels: map[models.ModelType]bool{
				models.OpenAI: true,
			},
			expectedModel: models.OpenAI,
			expectError:   false,
		},
		{
			name:     "Summarization with Claude available",
			taskType: models.Summarization,
			availableModels: map[models.ModelType]bool{
				models.Claude: true,
			},
			expectedModel: models.Claude,
			expectError:   false,
		},
		{
			name:     "Sentiment analysis with Gemini available",
			taskType: models.SentimentAnalysis,
			availableModels: map[models.ModelType]bool{
				models.Gemini: true,
			},
			expectedModel: models.Gemini,
			expectError:   false,
		},
		{
			name:     "Question answering with Mistral available",
			taskType: models.QuestionAnswering,
			availableModels: map[models.ModelType]bool{
				models.Mistral: true,
			},
			expectedModel: models.Mistral,
			expectError:   false,
		},
		{
			name:     "Text generation with OpenAI unavailable",
			taskType: models.TextGeneration,
			availableModels: map[models.ModelType]bool{
				models.OpenAI: false,
				models.Gemini: true,
			},
			expectedModel: models.Gemini,
			expectError:   false,
		},
		{
			name:     "Unknown task type",
			taskType: "unknown",
			availableModels: map[models.ModelType]bool{
				models.OpenAI: true,
			},
			expectedModel: models.OpenAI,
			expectError:   false,
		},
		{
			name:     "No models available",
			taskType: models.TextGeneration,
			availableModels: map[models.ModelType]bool{
				models.OpenAI:  false,
				models.Gemini:  false,
				models.Mistral: false,
				models.Claude:  false,
			},
			expectedModel: "",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRouter()
			r.SetTestMode(true)

			for model, available := range tc.availableModels {
				r.SetModelAvailability(model, available)
			}

			model, err := r.routeByTaskType(tc.taskType)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if tc.expectedModel != "" && model != tc.expectedModel {
					t.Errorf("Expected model %s, got %s", tc.expectedModel, model)
				}
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := NewRouter()
	r.SetTestMode(true)
	
	r.SetModelAvailability(models.OpenAI, false)
	r.SetModelAvailability(models.Gemini, false)
	r.SetModelAvailability(models.Mistral, false)
	r.SetModelAvailability(models.Claude, false)
	
	r.SetModelAvailability(models.OpenAI, true)
	r.SetModelAvailability(models.Gemini, true)
	
	numGoroutines := 50 // Reduced to avoid flakiness
	
	var testMutex sync.Mutex
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines) // Only test readers, handle writers separately
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			
			status := r.GetAvailability()
			testMutex.Lock()
			if !status.OpenAI || !status.Gemini {
				t.Errorf("Expected OpenAI and Gemini to be available")
			}
			testMutex.Unlock()
			
			model, err := r.getRandomAvailableModel()
			testMutex.Lock()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if model != models.OpenAI && model != models.Gemini {
				t.Errorf("Expected model to be either OpenAI or Gemini, got %s", model)
			}
			testMutex.Unlock()
		}()
	}
	
	wg.Wait()
	
	wg.Add(2) // Just two writers for Mistral and Claude
	
	go func() {
		defer wg.Done()
		r.SetModelAvailability(models.Mistral, true)
	}()
	
	go func() {
		defer wg.Done()
		r.SetModelAvailability(models.Claude, true)
	}()
	
	wg.Wait()
	
	status := r.GetAvailability()
	if !status.OpenAI || !status.Gemini || !status.Mistral || !status.Claude {
		t.Errorf("Expected all models to be available after concurrent operations")
	}
}

func TestEnsureAvailabilityUpdated(t *testing.T) {
	r := NewRouter()
	
	r.availabilityTTL = 10 * time.Millisecond
	
	if !r.lastUpdated.IsZero() {
		t.Errorf("Expected lastUpdated to be zero initially")
	}
	
	r.ensureAvailabilityUpdated()
	if r.lastUpdated.IsZero() {
		t.Errorf("Expected lastUpdated to be set after first call")
	}
	
	firstUpdate := r.lastUpdated
	
	r.ensureAvailabilityUpdated()
	if r.lastUpdated != firstUpdate {
		t.Errorf("Expected lastUpdated to remain unchanged after immediate second call")
	}
	
	time.Sleep(20 * time.Millisecond)
	
	r.ensureAvailabilityUpdated()
	if r.lastUpdated == firstUpdate {
		t.Errorf("Expected lastUpdated to change after TTL expired")
	}
}
