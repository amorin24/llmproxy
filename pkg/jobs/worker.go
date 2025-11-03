package jobs

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/pricing"
	"github.com/amorin24/llmproxy/pkg/router"
	"github.com/sirupsen/logrus"
)

type JobWorker struct {
	store         *JobStore
	router        *router.Router
	catalogLoader *pricing.CatalogLoader
	costEstimator *pricing.CostEstimator
	maxWorkers    int
	jobQueue      chan string
	stopCh        chan struct{}
}

func NewJobWorker(store *JobStore, router *router.Router, catalogLoader *pricing.CatalogLoader, maxWorkers int) *JobWorker {
	return &JobWorker{
		store:         store,
		router:        router,
		catalogLoader: catalogLoader,
		costEstimator: pricing.NewCostEstimator(catalogLoader),
		maxWorkers:    maxWorkers,
		jobQueue:      make(chan string, 100),
		stopCh:        make(chan struct{}),
	}
}

func (w *JobWorker) Start() {
	for i := 0; i < w.maxWorkers; i++ {
		go w.worker(i)
	}
	
	logrus.WithField("max_workers", w.maxWorkers).Info("Job worker started")
}

func (w *JobWorker) Stop() {
	close(w.stopCh)
	logrus.Info("Job worker stopped")
}

func (w *JobWorker) SubmitJob(jobID string) {
	select {
	case w.jobQueue <- jobID:
		logrus.WithField("job_id", jobID).Debug("Job submitted to queue")
	default:
		logrus.WithField("job_id", jobID).Warn("Job queue full, dropping job")
		w.store.UpdateJobError(jobID, "job queue full")
	}
}

func (w *JobWorker) worker(id int) {
	logrus.WithField("worker_id", id).Debug("Worker started")
	
	for {
		select {
		case jobID := <-w.jobQueue:
			w.processJob(jobID)
		case <-w.stopCh:
			logrus.WithField("worker_id", id).Debug("Worker stopped")
			return
		}
	}
}

func (w *JobWorker) processJob(jobID string) {
	job, err := w.store.GetJob(jobID)
	if err != nil {
		logrus.WithError(err).WithField("job_id", jobID).Error("Failed to get job")
		return
	}
	
	logrus.WithFields(logrus.Fields{
		"job_id": jobID,
		"query":  job.Query,
		"model":  job.Model,
	}).Info("Processing job")
	
	if err := w.store.UpdateJobStatus(jobID, JobStatusRunning); err != nil {
		logrus.WithError(err).WithField("job_id", jobID).Error("Failed to update job status")
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	req := models.QueryRequest{
		Query:        job.Query,
		Model:        job.Model,
		ModelVersion: job.ModelVersion,
		RequestID:    job.RequestID,
	}
	
	selectedModel, err := w.router.RouteRequest(ctx, req)
	if err != nil {
		logrus.WithError(err).WithField("job_id", jobID).Error("Failed to route request")
		w.store.UpdateJobError(jobID, fmt.Sprintf("routing failed: %v", err))
		w.sendWebhook(job, nil, err)
		return
	}
	
	client, err := llm.Factory(selectedModel)
	if err != nil {
		logrus.WithError(err).WithField("job_id", jobID).Error("Failed to create client")
		w.store.UpdateJobError(jobID, fmt.Sprintf("client creation failed: %v", err))
		w.sendWebhook(job, nil, err)
		return
	}
	
	modelVersion := llm.ValidateModelVersion(selectedModel, job.ModelVersion)
	
	result, err := client.Query(ctx, job.Query, modelVersion)
	if err != nil {
		logrus.WithError(err).WithField("job_id", jobID).Error("Failed to query LLM")
		w.store.UpdateJobError(jobID, fmt.Sprintf("query failed: %v", err))
		w.sendWebhook(job, nil, err)
		return
	}
	
	provider := pricing.MapModelTypeToProvider(selectedModel)
	actualCost := 0.0
	
	if result.InputTokens > 0 && result.OutputTokens > 0 {
		costEstimate, err := w.costEstimator.EstimatePostCall(provider, modelVersion, result.InputTokens, result.OutputTokens)
		if err == nil {
			actualCost = costEstimate.EstimatedCostUSD
		}
	}
	
	if err := w.store.UpdateJobResult(jobID, result.Response, actualCost, result.InputTokens, result.OutputTokens, provider); err != nil {
		logrus.WithError(err).WithField("job_id", jobID).Error("Failed to update job result")
		return
	}
	
	logrus.WithFields(logrus.Fields{
		"job_id":      jobID,
		"provider":    provider,
		"cost":        actualCost,
		"input_tokens": result.InputTokens,
		"output_tokens": result.OutputTokens,
	}).Info("Job completed successfully")
	
	w.sendWebhook(job, result, nil)
}

func (w *JobWorker) sendWebhook(job *Job, result *llm.QueryResult, err error) {
	if job.CallbackURL == "" {
		return
	}
	
	payload := map[string]interface{}{
		"job_id": job.ID,
		"status": string(job.Status),
	}
	
	if result != nil {
		payload["result"] = result.Response
		payload["actual_cost_usd"] = job.ActualCostUSD
		payload["input_tokens"] = result.InputTokens
		payload["output_tokens"] = result.OutputTokens
		payload["provider"] = job.Provider
	}
	
	if err != nil {
		payload["error"] = err.Error()
	}
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		req, err := http.NewRequestWithContext(ctx, "POST", job.CallbackURL, nil)
		if err != nil {
			logrus.WithError(err).WithField("job_id", job.ID).Error("Failed to create webhook request")
			return
		}
		
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logrus.WithError(err).WithField("job_id", job.ID).Error("Failed to send webhook")
			return
		}
		defer resp.Body.Close()
		
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logrus.WithFields(logrus.Fields{
				"job_id":      job.ID,
				"callback_url": job.CallbackURL,
				"status_code": resp.StatusCode,
			}).Info("Webhook delivered successfully")
		} else {
			logrus.WithFields(logrus.Fields{
				"job_id":      job.ID,
				"callback_url": job.CallbackURL,
				"status_code": resp.StatusCode,
			}).Warn("Webhook delivery failed")
		}
	}()
}
