// Package settings provides HTTP handlers for application settings endpoints.
package settings

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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

// ThemeTokens represents customizable CSS token overrides organized by category.
// @Description Customizable CSS design token overrides grouped by UI category.
type ThemeTokens struct {
	Backgrounds map[string]string `json:"backgrounds,omitempty"`
	Text        map[string]string `json:"text,omitempty"`
	Borders     map[string]string `json:"borders,omitempty"`
	Buttons     map[string]string `json:"buttons,omitempty"`
	Inputs      map[string]string `json:"inputs,omitempty"`
	Sidebar     map[string]string `json:"sidebar,omitempty"`
	Status      map[string]string `json:"status,omitempty"`
	Charts      map[string]string `json:"charts,omitempty"`
	Typography  map[string]string `json:"typography,omitempty"`
	Spacing     map[string]string `json:"spacing,omitempty"`
	Effects     map[string]string `json:"effects,omitempty"`
}

// ThemeDefinition represents a complete theme configuration.
// @Description A complete theme with metadata and CSS token overrides.
type ThemeDefinition struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	BaseMode    string      `json:"base_mode"`
	Version     int         `json:"version"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
	BuiltIn     bool        `json:"built_in"`
	Tokens      ThemeTokens `json:"tokens"`
}

// ActiveThemeResponse represents the currently active theme reference.
// @Description Response containing the active theme ID.
type ActiveThemeResponse struct {
	ThemeID string `json:"theme_id" example:"builtin-forest-dark"`
}

// ActiveThemeRequest represents a request to set the active theme.
// @Description Request body for setting the active theme.
type ActiveThemeRequest struct {
	ThemeID string `json:"theme_id" example:"builtin-forest-dark"`
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

	// Theme endpoints (literal paths before wildcard)
	mux.HandleFunc("GET /api/v1/settings/themes", h.handleListThemes)
	mux.HandleFunc("GET /api/v1/settings/themes/active", h.handleGetActiveTheme)
	mux.HandleFunc("PUT /api/v1/settings/themes/active", h.handleSetActiveTheme)
	mux.HandleFunc("POST /api/v1/settings/themes", h.handleCreateTheme)
	mux.HandleFunc("GET /api/v1/settings/themes/{id}", h.handleGetTheme)
	mux.HandleFunc("PUT /api/v1/settings/themes/{id}", h.handleUpdateTheme)
	mux.HandleFunc("DELETE /api/v1/settings/themes/{id}", h.handleDeleteTheme)
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

// ---------- Theme endpoints ----------

const (
	themeKeyPrefix    = "theme:"
	themeActiveKey    = "theme:active"
	themeSeededKey    = "theme:builtin:seeded"
	defaultThemeID   = "builtin-forest-dark"
)

// handleListThemes returns all stored themes.
//
//	@Summary		List themes
//	@Description	Get all available themes (built-in and custom).
//	@Tags			settings
//	@Produce		json
//	@Success		200	{array}		ThemeDefinition			"List of themes"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes [get]
func (h *Handler) handleListThemes(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureBuiltInThemes(r.Context()); err != nil {
		h.logger.Error("failed to seed built-in themes", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to initialise themes")
		return
	}

	all, err := h.settings.GetAll(r.Context())
	if err != nil {
		h.logger.Error("failed to list settings for themes", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to list themes")
		return
	}

	themes := make([]ThemeDefinition, 0)
	for i := range all {
		key := all[i].Key
		if !strings.HasPrefix(key, themeKeyPrefix) {
			continue
		}
		if key == themeActiveKey || key == themeSeededKey {
			continue
		}
		var td ThemeDefinition
		if err := json.Unmarshal([]byte(all[i].Value), &td); err != nil {
			h.logger.Warn("skipping unparsable theme", zap.String("key", key), zap.Error(err))
			continue
		}
		themes = append(themes, td)
	}

	writeJSON(w, http.StatusOK, themes)
}

// handleGetTheme returns a single theme by ID.
//
//	@Summary		Get theme
//	@Description	Get a theme by its ID.
//	@Tags			settings
//	@Produce		json
//	@Param			id	path		string					true	"Theme ID"
//	@Success		200	{object}	ThemeDefinition			"Theme details"
//	@Failure		404	{object}	SettingsProblemDetail	"Theme not found"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/{id} [get]
func (h *Handler) handleGetTheme(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	setting, err := h.settings.Get(r.Context(), themeKeyPrefix+id)
	if err != nil {
		if err == services.ErrNotFound {
			writeSettingsError(w, http.StatusNotFound, "theme not found")
			return
		}
		h.logger.Error("failed to get theme", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get theme")
		return
	}

	var td ThemeDefinition
	if err := json.Unmarshal([]byte(setting.Value), &td); err != nil {
		h.logger.Error("failed to parse theme", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to parse theme")
		return
	}
	writeJSON(w, http.StatusOK, td)
}

// handleCreateTheme creates a new custom theme.
//
//	@Summary		Create theme
//	@Description	Create a new custom theme with CSS token overrides.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ThemeDefinition			true	"Theme definition (id, created_at, updated_at, version, built_in are ignored)"
//	@Success		201		{object}	ThemeDefinition			"Created theme"
//	@Failure		400		{object}	SettingsProblemDetail	"Validation error"
//	@Failure		500		{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes [post]
func (h *Handler) handleCreateTheme(w http.ResponseWriter, r *http.Request) {
	var req ThemeDefinition
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSettingsError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeSettingsError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.BaseMode != "dark" && req.BaseMode != "light" {
		writeSettingsError(w, http.StatusBadRequest, "base_mode must be \"dark\" or \"light\"")
		return
	}

	id, err := generateID()
	if err != nil {
		h.logger.Error("failed to generate theme ID", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to generate theme ID")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	td := ThemeDefinition{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		BaseMode:    req.BaseMode,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		BuiltIn:     false,
		Tokens:      req.Tokens,
	}

	data, err := json.Marshal(td)
	if err != nil {
		h.logger.Error("failed to marshal theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to store theme")
		return
	}

	if err := h.settings.Set(r.Context(), themeKeyPrefix+td.ID, string(data)); err != nil {
		h.logger.Error("failed to save theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to store theme")
		return
	}

	writeJSON(w, http.StatusCreated, td)
}

// handleUpdateTheme updates an existing custom theme.
//
//	@Summary		Update theme
//	@Description	Update a custom theme. Built-in themes cannot be modified.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Theme ID"
//	@Param			request	body		ThemeDefinition			true	"Fields to update"
//	@Success		200		{object}	ThemeDefinition			"Updated theme"
//	@Failure		403		{object}	SettingsProblemDetail	"Cannot modify built-in theme"
//	@Failure		404		{object}	SettingsProblemDetail	"Theme not found"
//	@Failure		500		{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/{id} [put]
func (h *Handler) handleUpdateTheme(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	setting, err := h.settings.Get(r.Context(), themeKeyPrefix+id)
	if err != nil {
		if err == services.ErrNotFound {
			writeSettingsError(w, http.StatusNotFound, "theme not found")
			return
		}
		h.logger.Error("failed to get theme for update", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get theme")
		return
	}

	var existing ThemeDefinition
	if err := json.Unmarshal([]byte(setting.Value), &existing); err != nil {
		h.logger.Error("failed to parse theme for update", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to parse theme")
		return
	}

	if existing.BuiltIn {
		writeSettingsError(w, http.StatusForbidden, "cannot modify built-in theme")
		return
	}

	var patch ThemeDefinition
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeSettingsError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Merge only provided fields.
	if patch.Name != "" {
		existing.Name = patch.Name
	}
	if patch.Description != "" {
		existing.Description = patch.Description
	}
	if patch.BaseMode != "" {
		if patch.BaseMode != "dark" && patch.BaseMode != "light" {
			writeSettingsError(w, http.StatusBadRequest, "base_mode must be \"dark\" or \"light\"")
			return
		}
		existing.BaseMode = patch.BaseMode
	}
	if patch.Tokens.Backgrounds != nil {
		existing.Tokens.Backgrounds = patch.Tokens.Backgrounds
	}
	if patch.Tokens.Text != nil {
		existing.Tokens.Text = patch.Tokens.Text
	}
	if patch.Tokens.Borders != nil {
		existing.Tokens.Borders = patch.Tokens.Borders
	}
	if patch.Tokens.Buttons != nil {
		existing.Tokens.Buttons = patch.Tokens.Buttons
	}
	if patch.Tokens.Inputs != nil {
		existing.Tokens.Inputs = patch.Tokens.Inputs
	}
	if patch.Tokens.Sidebar != nil {
		existing.Tokens.Sidebar = patch.Tokens.Sidebar
	}
	if patch.Tokens.Status != nil {
		existing.Tokens.Status = patch.Tokens.Status
	}
	if patch.Tokens.Charts != nil {
		existing.Tokens.Charts = patch.Tokens.Charts
	}
	if patch.Tokens.Typography != nil {
		existing.Tokens.Typography = patch.Tokens.Typography
	}
	if patch.Tokens.Spacing != nil {
		existing.Tokens.Spacing = patch.Tokens.Spacing
	}
	if patch.Tokens.Effects != nil {
		existing.Tokens.Effects = patch.Tokens.Effects
	}

	existing.Version++
	existing.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.Marshal(existing)
	if err != nil {
		h.logger.Error("failed to marshal updated theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to store theme")
		return
	}

	if err := h.settings.Set(r.Context(), themeKeyPrefix+id, string(data)); err != nil {
		h.logger.Error("failed to save updated theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to store theme")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// handleDeleteTheme deletes a custom theme.
//
//	@Summary		Delete theme
//	@Description	Delete a custom theme. Built-in themes cannot be deleted.
//	@Tags			settings
//	@Param			id	path	string	true	"Theme ID"
//	@Success		204	"Theme deleted"
//	@Failure		403	{object}	SettingsProblemDetail	"Cannot delete built-in theme"
//	@Failure		404	{object}	SettingsProblemDetail	"Theme not found"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/{id} [delete]
func (h *Handler) handleDeleteTheme(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	setting, err := h.settings.Get(r.Context(), themeKeyPrefix+id)
	if err != nil {
		if err == services.ErrNotFound {
			writeSettingsError(w, http.StatusNotFound, "theme not found")
			return
		}
		h.logger.Error("failed to get theme for delete", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get theme")
		return
	}

	var existing ThemeDefinition
	if err := json.Unmarshal([]byte(setting.Value), &existing); err != nil {
		h.logger.Error("failed to parse theme for delete", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to parse theme")
		return
	}

	if existing.BuiltIn {
		writeSettingsError(w, http.StatusForbidden, "cannot delete built-in theme")
		return
	}

	if err := h.settings.Delete(r.Context(), themeKeyPrefix+id); err != nil {
		h.logger.Error("failed to delete theme", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to delete theme")
		return
	}

	// If the deleted theme was the active theme, reset to default.
	active, err := h.settings.Get(r.Context(), themeActiveKey)
	if err == nil && active.Value == id {
		_ = h.settings.Set(r.Context(), themeActiveKey, defaultThemeID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetActiveTheme returns the currently active theme ID.
//
//	@Summary		Get active theme
//	@Description	Get the ID of the currently active theme.
//	@Tags			settings
//	@Produce		json
//	@Success		200	{object}	ActiveThemeResponse		"Active theme ID"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/active [get]
func (h *Handler) handleGetActiveTheme(w http.ResponseWriter, r *http.Request) {
	setting, err := h.settings.Get(r.Context(), themeActiveKey)
	if err != nil {
		if err == services.ErrNotFound {
			writeJSON(w, http.StatusOK, ActiveThemeResponse{ThemeID: defaultThemeID})
			return
		}
		h.logger.Error("failed to get active theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get active theme")
		return
	}
	writeJSON(w, http.StatusOK, ActiveThemeResponse{ThemeID: setting.Value})
}

// handleSetActiveTheme sets the active theme.
//
//	@Summary		Set active theme
//	@Description	Set which theme is currently active.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ActiveThemeRequest		true	"Theme ID to activate"
//	@Success		200		{object}	ActiveThemeResponse		"Active theme set"
//	@Failure		400		{object}	SettingsProblemDetail	"Validation error"
//	@Failure		404		{object}	SettingsProblemDetail	"Theme not found"
//	@Failure		500		{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/active [put]
func (h *Handler) handleSetActiveTheme(w http.ResponseWriter, r *http.Request) {
	var req ActiveThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSettingsError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.ThemeID) == "" {
		writeSettingsError(w, http.StatusBadRequest, "theme_id is required")
		return
	}

	// Verify the theme exists.
	if _, err := h.settings.Get(r.Context(), themeKeyPrefix+req.ThemeID); err != nil {
		if err == services.ErrNotFound {
			writeSettingsError(w, http.StatusNotFound, "theme not found")
			return
		}
		h.logger.Error("failed to verify theme for activation", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to verify theme")
		return
	}

	if err := h.settings.Set(r.Context(), themeActiveKey, req.ThemeID); err != nil {
		h.logger.Error("failed to set active theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to set active theme")
		return
	}

	writeJSON(w, http.StatusOK, ActiveThemeResponse(req))
}

// ensureBuiltInThemes seeds built-in themes, adding any new ones that don't exist yet.
func (h *Handler) ensureBuiltInThemes(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)
	builtins := []ThemeDefinition{
		{
			ID:          "builtin-forest-dark",
			Name:        "Forest Dark",
			Description: "Default dark theme with forest green accents.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Tokens:      ThemeTokens{},
		},
		{
			ID:          "builtin-forest-light",
			Name:        "Forest Light",
			Description: "Light theme with forest green accents.",
			BaseMode:    "light",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Tokens:      ThemeTokens{},
		},
		{
			ID:          "builtin-navy-copper",
			Name:        "Navy Copper",
			Description: "Dark navy theme with copper accents and a natural green palette.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Tokens:      navyCopperTokens(),
		},
		{
			ID:          "builtin-classic-dark",
			Name:        "Classic Dark",
			Description: "Neutral dark theme with slate grays and blue accents.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Tokens:      classicDarkTokens(),
		},
		{
			ID:          "builtin-classic-light",
			Name:        "Classic Light",
			Description: "Clean light theme with neutral grays and blue accents.",
			BaseMode:    "light",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Tokens:      classicLightTokens(),
		},
	}

	for i := range builtins {
		// Skip themes that already exist.
		if _, err := h.settings.Get(ctx, themeKeyPrefix+builtins[i].ID); err == nil {
			continue
		}
		data, err := json.Marshal(builtins[i])
		if err != nil {
			return fmt.Errorf("marshal built-in theme %s: %w", builtins[i].ID, err)
		}
		if err := h.settings.Set(ctx, themeKeyPrefix+builtins[i].ID, string(data)); err != nil {
			return fmt.Errorf("save built-in theme %s: %w", builtins[i].ID, err)
		}
	}

	// Keep the seeded marker for backward compatibility.
	if _, err := h.settings.Get(ctx, themeSeededKey); err != nil {
		return h.settings.Set(ctx, themeSeededKey, "true")
	}
	return nil
}

// navyCopperTokens returns the complete token overrides for the Navy Copper theme.
func navyCopperTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root":     "#0D2238",
			"bg-surface":  "#122E4E",
			"bg-card":     "#1A4674",
			"bg-elevated": "#1E5080",
			"bg-hover":    "rgba(98, 203, 100, 0.06)",
			"bg-active":   "rgba(98, 203, 100, 0.10)",
			"bg-selected": "rgba(98, 203, 100, 0.08)",
		},
		Text: map[string]string{
			"text-primary":   "#E8EDF2",
			"text-secondary": "#8BA87A",
			"text-muted":     "#658646",
			"text-accent":    "#62CB64",
			"text-warm":      "#D59958",
			"text-inverse":   "#0D2238",
		},
		Borders: map[string]string{
			"border-subtle":  "rgba(98, 203, 100, 0.08)",
			"border-default": "rgba(98, 203, 100, 0.15)",
			"border-strong":  "rgba(98, 203, 100, 0.25)",
			"border-focus":   "#62CB64",
		},
		Buttons: map[string]string{
			"btn-primary-bg":    "#62CB64",
			"btn-primary-hover": "#78D67A",
			"btn-primary-text":  "#0D2238",
			"btn-danger-bg":     "#991b1b",
			"btn-danger-hover":  "#b91c1c",
			"btn-danger-text":   "#fecaca",
		},
		Inputs: map[string]string{
			"input-bg":          "#0D2238",
			"input-border":      "rgba(98, 203, 100, 0.15)",
			"input-focus":       "#62CB64",
			"input-text":        "#E8EDF2",
			"input-placeholder": "#658646",
		},
		Sidebar: map[string]string{
			"sidebar-bg":        "#0D2238",
			"sidebar-item":      "#8BA87A",
			"sidebar-active":    "#62CB64",
			"sidebar-active-bg": "rgba(98, 203, 100, 0.10)",
		},
		Status: map[string]string{
			"status-online":   "#62CB64",
			"status-degraded": "#D59958",
			"status-offline":  "#ef4444",
			"status-unknown":  "#658646",
		},
		Charts: map[string]string{
			"chart-green": "#62CB64",
			"chart-amber": "#D59958",
			"chart-sage":  "#658646",
			"chart-red":   "#ef4444",
			"chart-blue":  "#4A90D9",
			"chart-grid":  "rgba(98, 203, 100, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm":   "0 1px 2px rgba(0, 0, 0, 0.4)",
			"shadow-md":   "0 4px 6px rgba(0, 0, 0, 0.35)",
			"shadow-lg":   "0 10px 25px rgba(0, 0, 0, 0.4)",
			"shadow-glow": "0 0 20px rgba(98, 203, 100, 0.12)",
		},
	}
}

// classicDarkTokens returns the complete token overrides for the Classic Dark theme.
func classicDarkTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root":     "#1a1b26",
			"bg-surface":  "#1e1f2b",
			"bg-card":     "#252736",
			"bg-elevated": "#2a2b3d",
			"bg-hover":    "rgba(91, 156, 246, 0.06)",
			"bg-active":   "rgba(91, 156, 246, 0.10)",
			"bg-selected": "rgba(91, 156, 246, 0.08)",
		},
		Text: map[string]string{
			"text-primary":   "#e1e2e7",
			"text-secondary": "#a0a4b8",
			"text-muted":     "#6b6f85",
			"text-accent":    "#5b9cf6",
			"text-warm":      "#e6a855",
			"text-inverse":   "#1a1b26",
		},
		Borders: map[string]string{
			"border-subtle":  "rgba(91, 156, 246, 0.08)",
			"border-default": "rgba(91, 156, 246, 0.15)",
			"border-strong":  "rgba(91, 156, 246, 0.25)",
			"border-focus":   "#5b9cf6",
		},
		Buttons: map[string]string{
			"btn-primary-bg":    "#5b9cf6",
			"btn-primary-hover": "#7ab3f8",
			"btn-primary-text":  "#ffffff",
			"btn-danger-bg":     "#991b1b",
			"btn-danger-hover":  "#b91c1c",
			"btn-danger-text":   "#fecaca",
		},
		Inputs: map[string]string{
			"input-bg":          "#1a1b26",
			"input-border":      "rgba(91, 156, 246, 0.15)",
			"input-focus":       "#5b9cf6",
			"input-text":        "#e1e2e7",
			"input-placeholder": "#6b6f85",
		},
		Sidebar: map[string]string{
			"sidebar-bg":        "#1a1b26",
			"sidebar-item":      "#a0a4b8",
			"sidebar-active":    "#5b9cf6",
			"sidebar-active-bg": "rgba(91, 156, 246, 0.10)",
		},
		Status: map[string]string{
			"status-online":   "#4ade80",
			"status-degraded": "#e6a855",
			"status-offline":  "#ef4444",
			"status-unknown":  "#a0a4b8",
		},
		Charts: map[string]string{
			"chart-green": "#4ade80",
			"chart-amber": "#e6a855",
			"chart-sage":  "#a0a4b8",
			"chart-red":   "#ef4444",
			"chart-blue":  "#5b9cf6",
			"chart-grid":  "rgba(91, 156, 246, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm":   "0 1px 2px rgba(0, 0, 0, 0.4)",
			"shadow-md":   "0 4px 6px rgba(0, 0, 0, 0.35)",
			"shadow-lg":   "0 10px 25px rgba(0, 0, 0, 0.4)",
			"shadow-glow": "0 0 20px rgba(91, 156, 246, 0.12)",
		},
	}
}

// classicLightTokens returns the complete token overrides for the Classic Light theme.
func classicLightTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root":     "#f6f8fa",
			"bg-surface":  "#eef1f5",
			"bg-card":     "#ffffff",
			"bg-elevated": "#ffffff",
			"bg-hover":    "rgba(9, 105, 218, 0.04)",
			"bg-active":   "rgba(9, 105, 218, 0.08)",
			"bg-selected": "rgba(9, 105, 218, 0.06)",
		},
		Text: map[string]string{
			"text-primary":   "#1f2328",
			"text-secondary": "#656d76",
			"text-muted":     "#8b949e",
			"text-accent":    "#0969da",
			"text-warm":      "#bf8700",
			"text-inverse":   "#f6f8fa",
		},
		Borders: map[string]string{
			"border-subtle":  "rgba(9, 105, 218, 0.06)",
			"border-default": "rgba(9, 105, 218, 0.15)",
			"border-strong":  "rgba(9, 105, 218, 0.25)",
			"border-focus":   "#0969da",
		},
		Buttons: map[string]string{
			"btn-primary-bg":    "#0969da",
			"btn-primary-hover": "#0757b5",
			"btn-primary-text":  "#ffffff",
			"btn-danger-bg":     "#cf222e",
			"btn-danger-hover":  "#a40e26",
			"btn-danger-text":   "#ffffff",
		},
		Inputs: map[string]string{
			"input-bg":          "#ffffff",
			"input-border":      "rgba(9, 105, 218, 0.15)",
			"input-focus":       "#0969da",
			"input-text":        "#1f2328",
			"input-placeholder": "#8b949e",
		},
		Sidebar: map[string]string{
			"sidebar-bg":        "#eef1f5",
			"sidebar-item":      "#656d76",
			"sidebar-active":    "#0969da",
			"sidebar-active-bg": "rgba(9, 105, 218, 0.08)",
		},
		Status: map[string]string{
			"status-online":   "#1a7f37",
			"status-degraded": "#bf8700",
			"status-offline":  "#cf222e",
			"status-unknown":  "#8b949e",
		},
		Charts: map[string]string{
			"chart-green": "#1a7f37",
			"chart-amber": "#bf8700",
			"chart-sage":  "#8b949e",
			"chart-red":   "#cf222e",
			"chart-blue":  "#0969da",
			"chart-grid":  "rgba(9, 105, 218, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm":   "0 1px 2px rgba(0, 0, 0, 0.05)",
			"shadow-md":   "0 4px 6px rgba(0, 0, 0, 0.07)",
			"shadow-lg":   "0 10px 25px rgba(0, 0, 0, 0.1)",
			"shadow-glow": "0 0 20px rgba(9, 105, 218, 0.08)",
		},
	}
}

// generateID returns a random 32-character hex string suitable for use as a theme ID.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
