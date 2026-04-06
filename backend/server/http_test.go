package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"ts-escpos/backend/config"
	"ts-escpos/backend/jobs"
	"ts-escpos/backend/printer"
	"ts-escpos/backend/prints"
	"ts-escpos/backend/receipt"
)

func TestHandlePrintBlocksDuplicatesUnlessOverrideIsEnabled(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["Test Printer"] = printer.PrinterInfo{Name: "Test Printer", Status: "Ready"}

	var mu sync.Mutex
	printCallCount := 0
	printedPayloads := make([][]byte, 0, 2)
	printDone := make(chan struct{}, 2)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		mu.Lock()
		defer mu.Unlock()
		printCallCount++
		printedPayloads = append(printedPayloads, append([]byte(nil), data...))
		printDone <- struct{}{}
		if printCallCount == 1 {
			return errors.New("printer jam")
		}
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	requestBody := PrintRequest{
		MachineID:   machineID,
		PrinterName: "Test Printer",
		PrinterSize: "80mm",
		ReceiptType: "bill",
		OrderData: receipt.OrderData{
			InvoiceNo: "INV-1001",
			Date:      "2026-03-21",
			StoreInfo: receipt.StoreInfo{StoreCode: "STORE-7"},
			Items:     []receipt.OrderItem{{Name: "Tea", Quantity: 1, Price: 10}},
		},
	}

	firstResponse := performPrintRequest(t, srv, requestBody)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("expected first print to succeed, got status %d body %s", firstResponse.Code, firstResponse.Body.String())
	}
	waitForPrint(t, printDone)
	waitForStatus(t, printStore, buildPrintDedupeKey("bill", "INV-1001", "STORE-7", "2026-03-21", nil, "Test Printer", ""), jobs.StatusFailed)

	secondResponse := performPrintRequest(t, srv, requestBody)
	if secondResponse.Code != http.StatusConflict {
		t.Fatalf("expected duplicate print to be blocked, got status %d body %s", secondResponse.Code, secondResponse.Body.String())
	}

	var duplicateResponse PrintResponse
	if err := json.Unmarshal(secondResponse.Body.Bytes(), &duplicateResponse); err != nil {
		t.Fatalf("decode duplicate response: %v", err)
	}
	if duplicateResponse.Error == "" {
		t.Fatal("expected duplicate response error message")
	}

	requestBody.AllowDuplicatePrint = true
	thirdResponse := performPrintRequest(t, srv, requestBody)
	if thirdResponse.Code != http.StatusOK {
		t.Fatalf("expected override print to succeed, got status %d body %s", thirdResponse.Code, thirdResponse.Body.String())
	}
	waitForPrint(t, printDone)
	waitForStatus(t, printStore, buildPrintDedupeKey("bill", "INV-1001", "STORE-7", "2026-03-21", nil, "Test Printer", ""), jobs.StatusSuccess)

	mu.Lock()
	defer mu.Unlock()
	if printCallCount != 2 {
		t.Fatalf("expected 2 print attempts, got %d", printCallCount)
	}
	if len(printedPayloads) != 2 {
		t.Fatalf("expected 2 captured payloads, got %d", len(printedPayloads))
	}
	if bytes.Contains(printedPayloads[0], []byte("DUPLICATE")) {
		t.Fatal("did not expect DUPLICATE marker on the first print")
	}
	if !bytes.Contains(printedPayloads[1], []byte("DUPLICATE")) {
		t.Fatal("expected DUPLICATE marker on the override print")
	}

	recentRecords, err := printStore.ListRecent(10)
	if err != nil {
		t.Fatalf("list recent records: %v", err)
	}
	if len(recentRecords) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(recentRecords))
	}
	if recentRecords[0].Status != jobs.StatusSuccess {
		t.Fatalf("expected latest status %s, got %s", jobs.StatusSuccess, recentRecords[0].Status)
	}
	if recentRecords[1].Status != jobs.StatusFailed {
		t.Fatalf("expected first status %s, got %s", jobs.StatusFailed, recentRecords[1].Status)
	}
}

