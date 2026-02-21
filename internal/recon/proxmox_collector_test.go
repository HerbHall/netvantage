package recon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func newTestProxmoxServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestProxmoxCollector_CollectNodes(t *testing.T) {
	tests := []struct {
		name       string
		response   any
		statusCode int
		wantCount  int
		wantErr    bool
	}{
		{
			name: "two nodes",
			response: map[string]any{
				"data": []map[string]any{
					{
						"node":    "pve1",
						"status":  "online",
						"cpu":     0.15,
						"maxcpu":  16,
						"mem":     34359738368,
						"maxmem":  68719476736,
						"disk":    21474836480,
						"maxdisk": 500107862016,
						"uptime":  1234567,
					},
					{
						"node":    "pve2",
						"status":  "online",
						"cpu":     0.42,
						"maxcpu":  8,
						"mem":     8589934592,
						"maxmem":  17179869184,
						"disk":    10737418240,
						"maxdisk": 250053631008,
						"uptime":  987654,
					},
				},
			},
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name: "empty cluster",
			response: map[string]any{
				"data": []map[string]any{},
			},
			statusCode: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "API error",
			response:   "Unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestProxmoxServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api2/json/nodes" {
					http.NotFound(w, r)
					return
				}
				w.WriteHeader(tt.statusCode)
				if err := json.NewEncoder(w).Encode(tt.response); err != nil {
					t.Fatalf("encode response: %v", err)
				}
			})

			c := NewProxmoxCollector(srv.URL, "test@pve!token", "secret", zap.NewNop())
			nodes, err := c.CollectNodes(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(nodes) != tt.wantCount {
				t.Errorf("got %d nodes, want %d", len(nodes), tt.wantCount)
			}
			if tt.wantCount >= 1 {
				if nodes[0].Node != "pve1" {
					t.Errorf("node[0].Node = %q, want %q", nodes[0].Node, "pve1")
				}
				if nodes[0].Maxcpu != 16 {
					t.Errorf("node[0].Maxcpu = %d, want %d", nodes[0].Maxcpu, 16)
				}
			}
		})
	}
}

func TestProxmoxCollector_CollectNodeHardware(t *testing.T) {
	tests := []struct {
		name         string
		statusResp   any
		disksResp    any
		wantCPUModel string
		wantRAMMB    int
		wantDisks    int
		wantErr      bool
	}{
		{
			name: "node with disks",
			statusResp: map[string]any{
				"data": map[string]any{
					"cpuinfo": map[string]any{
						"model": "Intel(R) Xeon(R) E-2288G CPU @ 3.70GHz",
						"cores": 8,
						"cpus":  16,
					},
					"memory": map[string]any{
						"total": 68719476736, // 64 GB
					},
					"uptime": 1234567,
				},
			},
			disksResp: map[string]any{
				"data": []map[string]any{
					{
						"devpath": "/dev/sda",
						"model":   "Samsung SSD 870",
						"serial":  "S123456",
						"size":    1000204886016,
						"type":    "ssd",
					},
					{
						"devpath": "/dev/nvme0n1",
						"model":   "Samsung 990 PRO",
						"serial":  "N789012",
						"size":    2000398934016,
						"type":    "nvme",
					},
				},
			},
			wantCPUModel: "Intel(R) Xeon(R) E-2288G CPU @ 3.70GHz",
			wantRAMMB:    65536,
			wantDisks:    2,
		},
		{
			name: "node without disk access",
			statusResp: map[string]any{
				"data": map[string]any{
					"cpuinfo": map[string]any{
						"model": "AMD EPYC 7543",
						"cores": 32,
						"cpus":  64,
					},
					"memory": map[string]any{
						"total": 274877906944, // 256 GB
					},
				},
			},
			disksResp:    nil, // disk endpoint returns 403
			wantCPUModel: "AMD EPYC 7543",
			wantRAMMB:    262144,
			wantDisks:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestProxmoxServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api2/json/nodes/testnode/status":
					w.WriteHeader(http.StatusOK)
					if err := json.NewEncoder(w).Encode(tt.statusResp); err != nil {
						t.Fatalf("encode status: %v", err)
					}
				case "/api2/json/nodes/testnode/disks/list":
					if tt.disksResp == nil {
						http.Error(w, "Forbidden", http.StatusForbidden)
						return
					}
					w.WriteHeader(http.StatusOK)
					if err := json.NewEncoder(w).Encode(tt.disksResp); err != nil {
						t.Fatalf("encode disks: %v", err)
					}
				default:
					http.NotFound(w, r)
				}
			})

			c := NewProxmoxCollector(srv.URL, "test@pve!token", "secret", zap.NewNop())
			hw, storage, err := c.CollectNodeHardware(context.Background(), "testnode")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if hw.CPUModel != tt.wantCPUModel {
				t.Errorf("CPUModel = %q, want %q", hw.CPUModel, tt.wantCPUModel)
			}
			if hw.RAMTotalMB != tt.wantRAMMB {
				t.Errorf("RAMTotalMB = %d, want %d", hw.RAMTotalMB, tt.wantRAMMB)
			}
			if hw.CollectionSource != "proxmox-api" {
				t.Errorf("CollectionSource = %q, want %q", hw.CollectionSource, "proxmox-api")
			}
			if hw.Hypervisor != "proxmox" {
				t.Errorf("Hypervisor = %q, want %q", hw.Hypervisor, "proxmox")
			}
			if len(storage) != tt.wantDisks {
				t.Errorf("got %d storage devices, want %d", len(storage), tt.wantDisks)
			}
			if tt.wantDisks >= 2 {
				if storage[0].DiskType != "SSD" {
					t.Errorf("storage[0].DiskType = %q, want %q", storage[0].DiskType, "SSD")
				}
				if storage[1].DiskType != "NVMe" {
					t.Errorf("storage[1].DiskType = %q, want %q", storage[1].DiskType, "NVMe")
				}
				if storage[1].CollectionSource != "proxmox-api" {
					t.Errorf("storage[1].CollectionSource = %q, want %q", storage[1].CollectionSource, "proxmox-api")
				}
			}
		})
	}
}

