package jobs

import (
	"sync"
	"time"
)

type JobStatus string

const (
	StatusSuccess    JobStatus = "success"
	StatusFailed     JobStatus = "failure"
	StatusProcessing JobStatus = "printing"
	MaxStoredJobs              = 50
)

type PrintJob struct {
	ID          string    `json:"id"`
	InvoiceNo   string    `json:"invoiceNo"`
	StoreID     string    `json:"storeId,omitempty"`
	Date        string    `json:"date,omitempty"`
	KOTNumber   *int      `json:"kotNumber,omitempty"`
	PrinterName string    `json:"printerName"`
	Status      JobStatus `json:"status"`
	Error       string    `json:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	ReceiptType string    `json:"receiptType"`
}

type Store struct {
	mu   sync.RWMutex
	jobs []PrintJob
}

func NewStore() *Store {
	return &Store{
		jobs: make([]PrintJob, 0),
	}
}

func (s *Store) AddJob(job PrintJob) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if job exists and update it
	for i, j := range s.jobs {
		if j.ID == job.ID {
			s.jobs[i] = job
			return
		}
	}

	s.jobs = append(s.jobs, job)

	// Keep only the last MaxStoredJobs
	if len(s.jobs) > MaxStoredJobs {
		s.jobs = s.jobs[len(s.jobs)-MaxStoredJobs:]
	}
}

func (s *Store) GetJobs() []PrintJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return copy
	jobs := make([]PrintJob, len(s.jobs))
	copy(jobs, s.jobs)

	// Reverse order to have newest first (LIFO)
	for i, j := 0, len(jobs)-1; i < j; i, j = i+1, j-1 {
		jobs[i], jobs[j] = jobs[j], jobs[i]
	}
	return jobs
}

func (s *Store) ClearJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = make([]PrintJob, 0)
}
