package ollama

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/HerbHall/subnetree/pkg/llm"
)

// ollamaStatusError represents an HTTP error response from the Ollama API.
type ollamaStatusError struct {
	StatusCode int
	Message    string
}

func (e *ollamaStatusError) Error() string {
	return fmt.Sprintf("ollama: %d %s", e.StatusCode, e.Message)
}

// mapError translates Ollama and network errors into typed llm.ProviderError values.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	// Context errors.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return llm.NewProviderError(llm.ErrCodeTimeout, "request timed out or cancelled", err)
	}

	// Ollama HTTP error responses.
	var se *ollamaStatusError
	if errors.As(err, &se) {
		switch {
		case se.StatusCode == 401:
			return llm.NewProviderError(llm.ErrCodeAuthentication, se.Message, err)
		case se.StatusCode == 404 && strings.Contains(strings.ToLower(se.Message), "model"):
			return llm.NewProviderError(llm.ErrCodeModelNotFound, se.Message, err)
		case se.StatusCode >= 500:
			return llm.NewProviderError(llm.ErrCodeServerError, se.Message, err)
		case se.StatusCode >= 400:
			return llm.NewProviderError(llm.ErrCodeInvalidRequest, se.Message, err)
		}
	}

	// Connection refused, DNS errors, etc.
	msg := err.Error()
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "dial tcp") {
		return llm.NewProviderError(llm.ErrCodeServerError, "ollama server unreachable", err)
	}

	return llm.NewProviderError(llm.ErrCodeServerError, "ollama error", err)
}