func TestHandlePrintIncludesCINFromUIBody(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["BILL"] = printer.PrinterInfo{Name: "BILL", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	payload := map[string]interface{}{
		"machineId":   machineID,
		"printerName": "BILL",
		"printerSize": "80mm",
		"printerPort": 9100,
		"printerType": "ESC/POS",
		"receiptType": "bill",
		"orderData": map[string]interface{}{
			"invoiceNo":       "25-26/22",
			"tableNo":         "",
			"customerName":    "Walk-in",
			"customerContact": "",
			"date":            "22 Mar 2026, 05:36 pm IST",
			"items": []map[string]interface{}{
				{
					"id":          "fedf458d-f224-42c9-af34-f7dad838b11e",
					"name":        "Chickoo Ice Cream",
					"productName": "Chickoo Ice Cream",
					"quantity":    1.0,
					"unitPrice":   228.58,
					"sku":         "tenant:3:code:chickoo_ice_cream",
					"lineTotal":   228.58,
					"taxAmount":   11.42,
					"finalAmount": 228.58,
					"itemType":    "PRODUCT",
					"status":      "PENDING",
					"children": []map[string]interface{}{
						{
							"id":          "0",
							"name":        "Chickoo Ice Cream - Shake",
							"productName": "Chickoo Ice Cream - Shake",
							"quantity":    1.0,
							"unitPrice":   228.58,
							"sku":         "tenant:3:brand:3:prod:chickoo_ice_cream:var:shake",
							"lineTotal":   228.58,
							"finalAmount": 228.58,
							"itemType":    "VARIANT",
							"status":      "PENDING",
						},
					},
				},
			},
			"orderType": "TakeAway",
			"storeInfo": map[string]interface{}{
				"name":      "Mira Road (Dummy Outlet)",
				"storeCode": "N00045",
				"address":   "TEST ADDRESS",
				"location":  "TEST LOCATION",
				"firmName":  "TEST FIRM NAME",
				"mobile":    "9890442752",
				"gst":       "27ABCDE1234F1Z5",
				"fssai":     "11223344556677",
				"cin":       "U12345MH2020PTC123456",
			},
			"headerText":  "Shop 13 & 14, Krishna Towers, Shanti Park, Mira Road, Mumbai",
			"footerText":  "Whatsapp \"Hi\" on 8080801984 to order online\nEmail : customercare@naturalicecreams.in\nE&OE Thanks and visit again",
			"subTotal":    228.58,
			"tax":         11.42,
			"total":       240.0,
			"paymentMode": "Cash",
			"orderSource": "POS (Deepak's Laptop)",
			"cashierName": "Deepak Sharma",
			"taxBreakdown": []map[string]interface{}{
				{"name": "CGST", "rate": 2.5, "amount": 5.71},
				{"name": "SGST/UTGST", "rate": 2.5, "amount": 5.71},
			},
			"discountBreakdown": []map[string]interface{}{},
			"charges":           []map[string]interface{}{},
			"payments":          []map[string]interface{}{{"mode": "Cash", "amount": 240.0}},
			"displayOptions": map[string]interface{}{
				"showTaxBreakdown":      true,
				"showDiscountBreakdown": true,
				"showPaymentDetails":    true,
				"showCustomerInfo":      false,
				"showBarcode":           true,
				"showQRCode":            false,
			},
		},
		"printConfig": map[string]interface{}{
			"showTaxBreakdown":      true,
			"showDiscountBreakdown": true,
			"showPaymentDetails":    true,
			"showCustomerInfo":      false,
			"showBarcode":           true,
			"showQRCode":            false,
			"qrCodeData":            "",
			"printerSettings": map[string]interface{}{
				"printerName":        "BILL",
				"printerIP":          "",
				"printerPort":        9100,
				"printerType":        "ESC/POS",
				"paperSize":          "80mm",
				"copies":             1,
				"autoPrint":          true,
				"printHeader":        true,
				"printFooter":        true,
				"headerText":         "Shop 13 & 14, Krishna Towers, Shanti Park, Mira Road, Mumbai",
				"footerText":         "Whatsapp \"Hi\" on 8080801984 to order online\nEmail : customercare@naturalicecreams.in\nE&OE Thanks and visit again",
				"showLogo":           false,
				"logoURL":            "",
				"fontSize":           0,
				"charactersPerLine":  48,
				"additionalSettings": nil,
				"__typename":         "PrinterSettings",
			},
			"__typename": "BillPrintConfig",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/print", bytes.NewReader(body))
	response := httptest.NewRecorder()
	srv.handlePrint(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		if !bytes.Contains(raw, []byte("LLPIN/CIN: U12345MH2020PTC123456")) {
			t.Fatalf("expected printed payload to include CIN line, got %q", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func TestHandlePrintBillIncludesCashDrawerPulse(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["BILL"] = printer.PrinterInfo{Name: "BILL", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	requestBody := PrintRequest{
		MachineID:   machineID,
		PrinterName: "BILL",
		PrinterSize: "80mm",
		ReceiptType: "bill",
		OrderData: receipt.OrderData{
			InvoiceNo: "INV-DRAWER-1",
			Date:      "2026-04-06",
			Items:     []receipt.OrderItem{{Name: "Tea", Quantity: 1, Price: 10}},
			StoreInfo: receipt.StoreInfo{FirmName: "Test Store"},
			SubTotal:  10,
			Total:     10,
		},
	}

	response := performPrintRequest(t, srv, requestBody)
	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		cashDrawerPulse := []byte{0x1B, 0x70, 0x00, 0x19, 0xFA}
		if !bytes.Contains(raw, cashDrawerPulse) {
			t.Fatalf("expected bill payload with empty payment mode to include cash drawer pulse, got %v", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func TestHandlePrintBillIncludesCashDrawerPulseForCashPaymentMode(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["BILL"] = printer.PrinterInfo{Name: "BILL", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	requestBody := PrintRequest{
		MachineID:   machineID,
		PrinterName: "BILL",
		PrinterSize: "80mm",
		ReceiptType: "bill",
		OrderData: receipt.OrderData{
			InvoiceNo:   "INV-DRAWER-2",
			Date:        "2026-04-06",
			PaymentMode: "CASH",
			Items:       []receipt.OrderItem{{Name: "Tea", Quantity: 1, Price: 10}},
			StoreInfo:   receipt.StoreInfo{FirmName: "Test Store"},
			SubTotal:    10,
			Total:       10,
		},
	}

	response := performPrintRequest(t, srv, requestBody)
	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		cashDrawerPulse := []byte{0x1B, 0x70, 0x00, 0x19, 0xFA}
		if !bytes.Contains(raw, cashDrawerPulse) {
			t.Fatalf("expected bill payload with cash payment mode to include cash drawer pulse, got %v", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func TestHandlePrintBillIncludesCashDrawerPulseForNullPaymentMode(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["BILL"] = printer.PrinterInfo{Name: "BILL", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	payload := map[string]interface{}{
		"machineId":   machineID,
		"printerName": "BILL",
		"printerSize": "80mm",
		"receiptType": "bill",
		"orderData": map[string]interface{}{
			"invoiceNo":   "INV-DRAWER-3",
			"date":        "2026-04-06",
			"paymentMode": nil,
			"items":       []map[string]interface{}{{"name": "Tea", "quantity": 1.0, "price": 10.0}},
			"storeInfo":   map[string]interface{}{"firmName": "Test Store"},
			"subTotal":    10.0,
			"total":       10.0,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/print", bytes.NewReader(body))
	response := httptest.NewRecorder()
	srv.handlePrint(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		cashDrawerPulse := []byte{0x1B, 0x70, 0x00, 0x19, 0xFA}
		if !bytes.Contains(raw, cashDrawerPulse) {
			t.Fatalf("expected bill payload with null payment mode to include cash drawer pulse, got %v", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func TestHandlePrintBillSkipsCashDrawerPulseForNonCashPaymentMode(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["BILL"] = printer.PrinterInfo{Name: "BILL", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	requestBody := PrintRequest{
		MachineID:   machineID,
		PrinterName: "BILL",
		PrinterSize: "80mm",
		ReceiptType: "bill",
		OrderData: receipt.OrderData{
			InvoiceNo:   "INV-DRAWER-4",
			Date:        "2026-04-06",
			PaymentMode: "Card",
			Items:       []receipt.OrderItem{{Name: "Tea", Quantity: 1, Price: 10}},
			StoreInfo:   receipt.StoreInfo{FirmName: "Test Store"},
			SubTotal:    10,
			Total:       10,
		},
	}

	response := performPrintRequest(t, srv, requestBody)
	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		cashDrawerPulse := []byte{0x1B, 0x70, 0x00, 0x19, 0xFA}
		if bytes.Contains(raw, cashDrawerPulse) {
			t.Fatalf("did not expect bill payload with non-cash payment mode to include cash drawer pulse, got %v", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func TestHandlePrintKOTDoesNotIncludeCashDrawerPulse(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["KOT"] = printer.PrinterInfo{Name: "KOT", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	requestBody := PrintRequest{
		MachineID:   machineID,
		PrinterName: "KOT",
		PrinterSize: "80mm",
		ReceiptType: "kot",
		OrderData: receipt.OrderData{
			InvoiceNo: "INV-KOT-1",
			Date:      "2026-04-06",
			Items:     []receipt.OrderItem{{Name: "Tea", Quantity: 1}},
		},
	}

	response := performPrintRequest(t, srv, requestBody)
	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		cashDrawerPulse := []byte{0x1B, 0x70, 0x00, 0x19, 0xFA}
		if bytes.Contains(raw, cashDrawerPulse) {
			t.Fatalf("did not expect KOT payload to include cash drawer pulse, got %v", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func TestHandlePrintIncludesKOTExternalIDAndPaxFromUIBody(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["KOT"] = printer.PrinterInfo{Name: "KOT", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	payload := map[string]interface{}{
		"machineId":   machineID,
		"printerName": "KOT",
		"printerSize": "80mm",
		"receiptType": "kot",
		"orderData": map[string]interface{}{
			"invoiceNo":   "INV-2002",
			"externalId":  "KOT-EXT-12345678",
			"pax":         3,
			"cashierName": "Deepak Sharma",
			"date":        "22 Mar 2026, 05:36 pm IST",
			"items": []map[string]interface{}{
				{
					"name":     "Chickoo Ice Cream",
					"quantity": 1.0,
					"children": []map[string]interface{}{},
				},
			},
			"displayOptions": map[string]interface{}{
				"showOrderNumber": true,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/print", bytes.NewReader(body))
	response := httptest.NewRecorder()
	srv.handlePrint(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		if !bytes.Contains(raw, []byte("KOT-EXT-1234")) {
			t.Fatalf("expected printed payload to include KOT external ID prefix, got %q", raw)
		}
		if !bytes.Contains(raw, []byte(" 5678")) {
			t.Fatalf("expected printed payload to include KOT external ID suffix, got %q", raw)
		}
		if !bytes.Contains(raw, []byte("Order #: INV-2002  Pax: 3\n")) {
			t.Fatalf("expected printed payload to include pax after invoice, got %q", raw)
		}
		if !bytes.Contains(raw, []byte{0x1D, 0x42, 0x01}) {
			t.Fatalf("expected KOT external ID payload to use reverse mode for last 4 digits, got %q", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func TestHandlePrintKOTUsesExternalOrderIDField(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["KOT"] = printer.PrinterInfo{Name: "KOT", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	payload := map[string]interface{}{
		"machineId":   machineID,
		"printerName": "KOT",
		"printerSize": "80mm",
		"receiptType": "kot",
		"orderData": map[string]interface{}{
			"invoiceNo":       "25-26/102e59",
			"externalOrderId": "7941165327",
			"cashierName":     "System",
			"date":            "28 Mar 2026, 12:34 am IST",
			"items": []map[string]interface{}{
				{
					"name":     "Kesar Pista Ice Cream",
					"quantity": 1.0,
					"children": []map[string]interface{}{},
				},
			},
			"displayOptions": map[string]interface{}{
				"showOrderNumber": true,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/print", bytes.NewReader(body))
	response := httptest.NewRecorder()
	srv.handlePrint(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		externalIDIndex := bytes.Index(raw, []byte("794116"))
		if externalIDIndex < 0 {
			t.Fatalf("expected printed payload to include externalOrderId prefix, got %q", raw)
		}
		if !bytes.Contains(raw, []byte(" 5327")) {
			t.Fatalf("expected printed payload to include externalOrderId suffix, got %q", raw)
		}
		titleIndex := bytes.Index(raw, []byte("KOT\n"))
		if titleIndex < 0 || titleIndex <= externalIDIndex {
			t.Fatalf("expected printed payload to include KOT title after externalOrderId, got %q", raw)
		}
		if !bytes.Contains(raw, []byte("Order #: 25-26/102e59\n")) {
			t.Fatalf("expected printed payload to include KOT order number, got %q", raw)
		}
		if !bytes.Contains(raw, []byte{0x1D, 0x42, 0x01}) {
			t.Fatalf("expected KOT externalOrderId payload to reverse last 4 digits, got %q", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func TestHandlePrintBillUsesExternalOrderIDField(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["BILL"] = printer.PrinterInfo{Name: "BILL", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	payload := map[string]interface{}{
		"machineId":   machineID,
		"printerName": "BILL",
		"printerSize": "80mm",
		"receiptType": "bill",
		"orderData": map[string]interface{}{
			"invoiceNo":       "25-26/2",
			"externalOrderId": "7896135193",
			"date":            "26 Mar 2026, 08:22 pm IST",
			"items":           []map[string]interface{}{},
			"subTotal":        0,
			"tax":             4.66,
			"total":           4.66,
			"storeInfo": map[string]interface{}{
				"firmName": "Test Store",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/print", bytes.NewReader(body))
	response := httptest.NewRecorder()
	srv.handlePrint(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		if !bytes.Contains(raw, []byte("789613")) {
			t.Fatalf("expected printed payload to include external order ID prefix, got %q", raw)
		}
		if !bytes.Contains(raw, []byte("5193")) {
			t.Fatalf("expected printed payload to include external order ID bold suffix, got %q", raw)
		}
		if !bytes.Contains(raw, []byte("TAX INVOICE")) {
			t.Fatalf("expected printed payload to include TAX INVOICE title, got %q", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func TestHandlePrintBillUsesNumericExternalOrderIDField(t *testing.T) {
	tempDir := t.TempDir()
	jobStore := jobs.NewStore()
	printStore, err := prints.OpenStore(filepath.Join(tempDir, "prints.db"))
	if err != nil {
		t.Fatalf("open print store: %v", err)
	}
	defer func() {
		if err := printStore.Close(); err != nil {
			t.Fatalf("close print store: %v", err)
		}
	}()

	srv := NewServer(jobStore, printStore, &config.Config{HTTPPort: 9100}, "test")
	srv.SetContext(context.Background())
	srv.printers["BILL"] = printer.PrinterInfo{Name: "BILL", Status: "Ready"}

	printedPayload := make(chan []byte, 1)
	srv.printRaw = func(ctx context.Context, printerName string, data []byte) error {
		printedPayload <- append([]byte(nil), data...)
		return nil
	}

	machineID, machineIDErr := config.GetMachineID()
	if machineIDErr != nil {
		machineID = ""
	}

	payload := map[string]interface{}{
		"machineId":   machineID,
		"printerName": "BILL",
		"printerSize": "80mm",
		"receiptType": "bill",
		"orderData": map[string]interface{}{
			"invoiceNo":       "25-26/3",
			"externalOrderId": 7896135193,
			"date":            "26 Mar 2026, 08:22 pm IST",
			"items":           []map[string]interface{}{},
			"subTotal":        0,
			"tax":             4.66,
			"total":           4.66,
			"storeInfo": map[string]interface{}{
				"firmName": "Test Store",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/print", bytes.NewReader(body))
	response := httptest.NewRecorder()
	srv.handlePrint(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected print request to succeed, got status %d body %s", response.Code, response.Body.String())
	}

	select {
	case raw := <-printedPayload:
		titleIndex := bytes.Index(raw, []byte("TAX INVOICE"))
		if titleIndex < 0 {
			t.Fatalf("expected printed payload to include TAX INVOICE title, got %q", raw)
		}
		headerBlock := raw[:titleIndex]
		if !bytes.Contains(raw, []byte("789613")) {
			t.Fatalf("expected printed payload to include numeric external order ID prefix, got %q", raw)
		}
		if !bytes.Contains(raw, []byte("5193")) {
			t.Fatalf("expected printed payload to include numeric external order ID suffix, got %q", raw)
		}
		if bytes.Contains(headerBlock, []byte("e+")) || bytes.Contains(headerBlock, []byte(".0")) {
			t.Fatalf("expected numeric external order ID without float formatting, got %q", headerBlock)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for printed payload")
	}
}

func performPrintRequest(t *testing.T, srv *Server, requestBody PrintRequest) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/print", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	srv.handlePrint(response, request)
	return response
}

func waitForPrint(t *testing.T, printDone <-chan struct{}) {
	t.Helper()

	select {
	case <-printDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for print attempt")
	}
}

func waitForStatus(t *testing.T, printStore *prints.Store, dedupeKey string, expectedStatus jobs.JobStatus) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		record, err := printStore.FindLatestByKey(dedupeKey)
		if err != nil {
			t.Fatalf("find latest status: %v", err)
		}
		if record != nil && record.Status == expectedStatus {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for status %s", expectedStatus)
}
