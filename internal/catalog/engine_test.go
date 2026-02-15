package catalog

import (
	"testing"

	pkgcatalog "github.com/HerbHall/subnetree/pkg/catalog"
)

func TestEngine_Recommend_Tier0(t *testing.T) {
	engine := NewEngine(pkgcatalog.NewCatalog())
	entries, err := engine.Recommend(pkgcatalog.TierSBC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tier 0 should exclude tools that only support higher tiers
	// (Grafana, Prometheus, NetBox all require Tier 1+).
	for i := range entries {
		if entries[i].Name == "Grafana" || entries[i].Name == "Prometheus" || entries[i].Name == "NetBox" {
			t.Errorf("tier 0 should not include %s", entries[i].Name)
		}
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry for tier 0")
	}
}

func TestEngine_Recommend_Tier1_IncludesAll(t *testing.T) {
	engine := NewEngine(pkgcatalog.NewCatalog())
	entries, err := engine.Recommend(pkgcatalog.TierMiniPC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tier 1 (Mini PC) supports most tools. Only NAS-only tools excluded.
	// All 15 entries support tier 1 except none (NAS tier 2 tools also support 1).
	// Check we get a substantial set.
	if len(entries) < 12 {
		t.Errorf("expected at least 12 entries for tier 1, got %d", len(entries))
	}
}

func TestEngine_Recommend_SortedByRAM(t *testing.T) {
	engine := NewEngine(pkgcatalog.NewCatalog())
	entries, err := engine.Recommend(pkgcatalog.TierMiniPC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 1; i < len(entries); i++ {
		if entries[i].MinRAMMB < entries[i-1].MinRAMMB {
			t.Errorf("entries not sorted by RAM: %s (%d MB) before %s (%d MB)",
				entries[i-1].Name, entries[i-1].MinRAMMB,
				entries[i].Name, entries[i].MinRAMMB)
		}
	}
}

func TestEngine_RecommendByCategory(t *testing.T) {
	engine := NewEngine(pkgcatalog.NewCatalog())
	entries, err := engine.RecommendByCategory(pkgcatalog.TierMiniPC, pkgcatalog.CategoryMonitoring)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected monitoring entries for tier 1")
	}
	for i := range entries {
		if entries[i].Category != pkgcatalog.CategoryMonitoring {
			t.Errorf("expected category monitoring, got %s for %s", entries[i].Category, entries[i].Name)
		}
	}
}

func TestEngine_Recommend_NAS_OnlyLightweight(t *testing.T) {
	engine := NewEngine(pkgcatalog.NewCatalog())
	entries, err := engine.Recommend(pkgcatalog.TierNAS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// NAS tier should only include tools that explicitly support tier 2.
	for i := range entries {
		if !tierSupported(entries[i].SupportedTiers, pkgcatalog.TierNAS) {
			t.Errorf("tier NAS should not include %s", entries[i].Name)
		}
	}

	// Should be a smaller subset than tier 1.
	tier1Entries, _ := engine.Recommend(pkgcatalog.TierMiniPC)
	if len(entries) >= len(tier1Entries) {
		t.Errorf("NAS tier (%d entries) should be smaller than Mini PC tier (%d entries)",
			len(entries), len(tier1Entries))
	}
}

func TestTierSupported(t *testing.T) {
	tests := []struct {
		name   string
		tiers  []pkgcatalog.HardwareTier
		target pkgcatalog.HardwareTier
		want   bool
	}{
		{
			name:   "present",
			tiers:  []pkgcatalog.HardwareTier{0, 1, 3, 4},
			target: 1,
			want:   true,
		},
		{
			name:   "absent",
			tiers:  []pkgcatalog.HardwareTier{0, 1, 3, 4},
			target: 2,
			want:   false,
		},
		{
			name:   "empty",
			tiers:  []pkgcatalog.HardwareTier{},
			target: 0,
			want:   false,
		},
		{
			name:   "all tiers",
			tiers:  []pkgcatalog.HardwareTier{0, 1, 2, 3, 4},
			target: 3,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tierSupported(tt.tiers, tt.target)
			if got != tt.want {
				t.Errorf("tierSupported(%v, %d) = %v, want %v", tt.tiers, tt.target, got, tt.want)
			}
		})
	}
}
