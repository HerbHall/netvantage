package recon

import "testing"

func TestParseLinuxARP(t *testing.T) {
	output := `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.1      0x1         0x2         aa:bb:cc:dd:ee:ff     *        eth0
192.168.1.2      0x1         0x2         11:22:33:44:55:66     *        eth0
192.168.1.3      0x1         0x0         00:00:00:00:00:00     *        eth0
`
	table := ParseARPOutput(output, "linux")
	if len(table) != 2 {
		t.Errorf("entry count = %d, want 2 (incomplete entry skipped)", len(table))
	}
	if table["192.168.1.1"] != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("192.168.1.1 = %q, want AA:BB:CC:DD:EE:FF", table["192.168.1.1"])
	}
	if table["192.168.1.2"] != "11:22:33:44:55:66" {
		t.Errorf("192.168.1.2 = %q, want 11:22:33:44:55:66", table["192.168.1.2"])
	}
}

func TestParseWindowsARP(t *testing.T) {
	output := `
Interface: 192.168.1.100 --- 0x4
  Internet Address      Physical Address      Type
  192.168.1.1           aa-bb-cc-dd-ee-ff     dynamic
  192.168.1.2           11-22-33-44-55-66     dynamic
  192.168.1.255         ff-ff-ff-ff-ff-ff     static
`
	table := ParseARPOutput(output, "windows")
	if len(table) != 2 {
		t.Errorf("entry count = %d, want 2 (broadcast skipped)", len(table))
	}
	if table["192.168.1.1"] != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("192.168.1.1 = %q, want AA:BB:CC:DD:EE:FF", table["192.168.1.1"])
	}
}

func TestParseDarwinARP(t *testing.T) {
	output := `? (192.168.1.1) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]
? (192.168.1.2) at 11:22:33:44:55:66 on en0 ifscope [ethernet]
? (192.168.1.3) at (incomplete) on en0 ifscope [ethernet]
`
	table := ParseARPOutput(output, "darwin")
	if len(table) != 2 {
		t.Errorf("entry count = %d, want 2 (incomplete skipped)", len(table))
	}
	if table["192.168.1.1"] != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("192.168.1.1 = %q, want AA:BB:CC:DD:EE:FF", table["192.168.1.1"])
	}
}

func TestParseARP_EmptyOutput(t *testing.T) {
	for _, platform := range []string{"linux", "windows", "darwin"} {
		t.Run(platform, func(t *testing.T) {
			table := ParseARPOutput("", platform)
			if len(table) != 0 {
				t.Errorf("expected empty table, got %d entries", len(table))
			}
		})
	}
}

func TestParseARP_UnknownPlatform(t *testing.T) {
	table := ParseARPOutput("anything", "freebsd")
	if len(table) != 0 {
		t.Errorf("expected empty table for unknown platform, got %d entries", len(table))
	}
}
