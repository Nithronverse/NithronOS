package backup

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nithronos/backend/nosd/internal/fsatomic"
)

// Destination represents a backup destination
type Destination struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // local, ssh, s3, b2, webdav
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	LastBackup  *time.Time             `json:"lastBackup,omitempty"`
	LastStatus  string                 `json:"lastStatus,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// Job represents a backup job
type Job struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	DestinationID string     `json:"destinationId"`
	SourcePaths   []string   `json:"sourcePaths"`
	Schedule      string     `json:"schedule"` // cron expression
	Retention     int        `json:"retention"` // days
	Encryption    bool       `json:"encryption"`
	Compression   bool       `json:"compression"`
	Enabled       bool       `json:"enabled"`
	LastRun       *time.Time `json:"lastRun,omitempty"`
	NextRun       *time.Time `json:"nextRun,omitempty"`
	LastStatus    string     `json:"lastStatus,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// Run represents a single backup execution
type Run struct {
	ID          string     `json:"id"`
	JobID       string     `json:"jobId"`
	Status      string     `json:"status"` // running, success, failed
	StartTime   time.Time  `json:"startTime"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	BytesBackup int64      `json:"bytesBackup"`
	FilesBackup int        `json:"filesBackup"`
	Errors      []string   `json:"errors,omitempty"`
	Log         string     `json:"log"`
}

// Store manages backup configurations
type Store struct {
	path         string
	destinations map[string]*Destination
	jobs         map[string]*Job
	runs         map[string]*Run
	mu           sync.RWMutex
}

// NewStore creates a new backup store
func NewStore(path string) (*Store, error) {
	s := &Store{
		path:         path,
		destinations: make(map[string]*Destination),
		jobs:         make(map[string]*Job),
		runs:         make(map[string]*Run),
	}
	
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Load destinations
	destPath := filepath.Join(s.path, "destinations.json")
	var destinations []*Destination
	if ok, err := fsatomic.LoadJSON(destPath, &destinations); err != nil {
		return err
	} else if ok {
		for _, dest := range destinations {
			s.destinations[dest.ID] = dest
		}
	}
	
	// Load jobs
	jobsPath := filepath.Join(s.path, "jobs.json")
	var jobs []*Job
	if ok, err := fsatomic.LoadJSON(jobsPath, &jobs); err != nil {
		return err
	} else if ok {
		for _, job := range jobs {
			s.jobs[job.ID] = job
		}
	}
	
	// Load recent runs
	runsPath := filepath.Join(s.path, "runs.json")
	var runs []*Run
	if ok, err := fsatomic.LoadJSON(runsPath, &runs); err != nil {
		return err
	} else if ok {
		for _, run := range runs {
			s.runs[run.ID] = run
		}
	}
	
	return nil
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Save destinations
	destinations := make([]*Destination, 0, len(s.destinations))
	for _, dest := range s.destinations {
		destinations = append(destinations, dest)
	}
	destPath := filepath.Join(s.path, "destinations.json")
	if err := fsatomic.SaveJSON(context.Background(), destPath, destinations, 0600); err != nil {
		return err
	}
	
	// Save jobs
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	jobsPath := filepath.Join(s.path, "jobs.json")
	if err := fsatomic.SaveJSON(context.Background(), jobsPath, jobs, 0600); err != nil {
		return err
	}
	
	// Save runs (keep only last 100)
	runs := make([]*Run, 0, len(s.runs))
	for _, run := range s.runs {
		runs = append(runs, run)
	}
	if len(runs) > 100 {
		runs = runs[len(runs)-100:]
	}
	runsPath := filepath.Join(s.path, "runs.json")
	return fsatomic.SaveJSON(context.Background(), runsPath, runs, 0600)
}

// Destinations
func (s *Store) GetDestination(id string) (*Destination, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dest, ok := s.destinations[id]
	return dest, ok
}

func (s *Store) ListDestinations() []*Destination {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	list := make([]*Destination, 0, len(s.destinations))
	for _, dest := range s.destinations {
		list = append(list, dest)
	}
	return list
}

func (s *Store) AddDestination(dest *Destination) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.destinations[dest.ID] = dest
}

func (s *Store) UpdateDestination(id string, dest *Destination) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.destinations[id] = dest
}

func (s *Store) DeleteDestination(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.destinations, id)
}

// Jobs
func (s *Store) GetJob(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}

func (s *Store) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	list := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		list = append(list, job)
	}
	return list
}

func (s *Store) AddJob(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

func (s *Store) UpdateJob(id string, job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[id] = job
}

func (s *Store) DeleteJob(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, id)
}

// Runs
func (s *Store) GetRun(id string) (*Run, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[id]
	return run, ok
}

func (s *Store) ListRuns(jobID string) []*Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	list := make([]*Run, 0)
	for _, run := range s.runs {
		if jobID == "" || run.JobID == jobID {
			list = append(list, run)
		}
	}
	return list
}

func (s *Store) AddRun(run *Run) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.ID] = run
}

func (s *Store) UpdateRun(id string, run *Run) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[id] = run
}

// HasJobsForDestination checks if destination is used by any jobs
func (s *Store) HasJobsForDestination(destID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, job := range s.jobs {
		if job.DestinationID == destID {
			return true
		}
	}
	return false
}
