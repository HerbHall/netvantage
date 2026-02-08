package pulse

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// alerterTestStore creates an in-memory PulseStore for alerter tests.
// Named differently from newTestStore to avoid redeclaration in the same test binary.
func alerterTestStore(t *testing.T) *PulseStore {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()
	if err := db.Migrate(ctx, "pulse", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewPulseStore(db.DB())
}

// mockEventBus collects published events for verification.
type mockEventBus struct {
	events []plugin.Event
}

func (m *mockEventBus) Publish(_ context.Context, event plugin.Event) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventBus) PublishAsync(_ context.Context, event plugin.Event) {
	m.events = append(m.events, event)
}

func (m *mockEventBus) Subscribe(_ string, _ plugin.EventHandler) func() {
	return func() {}
}

func (m *mockEventBus) SubscribeAll(_ plugin.EventHandler) func() {
	return func() {}
}

// makeTestCheck inserts a check into the store and returns it.
// Named differently from insertTestCheck in store_test.go to avoid redeclaration.
func makeTestCheck(t *testing.T, ps *PulseStore, deviceID, checkType, target string) Check {
	t.Helper()
	ctx := context.Background()
	check := Check{
		ID:              "check-" + deviceID + "-" + checkType,
		DeviceID:        deviceID,
		CheckType:       checkType,
		Target:          target,
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	if err := ps.InsertCheck(ctx, &check); err != nil {
		t.Fatalf("insert check: %v", err)
	}
	return check
}

func TestAlerter_SingleFailure_NoAlert(t *testing.T) {
	ps := alerterTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	check := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	ctx := context.Background()

	result := &CheckResult{
		CheckID:      check.ID,
		DeviceID:     check.DeviceID,
		Success:      false,
		ErrorMessage: "timeout",
		CheckedAt:    time.Now().UTC(),
	}

	alerter.ProcessResult(ctx, check, result)

	// No alert should be created (below threshold).
	alert, err := ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if alert != nil {
		t.Errorf("got alert = %v, want nil (below threshold)", alert)
	}

	// No events should be published.
	if len(bus.events) != 0 {
		t.Errorf("got %d events, want 0", len(bus.events))
	}
}

func TestAlerter_ConsecutiveFailures_TriggersAlert(t *testing.T) {
	ps := alerterTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	check := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	ctx := context.Background()

	result := &CheckResult{
		CheckID:      check.ID,
		DeviceID:     check.DeviceID,
		Success:      false,
		ErrorMessage: "timeout",
		CheckedAt:    time.Now().UTC(),
	}

	// Send threshold failures.
	for i := 0; i < threshold; i++ {
		alerter.ProcessResult(ctx, check, result)
	}

	// Alert should be created.
	alert, err := ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if alert == nil {
		t.Fatal("got nil alert, want alert to be created")
	}

	if alert.CheckID != check.ID {
		t.Errorf("alert.CheckID = %q, want %q", alert.CheckID, check.ID)
	}
	if alert.DeviceID != check.DeviceID {
		t.Errorf("alert.DeviceID = %q, want %q", alert.DeviceID, check.DeviceID)
	}
	if alert.Severity != "warning" {
		t.Errorf("alert.Severity = %q, want %q", alert.Severity, "warning")
	}
	if alert.ConsecutiveFailures != threshold {
		t.Errorf("alert.ConsecutiveFailures = %d, want %d", alert.ConsecutiveFailures, threshold)
	}
	if alert.ResolvedAt != nil {
		t.Errorf("alert.ResolvedAt = %v, want nil", alert.ResolvedAt)
	}

	// Event should be published.
	if len(bus.events) != 1 {
		t.Fatalf("got %d events, want 1", len(bus.events))
	}
	if bus.events[0].Topic != TopicAlertTriggered {
		t.Errorf("event.Topic = %q, want %q", bus.events[0].Topic, TopicAlertTriggered)
	}
	if bus.events[0].Source != "pulse" {
		t.Errorf("event.Source = %q, want %q", bus.events[0].Source, "pulse")
	}
}

func TestAlerter_Success_ResolvesAlert(t *testing.T) {
	ps := alerterTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	check := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	ctx := context.Background()

	failureResult := &CheckResult{
		CheckID:      check.ID,
		DeviceID:     check.DeviceID,
		Success:      false,
		ErrorMessage: "timeout",
		CheckedAt:    time.Now().UTC(),
	}

	// Trigger alert.
	for i := 0; i < threshold; i++ {
		alerter.ProcessResult(ctx, check, failureResult)
	}

	// Verify alert exists.
	alert, err := ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if alert == nil {
		t.Fatal("got nil alert, want alert to exist")
	}

	// Reset event bus.
	bus.events = nil

	// Send success.
	successResult := &CheckResult{
		CheckID:   check.ID,
		DeviceID:  check.DeviceID,
		Success:   true,
		CheckedAt: time.Now().UTC(),
	}
	alerter.ProcessResult(ctx, check, successResult)

	// Alert should be resolved.
	alert, err = ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert after success: %v", err)
	}
	if alert != nil {
		t.Errorf("got alert = %v, want nil (should be resolved)", alert)
	}

	// Resolved event should be published.
	if len(bus.events) != 1 {
		t.Fatalf("got %d events, want 1", len(bus.events))
	}
	if bus.events[0].Topic != TopicAlertResolved {
		t.Errorf("event.Topic = %q, want %q", bus.events[0].Topic, TopicAlertResolved)
	}
}

