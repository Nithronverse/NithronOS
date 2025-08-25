package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Job represents a background job or task
type Job struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"` // pending, running, completed, failed
	Progress    *int       `json:"progress,omitempty"`
	Message     string     `json:"message,omitempty"`
	StartedAt   string     `json:"startedAt"`
	CompletedAt *string    `json:"completedAt,omitempty"`
	Error       *string    `json:"error,omitempty"`
}

// JobsHandler handles job-related endpoints
type JobsHandler struct {
	// In a real implementation, this would have a job queue/store
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler() *JobsHandler {
	return &JobsHandler{}
}

// Routes registers the jobs routes
func (h *JobsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	
	r.Get("/recent", h.GetRecentJobs)
	r.Get("/{id}", h.GetJob)
	
	return r
}

// GetRecentJobs returns recent jobs
// GET /api/v1/jobs/recent?limit=10
func (h *JobsHandler) GetRecentJobs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	// In a real implementation, this would query the job store
	// For now, return sample jobs
	
	now := time.Now()
	jobs := []Job{
		{
			ID:        "job-1",
			Type:      "pool.scrub",
			Status:    "completed",
			Message:   "Scrub completed on pool 'main'",
			StartedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
			CompletedAt: strPtr(now.Add(-1 * time.Hour).Format(time.RFC3339)),
		},
		{
			ID:        "job-2",
			Type:      "smart.scan",
			Status:    "completed",
			Message:   "SMART scan completed on 4 devices",
			StartedAt: now.Add(-3 * time.Hour).Format(time.RFC3339),
			CompletedAt: strPtr(now.Add(-2*time.Hour - 30*time.Minute).Format(time.RFC3339)),
		},
		{
			ID:        "job-3",
			Type:      "app.install",
			Status:    "completed",
			Message:   "Installed app: Nextcloud",
			StartedAt: now.Add(-5 * time.Hour).Format(time.RFC3339),
			CompletedAt: strPtr(now.Add(-4*time.Hour - 45*time.Minute).Format(time.RFC3339)),
		},
		{
			ID:        "job-4",
			Type:      "share.create",
			Status:    "completed",
			Message:   "Created share: Documents",
			StartedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
			CompletedAt: strPtr(now.Add(-23*time.Hour - 55*time.Minute).Format(time.RFC3339)),
		},
		{
			ID:        "job-5",
			Type:      "backup.snapshot",
			Status:    "running",
			Progress:  intPtr(67),
			Message:   "Creating snapshot for backup",
			StartedAt: now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
	}
	
	// Limit the results
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		log.Error().Err(err).Msg("Failed to encode jobs")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetJob returns a specific job by ID
// GET /api/v1/jobs/{id}
func (h *JobsHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	
	// In a real implementation, this would query the job store
	// For now, return a sample job
	
	job := Job{
		ID:        jobID,
		Type:      "pool.scrub",
		Status:    "completed",
		Message:   "Scrub completed on pool 'main'",
		StartedAt: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		CompletedAt: strPtr(time.Now().Add(-1 * time.Hour).Format(time.RFC3339)),
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(job); err != nil {
		log.Error().Err(err).Msg("Failed to encode job")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
