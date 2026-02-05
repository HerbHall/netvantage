package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteProblem(t *testing.T) {
	w := httptest.NewRecorder()

	WriteProblem(w, Problem{
		Type:     ProblemTypeNotFound,
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   "device xyz not found",
		Instance: "/api/v1/devices/xyz",
	})

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want %q", ct, "application/problem+json")
	}

	var p Problem
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if p.Type != ProblemTypeNotFound {
		t.Errorf("type = %q, want %q", p.Type, ProblemTypeNotFound)
	}
	if p.Title != "Not Found" {
		t.Errorf("title = %q, want %q", p.Title, "Not Found")
	}
	if p.Status != 404 {
		t.Errorf("status = %d, want 404", p.Status)
	}
	if p.Detail != "device xyz not found" {
		t.Errorf("detail = %q, want %q", p.Detail, "device xyz not found")
	}
	if p.Instance != "/api/v1/devices/xyz" {
		t.Errorf("instance = %q, want %q", p.Instance, "/api/v1/devices/xyz")
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	NotFound(w, "missing", "/test")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var p Problem
	json.NewDecoder(w.Body).Decode(&p)
	if p.Type != ProblemTypeNotFound {
		t.Errorf("type = %q, want %q", p.Type, ProblemTypeNotFound)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	BadRequest(w, "invalid input", "/test")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var p Problem
	json.NewDecoder(w.Body).Decode(&p)
	if p.Type != ProblemTypeBadRequest {
		t.Errorf("type = %q, want %q", p.Type, ProblemTypeBadRequest)
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	InternalError(w, "something broke", "/test")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var p Problem
	json.NewDecoder(w.Body).Decode(&p)
	if p.Type != ProblemTypeInternal {
		t.Errorf("type = %q, want %q", p.Type, ProblemTypeInternal)
	}
}

func TestRateLimited(t *testing.T) {
	w := httptest.NewRecorder()
	RateLimited(w, "slow down", "/test")

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	var p Problem
	json.NewDecoder(w.Body).Decode(&p)
	if p.Type != ProblemTypeRateLimited {
		t.Errorf("type = %q, want %q", p.Type, ProblemTypeRateLimited)
	}
}

func TestWriteProblem_OmitsEmptyOptionalFields(t *testing.T) {
	w := httptest.NewRecorder()

	WriteProblem(w, Problem{
		Type:   ProblemTypeInternal,
		Title:  "Internal Server Error",
		Status: 500,
	})

	var raw map[string]interface{}
	json.NewDecoder(w.Body).Decode(&raw)

	if _, ok := raw["detail"]; ok {
		t.Error("expected detail to be omitted when empty")
	}
	if _, ok := raw["instance"]; ok {
		t.Error("expected instance to be omitted when empty")
	}
}
