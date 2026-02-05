package recon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// newTestModule creates a Module wired with in-memory SQLite and mock scanners.
func newTestModule(t *testing.T) *Module {
	t.Helper()

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	reconStore := NewReconStore(db.DB())
	oui := NewOUITable()
	bus := newTestBus(logger)

	pinger := &mockPingScanner{results: []HostResult{}}
	arp := &mockARPReader{table: map[string]string{}}

	m := &Module{
		logger:      logger,
		cfg:         DefaultConfig(),
		store:       reconStore,
		bus:         bus,
		oui:         oui,
		orchestrator: NewScanOrchestrator(reconStore, bus, oui, pinger, arp, logger),
	}
	// Start sets up scanCtx.
	m.scanCtx, m.scanCancel = context.WithCancel(context.Background())
	t.Cleanup(func() { m.scanCancel() })

	return m
}

func TestHandleScan_ValidCIDR(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`{"subnet":"192.168.1.0/24"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusAccepted, w.Body.String())
	}

	var resp models.ScanResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty scan ID")
	}
	if resp.Status != "running" {
		t.Errorf("status = %q, want running", resp.Status)
	}

	// Wait for background scan to finish.
	m.wg.Wait()
}

func TestHandleScan_InvalidCIDR(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`{"subnet":"not-a-cidr"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}

func TestHandleScan_MissingSubnet(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleScan_SubnetTooLarge(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`{"subnet":"10.0.0.0/8"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleScan_InvalidBody(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleListScans_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("GET", "/scans", http.NoBody)
	w := httptest.NewRecorder()
	m.handleListScans(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var scans []models.ScanResult
	if err := json.NewDecoder(w.Body).Decode(&scans); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(scans) != 0 {
		t.Errorf("scan count = %d, want 0", len(scans))
	}
}

func TestHandleListScans_WithData(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()

	// Create some scans.
	for i := 0; i < 3; i++ {
		_ = m.store.CreateScan(ctx, &models.ScanResult{Subnet: "10.0.0.0/24"})
	}

	req := httptest.NewRequest("GET", "/scans?limit=2&offset=0", http.NoBody)
	w := httptest.NewRecorder()
	m.handleListScans(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var scans []models.ScanResult
	_ = json.NewDecoder(w.Body).Decode(&scans)
	if len(scans) != 2 {
		t.Errorf("scan count = %d, want 2 (paginated)", len(scans))
	}
}

func TestHandleGetScan_Found(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()

	scan := &models.ScanResult{ID: "test-scan-1", Subnet: "10.0.0.0/24", Status: "completed"}
	_ = m.store.CreateScan(ctx, scan)

	// Use Go 1.22+ mux for path parameter support.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /scans/{id}", m.handleGetScan)

	req := httptest.NewRequest("GET", "/scans/test-scan-1", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var got models.ScanResult
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.ID != "test-scan-1" {
		t.Errorf("scan ID = %q, want test-scan-1", got.ID)
	}
}

func TestHandleGetScan_NotFound(t *testing.T) {
	m := newTestModule(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /scans/{id}", m.handleGetScan)

	req := httptest.NewRequest("GET", "/scans/nonexistent", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleTopology_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("GET", "/topology", http.NoBody)
	w := httptest.NewRecorder()
	m.handleTopology(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var graph TopologyGraph
	if err := json.NewDecoder(w.Body).Decode(&graph); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("nodes = %d, want 0", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Errorf("edges = %d, want 0", len(graph.Edges))
	}
}

func TestHandleTopology_WithData(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()

	d1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:00:00:00:00:01",
		Hostname: "router", DeviceType: models.DeviceTypeRouter,
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryARP,
	}
	d2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:00:00:00:00:02",
		Hostname: "switch", DeviceType: models.DeviceTypeSwitch,
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryARP,
	}
	_, _ = m.store.UpsertDevice(ctx, d1)
	_, _ = m.store.UpsertDevice(ctx, d2)
	_ = m.store.UpsertTopologyLink(ctx, &TopologyLink{
		SourceDeviceID: d1.ID, TargetDeviceID: d2.ID, LinkType: "arp",
	})

	req := httptest.NewRequest("GET", "/topology", http.NoBody)
	w := httptest.NewRecorder()
	m.handleTopology(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var graph TopologyGraph
	_ = json.NewDecoder(w.Body).Decode(&graph)
	if len(graph.Nodes) != 2 {
		t.Errorf("nodes = %d, want 2", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Errorf("edges = %d, want 1", len(graph.Edges))
	}
}

func TestWriteError_Format(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["detail"] != "test error" {
		t.Errorf("detail = %v, want test error", resp["detail"])
	}
	if resp["status"] != float64(http.StatusBadRequest) {
		t.Errorf("status = %v, want %d", resp["status"], http.StatusBadRequest)
	}
}
