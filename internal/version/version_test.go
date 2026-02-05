package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestInfo(t *testing.T) {
	info := Info()
	if !strings.Contains(info, "SubNetree") {
		t.Errorf("Info() should contain 'SubNetree', got: %s", info)
	}
	if !strings.Contains(info, runtime.Version()) {
		t.Errorf("Info() should contain Go version, got: %s", info)
	}
}

func TestShort(t *testing.T) {
	if got := Short(); got != "dev" {
		t.Errorf("Short() = %q, want %q (default)", got, "dev")
	}
}

func TestMap(t *testing.T) {
	m := Map()

	requiredKeys := []string{"version", "git_commit", "build_date", "go_version", "os", "arch"}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("Map() missing key %q", key)
		}
	}

	if m["version"] != "dev" {
		t.Errorf("Map()[\"version\"] = %q, want %q", m["version"], "dev")
	}
	if m["go_version"] != runtime.Version() {
		t.Errorf("Map()[\"go_version\"] = %q, want %q", m["go_version"], runtime.Version())
	}
}
