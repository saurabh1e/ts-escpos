package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"ts-escpos/backend/config"
	"ts-escpos/backend/jobs"
	"ts-escpos/backend/printer"
	"ts-escpos/backend/receipt"
)

type EventEmitter func(ctx context.Context, eventName string, optionalData ...interface{})

type Server struct {
	store          *jobs.Store
	config         *config.Config
	ctx            context.Context
	clients        map[*websocket.Conn]bool
	clientsMux     sync.Mutex
	upgrader       websocket.Upgrader
	printers       map[string]printer.PrinterInfo
	defaultPrinter string
	printersMux    sync.RWMutex
	eventEmitter   EventEmitter
	version        string
}

func NewServer(store *jobs.Store, cfg *config.Config, version string) *Server {
	return &Server{
		store:    store,
		config:   cfg,
		version:  version,
		clients:  make(map[*websocket.Conn]bool),
		printers: make(map[string]printer.PrinterInfo),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all CORS for now to support development from various origins
				// In production, check r.Header.Get("Origin") against cfg.AllowedCors
				return true
			},
		},
	}
}

func (s *Server) SetEventEmitter(emitter EventEmitter) {
	s.eventEmitter = emitter
}

func (s *Server) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *Server) refreshPrinters() {
	list, err := printer.GetPrinters()
	if err != nil {
		fmt.Printf("Failed to refresh printers: %v\n", err)
		return
	}

	s.printersMux.Lock()
	defer s.printersMux.Unlock()

	s.printers = make(map[string]printer.PrinterInfo)
	s.defaultPrinter = ""
	if len(list) > 0 {
		s.defaultPrinter = list[0].Name // Default to first printer
	}
	for _, p := range list {
		s.printers[p.Name] = p
	}
	fmt.Printf("Printers refreshed. Found %d printers. Default: %s\n", len(s.printers), s.defaultPrinter)
}

func (s *Server) Start() {
	s.refreshPrinters()

	mux := http.NewServeMux()

	fmt.Println("Registering routes...")
	// Register identifier first to test priority
	mux.HandleFunc("/api/identifier", s.handleGetIdentifier)
	mux.HandleFunc("/api/identifier/", s.handleGetIdentifier)

	mux.HandleFunc("/api/print", s.handlePrint)
	mux.HandleFunc("/api/printers", s.handleGetPrinters)
	mux.HandleFunc("/api/version", s.handleGetVersion)
	mux.HandleFunc("/api/validate", s.handleValidate)
	mux.HandleFunc("/api/test-notification", s.handleTestNotification)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	// Catch-all for debugging
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("404 Debug: No route matched for %s\n", r.URL.Path)
		http.NotFound(w, r)
	})

	// Wrap with CORS
	handler := s.corsMiddleware(mux)

	addr := fmt.Sprintf(":%d", s.config.HTTPPort)
	fmt.Printf("Starting HTTP server on %s\n", addr)
	go func() {
		if err := http.ListenAndServe(addr, handler); err != nil {
			fmt.Printf("HTTP Server failed: %v\n", err)
		}
	}()
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// fmt.Printf("Incoming request: %s %s\n", r.Method, r.URL.Path) // Verbose logging disabled

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Private-Network")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			// fmt.Printf("Request: %s %s | Status: %d | Duration: %v\n", r.Method, r.URL.Path, http.StatusOK, time.Since(start))
			return
		}

		lrw := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(lrw, r)

		code := lrw.statusCode
		if code == 0 {
			code = http.StatusOK
		}

		// Only log errors to reduce console I/O overhead
		if code >= 400 {
			fmt.Printf("Request: %s %s | Status: %d | Duration: %v\n", r.Method, r.URL.Path, code, time.Since(start))
		}
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}
	return lrw.ResponseWriter.Write(b)
}

func (lrw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("hijack not supported")
}

