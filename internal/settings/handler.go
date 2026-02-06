// Package settings provides HTTP handlers for application settings endpoints.
package settings

import (
	"encoding/json"
	"net/http"

	"github.com/HerbHall/subnetree/internal/services"
	"go.uber.org/zap"
)

// ScanInterfaceRequest represents a request to set the scan interface.
// @Description Request body for setting the network scan interface.
type ScanInterfaceRequest struct {
	InterfaceName string `json:"interface_name" example:"eth0"`
}

// ScanInterfaceResponse represents the current scan interface setting.
// @Description Response containing the configured scan interface.
type ScanInterfaceResponse struct {
	InterfaceName string `json:"interface_name" example:"eth0"`
}

// SettingsProblemDetail represents an RFC 7807 error response for settings endpoints.
// @Description RFC 7807 Problem Details error response.
type SettingsProblemDetail struct {
	Type   string `json:"type" example:"https://subnetree.com/problems/settings-error"`
	Title  string `json:"title" example:"Bad Request"`
	Status int    `json:"status" example:"400"`
	Detail string `json:"detail" example:"interface not found: eth99"`
}

// Handler provides HTTP handlers for settings endpoints.
type Handler struct {
	interfaces *services.InterfaceService
	settings   services.SettingsRepository
	logger     *zap.Logger
}

// NewHandler creates a settings Handler.
func NewHandler(settings services.SettingsRepository, logger *zap.Logger) *Handler {
	return &Handler{
		interfaces: services.NewInterfaceService(),
		settings:   settings,
		logger:     logger,
	}
}

// RegisterRoutes registers settings-related routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Network interface endpoints (public during setup)
	mux.HandleFunc("GET /api/v1/settings/interfaces", h.handleListInterfaces)
	mux.HandleFunc("GET /api/v1/settings/scan-interface", h.handleGetScanInterface)
	mux.HandleFunc("POST /api/v1/settings/scan-interface", h.handleSetScanInterface)
}

// handleListInterfaces returns all available network interfaces.
//
//	@Summary		List network interfaces
//	@Description	Get a list of all network interfaces available on the server.
//	@Tags			settings
//	@Produce		json
//	@Success		200	{array}		services.NetworkInterface	"List of interfaces"
//	@Failure		500	{object}	SettingsProblemDetail		"Internal server error"
//	@Router			/settings/interfaces [get]
func (h *Handler) handleListInterfaces(w http.ResponseWriter, _ *http.Request) {
	interfaces, err := h.interfaces.ListNetworkInterfaces()
	if err != nil {
		h.logger.Error("failed to list interfaces", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to list network interfaces")
		return
	}

	writeJSON(w, http.StatusOK, interfaces)
}

// handleGetScanInterface returns the currently configured scan interface.
//
//	@Summary		Get scan interface
//	@Description	Get the currently configured network interface for scanning.
//	@Tags			settings
//	@Produce		json
//	@Success		200	{object}	ScanInterfaceResponse	"Current scan interface"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/scan-interface [get]
func (h *Handler) handleGetScanInterface(w http.ResponseWriter, r *http.Request) {
	setting, err := h.settings.Get(r.Context(), "scan_interface")
	if err != nil {
		if err == services.ErrNotFound {
			// No interface configured yet -- return empty response
			writeJSON(w, http.StatusOK, map[string]string{"interface_name": ""})
			return
		}
		h.logger.Error("failed to get scan interface setting", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get scan interface")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"interface_name": setting.Value})
}

// handleSetScanInterface saves the selected scan interface.
//
//	@Summary		Set scan interface
//	@Description	Configure which network interface to use for scanning.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ScanInterfaceRequest	true	"Interface to use"
//	@Success		200		{object}	ScanInterfaceResponse	"Interface configured"
//	@Failure		400		{object}	SettingsProblemDetail	"Invalid request or interface not found"
//	@Failure		500		{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/scan-interface [post]
func (h *Handler) handleSetScanInterface(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InterfaceName string `json:"interface_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSettingsError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate that the interface exists (if non-empty)
	if req.InterfaceName != "" {
		interfaces, err := h.interfaces.ListNetworkInterfaces()
		if err != nil {
			h.logger.Error("failed to list interfaces for validation", zap.Error(err))
			writeSettingsError(w, http.StatusInternalServerError, "failed to validate interface")
			return
		}
		found := false
		for i := range interfaces {
			if interfaces[i].Name == req.InterfaceName {
				found = true
				break
			}
		}
		if !found {
			writeSettingsError(w, http.StatusBadRequest, "interface not found: "+req.InterfaceName)
			return
		}
	}

	if err := h.settings.Set(r.Context(), "scan_interface", req.InterfaceName); err != nil {
		h.logger.Error("failed to set scan interface", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to save scan interface")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"interface_name": req.InterfaceName})
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeSettingsError writes an RFC 7807 problem response.
func writeSettingsError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/settings-error",
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
