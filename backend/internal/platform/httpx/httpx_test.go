package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, map[string]string{"code": "abc"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["code"] != "abc" {
		t.Fatalf("body = %v", body)
	}
}

func TestWriteError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusBadRequest, "invalid json")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "invalid json" {
		t.Fatalf("body = %v", body)
	}
}