func TestAlerter_Success_ResetsCounter(t *testing.T) {
	ps := alerterTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	check := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	ctx := context.Background()

	failureResult := &CheckResult{
		CheckID:      check.ID,
		DeviceID:     check.DeviceID,
		Success:      false,
		ErrorMessage: "timeout",
		CheckedAt:    time.Now().UTC(),
	}

	// Send threshold-1 failures.
	for i := 0; i < threshold-1; i++ {
		alerter.ProcessResult(ctx, check, failureResult)
	}

	// No alert should exist.
	alert, err := ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if alert != nil {
		t.Errorf("got alert = %v, want nil (below threshold)", alert)
	}

	// Send success to reset counter.
	successResult := &CheckResult{
		CheckID:   check.ID,
		DeviceID:  check.DeviceID,
		Success:   true,
		CheckedAt: time.Now().UTC(),
	}
	alerter.ProcessResult(ctx, check, successResult)

	// Send threshold failures again.
	for i := 0; i < threshold; i++ {
		alerter.ProcessResult(ctx, check, failureResult)
	}

	// Alert should be created only after threshold failures AFTER reset.
	alert, err = ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert after reset: %v", err)
	}
	if alert == nil {
		t.Fatal("got nil alert, want alert to be created after reset")
	}
	if alert.ConsecutiveFailures != threshold {
		t.Errorf("alert.ConsecutiveFailures = %d, want %d", alert.ConsecutiveFailures, threshold)
	}
}

func TestAlerter_NoDoubleAlert(t *testing.T) {
	ps := alerterTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	check := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	ctx := context.Background()

	failureResult := &CheckResult{
		CheckID:      check.ID,
		DeviceID:     check.DeviceID,
		Success:      false,
		ErrorMessage: "timeout",
		CheckedAt:    time.Now().UTC(),
	}

	// Send threshold + 5 failures.
	for i := 0; i < threshold+5; i++ {
		alerter.ProcessResult(ctx, check, failureResult)
	}

	// Only one alert triggered event should be published.
	alertEvents := 0
	for _, e := range bus.events {
		if e.Topic == TopicAlertTriggered {
			alertEvents++
		}
	}
	if alertEvents != 1 {
		t.Errorf("got %d alert.triggered events, want 1", alertEvents)
	}
}

func TestAlerter_SeverityWarning(t *testing.T) {
	ps := alerterTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	check := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	ctx := context.Background()

	failureResult := &CheckResult{
		CheckID:      check.ID,
		DeviceID:     check.DeviceID,
		Success:      false,
		ErrorMessage: "timeout",
		CheckedAt:    time.Now().UTC(),
	}

	// Send exactly threshold failures.
	for i := 0; i < threshold; i++ {
		alerter.ProcessResult(ctx, check, failureResult)
	}

	alert, err := ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if alert == nil {
		t.Fatal("got nil alert, want alert to exist")
	}

	if alert.Severity != "warning" {
		t.Errorf("alert.Severity = %q, want %q", alert.Severity, "warning")
	}
}

