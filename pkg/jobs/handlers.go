package jobs

import (
	"encoding/json"
	"net/http"

	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/pricing"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type JobHandler struct {
	store         *JobStore
	worker        *JobWorker
	catalogLoader *pricing.CatalogLoader
	costEstimator *pricing.CostEstimator
}

func NewJobHandler(store *JobStore, worker *JobWorker, catalogLoader *pricing.CatalogLoader) *JobHandler {
	return &JobHandler{
		store:         store,
		worker:        worker,
		catalogLoader: catalogLoader,
		costEstimator: pricing.NewCostEstimator(catalogLoader),
	}
}

func (h *JobHandler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	var req JobSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = models.OpenAI
	}

	modelVersion := req.ModelVersion

	estimatedCost := 0.0
	inputTokens := pricing.EstimateTokenCount(req.Query)
	expectedOutputTokens := 500

	provider := pricing.MapModelTypeToProvider(model)
	if provider != "" {
		if modelVersion == "" {
			modelVersion = pricing.GetDefaultModelVersion(model)
		}

		estimate, err := h.costEstimator.EstimatePreCall(provider, modelVersion, inputTokens, expectedOutputTokens)
		if err == nil {
			estimatedCost = estimate.EstimatedCostUSD
		}
	}

	job := h.store.CreateJob(req.Query, string(model), modelVersion, req.CallbackURL, estimatedCost)

	h.worker.SubmitJob(job.ID)

	response := JobSubmitResponse{
		JobID:            job.ID,
		Status:           string(job.Status),
		EstimatedCostUSD: job.EstimatedCostUSD,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)

	logrus.WithFields(logrus.Fields{
		"job_id":         job.ID,
		"model":          model,
		"estimated_cost": estimatedCost,
	}).Info("Job submitted")
}

func (h *JobHandler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	job, err := h.store.GetJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	response := JobStatusResponse{
		JobID:            job.ID,
		Status:           string(job.Status),
		EstimatedCostUSD: job.EstimatedCostUSD,
		ActualCostUSD:    job.ActualCostUSD,
		CreatedAt:        job.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Error:            job.Error,
	}

	if job.StartedAt != nil {
		response.StartedAt = job.StartedAt.Format("2006-01-02T15:04:05Z")
	}

	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt.Format("2006-01-02T15:04:05Z")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *JobHandler) GetJobResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	job, err := h.store.GetJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	if job.Status == JobStatusPending || job.Status == JobStatusRunning {
		http.Error(w, "Job not completed yet", http.StatusAccepted)
		return
	}

	response := JobResultResponse{
		JobID:         job.ID,
		Status:        string(job.Status),
		Result:        job.Result,
		ActualCostUSD: job.ActualCostUSD,
		InputTokens:   job.InputTokens,
		OutputTokens:  job.OutputTokens,
		TotalTokens:   job.TotalTokens,
		Provider:      job.Provider,
		Error:         job.Error,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs := h.store.ListJobs()

	responses := make([]JobStatusResponse, 0, len(jobs))
	for _, job := range jobs {
		response := JobStatusResponse{
			JobID:            job.ID,
			Status:           string(job.Status),
			EstimatedCostUSD: job.EstimatedCostUSD,
			ActualCostUSD:    job.ActualCostUSD,
			CreatedAt:        job.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Error:            job.Error,
		}

		if job.StartedAt != nil {
			response.StartedAt = job.StartedAt.Format("2006-01-02T15:04:05Z")
		}

		if job.CompletedAt != nil {
			response.CompletedAt = job.CompletedAt.Format("2006-01-02T15:04:05Z")
		}

		responses = append(responses, response)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}
