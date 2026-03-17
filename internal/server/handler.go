package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/hra4h03/snowflake-uuid/snowflake"
)

// Handler holds the dependencies for HTTP handlers.
type Handler struct {
	gen       *snowflake.Generator
	epoch     time.Time
	startTime time.Time
}

// NewHandler creates a Handler wrapping the given generator.
func NewHandler(gen *snowflake.Generator, epoch time.Time) *Handler {
	return &Handler{
		gen:       gen,
		epoch:     epoch,
		startTime: time.Now(),
	}
}

// RegisterRoutes adds all endpoint handlers to the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /id", h.handleGenerateID)
	mux.HandleFunc("GET /id/batch", h.handleBatchIDs)
	mux.HandleFunc("GET /id/parse", h.handleParseID)
	mux.HandleFunc("GET /health", h.handleHealth)
}

func (h *Handler) handleGenerateID(w http.ResponseWriter, r *http.Request) {
	id, err := h.gen.Generate()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id.String()})
}

func (h *Handler) handleBatchIDs(w http.ResponseWriter, r *http.Request) {
	countStr := r.URL.Query().Get("count")
	if countStr == "" {
		countStr = "1"
	}

	count, err := strconv.Atoi(countStr)
	if err != nil || count < 1 || count > 1000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "count must be between 1 and 1000",
		})
		return
	}

	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		id, err := h.gen.Generate()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		ids = append(ids, id.String())
	}

	writeJSON(w, http.StatusOK, map[string][]string{"ids": ids})
}

func (h *Handler) handleParseID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id parameter is required"})
		return
	}

	v, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	id := snowflake.ID(v)
	generatedAt := id.Time(h.epoch)

	writeJSON(w, http.StatusOK, map[string]any{
		"id":           id.String(),
		"timestamp_ms": id.Timestamp(),
		"node_id":      id.NodeID(),
		"sequence":     id.Sequence(),
		"generated_at": generatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"node_id":        h.gen.NodeID(),
		"uptime_seconds": int(time.Since(h.startTime).Seconds()),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
