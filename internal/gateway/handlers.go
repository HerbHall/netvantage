package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/sessions", Handler: m.handleListSessions},
		{Method: "GET", Path: "/sessions/{id}", Handler: m.handleGetSession},
		{Method: "DELETE", Path: "/sessions/{id}", Handler: m.handleDeleteSession},
		{Method: "GET", Path: "/status", Handler: m.handleStatus},
		{Method: "GET", Path: "/audit", Handler: m.handleListAudit},
	}
}

// handleListSessions returns all active sessions.
func (m *Module) handleListSessions(w http.ResponseWriter, _ *http.Request) {
	if m.sessions == nil {
		gatewayWriteJSON(w, http.StatusOK, []any{})
		return
	}

	sessions := m.sessions.List()
	views := make([]sessionView, len(sessions))
	for i, s := range sessions {
		views[i] = s.toView()
	}
	gatewayWriteJSON(w, http.StatusOK, views)
}

// handleGetSession returns a single session by ID.
func (m *Module) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		gatewayWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if m.sessions == nil {
		gatewayWriteError(w, http.StatusNotFound, "session not found")
		return
	}

	session, ok := m.sessions.Get(id)
	if !ok {
		gatewayWriteError(w, http.StatusNotFound, "session not found")
		return
	}

	gatewayWriteJSON(w, http.StatusOK, session.toView())
}

// handleDeleteSession closes and removes a session.
func (m *Module) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		gatewayWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if m.sessions == nil {
		gatewayWriteError(w, http.StatusNotFound, "session not found")
		return
	}

	session, ok := m.sessions.Get(id)
	if !ok {
		gatewayWriteError(w, http.StatusNotFound, "session not found")
		return
	}

	m.sessions.Delete(id)
	m.logSessionClosed(session, "manual")

	w.WriteHeader(http.StatusNoContent)
}

// handleStatus returns gateway status including session count and capacity.
func (m *Module) handleStatus(w http.ResponseWriter, _ *http.Request) {
	sessionCount := 0
	if m.sessions != nil {
		sessionCount = m.sessions.Count()
	}

	storeStatus := "unavailable"
	if m.store != nil {
		storeStatus = "connected"
	}

	gatewayWriteJSON(w, http.StatusOK, map[string]any{
		"active_sessions": sessionCount,
		"max_sessions":    m.cfg.MaxSessions,
		"store":           storeStatus,
	})
}

// handleListAudit returns audit log entries with optional device filtering.
func (m *Module) handleListAudit(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		gatewayWriteError(w, http.StatusServiceUnavailable, "gateway store not available")
		return
	}

	deviceID := r.URL.Query().Get("device_id")
	limit := gatewayParseLimit(r, 100)

	entries, err := m.store.ListAuditEntries(r.Context(), deviceID, limit)
	if err != nil {
		m.logger.Warn("failed to list gateway audit entries", zap.Error(err))
		gatewayWriteError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	gatewayWriteJSON(w, http.StatusOK, entries)
}

// --- Helpers ---

// gatewayWriteJSON writes a JSON response with the given status code.
func gatewayWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// gatewayWriteError writes a problem+json error response.
func gatewayWriteError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   fmt.Sprintf("https://subnetree.com/problems/%s", http.StatusText(status)),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

// gatewayParseLimit extracts a limit query parameter with a default value.
func gatewayParseLimit(r *http.Request, defaultLimit int) int {
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 1000 {
			return n
		}
	}
	return defaultLimit
}