func (lrw *loggingResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

type PrintRequest struct {
	MachineID   string            `json:"machineId"`
	PrinterName string            `json:"printerName"`
	OrderData   receipt.OrderData `json:"orderData"`
	PrinterSize string            `json:"printerSize"`
	ReceiptType string            `json:"receiptType"`
}

type PrintResponse struct {
	Success bool   `json:"success"`
	JobID   string `json:"jobId"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) notifyError(title, message, icon string, sound bool) {
	logMsg := fmt.Sprintf("[Notification] Title: %s | Message: %s", title, message)
	fmt.Println(logMsg)
	if s.ctx != nil && s.eventEmitter != nil {
		s.eventEmitter(s.ctx, "backend_log", logMsg)
	}

	// 1. Notify Wails Frontend
	if s.ctx != nil && s.eventEmitter != nil {
		s.eventEmitter(s.ctx, "error_notification", map[string]string{
			"title":   title,
			"message": message,
			"icon":    icon,
		})
	}

	// 2. Play Sound if requested
	if sound {
		go playSystemSound()
	}

	// 3. Show System Notification (Windows Toast / macOS Notification)
	go func() {
		// Empty string for icon uses default/system icon
		if err := beeep.Notify(title, message, icon); err != nil {
			fmt.Printf("Failed to send system notification: %v\n", err)
		}
	}()
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade to websocket: %v\n", err)
		return
	}

	// Register client
	s.clientsMux.Lock()
	s.clients[ws] = true
	s.clientsMux.Unlock()

	fmt.Println("New WebSocket client connected")

	// Send initial welcome message
	ws.WriteJSON(map[string]string{"type": "connected", "message": "Connected to TS-ESCPOS Printer Service"})

	// Keep connection alive / Listen for messages
	go func() {
		defer func() {
			s.clientsMux.Lock()
			delete(s.clients, ws)
			s.clientsMux.Unlock()
			ws.Close()
		}()

		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (s *Server) handlePrint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PrintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("Print request decode error: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Printf("[Print] Received print request for printer: %s, invoiceNo: %s\n", req.PrinterName, req.OrderData.GetInvoiceNo())

	// 1. Unique ID Validation
	storedMachineID, err := config.GetMachineID()
	if err == nil && req.MachineID != storedMachineID {
		fmt.Printf("Print request validation failed: Invalid Machine ID (got: %s, expected: %s)\n", req.MachineID, storedMachineID)
		s.notifyError("Validation Failed", "Unique ID validation failed.", "", false)
		http.Error(w, "Invalid Machine ID", http.StatusUnauthorized)
		return
	}

	// Resolve printer from cache; fall back to default ESC/POS printer when not found
	s.printersMux.RLock()
	targetPrinterName := req.PrinterName
	selectedPrinter, exists := s.printers[targetPrinterName]
	cacheSize := len(s.printers)
	s.printersMux.RUnlock()

	fmt.Printf("[Print] Printer cache size: %d, printer '%s' found in cache: %v\n", cacheSize, targetPrinterName, exists)

	if !exists {
		// Attempt to use system default printer if requested printer is not found (and not in our cache because it might be unsupported)
		// Or if we specifically want fallback behavior.
		// The prompt says: "always choose the default printer if name not mactehed. irrespective whether the printer is esc pos or not"

		fmt.Printf("[Print] Printer '%s' not found in supported list. Attempting fallback to system default.\n", targetPrinterName)

		defaultName, err := printer.GetDefaultPrinterName()
		if err == nil && defaultName != "" {
			fmt.Printf("[Print] System default printer found: '%s'\n", defaultName)
			targetPrinterName = defaultName

			// We don't have status info for this printer if it wasn't in cache, but we proceed anyway.
			// Construct a dummy info for logging purposes
			selectedPrinter = printer.PrinterInfo{
				Name:      defaultName,
				Status:    "Unknown (Fallback)",
				WindowsID: "Unknown",
			}
			exists = true // Treat as found
		} else {
			fmt.Printf("[Print] Failed to get system default printer: %v\n", err)

			// Try internal cache default as last resort (if any supported printer exists)
			s.printersMux.RLock()
			fallback := s.defaultPrinter
			fallbackPrinter, fallbackExists := s.printers[fallback]
			s.printersMux.RUnlock()

			if fallbackExists && fallback != "" {
				fmt.Printf("[Print] Falling back to internal default supported printer: '%s'\n", fallback)
				targetPrinterName = fallback
				selectedPrinter = fallbackPrinter
				exists = true
			}
		}
	}

	if !exists {
		msg := fmt.Sprintf("Printer '%s' not found and no default printer available.", req.PrinterName)
		fmt.Printf("Print failed: %s\n", msg)
		s.notifyError("Printer Not Found", msg, "", true)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	// Initialize job for tracking
	jobID := uuid.New().String()
	job := jobs.PrintJob{
		ID:          jobID,
		InvoiceNo:   req.OrderData.GetInvoiceNo(),
		PrinterName: targetPrinterName,
		ReceiptType: req.ReceiptType,
		Timestamp:   time.Now(),
		Status:      jobs.StatusFailed,
	}

	// 4. Respond to client immediately (Async processing)
	resp := PrintResponse{
		Success: true,
		JobID:   jobID,
		Message: "Print job submitted successfully. Processing in background.",
	}
	job.Status = jobs.StatusProcessing
	s.store.AddJob(job)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	// 5. Background Printing Process
	go func() {
		fmt.Printf("[Job %s] Starting background print for %s\n", jobID, targetPrinterName)
		defer func() {
			s.store.AddJob(job) // Update final status
		}()

		// 3. Status Check (using cached status)
		statusLower := strings.ToLower(selectedPrinter.Status)
		blockingStatuses := []string{"offline", "not available"}
		for _, bs := range blockingStatuses {
			if strings.Contains(statusLower, bs) {
				msg := fmt.Sprintf("Printer '%s' status is %s. Might fail.", targetPrinterName, selectedPrinter.Status)
				fmt.Printf("Warning: %s\n", msg)
			}
		}

		adapter := printer.NewEscposAdapter()
		if req.ReceiptType == "kot" {
			receipt.RenderKOT(adapter, req.OrderData, req.PrinterSize)
		} else {
			receipt.RenderBill(adapter, req.OrderData, req.PrinterSize)
		}

		bytesToPrint := adapter.GetBytes()
		fmt.Printf("[Job %s] Generic ESC/POS bytes generated (%d bytes)\n", jobID, len(bytesToPrint))

		// Use s.ctx to allow logging to frontend
		err := printer.PrintRaw(s.ctx, targetPrinterName, bytesToPrint)
		if err != nil {
			fmt.Printf("[Job %s] PRINT FAILED: %v\n", jobID, err)
			job.Status = jobs.StatusFailed
			job.Error = err.Error()

			s.notifyError("Print Failed", fmt.Sprintf("Failed to print on %s: %v", targetPrinterName, err), "", true)
		} else {
			fmt.Printf("[Job %s] PRINT SUCCESS\n", jobID)
			job.Status = jobs.StatusSuccess
		}
	}()
}

func (s *Server) handleGetPrinters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.printersMux.RLock()
	defer s.printersMux.RUnlock()

	var printerList []printer.PrinterInfo
	for _, p := range s.printers {
		printerList = append(printerList, p)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"printers": printerList,
	})
}

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"version": s.version,
	})
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MachineID string `json:"machineId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id, err := config.GetMachineID()
	if err != nil {
		http.Error(w, "Failed to get machine ID", http.StatusInternalServerError)
		return
	}

	isValid := req.MachineID == id

	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":     isValid,
		"machineId": id,
	})
}

