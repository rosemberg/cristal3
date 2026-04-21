package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bergmaia/cristal-backend/internal/orchestrator"
)

// Handler handles HTTP requests
type Handler struct {
	orchestrator *orchestrator.Orchestrator
	logger       *slog.Logger
}

// NewHandler creates a new handler
func NewHandler(orch *orchestrator.Orchestrator, logger *slog.Logger) *Handler {
	return &Handler{
		orchestrator: orch,
		logger:       logger,
	}
}

// HandleChat handles POST /chat requests
func (h *Handler) HandleChat(w http.ResponseWriter, r *http.Request) {
	// Only accept POST
	if r.Method != http.MethodPost {
		h.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode request
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request JSON", "error", err)
		h.sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate message
	if req.Message == "" {
		h.sendError(w, "message is required", http.StatusBadRequest)
		return
	}

	h.logger.Info("received chat request", "message_len", len(req.Message))

	// Process query
	result, err := h.orchestrator.ProcessQuery(r.Context(), req.Message)
	if err != nil {
		h.logger.Error("query processing failed", "error", err)
		h.sendError(w, fmt.Sprintf("processing error: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert orchestrator citations to server citations
	citations := make([]Citation, len(result.Citations))
	for i, c := range result.Citations {
		citations[i] = Citation{
			ID:         c.ID,
			Title:      c.Title,
			Breadcrumb: c.Breadcrumb,
			URL:        c.URL,
		}
	}

	// Send success response (frontend-compatible format)
	h.sendJSON(w, ChatResponse{
		Response:  result.Response,
		Citations: citations,
	}, http.StatusOK)
}

// HandleHealth handles GET /health requests
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.sendJSON(w, HealthResponse{Status: "ok"}, http.StatusOK)
}

// sendJSON sends a JSON response
func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON", "error", err)
	}
}

// sendError sends an error response
func (h *Handler) sendError(w http.ResponseWriter, message string, statusCode int) {
	h.sendJSON(w, ChatResponse{
		Status: "error",
		Error:  message,
	}, statusCode)
}
