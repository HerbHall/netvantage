//go:build !windows

package profiler

import (
	"testing"
)

func TestParseCPUInfo(t *testing.T) {
	// Fixture: dual-core hyperthreaded CPU (2 physical cores, 4 threads).
	content := `processor	: 0
vendor_id	: GenuineIntel
cpu family	: 6
model		: 142
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
physical id	: 0
core id		: 0

processor	: 1
vendor_id	: GenuineIntel
cpu family	: 6
model		: 142
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
physical id	: 0
core id		: 1

processor	: 2
vendor_id	: GenuineIntel
cpu family	: 6
model		: 142
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
physical id	: 0
core id		: 0

processor	: 3
vendor_id	: GenuineIntel
cpu family	: 6
model		: 142
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
physical id	: 0
core id		: 1
`

	tests := []struct {
		name        string
		content     string
		logical     int32
		wantModel   string
		wantPhys    int32
		wantLogical int32
	}{
		{
			name:        "hyperthreaded dual-core",
			content:     content,
			logical:     4,
			wantModel:   "Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz",
			wantPhys:    2,
			wantLogical: 4,
		},
		{
			name: "single processor no physical id",
			content: `processor	: 0
model name	: ARM Cortex-A72
`,
			logical:     1,
			wantModel:   "ARM Cortex-A72",
			wantPhys:    1, // Falls back to logical count.
			wantLogical: 1,
		},
		{
			name:        "empty content",
			content:     "",
			logical:     4,
			wantModel:   "",
			wantPhys:    4, // Falls back to logical count.
			wantLogical: 4,
		},
		{
			name: "dual socket",
			content: `processor	: 0
model name	: Intel Xeon E5-2680 v4
physical id	: 0
core id		: 0

processor	: 1
model name	: Intel Xeon E5-2680 v4
physical id	: 0
core id		: 1

processor	: 2
model name	: Intel Xeon E5-2680 v4
physical id	: 1
core id		: 0

processor	: 3
model name	: Intel Xeon E5-2680 v4
physical id	: 1
core id		: 1
`,
			logical:     4,
			wantModel:   "Intel Xeon E5-2680 v4",
			wantPhys:    4, // 2 sockets x 2 cores = 4 unique (physID, coreID) pairs.
			wantLogical: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, phys, logical := parseCPUInfo(tt.content, tt.logical)
			if model != tt.wantModel {
				t.Errorf("model: got %q, want %q", model, tt.wantModel)
			}
			if phys != tt.wantPhys {
				t.Errorf("physCores: got %d, want %d", phys, tt.wantPhys)
			}
			if logical != tt.wantLogical {
				t.Errorf("logicalCPUs: got %d, want %d", logical, tt.wantLogical)
			}
		})
	}
}

func TestClassifyLinuxDiskType(t *testing.T) {
	tests := []struct {
		name     string
		devName  string
		wantType string
	}{
		{name: "nvme device", devName: "nvme0n1", wantType: "NVMe"},
		{name: "nvme partition", devName: "nvme1n1", wantType: "NVMe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For NVMe, the function returns early based on name alone
			// without reading /sys/block files.
			got := classifyLinuxDiskType(tt.devName, "/nonexistent")
			if got != tt.wantType {
				t.Errorf("classifyLinuxDiskType(%q): got %q, want %q", tt.devName, got, tt.wantType)
			}
		})
	}
}

