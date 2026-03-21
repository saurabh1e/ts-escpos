package prints

import (
	"path/filepath"
	"testing"
	"time"

	"ts-escpos/backend/jobs"
)

func TestStoreReserveAndUpdateStatus(t *testing.T) {
	tempDir := t.TempDir()
	store, err := OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	kotNumber := 9
	record := PrintRecord{
		ID:          "job-1",
		DedupeKey:   "kot|INV-1|STORE-1|2026-03-21|9",
		InvoiceNo:   "INV-1",
		StoreID:     "STORE-1",
		Date:        "2026-03-21",
		KOTNumber:   &kotNumber,
		PrinterName: "Counter Printer",
		ReceiptType: "kot",
		Status:      jobs.StatusProcessing,
		CreatedAt:   time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC),
	}

	existingRecord, err := store.Reserve(record, false)
	if err != nil {
		t.Fatalf("reserve record: %v", err)
	}
	if existingRecord != nil {
		t.Fatalf("expected no existing record, got %+v", existingRecord)
	}

	duplicateRecord, err := store.Reserve(record, false)
	if err != ErrDuplicatePrint {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	if duplicateRecord == nil {
		t.Fatal("expected duplicate record details")
	}
	if duplicateRecord.ID != record.ID {
		t.Fatalf("expected duplicate id %s, got %s", record.ID, duplicateRecord.ID)
	}

	if err := store.UpdateStatus(record.ID, jobs.StatusSuccess, ""); err != nil {
		t.Fatalf("update status: %v", err)
	}

	storedRecord, err := store.FindLatestByKey(record.DedupeKey)
	if err != nil {
		t.Fatalf("find latest: %v", err)
	}
	if storedRecord == nil {
		t.Fatal("expected stored record")
	}
	if storedRecord.Status != jobs.StatusSuccess {
		t.Fatalf("expected status %s, got %s", jobs.StatusSuccess, storedRecord.Status)
	}
	if storedRecord.KOTNumber == nil || *storedRecord.KOTNumber != kotNumber {
		t.Fatalf("expected KOT number %d, got %+v", kotNumber, storedRecord.KOTNumber)
	}

	recentRecords, err := store.ListRecent(10)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(recentRecords) != 1 {
		t.Fatalf("expected 1 recent record, got %d", len(recentRecords))
	}
}

func TestStoreReserveWithOverrideKeepsHistory(t *testing.T) {
	tempDir := t.TempDir()
	store, err := OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}()

	firstRecord := PrintRecord{
		ID:          "job-1",
		DedupeKey:   "bill|INV-2|STORE-1|2026-03-21|-",
		InvoiceNo:   "INV-2",
		StoreID:     "STORE-1",
		Date:        "2026-03-21",
		PrinterName: "Counter Printer",
		ReceiptType: "bill",
		Status:      jobs.StatusFailed,
		CreatedAt:   time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC),
	}

	secondRecord := PrintRecord{
		ID:          "job-2",
		DedupeKey:   firstRecord.DedupeKey,
		InvoiceNo:   firstRecord.InvoiceNo,
		StoreID:     firstRecord.StoreID,
		Date:        firstRecord.Date,
		PrinterName: firstRecord.PrinterName,
		ReceiptType: firstRecord.ReceiptType,
		Status:      jobs.StatusProcessing,
		CreatedAt:   time.Date(2026, time.March, 21, 12, 5, 0, 0, time.UTC),
	}

	if _, err := store.Reserve(firstRecord, false); err != nil {
		t.Fatalf("reserve first record: %v", err)
	}
	if _, err := store.Reserve(secondRecord, true); err != nil {
		t.Fatalf("reserve second record with override: %v", err)
	}

	latestRecord, err := store.FindLatestByKey(firstRecord.DedupeKey)
	if err != nil {
		t.Fatalf("find latest: %v", err)
	}
	if latestRecord == nil || latestRecord.ID != secondRecord.ID {
		t.Fatalf("expected latest record %s, got %+v", secondRecord.ID, latestRecord)
	}

	recentRecords, err := store.ListRecent(10)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(recentRecords) != 2 {
		t.Fatalf("expected 2 recent records, got %d", len(recentRecords))
	}
	if recentRecords[0].ID != secondRecord.ID || recentRecords[1].ID != firstRecord.ID {
		t.Fatalf("unexpected record order: %+v", recentRecords)
	}
}

func TestMemoryStoreFallbackSupportsDuplicateChecks(t *testing.T) {
	store := newMemoryStore("persistence disabled")

	firstRecord := PrintRecord{
		ID:        "job-1",
		DedupeKey: "bill|INV-3|STORE-1|2026-03-21|-",
		InvoiceNo: "INV-3",
		StoreID:   "STORE-1",
		Date:      "2026-03-21",
		Status:    jobs.StatusProcessing,
	}

	if store.IsPersistent() {
		t.Fatal("expected fallback store to be non-persistent")
	}
	if store.Warning() == "" {
		t.Fatal("expected fallback warning")
	}

	if _, err := store.Reserve(firstRecord, false); err != nil {
		t.Fatalf("reserve first record: %v", err)
	}

	existingRecord, err := store.Reserve(PrintRecord{
		ID:        "job-2",
		DedupeKey: firstRecord.DedupeKey,
		InvoiceNo: firstRecord.InvoiceNo,
		StoreID:   firstRecord.StoreID,
		Date:      firstRecord.Date,
		Status:    jobs.StatusProcessing,
	}, false)
	if err != ErrDuplicatePrint {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	if existingRecord == nil || existingRecord.ID != firstRecord.ID {
		t.Fatalf("expected existing fallback record %s, got %+v", firstRecord.ID, existingRecord)
	}

	if err := store.UpdateStatus(firstRecord.ID, jobs.StatusSuccess, ""); err != nil {
		t.Fatalf("update status: %v", err)
	}

	latestRecord, err := store.FindLatestByKey(firstRecord.DedupeKey)
	if err != nil {
		t.Fatalf("find latest: %v", err)
	}
	if latestRecord == nil || latestRecord.Status != jobs.StatusSuccess {
		t.Fatalf("expected updated fallback record, got %+v", latestRecord)
	}
}
