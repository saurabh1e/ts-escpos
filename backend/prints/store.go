package prints

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"

	"ts-escpos/backend/jobs"
)

var ErrDuplicatePrint = errors.New("duplicate print record")

var (
	recordsBucket = []byte("print_records")
	indexBucket   = []byte("print_records_by_key")
	orderBucket   = []byte("print_records_order")
)

type PrintRecord struct {
	ID          string         `json:"id"`
	DedupeKey   string         `json:"dedupeKey"`
	InvoiceNo   string         `json:"invoiceNo"`
	StoreID     string         `json:"storeId"`
	Date        string         `json:"date"`
	KOTNumber   *int           `json:"kotNumber,omitempty"`
	PrinterName string         `json:"printerName"`
	ReceiptType string         `json:"receiptType"`
	Status      jobs.JobStatus `json:"status"`
	Error       string         `json:"error,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

func (r PrintRecord) ToJob() jobs.PrintJob {
	return jobs.PrintJob{
		ID:          r.ID,
		InvoiceNo:   r.InvoiceNo,
		StoreID:     r.StoreID,
		Date:        r.Date,
		KOTNumber:   r.KOTNumber,
		PrinterName: r.PrinterName,
		Status:      r.Status,
		Error:       r.Error,
		Timestamp:   r.CreatedAt,
		ReceiptType: r.ReceiptType,
	}
}

type Store struct {
	db              *bolt.DB
	mu              sync.RWMutex
	records         map[string]PrintRecord
	index           map[string]string
	order           []string
	isPersistent    bool
	fallbackWarning string
}

func OpenStore(path string) (*Store, error) {
	store := &Store{isPersistent: true}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return newMemoryStore(fmt.Sprintf("print store persistence disabled: create directory failed: %v", err)), nil
	}

	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: openTimeout()})
	if err != nil {
		return newMemoryStore(fmt.Sprintf("print store persistence disabled: open database failed: %v", err)), nil
	}
	store.db = db

	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(recordsBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(indexBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(orderBucket); err != nil {
			return err
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return newMemoryStore(fmt.Sprintf("print store persistence disabled: initialize database failed: %v", err)), nil
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) IsPersistent() bool {
	if s == nil {
		return false
	}
	return s.isPersistent
}

func (s *Store) Warning() string {
	if s == nil {
		return ""
	}
	return s.fallbackWarning
}

func (s *Store) Reserve(record PrintRecord, allowDuplicate bool) (*PrintRecord, error) {
	if s.db == nil {
		return s.reserveInMemory(record, allowDuplicate)
	}

	var existingRecord *PrintRecord

	err := s.db.Update(func(tx *bolt.Tx) error {
		records := tx.Bucket(recordsBucket)
		index := tx.Bucket(indexBucket)
		order := tx.Bucket(orderBucket)

		now := time.Now().UTC()
		if record.CreatedAt.IsZero() {
			record.CreatedAt = now
		} else {
			record.CreatedAt = record.CreatedAt.UTC()
		}
		record.UpdatedAt = now

		if !allowDuplicate {
			existingID := index.Get([]byte(record.DedupeKey))
			if len(existingID) > 0 {
				existingPayload := records.Get(existingID)
				if len(existingPayload) > 0 {
					var storedRecord PrintRecord
					if err := json.Unmarshal(existingPayload, &storedRecord); err == nil {
						existingRecord = &storedRecord
					}
				}
				return ErrDuplicatePrint
			}
		}

		payload, err := json.Marshal(record)
		if err != nil {
			return err
		}

		if err := records.Put([]byte(record.ID), payload); err != nil {
			return err
		}
		if err := index.Put([]byte(record.DedupeKey), []byte(record.ID)); err != nil {
			return err
		}
		if err := order.Put([]byte(buildOrderKey(record.CreatedAt, record.ID)), []byte(record.ID)); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return existingRecord, err
	}

	return nil, nil
}

func (s *Store) UpdateStatus(recordID string, status jobs.JobStatus, errorMessage string) error {
	if s.db == nil {
		return s.updateStatusInMemory(recordID, status, errorMessage)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		records := tx.Bucket(recordsBucket)
		payload := records.Get([]byte(recordID))
		if len(payload) == 0 {
			return fmt.Errorf("print record not found: %s", recordID)
		}

		var record PrintRecord
		if err := json.Unmarshal(payload, &record); err != nil {
			return err
		}

		record.Status = status
		record.Error = errorMessage
		record.UpdatedAt = time.Now().UTC()

		updatedPayload, err := json.Marshal(record)
		if err != nil {
			return err
		}

		return records.Put([]byte(recordID), updatedPayload)
	})
}

func (s *Store) FindLatestByKey(dedupeKey string) (*PrintRecord, error) {
	if s.db == nil {
		return s.findLatestByKeyInMemory(dedupeKey)
	}

	var record *PrintRecord

	err := s.db.View(func(tx *bolt.Tx) error {
		records := tx.Bucket(recordsBucket)
		index := tx.Bucket(indexBucket)

		recordID := index.Get([]byte(dedupeKey))
		if len(recordID) == 0 {
			return nil
		}

		payload := records.Get(recordID)
		if len(payload) == 0 {
			return nil
		}

		var storedRecord PrintRecord
		if err := json.Unmarshal(payload, &storedRecord); err != nil {
			return err
		}

		record = &storedRecord
		return nil
	})

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *Store) ListRecent(limit int) ([]PrintRecord, error) {
	if limit <= 0 {
		limit = jobs.MaxStoredJobs
	}

	if s.db == nil {
		return s.listRecentInMemory(limit), nil
	}

	recordsList := make([]PrintRecord, 0, limit)

	err := s.db.View(func(tx *bolt.Tx) error {
		records := tx.Bucket(recordsBucket)
		order := tx.Bucket(orderBucket)
		cursor := order.Cursor()

		for orderKey, recordID := cursor.Last(); orderKey != nil && len(recordsList) < limit; orderKey, recordID = cursor.Prev() {
			payload := records.Get(recordID)
			if len(payload) == 0 {
				continue
			}

			var record PrintRecord
			if err := json.Unmarshal(payload, &record); err != nil {
				return err
			}

			recordsList = append(recordsList, record)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return recordsList, nil
}

func buildOrderKey(createdAt time.Time, recordID string) string {
	return fmt.Sprintf("%020d:%s", createdAt.UTC().UnixNano(), recordID)
}

func openTimeout() time.Duration {
	if runtime.GOOS == "windows" {
		return 5 * time.Second
	}
	return 1 * time.Second
}

func newMemoryStore(warning string) *Store {
	return &Store{
		records:         make(map[string]PrintRecord),
		index:           make(map[string]string),
		order:           make([]string, 0),
		isPersistent:    false,
		fallbackWarning: warning,
	}
}

func (s *Store) reserveInMemory(record PrintRecord, allowDuplicate bool) (*PrintRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	} else {
		record.CreatedAt = record.CreatedAt.UTC()
	}
	record.UpdatedAt = now

	if !allowDuplicate {
		existingID, hasExistingRecord := s.index[record.DedupeKey]
		if hasExistingRecord {
			existingRecord := s.records[existingID]
			return &existingRecord, ErrDuplicatePrint
		}
	}

	s.records[record.ID] = record
	s.index[record.DedupeKey] = record.ID
	s.order = append(s.order, record.ID)

	return nil, nil
}

func (s *Store) updateStatusInMemory(recordID string, status jobs.JobStatus, errorMessage string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, hasRecord := s.records[recordID]
	if !hasRecord {
		return fmt.Errorf("print record not found: %s", recordID)
	}

	record.Status = status
	record.Error = errorMessage
	record.UpdatedAt = time.Now().UTC()
	s.records[recordID] = record

	return nil
}

func (s *Store) findLatestByKeyInMemory(dedupeKey string) (*PrintRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recordID, hasRecord := s.index[dedupeKey]
	if !hasRecord {
		return nil, nil
	}

	record := s.records[recordID]
	return &record, nil
}

func (s *Store) listRecentInMemory(limit int) []PrintRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recordsList := make([]PrintRecord, 0, limit)
	for i := len(s.order) - 1; i >= 0 && len(recordsList) < limit; i-- {
		recordID := s.order[i]
		record, hasRecord := s.records[recordID]
		if !hasRecord {
			continue
		}
		recordsList = append(recordsList, record)
	}

	return recordsList
}
