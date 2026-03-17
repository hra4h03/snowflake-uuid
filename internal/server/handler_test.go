package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hra4h03/snowflake-uuid/snowflake"
)

func setupTestHandler(t *testing.T) (*http.ServeMux, *snowflake.Generator) {
	t.Helper()
	gen, err := snowflake.New(5)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h := NewHandler(gen, snowflake.DefaultEpoch)
	h.RegisterRoutes(mux)
	return mux, gen
}

func TestHandleGenerateID(t *testing.T) {
	mux, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/id", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp["id"] == "" {
		t.Error("expected non-empty id")
	}
}

func TestHandleBatchIDs(t *testing.T) {
	mux, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/id/batch?count=5", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string][]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(resp["ids"]) != 5 {
		t.Errorf("got %d ids, want 5", len(resp["ids"]))
	}

	// Verify uniqueness
	seen := make(map[string]bool)
	for _, id := range resp["ids"] {
		if seen[id] {
			t.Errorf("duplicate id in batch: %s", id)
		}
		seen[id] = true
	}
}

func TestHandleBatchIDsInvalidCount(t *testing.T) {
	mux, _ := setupTestHandler(t)

	tests := []struct {
		name  string
		query string
	}{
		{"zero", "count=0"},
		{"negative", "count=-1"},
		{"too_large", "count=2000"},
		{"non_numeric", "count=abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/id/batch?"+tt.query, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleParseID(t *testing.T) {
	mux, gen := setupTestHandler(t)

	// First generate an ID
	id, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/id/parse?id="+id.String(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp["id"] != id.String() {
		t.Errorf("parsed id = %v, want %s", resp["id"], id.String())
	}

	// node_id should be 5 (what we set up)
	if int64(resp["node_id"].(float64)) != 5 {
		t.Errorf("node_id = %v, want 5", resp["node_id"])
	}
}

func TestHandleParseIDMissing(t *testing.T) {
	mux, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/id/parse", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleParseIDInvalid(t *testing.T) {
	mux, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/id/parse?id=not-a-number", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleHealth(t *testing.T) {
	mux, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
	if int64(resp["node_id"].(float64)) != 5 {
		t.Errorf("node_id = %v, want 5", resp["node_id"])
	}
}

func TestHandleBatchDefaultCount(t *testing.T) {
	mux, _ := setupTestHandler(t)

	// No count param — should default to 1
	req := httptest.NewRequest(http.MethodGet, "/id/batch", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string][]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(resp["ids"]) != 1 {
		t.Errorf("default batch should return 1 id, got %d", len(resp["ids"]))
	}
}
