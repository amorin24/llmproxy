package jobs

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type JobStore struct {
	jobs      map[string]*Job
	mu        sync.RWMutex
	ttl       time.Duration
	cleanupCh chan struct{}
}

func NewJobStore(ttl time.Duration) *JobStore {
	store := &JobStore{
		jobs:      make(map[string]*Job),
		ttl:       ttl,
		cleanupCh: make(chan struct{}),
	}

	go store.cleanupExpiredJobs()

	return store
}

func (s *JobStore) CreateJob(query string, model string, modelVersion string, callbackURL string, estimatedCost float64) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := &Job{
		ID:               uuid.New().String(),
		Query:            query,
		Model:            model,
		ModelVersion:     modelVersion,
		Status:           JobStatusPending,
		EstimatedCostUSD: estimatedCost,
		CallbackURL:      callbackURL,
		CreatedAt:        time.Now(),
		RequestID:        uuid.New().String(),
	}

	s.jobs[job.ID] = job
	return job
}

func (s *JobStore) GetJob(jobID string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

func (s *JobStore) UpdateJobStatus(jobID string, status JobStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Status = status

	now := time.Now()
	if status == JobStatusRunning && job.StartedAt == nil {
		job.StartedAt = &now
	} else if (status == JobStatusCompleted || status == JobStatusFailed) && job.CompletedAt == nil {
		job.CompletedAt = &now
	}

	return nil
}

func (s *JobStore) UpdateJobResult(jobID string, result string, actualCost float64, inputTokens int, outputTokens int, provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Result = result
	job.ActualCostUSD = actualCost
	job.InputTokens = inputTokens
	job.OutputTokens = outputTokens
	job.TotalTokens = inputTokens + outputTokens
	job.Provider = provider
	job.Status = JobStatusCompleted

	now := time.Now()
	if job.CompletedAt == nil {
		job.CompletedAt = &now
	}

	return nil
}

func (s *JobStore) UpdateJobError(jobID string, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Error = errorMsg
	job.Status = JobStatusFailed

	now := time.Now()
	if job.CompletedAt == nil {
		job.CompletedAt = &now
	}

	return nil
}

func (s *JobStore) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

func (s *JobStore) DeleteJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[jobID]; !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	delete(s.jobs, jobID)
	return nil
}

func (s *JobStore) cleanupExpiredJobs() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.removeExpiredJobs()
		case <-s.cleanupCh:
			return
		}
	}
}

func (s *JobStore) removeExpiredJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for jobID, job := range s.jobs {
		if job.CompletedAt != nil && now.Sub(*job.CompletedAt) > s.ttl {
			delete(s.jobs, jobID)
		} else if job.Status == JobStatusPending && now.Sub(job.CreatedAt) > s.ttl {
			delete(s.jobs, jobID)
		}
	}
}

func (s *JobStore) Close() {
	close(s.cleanupCh)
}

func (s *JobStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.jobs)
}

func (s *JobStore) CountByStatus(status JobStatus) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, job := range s.jobs {
		if job.Status == status {
			count++
		}
	}

	return count
}
