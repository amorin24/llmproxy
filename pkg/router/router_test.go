package router

import (
	"testing"

	"github.com/amorin24/llmproxy/pkg/models"
)

func TestRouteRequest(t *testing.T) {
	r := NewRouter()
	
	r.SetTestMode(true)

	req := models.QueryRequest{
		Query: "Test query",
		Model: models.OpenAI,
	}

	r.SetModelAvailability(models.OpenAI, true)

	model, err := r.RouteRequest(req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if model != models.OpenAI {
		t.Errorf("Expected model %s, got %s", models.OpenAI, model)
	}

	req.Model = models.Gemini
	r.SetModelAvailability(models.Gemini, false)
	r.SetModelAvailability(models.OpenAI, true)

	model, err = r.RouteRequest(req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if model == models.Gemini {
		t.Errorf("Expected model not to be %s", models.Gemini)
	}

	req.Model = ""
	req.TaskType = models.Summarization
	r.SetModelAvailability(models.Claude, true)

	model, err = r.RouteRequest(req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if model != models.Claude {
		t.Errorf("Expected model %s, got %s", models.Claude, model)
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
