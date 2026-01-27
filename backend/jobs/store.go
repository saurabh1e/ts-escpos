package jobs

import (
	"sync"
	"time"
)

type JobStatus string

const (
	StatusSuccess JobStatus = "success"
	StatusFailed  JobStatus = "failed"
)

type PrintJob struct {
	ID          string    `json:"id"`
	InvoiceNo   string    `json:"invoiceNo"`
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
	s.jobs = append(s.jobs, job)
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
