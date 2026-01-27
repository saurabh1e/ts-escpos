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
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"ts-escpos/backend/config"
	"ts-escpos/backend/jobs"
	"ts-escpos/backend/printer"
	"ts-escpos/backend/receipt"
)

type Server struct {
	store      *jobs.Store
	config     *config.Config
	ctx        context.Context
	clients    map[*websocket.Conn]bool
	clientsMux sync.Mutex
	upgrader   websocket.Upgrader
}

func NewServer(store *jobs.Store, cfg *config.Config) *Server {
	return &Server{
		store:   store,
		config:  cfg,
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all CORS for now to support development from various origins
				// In production, check r.Header.Get("Origin") against cfg.AllowedCors
				return true
			},
		},
	}
}

func (s *Server) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	fmt.Println("Registering routes...")
	// Register identifier first to test priority
	mux.HandleFunc("/api/identifier", s.handleGetIdentifier)
	mux.HandleFunc("/api/identifier/", s.handleGetIdentifier)

	mux.HandleFunc("/api/print", s.handlePrint)
	mux.HandleFunc("/api/printers", s.handleGetPrinters)
	mux.HandleFunc("/api/validate", s.handleValidate)
	mux.HandleFunc("/api/test-notification", s.handleTestNotification)
	mux.HandleFunc("/ws", s.handleWebSocket)

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
		fmt.Printf("Incoming request: %s %s\n", r.Method, r.URL.Path)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			fmt.Printf("Request: %s %s | Status: %d | Duration: %v\n", r.Method, r.URL.Path, http.StatusOK, time.Since(start))
			return
		}

		lrw := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(lrw, r)

		code := lrw.statusCode
		if code == 0 {
			code = http.StatusOK
		}

		fmt.Printf("Request: %s %s | Status: %d | Duration: %v\n", r.Method, r.URL.Path, code, time.Since(start))
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
	// 1. Notify Wails Frontend
	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "error_notification", map[string]string{
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

	// 1. Unique ID Validation
	storedMachineID, err := config.GetMachineID()
	if err == nil && req.MachineID != storedMachineID {
		fmt.Printf("Print request validation failed: Invalid Machine ID\n")
		s.notifyError("Validation Failed", "Unique ID validation failed.", "", false)
		http.Error(w, "Invalid Machine ID", http.StatusUnauthorized)
		return
	}

	// Initialize job for tracking
	jobID := uuid.New().String()
	job := jobs.PrintJob{
		ID:          jobID,
		InvoiceNo:   req.OrderData.InvoiceNo,
		PrinterName: req.PrinterName,
		ReceiptType: req.ReceiptType,
		Timestamp:   time.Now(),
		Status:      jobs.StatusFailed,
	}

	// 2. Printer Validation
	printers, _ := printer.GetPrinters()
	var selectedPrinter printer.PrinterInfo
	found := false
	for _, p := range printers {
		if p.Name == req.PrinterName {
			selectedPrinter = p
			found = true
			break
		}
	}

	if !found {
		msg := fmt.Sprintf("Printer '%s' not found or installed.", req.PrinterName)
		fmt.Printf("Print failed: %s\n", msg)
		s.notifyError("Printer Not Found", msg, "", true)

		job.Error = msg
		s.store.AddJob(job)

		_ = json.NewEncoder(w).Encode(PrintResponse{
			Success: false,
			Error:   msg,
		})
		return
	}

	// Check for blocking statuses
	statusLower := strings.ToLower(selectedPrinter.Status)
	blockingStatuses := []string{"offline", "error", "paper jam", "paper out", "door open", "not available", "no toner", "out of memory"}
	for _, bs := range blockingStatuses {
		if strings.Contains(statusLower, bs) {
			msg := fmt.Sprintf("Printer '%s' is %s. Please check the device.", req.PrinterName, selectedPrinter.Status)
			fmt.Printf("Print failed: %s\n", msg)
			s.notifyError("Printer Error", msg, "", true)

			job.Error = msg
			s.store.AddJob(job)

			_ = json.NewEncoder(w).Encode(PrintResponse{
				Success: false,
				Error:   msg,
			})
			return
		}
	}

	// 3. Generate ESC/POS bytes
	adapter := printer.NewEscposAdapter()
	if req.ReceiptType == "kot" {
		receipt.RenderKOT(adapter, req.OrderData, req.PrinterSize)
	} else {
		receipt.RenderBill(adapter, req.OrderData, req.PrinterSize)
	}

	bytesToPrint := adapter.GetBytes()

	// 4. Send to printer
	err = printer.PrintRaw(req.PrinterName, bytesToPrint)

	resp := PrintResponse{
		Success: true,
		JobID:   jobID,
		Message: "Printed successfully",
	}

	job.Status = jobs.StatusSuccess

	if err != nil {
		fmt.Printf("Print failed: %v\n", err)
		job.Status = jobs.StatusFailed
		job.Error = err.Error()
		resp.Success = false
		resp.Message = ""
		resp.Error = err.Error()

		s.notifyError("Print Failed", fmt.Sprintf("Unable to print to '%s'. Device might be unresponsive.", req.PrinterName), "", true)
	} else {
		fmt.Printf("Print success: Job %s sent to %s\n", jobID, req.PrinterName)
		w.WriteHeader(http.StatusOK)
	}

	s.store.AddJob(job)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGetPrinters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	printers, err := printer.GetPrinters()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get printers: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"printers": printers,
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