func TestAlerter_SeverityCritical_FirstAlertAfterManyFailures(t *testing.T) {
	ps := alerterTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	check := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	ctx := context.Background()

	// Manually set failure counter to simulate many failures before alert creation.
	// This tests the code path where count >= threshold*2 when the alert is first created.
	alerter.mu.Lock()
	alerter.failures[check.ID] = threshold*2 - 1
	alerter.mu.Unlock()

	failureResult := &CheckResult{
		CheckID:      check.ID,
		DeviceID:     check.DeviceID,
		Success:      false,
		ErrorMessage: "timeout",
		CheckedAt:    time.Now().UTC(),
	}

	// This failure will push count to threshold*2, and since no alert exists yet,
	// it will create one with "critical" severity.
	alerter.ProcessResult(ctx, check, failureResult)

	alert, err := ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if alert == nil {
		t.Fatal("got nil alert, want alert to exist")
	}

	if alert.Severity != "critical" {
		t.Errorf("alert.Severity = %q, want %q", alert.Severity, "critical")
	}
	if alert.ConsecutiveFailures != threshold*2 {
		t.Errorf("alert.ConsecutiveFailures = %d, want %d", alert.ConsecutiveFailures, threshold*2)
	}
}

func TestAlerter_NilBus(t *testing.T) {
	ps := alerterTestStore(t)
	threshold := 3
	alerter := NewAlerter(ps, nil, threshold, zap.NewNop())

	check := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	ctx := context.Background()

	failureResult := &CheckResult{
		CheckID:      check.ID,
		DeviceID:     check.DeviceID,
		Success:      false,
		ErrorMessage: "timeout",
		CheckedAt:    time.Now().UTC(),
	}

	// Trigger alert with nil bus (should not panic).
	for i := 0; i < threshold; i++ {
		alerter.ProcessResult(ctx, check, failureResult)
	}

	// Alert should still be created.
	alert, err := ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if alert == nil {
		t.Fatal("got nil alert, want alert to exist")
	}

	// Resolve alert with nil bus (should not panic).
	successResult := &CheckResult{
		CheckID:   check.ID,
		DeviceID:  check.DeviceID,
		Success:   true,
		CheckedAt: time.Now().UTC(),
	}
	alerter.ProcessResult(ctx, check, successResult)

	// Alert should be resolved.
	alert, err = ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert after success: %v", err)
	}
	if alert != nil {
		t.Errorf("got alert = %v, want nil (should be resolved)", alert)
	}
}

func TestAlerter_MultipleChecks(t *testing.T) {
	ps := alerterTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	check1 := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	check2 := makeTestCheck(t, ps, "device2", "ping", "192.168.1.2")
	ctx := context.Background()

	// Fail check1 threshold-1 times.
	for i := 0; i < threshold-1; i++ {
		alerter.ProcessResult(ctx, check1, &CheckResult{
			CheckID:      check1.ID,
			DeviceID:     check1.DeviceID,
			Success:      false,
			ErrorMessage: "timeout",
			CheckedAt:    time.Now().UTC(),
		})
	}

	// Fail check2 threshold times.
	for i := 0; i < threshold; i++ {
		alerter.ProcessResult(ctx, check2, &CheckResult{
			CheckID:      check2.ID,
			DeviceID:     check2.DeviceID,
			Success:      false,
			ErrorMessage: "timeout",
			CheckedAt:    time.Now().UTC(),
		})
	}

	// Check1 should have no alert.
	alert1, err := ps.GetActiveAlert(ctx, check1.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert check1: %v", err)
	}
	if alert1 != nil {
		t.Errorf("check1: got alert = %v, want nil", alert1)
	}

	// Check2 should have an alert.
	alert2, err := ps.GetActiveAlert(ctx, check2.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert check2: %v", err)
	}
	if alert2 == nil {
		t.Fatal("check2: got nil alert, want alert to exist")
	}
	if alert2.CheckID != check2.ID {
		t.Errorf("check2: alert.CheckID = %q, want %q", alert2.CheckID, check2.ID)
	}
}

func TestAlerter_ErrorMessage(t *testing.T) {
	ps := alerterTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	check := makeTestCheck(t, ps, "device1", "ping", "192.168.1.1")
	ctx := context.Background()

	customError := "connection refused on port 22"
	failureResult := &CheckResult{
		CheckID:      check.ID,
		DeviceID:     check.DeviceID,
		Success:      false,
		ErrorMessage: customError,
		CheckedAt:    time.Now().UTC(),
	}

	// Trigger alert.
	for i := 0; i < threshold; i++ {
		alerter.ProcessResult(ctx, check, failureResult)
	}

	alert, err := ps.GetActiveAlert(ctx, check.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if alert == nil {
		t.Fatal("got nil alert, want alert to exist")
	}

	// Alert message should use the custom error message.
	if alert.Message != customError {
		t.Errorf("alert.Message = %q, want %q", alert.Message, customError)
	}
}
