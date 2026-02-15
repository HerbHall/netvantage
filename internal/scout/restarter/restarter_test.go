package restarter

import "testing"

func TestDetect_ReturnsNonNil(t *testing.T) {
	r := Detect()
	if r == nil {
		t.Fatal("Detect() returned nil, expected a Restarter")
	}
}

func TestDetect_HasName(t *testing.T) {
	r := Detect()
	if r == nil {
		t.Skip("Detect() returned nil")
	}
	name := r.Name()
	if name == "" {
		t.Error("Name() returned empty string")
	}
	t.Logf("detected restarter: %s", name)
}

func TestAllRestarters_HaveNames(t *testing.T) {
	tests := []struct {
		name      string
		restarter Restarter
	}{
		// Only test the exec restarter since it exists on all platforms.
		{"exec", &execRestarter{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.restarter.Name()
			if got == "" {
				t.Error("Name() returned empty string")
			}
			if got != tt.name {
				t.Errorf("Name() = %q, want %q", got, tt.name)
			}
		})
	}
}
