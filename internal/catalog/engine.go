// Package catalog provides the recommendation engine that filters the embedded
// tool catalog by hardware tier and category.
package catalog

import (
	"sort"

	pkgcatalog "github.com/HerbHall/subnetree/pkg/catalog"
)

// Engine filters catalog entries by hardware tier and category.
type Engine struct {
	cat *pkgcatalog.Catalog
}

// NewEngine creates a new recommendation engine backed by the given catalog.
func NewEngine(cat *pkgcatalog.Catalog) *Engine {
	return &Engine{cat: cat}
}

// Recommend returns catalog entries compatible with the given hardware tier,
// sorted by MinRAMMB ascending (lightest first).
func (e *Engine) Recommend(tier pkgcatalog.HardwareTier) ([]pkgcatalog.CatalogEntry, error) {
	entries, err := e.cat.Entries()
	if err != nil {
		return nil, err
	}

	result := make([]pkgcatalog.CatalogEntry, 0, len(entries))
	for i := range entries {
		if tierSupported(entries[i].SupportedTiers, tier) {
			result = append(result, entries[i])
		}
	}

	sort.Slice(result, func(a, b int) bool {
		return result[a].MinRAMMB < result[b].MinRAMMB
	})

	return result, nil
}

// RecommendByCategory returns catalog entries compatible with the given tier
// and matching the specified category, sorted by MinRAMMB ascending.
func (e *Engine) RecommendByCategory(tier pkgcatalog.HardwareTier, cat pkgcatalog.Category) ([]pkgcatalog.CatalogEntry, error) {
	entries, err := e.Recommend(tier)
	if err != nil {
		return nil, err
	}

	result := make([]pkgcatalog.CatalogEntry, 0, len(entries))
	for i := range entries {
		if entries[i].Category == cat {
			result = append(result, entries[i])
		}
	}

	return result, nil
}

// tierSupported checks whether target is present in the tiers slice.
func tierSupported(tiers []pkgcatalog.HardwareTier, target pkgcatalog.HardwareTier) bool {
	for i := range tiers {
		if tiers[i] == target {
			return true
		}
	}
	return false
}