func TestParseLspciOutput(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantCount int
		wantGPUs  []string // expected model strings
	}{
		{
			name: "nvidia and intel GPUs",
			output: `00:02.0 "VGA compatible controller" "Intel Corporation" "CoffeeLake-H GT2 [UHD Graphics 630]" "Dell" "CoffeeLake-H GT2 [UHD Graphics 630]"
01:00.0 "VGA compatible controller" "NVIDIA Corporation" "GA106 [GeForce RTX 3060 Lite Hash Rate]" "eVga.com. Corp." "GA106 [GeForce RTX 3060 12GB]"
`,
			wantCount: 2,
			wantGPUs: []string{
				"Intel Corporation CoffeeLake-H GT2 [UHD Graphics 630]",
				"NVIDIA Corporation GA106 [GeForce RTX 3060 Lite Hash Rate]",
			},
		},
		{
			name: "3D controller (compute GPU)",
			output: `41:00.0 "3D controller" "NVIDIA Corporation" "A100 PCIe 40GB" "NVIDIA" "A100"
`,
			wantCount: 1,
			wantGPUs:  []string{"NVIDIA Corporation A100 PCIe 40GB"},
		},
		{
			name: "no GPUs (network and storage only)",
			output: `00:1f.6 "Ethernet controller" "Intel Corporation" "Ethernet Connection (7) I219-V" "Dell" "Ethernet Connection (7) I219-V"
01:00.0 "Non-Volatile memory controller" "Samsung Electronics Co Ltd" "NVMe SSD Controller PM9A1/PM9A3/980PRO" "" ""
`,
			wantCount: 0,
			wantGPUs:  nil,
		},
		{
			name:      "empty output",
			output:    "",
			wantCount: 0,
			wantGPUs:  nil,
		},
		{
			name: "AMD GPU",
			output: `06:00.0 "VGA compatible controller" "Advanced Micro Devices, Inc. [AMD/ATI]" "Navi 10 [Radeon RX 5600 OXT / 5700 XT]" "XFX" "Navi 10 [Radeon RX 5600 XT]"
`,
			wantCount: 1,
			wantGPUs:  []string{"Advanced Micro Devices, Inc. [AMD/ATI] Navi 10 [Radeon RX 5600 OXT / 5700 XT]"},
		},
		{
			name: "mixed devices with GPU",
			output: `00:00.0 "Host bridge" "Intel Corporation" "Device 9a14" "" ""
00:02.0 "VGA compatible controller" "Intel Corporation" "TigerLake-LP GT2 [Iris Xe Graphics]" "Lenovo" "TigerLake-LP GT2 [Iris Xe Graphics]"
00:14.0 "USB controller" "Intel Corporation" "Tiger Lake-LP USB 3.2 Gen 2x1 xHCI Host Controller" "Lenovo" ""
`,
			wantCount: 1,
			wantGPUs:  []string{"Intel Corporation TigerLake-LP GT2 [Iris Xe Graphics]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpus := parseLspciOutput(tt.output)
			if len(gpus) != tt.wantCount {
				t.Fatalf("parseLspciOutput() returned %d GPUs, want %d", len(gpus), tt.wantCount)
			}
			for i, wantModel := range tt.wantGPUs {
				if gpus[i].Model != wantModel {
					t.Errorf("GPU[%d].Model = %q, want %q", i, gpus[i].Model, wantModel)
				}
			}
		})
	}
}

func TestParseQuotedFields(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantFields []string
	}{
		{
			name: "standard lspci line",
			line: `01:00.0 "VGA compatible controller" "NVIDIA Corporation" "GA106 [GeForce RTX 3060]" "eVga.com." "GA106"`,
			wantFields: []string{
				"01:00.0",
				"VGA compatible controller",
				"NVIDIA Corporation",
				"GA106 [GeForce RTX 3060]",
				"eVga.com.",
				"GA106",
			},
		},
		{
			name:       "slot only",
			line:       `00:02.0`,
			wantFields: []string{"00:02.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseQuotedFields(tt.line)
			if len(got) != len(tt.wantFields) {
				t.Fatalf("parseQuotedFields() returned %d fields, want %d: %v", len(got), len(tt.wantFields), got)
			}
			for i, want := range tt.wantFields {
				if got[i] != want {
					t.Errorf("field[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestClassifyLinuxNICType(t *testing.T) {
	tests := []struct {
		name       string
		ifName     string
		kernelType string
		wantType   string
	}{
		{name: "wifi wl prefix", ifName: "wlp3s0", kernelType: "1", wantType: "wifi"},
		{name: "wifi wlan prefix", ifName: "wlan0", kernelType: "1", wantType: "wifi"},
		{name: "virtual veth", ifName: "veth123abc", kernelType: "1", wantType: "virtual"},
		{name: "virtual docker", ifName: "docker0", kernelType: "1", wantType: "virtual"},
		{name: "virtual bridge", ifName: "br-abc123", kernelType: "1", wantType: "virtual"},
		{name: "ethernet", ifName: "eth0", kernelType: "1", wantType: "ethernet"},
		{name: "ethernet eno", ifName: "eno1", kernelType: "1", wantType: "ethernet"},
		{name: "wifi kernel type", ifName: "ath0", kernelType: "801", wantType: "wifi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLinuxNICType(tt.ifName, tt.kernelType)
			if got != tt.wantType {
				t.Errorf("classifyLinuxNICType(%q, %q): got %q, want %q", tt.ifName, tt.kernelType, got, tt.wantType)
			}
		})
	}
}
