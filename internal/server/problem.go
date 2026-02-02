package server

import (
	"encoding/json"
	"net/http"
)

// Problem types for RFC 7807 Problem Details responses.
const (
	ProblemTypeNotFound     = "https://netvantage.io/problems/not-found"
	ProblemTypeBadRequest   = "https://netvantage.io/problems/bad-request"
	ProblemTypeInternal     = "https://netvantage.io/problems/internal-error"
	ProblemTypeUnauthorized = "https://netvantage.io/problems/unauthorized"
	ProblemTypeForbidden    = "https://netvantage.io/problems/forbidden"
	ProblemTypeRateLimited  = "https://netvantage.io/problems/rate-limited"
	ProblemTypeConflict     = "https://netvantage.io/problems/conflict"
)

// Problem represents an RFC 7807 Problem Details response.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// WriteProblem writes an RFC 7807 Problem Details JSON response.
func WriteProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// NotFound writes a 404 problem response.
func NotFound(w http.ResponseWriter, detail, instance string) {
	WriteProblem(w, Problem{
		Type:     ProblemTypeNotFound,
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   detail,
		Instance: instance,
	})
}

// BadRequest writes a 400 problem response.
func BadRequest(w http.ResponseWriter, detail, instance string) {
	WriteProblem(w, Problem{
		Type:     ProblemTypeBadRequest,
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Detail:   detail,
		Instance: instance,
	})
}

// InternalError writes a 500 problem response.
func InternalError(w http.ResponseWriter, detail, instance string) {
	WriteProblem(w, Problem{
		Type:     ProblemTypeInternal,
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   detail,
		Instance: instance,
	})
}

// RateLimited writes a 429 problem response.
func RateLimited(w http.ResponseWriter, detail, instance string) {
	WriteProblem(w, Problem{
		Type:     ProblemTypeRateLimited,
		Title:    "Too Many Requests",
		Status:   http.StatusTooManyRequests,
		Detail:   detail,
		Instance: instance,
	})
}