func TestProxmoxCollector_CollectVMs(t *testing.T) {
	tests := []struct {
		name       string
		response   any
		statusCode int
		wantCount  int
		wantErr    bool
	}{
		{
			name: "two VMs",
			response: map[string]any{
				"data": []map[string]any{
					{
						"vmid":   100,
						"name":   "ubuntu-server",
						"status": "running",
						"cpus":   4,
						"maxmem": 8589934592,
					},
					{
						"vmid":   101,
						"name":   "windows-desktop",
						"status": "stopped",
						"cpus":   8,
						"maxmem": 17179869184,
					},
				},
			},
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name: "no VMs",
			response: map[string]any{
				"data": []map[string]any{},
			},
			statusCode: http.StatusOK,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestProxmoxServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api2/json/nodes/testnode/qemu" {
					http.NotFound(w, r)
					return
				}
				w.WriteHeader(tt.statusCode)
				if err := json.NewEncoder(w).Encode(tt.response); err != nil {
					t.Fatalf("encode response: %v", err)
				}
			})

			c := NewProxmoxCollector(srv.URL, "test@pve!token", "secret", zap.NewNop())
			vms, err := c.CollectVMs(context.Background(), "testnode")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(vms) != tt.wantCount {
				t.Errorf("got %d VMs, want %d", len(vms), tt.wantCount)
			}
			if tt.wantCount >= 1 {
				if vms[0].Name != "ubuntu-server" {
					t.Errorf("vm[0].Name = %q, want %q", vms[0].Name, "ubuntu-server")
				}
			}
		})
	}
}

func TestProxmoxCollector_CollectContainers(t *testing.T) {
	srv := newTestProxmoxServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/testnode/lxc" {
			http.NotFound(w, r)
			return
		}
		resp := map[string]any{
			"data": []map[string]any{
				{
					"vmid":   200,
					"name":   "nginx-proxy",
					"status": "running",
					"cpus":   2,
					"maxmem": 536870912,
				},
				{
					"vmid":   201,
					"name":   "pihole",
					"status": "running",
					"cpus":   1,
					"maxmem": 268435456,
				},
			},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	})

	c := NewProxmoxCollector(srv.URL, "test@pve!token", "secret", zap.NewNop())
	containers, err := c.CollectContainers(context.Background(), "testnode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(containers) != 2 {
		t.Fatalf("got %d containers, want 2", len(containers))
	}
	if containers[0].Name != "nginx-proxy" {
		t.Errorf("container[0].Name = %q, want %q", containers[0].Name, "nginx-proxy")
	}
	if containers[1].VMID != 201 {
		t.Errorf("container[1].VMID = %d, want %d", containers[1].VMID, 201)
	}
}

func TestProxmoxCollector_AuthHeader(t *testing.T) {
	var capturedAuth string

	srv := newTestProxmoxServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		resp := map[string]any{"data": []map[string]any{}}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	})

	c := NewProxmoxCollector(srv.URL, "admin@pve!monitoring", "aaaabbbb-cccc-dddd-eeee-ffffgggghhh", zap.NewNop())
	_, err := c.CollectNodes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantAuth := "PVEAPIToken=admin@pve!monitoring=aaaabbbb-cccc-dddd-eeee-ffffgggghhh"
	if capturedAuth != wantAuth {
		t.Errorf("Authorization header = %q, want %q", capturedAuth, wantAuth)
	}
}

func TestNormalizeDiskType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ssd", "SSD"},
		{"hdd", "HDD"},
		{"nvme", "NVMe"},
		{"unknown", "Unknown"},
		{"", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeDiskType(tt.input)
			if got != tt.want {
				t.Errorf("normalizeDiskType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
