package pulse

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// createCheckRequest is the JSON body for POST /checks.
type createCheckRequest struct {
	DeviceID        string `json:"device_id"`
	CheckType       string `json:"check_type"`
	Target          string `json:"target"`
	IntervalSeconds int    `json:"interval_seconds"`
}

// updateCheckRequest is the JSON body for PUT /checks/{id}.
type updateCheckRequest struct {
	Target          string `json:"target,omitempty"`
	CheckType       string `json:"check_type,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/checks", Handler: m.handleListChecks},
		{Method: "POST", Path: "/checks", Handler: m.handleCreateCheck},
		{Method: "GET", Path: "/checks/{device_id}", Handler: m.handleDeviceChecks},
		{Method: "PUT", Path: "/checks/{id}", Handler: m.handleUpdateCheck},
		{Method: "DELETE", Path: "/checks/{id}", Handler: m.handleDeleteCheck},
		{Method: "PATCH", Path: "/checks/{id}/toggle", Handler: m.handleToggleCheck},
		{Method: "GET", Path: "/results/{device_id}", Handler: m.handleDeviceResults},
		{Method: "GET", Path: "/alerts", Handler: m.handleListAlerts},
		{Method: "GET", Path: "/alerts/{id}", Handler: m.handleGetAlert},
		{Method: "POST", Path: "/alerts/{id}/acknowledge", Handler: m.handleAcknowledgeAlert},
		{Method: "POST", Path: "/alerts/{id}/resolve", Handler: m.handleResolveAlert},
		{Method: "GET", Path: "/status/{device_id}", Handler: m.handleDeviceStatus},
	}
}

// handleListChecks returns all registered monitoring checks.
//
//	@Summary		List checks
//	@Description	Returns all monitoring checks (enabled and disabled).
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200 {array} Check
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks [get]
func (m *Module) handleListChecks(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	checks, err := m.store.ListAllChecks(r.Context())
	if err != nil {
		m.logger.Warn("failed to list checks", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list checks")
		return
	}
	if checks == nil {
		checks = []Check{}
	}
	pulseWriteJSON(w, http.StatusOK, checks)
}

// handleCreateCheck creates a new monitoring check.
//
//	@Summary		Create check
//	@Description	Creates a new monitoring check for a device.
//	@Tags			pulse
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body body createCheckRequest true "Check definition"
//	@Success		201 {object} Check
//	@Failure		400 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks [post]
func (m *Module) handleCreateCheck(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	var req createCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pulseWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.DeviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	// Validate check_type.
	switch req.CheckType {
	case "icmp", "tcp", "http":
		// valid
	default:
		pulseWriteError(w, http.StatusBadRequest, "check_type must be icmp, tcp, or http")
		return
	}

	// Validate target based on check type.
	if req.Target == "" {
		pulseWriteError(w, http.StatusBadRequest, "target is required")
		return
	}
	if err := validateTarget(req.CheckType, req.Target); err != nil {
		pulseWriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = 30
	}

	now := time.Now().UTC()
	check := &Check{
		ID:              fmt.Sprintf("pulse-%s-%s-%d", req.DeviceID, req.CheckType, now.UnixMilli()),
		DeviceID:        req.DeviceID,
		CheckType:       req.CheckType,
		Target:          req.Target,
		IntervalSeconds: req.IntervalSeconds,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := m.store.InsertCheck(r.Context(), check); err != nil {
		m.logger.Warn("failed to create check", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to create check")
		return
	}

	pulseWriteJSON(w, http.StatusCreated, check)
}

// handleUpdateCheck updates an existing monitoring check.
//
//	@Summary		Update check
//	@Description	Updates fields on an existing monitoring check.
//	@Tags			pulse
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Check ID"
//	@Param			body body updateCheckRequest true "Fields to update"
//	@Success		200 {object} Check
//	@Failure		400 {object} map[string]any
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{id} [put]
func (m *Module) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	existing, err := m.store.GetCheck(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get check for update", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get check")
		return
	}
	if existing == nil {
		pulseWriteError(w, http.StatusNotFound, "check not found")
		return
	}

	var req updateCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pulseWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.CheckType != "" {
		switch req.CheckType {
		case "icmp", "tcp", "http":
			existing.CheckType = req.CheckType
		default:
			pulseWriteError(w, http.StatusBadRequest, "check_type must be icmp, tcp, or http")
			return
		}
	}
	if req.Target != "" {
		if err := validateTarget(existing.CheckType, req.Target); err != nil {
			pulseWriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		existing.Target = req.Target
	}
	if req.IntervalSeconds > 0 {
		existing.IntervalSeconds = req.IntervalSeconds
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := m.store.UpdateCheck(r.Context(), existing); err != nil {
		m.logger.Warn("failed to update check", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to update check")
		return
	}

	pulseWriteJSON(w, http.StatusOK, existing)
}

// handleDeleteCheck deletes a monitoring check and its results.
//
//	@Summary		Delete check
//	@Description	Deletes a monitoring check and cascade-deletes its results.
//	@Tags			pulse
//	@Security		BearerAuth
//	@Param			id path string true "Check ID"
//	@Success		204
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{id} [delete]
func (m *Module) handleDeleteCheck(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := m.store.DeleteCheck(r.Context(), id); err != nil {
		m.logger.Warn("failed to delete check", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to delete check")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleToggleCheck toggles the enabled state of a check.
//
//	@Summary		Toggle check
//	@Description	Toggles the enabled/disabled state of a monitoring check.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Check ID"
//	@Success		200 {object} Check
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{id}/toggle [patch]
func (m *Module) handleToggleCheck(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	existing, err := m.store.GetCheck(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get check for toggle", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get check")
		return
	}
	if existing == nil {
		pulseWriteError(w, http.StatusNotFound, "check not found")
		return
	}

	newEnabled := !existing.Enabled
	if err := m.store.UpdateCheckEnabled(r.Context(), id, newEnabled); err != nil {
		m.logger.Warn("failed to toggle check", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to toggle check")
		return
	}

	existing.Enabled = newEnabled
	pulseWriteJSON(w, http.StatusOK, existing)
}

// handleDeviceChecks returns checks for a specific device.
//
//	@Summary		Device checks
//	@Description	Returns monitoring checks for a specific device.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Success		200 {object} Check
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{device_id} [get]
func (m *Module) handleDeviceChecks(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	check, err := m.store.GetCheckByDeviceID(r.Context(), deviceID)
	if err != nil {
		m.logger.Warn("failed to get device check", zap.String("device_id", deviceID), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get check")
		return
	}
	if check == nil {
		pulseWriteError(w, http.StatusNotFound, "no check found for device")
		return
	}
	pulseWriteJSON(w, http.StatusOK, check)
}

// handleDeviceResults returns recent check results for a device.
//
//	@Summary		Device results
//	@Description	Returns recent check results for a specific device.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Param			limit query int false "Maximum results" default(100)
//	@Success		200 {array} CheckResult
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/results/{device_id} [get]
func (m *Module) handleDeviceResults(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	limit := pulseParseLimit(r, 100)
	results, err := m.store.ListResults(r.Context(), deviceID, limit)
	if err != nil {
		m.logger.Warn("failed to list results", zap.String("device_id", deviceID), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list results")
		return
	}
	if results == nil {
		results = []CheckResult{}
	}
	pulseWriteJSON(w, http.StatusOK, results)
}

// handleListAlerts returns alerts with optional filtering.
//
//	@Summary		List alerts
//	@Description	Returns monitoring alerts with optional filters.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id query string false "Filter by device ID"
//	@Param			severity query string false "Filter by severity (warning, critical)"
//	@Param			active query bool false "Only active (unresolved) alerts" default(true)
//	@Param			limit query int false "Maximum alerts" default(50)
//	@Success		200 {array} Alert
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts [get]
func (m *Module) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	filters := AlertFilters{
		DeviceID:   r.URL.Query().Get("device_id"),
		Severity:   r.URL.Query().Get("severity"),
		ActiveOnly: true,
		Limit:      pulseParseLimit(r, 50),
	}

	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		filters.ActiveOnly = activeStr != "false"
	}

	alerts, err := m.store.ListAlerts(r.Context(), filters)
	if err != nil {
		m.logger.Warn("failed to list alerts", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list alerts")
		return
	}
	if alerts == nil {
		alerts = []Alert{}
	}
	pulseWriteJSON(w, http.StatusOK, alerts)
}

// handleGetAlert returns a single alert by ID.
//
//	@Summary		Get alert
//	@Description	Returns a single monitoring alert by ID.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Alert ID"
//	@Success		200 {object} Alert
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts/{id} [get]
func (m *Module) handleGetAlert(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	alert, err := m.store.GetAlert(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get alert", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get alert")
		return
	}
	if alert == nil {
		pulseWriteError(w, http.StatusNotFound, "alert not found")
		return
	}

	pulseWriteJSON(w, http.StatusOK, alert)
}

// handleAcknowledgeAlert acknowledges an alert.
//
//	@Summary		Acknowledge alert
//	@Description	Marks an alert as acknowledged.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Alert ID"
//	@Success		200 {object} Alert
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts/{id}/acknowledge [post]
func (m *Module) handleAcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := m.store.AcknowledgeAlert(r.Context(), id); err != nil {
		m.logger.Warn("failed to acknowledge alert", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to acknowledge alert")
		return
	}

	alert, err := m.store.GetAlert(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get alert after acknowledge", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get alert")
		return
	}
	if alert == nil {
		pulseWriteError(w, http.StatusNotFound, "alert not found")
		return
	}

	pulseWriteJSON(w, http.StatusOK, alert)
}

// handleResolveAlert resolves an alert.
//
//	@Summary		Resolve alert
//	@Description	Marks an alert as resolved.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Alert ID"
//	@Success		200 {object} Alert
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts/{id}/resolve [post]
func (m *Module) handleResolveAlert(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	now := time.Now().UTC()
	if err := m.store.ResolveAlert(r.Context(), id, now); err != nil {
		m.logger.Warn("failed to resolve alert", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to resolve alert")
		return
	}

	alert, err := m.store.GetAlert(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get alert after resolve", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get alert")
		return
	}
	if alert == nil {
		pulseWriteError(w, http.StatusNotFound, "alert not found")
		return
	}

	pulseWriteJSON(w, http.StatusOK, alert)
}

// handleDeviceStatus returns composite monitoring status for a device.
//
//	@Summary		Monitoring status
//	@Description	Returns composite health status for a specific device, including latest check result and active alerts.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Success		200 {object} github_com_HerbHall_subnetree_pkg_roles.MonitorStatus
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/status/{device_id} [get]
func (m *Module) handleDeviceStatus(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	status, err := m.Status(r.Context(), deviceID)
	if err != nil {
		m.logger.Warn("failed to get device status", zap.String("device_id", deviceID), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get status")
		return
	}
	pulseWriteJSON(w, http.StatusOK, status)
}

// -- helpers --

func pulseWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func pulseWriteError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

func pulseParseLimit(r *http.Request, defaultLimit int) int {
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 1000 {
			return n
		}
	}
	return defaultLimit
}

// validateTarget validates a check target based on the check type.
func validateTarget(checkType, target string) error {
	switch checkType {
	case "icmp":
		if net.ParseIP(target) == nil {
			// Not an IP -- check it's a non-empty hostname.
			if strings.TrimSpace(target) == "" {
				return fmt.Errorf("icmp target must be a valid IP or hostname")
			}
		}
	case "tcp":
		if _, _, err := net.SplitHostPort(target); err != nil {
			return fmt.Errorf("tcp target must be host:port format")
		}
	case "http":
		u, err := url.Parse(target)
		if err != nil {
			return fmt.Errorf("http target must be a valid URL")
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("http target must have http or https scheme")
		}
	}
	return nil
}