func (s *Server) handleGetIdentifier(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling Identifier Request")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := config.GetMachineID()
	if err != nil {
		fmt.Printf("Error getting machine ID: %v\n", err)
		http.Error(w, "Failed to get machine ID", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"identifier": id,
	})
}

func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type NotificationRequest struct {
		Title   string `json:"title"`
		Message string `json:"message"`
		Icon    string `json:"icon"`
		Sound   bool   `json:"sound"`
	}

	var req NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = NotificationRequest{
			Title:   "Test Notification",
			Message: "This is a test notification from the backend.",
		}
	}

	if req.Title == "" {
		req.Title = "Test Notification"
	}
	if req.Message == "" {
		req.Message = "This is a test notification from the backend."
	}

	s.notifyError(req.Title, req.Message, req.Icon, req.Sound)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Notification sent",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// 1. Machine ID Check
	mid, err := config.GetMachineID()
	midStatus := "OK"
	if err != nil {
		midStatus = fmt.Sprintf("Error: %v", err)
		mid = "UNKNOWN"
	}

	// 2. Printer Count Check
	s.printersMux.RLock()
	printerCount := len(s.printers)
	printerList := make([]string, 0, len(s.printers))
	for name := range s.printers {
		printerList = append(printerList, name)
	}
	s.printersMux.RUnlock()

	// Intensive Check: Verify if we can actually construct a printer adapter
	adapterStatus := "OK"
	// Just a dummy check if we can instantiate it
	_ = printer.NewEscposAdapter()

	response := map[string]interface{}{
		"status":         "UP",
		"timestamp":      time.Now(),
		"machine_id":     mid,
		"machine_status": midStatus,
		"printers_count": printerCount,
		"printers_list":  printerList,
		"adapter_status": adapterStatus,
		"user_agent":     r.UserAgent(),
		"remote_addr":    r.RemoteAddr,
		"cors_access":    "enabled",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
