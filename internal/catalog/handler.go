package catalog

import (
	"encoding/json"
	"net/http"
	"strconv"

	pkgcatalog "github.com/HerbHall/subnetree/pkg/catalog"
	"go.uber.org/zap"
)

// RecommendationResponse is the response for GET /api/v1/catalog/recommendations.
type RecommendationResponse struct {
	Tier    int                       `json:"tier"`
	Count   int                       `json:"count"`
	Entries []pkgcatalog.CatalogEntry `json:"entries"`
}

// Handler serves the catalog recommendation API.
type Handler struct {
	engine *Engine
	logger *zap.Logger
}

// NewHandler creates a new catalog API handler.
func NewHandler(engine *Engine, logger *zap.Logger) *Handler {
	return &Handler{engine: engine, logger: logger}
}

// RegisterRoutes implements server.SimpleRouteRegistrar.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/catalog/recommendations", h.handleRecommendations)
	mux.HandleFunc("GET /api/v1/catalog/entries", h.handleListEntries)
}

// handleRecommendations returns recommended tools for a hardware tier.
//
//	@Summary		Get tool recommendations
//	@Description	Returns recommended homelab tools filtered by hardware tier and optional category, sorted by memory requirements ascending.
//	@Tags			catalog
//	@Produce		json
//	@Security		BearerAuth
//	@Param			tier query int false "Hardware tier (0=SBC, 1=Mini PC, 2=NAS, 3=Cluster, 4=SMB)" default(1)
//	@Param			category query string false "Filter by category (monitoring, infrastructure, dns, etc.)"
//	@Success		200 {object} RecommendationResponse
//	@Failure		400 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/catalog/recommendations [get]
func (h *Handler) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	tierStr := r.URL.Query().Get("tier")
	tier := 1 // default: Mini PC
	if tierStr != "" {
		parsed, err := strconv.Atoi(tierStr)
		if err != nil || parsed < 0 || parsed > 4 {
			writeError(w, http.StatusBadRequest, "tier must be an integer between 0 and 4")
			return
		}
		tier = parsed
	}

	category := r.URL.Query().Get("category")

	var entries []pkgcatalog.CatalogEntry
	var err error

	if category != "" {
		entries, err = h.engine.RecommendByCategory(
			pkgcatalog.HardwareTier(tier),
			pkgcatalog.Category(category),
		)
	} else {
		entries, err = h.engine.Recommend(pkgcatalog.HardwareTier(tier))
	}

	if err != nil {
		h.logger.Error("failed to get recommendations", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to load catalog")
		return
	}

	if entries == nil {
		entries = []pkgcatalog.CatalogEntry{}
	}

	writeJSON(w, http.StatusOK, RecommendationResponse{
		Tier:    tier,
		Count:   len(entries),
		Entries: entries,
	})
}

// handleListEntries returns all catalog entries.
//
//	@Summary		List all catalog entries
//	@Description	Returns the full tool catalog for client-side filtering.
//	@Tags			catalog
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200 {array} pkgcatalog.CatalogEntry
//	@Failure		500 {object} map[string]any
//	@Router			/catalog/entries [get]
func (h *Handler) handleListEntries(w http.ResponseWriter, r *http.Request) {
	entries, err := h.engine.cat.Entries()
	if err != nil {
		h.logger.Error("failed to load catalog", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to load catalog")
		return
	}

	if entries == nil {
		entries = []pkgcatalog.CatalogEntry{}
	}

	writeJSON(w, http.StatusOK, entries)
}

// -- helpers --

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
