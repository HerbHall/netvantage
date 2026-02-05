package recon

import "testing"

func TestOUITable_KnownPrefixes(t *testing.T) {
	oui := NewOUITable()

	tests := []struct {
		mac  string
		want string
	}{
		{"00:50:56:XX:XX:XX", "VMware, Inc."},
		{"00:0C:29:AB:CD:EF", "VMware, Inc."},
		{"DC:A6:32:00:11:22", "Raspberry Pi Trading Ltd"},
	}

	for _, tt := range tests {
		t.Run(tt.mac, func(t *testing.T) {
			got := oui.Lookup(tt.mac)
			if got != tt.want {
				t.Errorf("Lookup(%q) = %q, want %q", tt.mac, got, tt.want)
			}
		})
	}
}

func TestOUITable_UnknownMAC(t *testing.T) {
	oui := NewOUITable()
	got := oui.Lookup("FF:FF:FF:FF:FF:FF")
	if got != "" {
		t.Errorf("Lookup(FF:FF:FF:FF:FF:FF) = %q, want empty", got)
	}
}

func TestOUITable_Formats(t *testing.T) {
	oui := NewOUITable()

	// All these represent the same OUI prefix for VMware.
	formats := []string{
		"00:50:56:12:34:56",
		"00-50-56-12-34-56",
		"005056123456",
		"0050.5612.3456",
	}
	for _, mac := range formats {
		t.Run(mac, func(t *testing.T) {
			got := oui.Lookup(mac)
			if got != "VMware, Inc." {
				t.Errorf("Lookup(%q) = %q, want VMware, Inc.", mac, got)
			}
		})
	}
}

func TestOUITable_MalformedMAC(t *testing.T) {
	oui := NewOUITable()

	tests := []string{"", "AB", "not-a-mac", "ZZ:ZZ:ZZ:ZZ:ZZ:ZZ"}
	for _, mac := range tests {
		t.Run(mac, func(t *testing.T) {
			got := oui.Lookup(mac)
			if got != "" {
				t.Errorf("Lookup(%q) = %q, want empty for malformed MAC", mac, got)
			}
		})
	}
}

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"aa:bb:cc:dd:ee:ff", "AA:BB:CC"},
		{"AA-BB-CC-DD-EE-FF", "AA:BB:CC"},
		{"AABBCCDDEEFF", "AA:BB:CC"},
		{"aabb.ccdd.eeff", "AA:BB:CC"},
		{"", ""},
		{"AB", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeMAC(tt.input)
			if got != tt.want {
				t.Errorf("normalizeMAC(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
