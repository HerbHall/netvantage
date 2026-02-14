package pulse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// -- handleListChecks tests --

func TestHandleListChecks_Empty(t *testing.T) {
	m, _ := newTestModule(t)
	req := httptest.NewRequest(http.MethodGet, "/checks", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListChecks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var checks []Check
	if err := json.NewDecoder(w.Body).Decode(&checks); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(checks) != 0 {
		t.Errorf("len(checks) = %d, want 0", len(checks))
	}
}

func TestHandleListChecks_WithData(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	// Insert a disabled check -- ListAllChecks should return both.
	disabledCheck := &Check{
		ID:              "check-2",
		DeviceID:        "dev-2",
		CheckType:       "tcp",
		Target:          "192.168.1.2:22",
		IntervalSeconds: 60,
		Enabled:         false,
		CreatedAt:       now.Add(time.Second),
		UpdatedAt:       now.Add(time.Second),
	}
	if err := m.store.InsertCheck(context.Background(), disabledCheck); err != nil {
		t.Fatalf("insert disabled check: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/checks", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListChecks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var checks []Check
	if err := json.NewDecoder(w.Body).Decode(&checks); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(checks) != 2 {
		t.Fatalf("len(checks) = %d, want 2", len(checks))
	}
	if checks[0].ID != "check-1" {
		t.Errorf("checks[0].ID = %q, want %q", checks[0].ID, "check-1")
	}
}

func TestHandleListChecks_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/checks", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListChecks(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleCreateCheck tests --

func TestHandleCreateCheck_Success(t *testing.T) {
	m, _ := newTestModule(t)

	body := `{"device_id":"dev-1","check_type":"icmp","target":"192.168.1.1","interval_seconds":60}`
	req := httptest.NewRequest(http.MethodPost, "/checks", strings.NewReader(body))
	w := httptest.NewRecorder()

	m.handleCreateCheck(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var check Check
	if err := json.NewDecoder(w.Body).Decode(&check); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if check.DeviceID != "dev-1" {
		t.Errorf("check.DeviceID = %q, want %q", check.DeviceID, "dev-1")
	}
	if check.CheckType != "icmp" {
		t.Errorf("check.CheckType = %q, want %q", check.CheckType, "icmp")
	}
	if check.Target != "192.168.1.1" {
		t.Errorf("check.Target = %q, want %q", check.Target, "192.168.1.1")
	}
	if check.IntervalSeconds != 60 {
		t.Errorf("check.IntervalSeconds = %d, want %d", check.IntervalSeconds, 60)
	}
	if !check.Enabled {
		t.Error("check.Enabled = false, want true")
	}
	if check.ID == "" {
		t.Error("check.ID is empty, want non-empty")
	}
}

func TestHandleCreateCheck_DefaultInterval(t *testing.T) {
	m, _ := newTestModule(t)

	body := `{"device_id":"dev-1","check_type":"icmp","target":"192.168.1.1"}`
	req := httptest.NewRequest(http.MethodPost, "/checks", strings.NewReader(body))
	w := httptest.NewRecorder()

	m.handleCreateCheck(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var check Check
	if err := json.NewDecoder(w.Body).Decode(&check); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if check.IntervalSeconds != 30 {
		t.Errorf("check.IntervalSeconds = %d, want %d (default)", check.IntervalSeconds, 30)
	}
}

func TestHandleCreateCheck_InvalidType(t *testing.T) {
	m, _ := newTestModule(t)

	body := `{"device_id":"dev-1","check_type":"snmp","target":"192.168.1.1"}`
	req := httptest.NewRequest(http.MethodPost, "/checks", strings.NewReader(body))
	w := httptest.NewRecorder()

	m.handleCreateCheck(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCheck_InvalidTarget_TCP(t *testing.T) {
	m, _ := newTestModule(t)

	body := `{"device_id":"dev-1","check_type":"tcp","target":"no-port"}`
	req := httptest.NewRequest(http.MethodPost, "/checks", strings.NewReader(body))
	w := httptest.NewRecorder()

	m.handleCreateCheck(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCheck_InvalidTarget_HTTP(t *testing.T) {
	m, _ := newTestModule(t)

	body := `{"device_id":"dev-1","check_type":"http","target":"not-a-url"}`
	req := httptest.NewRequest(http.MethodPost, "/checks", strings.NewReader(body))
	w := httptest.NewRecorder()

	m.handleCreateCheck(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// -- handleUpdateCheck tests --

func TestHandleUpdateCheck_Success(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	body := `{"target":"10.0.0.1","interval_seconds":120}`
	req := httptest.NewRequest(http.MethodPut, "/checks/check-1", strings.NewReader(body))
	req.SetPathValue("id", "check-1")
	w := httptest.NewRecorder()

	m.handleUpdateCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var updated Check
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if updated.Target != "10.0.0.1" {
		t.Errorf("updated.Target = %q, want %q", updated.Target, "10.0.0.1")
	}
	if updated.IntervalSeconds != 120 {
		t.Errorf("updated.IntervalSeconds = %d, want %d", updated.IntervalSeconds, 120)
	}
}

func TestHandleUpdateCheck_NotFound(t *testing.T) {
	m, _ := newTestModule(t)

	body := `{"target":"10.0.0.1"}`
	req := httptest.NewRequest(http.MethodPut, "/checks/nonexistent", strings.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	m.handleUpdateCheck(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// -- handleDeleteCheck tests --

func TestHandleDeleteCheck_Success(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/checks/check-1", http.NoBody)
	req.SetPathValue("id", "check-1")
	w := httptest.NewRecorder()

	m.handleDeleteCheck(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify it's gone.
	got, err := m.store.GetCheck(context.Background(), "check-1")
	if err != nil {
		t.Fatalf("GetCheck after delete: %v", err)
	}
	if got != nil {
		t.Errorf("check still exists after delete")
	}
}

// -- handleToggleCheck tests --

func TestHandleToggleCheck_Success(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/checks/check-1/toggle", http.NoBody)
	req.SetPathValue("id", "check-1")
	w := httptest.NewRecorder()

	m.handleToggleCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var toggled Check
	if err := json.NewDecoder(w.Body).Decode(&toggled); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if toggled.Enabled {
		t.Error("toggled.Enabled = true, want false (was true before toggle)")
	}
}

func TestHandleToggleCheck_NotFound(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodPatch, "/checks/nonexistent/toggle", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	m.handleToggleCheck(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// -- handleDeviceChecks tests --

func TestHandleDeviceChecks_Found(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/checks/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceChecks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got Check
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "check-1" {
		t.Errorf("check.ID = %q, want %q", got.ID, "check-1")
	}
	if got.DeviceID != "dev-1" {
		t.Errorf("check.DeviceID = %q, want %q", got.DeviceID, "dev-1")
	}
}

func TestHandleDeviceChecks_NotFound(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/checks/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceChecks(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDeviceChecks_EmptyDeviceID(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/checks/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleDeviceChecks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceChecks_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/checks/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceChecks(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleDeviceResults tests --

func TestHandleDeviceResults_Empty(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/results/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var results []CheckResult
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestHandleDeviceResults_WithData(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	// Insert check first (foreign key constraint).
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	result := &CheckResult{
		CheckID:      "check-1",
		DeviceID:     "dev-1",
		Success:      true,
		LatencyMs:    12.5,
		PacketLoss:   0.0,
		ErrorMessage: "",
		CheckedAt:    now,
	}
	if err := m.store.InsertResult(context.Background(), result); err != nil {
		t.Fatalf("insert result: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/results/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var results []CheckResult
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].DeviceID != "dev-1" {
		t.Errorf("results[0].DeviceID = %q, want %q", results[0].DeviceID, "dev-1")
	}
	if results[0].Success != true {
		t.Errorf("results[0].Success = %v, want true", results[0].Success)
	}
}

func TestHandleDeviceResults_WithLimit(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	// Insert check first (foreign key constraint).
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	// Insert 10 results.
	for i := 0; i < 10; i++ {
		result := &CheckResult{
			CheckID:    "check-1",
			DeviceID:   "dev-1",
			Success:    true,
			LatencyMs:  float64(i) + 1.0,
			PacketLoss: 0.0,
			CheckedAt:  now.Add(time.Duration(i) * time.Second),
		}
		if err := m.store.InsertResult(context.Background(), result); err != nil {
			t.Fatalf("insert result %d: %v", i, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/results/dev-1?limit=5", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var results []CheckResult
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("len(results) = %d, want 5", len(results))
	}
}

func TestHandleDeviceResults_EmptyDeviceID(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/results/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceResults_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/results/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleListAlerts tests --

func TestHandleListAlerts_Empty(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/alerts", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var alerts []Alert
	if err := json.NewDecoder(w.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("len(alerts) = %d, want 0", len(alerts))
	}
}

func TestHandleListAlerts_WithData(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	// Insert check first (foreign key constraint).
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	alert := &Alert{
		ID:                  "alert-1",
		CheckID:             "check-1",
		DeviceID:            "dev-1",
		Severity:            "warning",
		Message:             "Device unreachable",
		TriggeredAt:         now,
		ResolvedAt:          nil,
		ConsecutiveFailures: 3,
	}
	if err := m.store.InsertAlert(context.Background(), alert); err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/alerts", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var alerts []Alert
	if err := json.NewDecoder(w.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(alerts))
	}
	if alerts[0].ID != "alert-1" {
		t.Errorf("alerts[0].ID = %q, want %q", alerts[0].ID, "alert-1")
	}
	if alerts[0].Message != "Device unreachable" {
		t.Errorf("alerts[0].Message = %q, want %q", alerts[0].Message, "Device unreachable")
	}
}

func TestHandleListAlerts_WithFilters(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check1 := &Check{
		ID: "check-1", DeviceID: "dev-1", CheckType: "icmp",
		Target: "192.168.1.1", IntervalSeconds: 60, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	check2 := &Check{
		ID: "check-2", DeviceID: "dev-2", CheckType: "icmp",
		Target: "192.168.1.2", IntervalSeconds: 60, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertCheck(context.Background(), check1); err != nil {
		t.Fatalf("insert check1: %v", err)
	}
	if err := m.store.InsertCheck(context.Background(), check2); err != nil {
		t.Fatalf("insert check2: %v", err)
	}

	alert1 := &Alert{
		ID: "alert-1", CheckID: "check-1", DeviceID: "dev-1",
		Severity: "warning", Message: "Device dev-1 unreachable",
		TriggeredAt: now, ConsecutiveFailures: 3,
	}
	alert2 := &Alert{
		ID: "alert-2", CheckID: "check-2", DeviceID: "dev-2",
		Severity: "critical", Message: "Device dev-2 unreachable",
		TriggeredAt: now.Add(time.Second), ConsecutiveFailures: 5,
	}
	if err := m.store.InsertAlert(context.Background(), alert1); err != nil {
		t.Fatalf("insert alert1: %v", err)
	}
	if err := m.store.InsertAlert(context.Background(), alert2); err != nil {
		t.Fatalf("insert alert2: %v", err)
	}

	// Filter by device_id.
	req := httptest.NewRequest(http.MethodGet, "/alerts?device_id=dev-1", http.NoBody)
	w := httptest.NewRecorder()
	m.handleListAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var alerts []Alert
	if err := json.NewDecoder(w.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(alerts))
	}
	if alerts[0].DeviceID != "dev-1" {
		t.Errorf("alerts[0].DeviceID = %q, want %q", alerts[0].DeviceID, "dev-1")
	}

	// Filter by severity.
	req = httptest.NewRequest(http.MethodGet, "/alerts?severity=critical", http.NoBody)
	w = httptest.NewRecorder()
	m.handleListAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if err := json.NewDecoder(w.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(alerts))
	}
	if alerts[0].Severity != "critical" {
		t.Errorf("alerts[0].Severity = %q, want %q", alerts[0].Severity, "critical")
	}
}

func TestHandleListAlerts_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/alerts", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListAlerts(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleGetAlert tests --

func TestHandleGetAlert_Found(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID: "check-1", DeviceID: "dev-1", CheckType: "icmp",
		Target: "192.168.1.1", IntervalSeconds: 60, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	alert := &Alert{
		ID: "alert-1", CheckID: "check-1", DeviceID: "dev-1",
		Severity: "critical", Message: "Host down",
		TriggeredAt: now, ConsecutiveFailures: 5,
	}
	if err := m.store.InsertAlert(context.Background(), alert); err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/alerts/alert-1", http.NoBody)
	req.SetPathValue("id", "alert-1")
	w := httptest.NewRecorder()

	m.handleGetAlert(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got Alert
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "alert-1" {
		t.Errorf("alert.ID = %q, want %q", got.ID, "alert-1")
	}
}

func TestHandleGetAlert_NotFound(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/alerts/nonexistent", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	m.handleGetAlert(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// -- handleAcknowledgeAlert tests --

func TestHandleAcknowledgeAlert_Success(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID: "check-1", DeviceID: "dev-1", CheckType: "icmp",
		Target: "192.168.1.1", IntervalSeconds: 60, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	alert := &Alert{
		ID: "alert-1", CheckID: "check-1", DeviceID: "dev-1",
		Severity: "warning", Message: "High latency",
		TriggeredAt: now, ConsecutiveFailures: 3,
	}
	if err := m.store.InsertAlert(context.Background(), alert); err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/alerts/alert-1/acknowledge", http.NoBody)
	req.SetPathValue("id", "alert-1")
	w := httptest.NewRecorder()

	m.handleAcknowledgeAlert(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got Alert
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.AcknowledgedAt == nil {
		t.Error("acknowledged_at should be set after acknowledging")
	}
}

// -- handleResolveAlert tests --

func TestHandleResolveAlert_Success(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID: "check-1", DeviceID: "dev-1", CheckType: "icmp",
		Target: "192.168.1.1", IntervalSeconds: 60, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	alert := &Alert{
		ID: "alert-1", CheckID: "check-1", DeviceID: "dev-1",
		Severity: "critical", Message: "Host down",
		TriggeredAt: now, ConsecutiveFailures: 5,
	}
	if err := m.store.InsertAlert(context.Background(), alert); err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/alerts/alert-1/resolve", http.NoBody)
	req.SetPathValue("id", "alert-1")
	w := httptest.NewRecorder()

	m.handleResolveAlert(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got Alert
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ResolvedAt == nil {
		t.Error("resolved_at should be set after resolving")
	}
}

// -- handleDeviceStatus tests --

func TestHandleDeviceStatus_WithData(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	// Insert check first (foreign key constraint).
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	result := &CheckResult{
		CheckID:      "check-1",
		DeviceID:     "dev-1",
		Success:      true,
		LatencyMs:    15.3,
		PacketLoss:   0.0,
		ErrorMessage: "",
		CheckedAt:    now,
	}
	if err := m.store.InsertResult(context.Background(), result); err != nil {
		t.Fatalf("insert result: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/status/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var status roles.MonitorStatus
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if status.DeviceID != "dev-1" {
		t.Errorf("status.DeviceID = %q, want %q", status.DeviceID, "dev-1")
	}
	if !status.Healthy {
		t.Errorf("status.Healthy = %v, want true", status.Healthy)
	}
	if status.Message == "" {
		t.Error("status.Message is empty, want non-empty")
	}
}

func TestHandleDeviceStatus_EmptyDeviceID(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/status/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleDeviceStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// -- pulseParseLimit tests --

func TestPulseParseLimit(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		defaultVal int
		want       int
	}{
		{
			name:       "no param returns default",
			query:      "",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "valid param",
			query:      "limit=50",
			defaultVal: 100,
			want:       50,
		},
		{
			name:       "out of range high returns default",
			query:      "limit=2000",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "out of range low returns default",
			query:      "limit=0",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "negative returns default",
			query:      "limit=-10",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "non-numeric returns default",
			query:      "limit=abc",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "max allowed value",
			query:      "limit=1000",
			defaultVal: 100,
			want:       1000,
		},
		{
			name:       "min allowed value",
			query:      "limit=1",
			defaultVal: 100,
			want:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, http.NoBody)
			got := pulseParseLimit(req, tt.defaultVal)
			if got != tt.want {
				t.Errorf("pulseParseLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}
