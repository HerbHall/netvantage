package pulse

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Tests for the Checker interface and ICMPChecker implementation.
//
// Testing Strategy:
//  - NewICMPChecker: Verify constructor sets fields correctly
//  - mockChecker: Reusable test double for integration tests in other files
//  - Interface tests: Verify both ICMPChecker and mockChecker implement Checker
//  - Contract tests: Success, failure, error, and context cancellation scenarios
//
// Note: ICMPChecker.Check() requires network permissions and can't be unit tested
// without real ICMP access. The mockChecker provides a testable implementation
// of the Checker interface for use in scheduler, alerter, and integration tests.

// mockChecker is a configurable mock implementation of the Checker interface.
// It can be configured to return specific results, errors, or respect context cancellation.
type mockChecker struct {
	result      *CheckResult
	err         error
	delay       time.Duration
	checkTarget bool // If true, validate that target matches expectedTarget
	targetWant  string
}

// newMockChecker creates a new mock checker that returns the given result and error.
// If delay > 0, the check will sleep for that duration (useful for testing context cancellation).
func newMockChecker(result *CheckResult, err error) *mockChecker {
	return &mockChecker{
		result: result,
		err:    err,
	}
}

// withDelay configures the mock to sleep for the given duration before returning.
func (m *mockChecker) withDelay(d time.Duration) *mockChecker {
	m.delay = d
	return m
}

// withTargetValidation configures the mock to validate that the target matches the expected value.
func (m *mockChecker) withTargetValidation(target string) *mockChecker {
	m.checkTarget = true
	m.targetWant = target
	return m
}

// Check implements the Checker interface.
func (m *mockChecker) Check(ctx context.Context, target string) (*CheckResult, error) {
	if m.checkTarget && target != m.targetWant {
		return nil, errors.New("unexpected target")
	}

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return &CheckResult{
				Success:      false,
				PacketLoss:   1.0,
				ErrorMessage: "check cancelled",
				CheckedAt:    time.Now().UTC(),
			}, nil
		}
	}

	return m.result, m.err
}

// Compile-time interface guard.
var _ Checker = (*mockChecker)(nil)

func TestNewICMPChecker(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		count       int
		wantTimeout time.Duration
		wantCount   int
	}{
		{
			name:        "default values",
			timeout:     5 * time.Second,
			count:       3,
			wantTimeout: 5 * time.Second,
			wantCount:   3,
		},
		{
			name:        "short timeout",
			timeout:     1 * time.Second,
			count:       1,
			wantTimeout: 1 * time.Second,
			wantCount:   1,
		},
		{
			name:        "high count",
			timeout:     10 * time.Second,
			count:       10,
			wantTimeout: 10 * time.Second,
			wantCount:   10,
		},
		{
			name:        "zero values",
			timeout:     0,
			count:       0,
			wantTimeout: 0,
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewICMPChecker(tt.timeout, tt.count)

			if checker == nil {
				t.Fatal("NewICMPChecker() returned nil")
			}

			if checker.timeout != tt.wantTimeout {
				t.Errorf("timeout = %v, want %v", checker.timeout, tt.wantTimeout)
			}

			if checker.count != tt.wantCount {
				t.Errorf("count = %v, want %v", checker.count, tt.wantCount)
			}
		})
	}
}

func TestMockChecker_Success(t *testing.T) {
	now := time.Now().UTC()
	want := &CheckResult{
		CheckID:    "check-1",
		DeviceID:   "device-1",
		Success:    true,
		LatencyMs:  12.5,
		PacketLoss: 0.0,
		CheckedAt:  now,
	}

	checker := newMockChecker(want, nil)
	got, err := checker.Check(context.Background(), "192.0.2.1")

	if err != nil {
		t.Errorf("Check() error = %v, want nil", err)
	}

	if got != want {
		t.Errorf("Check() = %+v, want %+v", got, want)
	}
}

func TestMockChecker_Failure(t *testing.T) {
	now := time.Now().UTC()
	want := &CheckResult{
		CheckID:      "check-2",
		DeviceID:     "device-2",
		Success:      false,
		LatencyMs:    0,
		PacketLoss:   1.0,
		ErrorMessage: "all packets lost",
		CheckedAt:    now,
	}

	checker := newMockChecker(want, nil)
	got, err := checker.Check(context.Background(), "192.0.2.1")

	if err != nil {
		t.Errorf("Check() error = %v, want nil", err)
	}

	if got != want {
		t.Errorf("Check() = %+v, want %+v", got, want)
	}

	if got.Success {
		t.Error("Check() Success = true, want false")
	}

	if got.PacketLoss != 1.0 {
		t.Errorf("Check() PacketLoss = %v, want 1.0", got.PacketLoss)
	}

	if got.ErrorMessage == "" {
		t.Error("Check() ErrorMessage is empty, want non-empty")
	}
}

func TestMockChecker_Error(t *testing.T) {
	wantErr := errors.New("network unreachable")

	checker := newMockChecker(nil, wantErr)
	got, err := checker.Check(context.Background(), "192.0.2.1")

	if err == nil {
		t.Error("Check() error = nil, want error")
	}

	if err != wantErr {
		t.Errorf("Check() error = %v, want %v", err, wantErr)
	}

	if got != nil {
		t.Errorf("Check() = %+v, want nil", got)
	}
}

func TestMockChecker_ContextCancelled(t *testing.T) {
	checker := newMockChecker(
		&CheckResult{Success: true, LatencyMs: 10.0},
		nil,
	).withDelay(500 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	got, err := checker.Check(ctx, "192.0.2.1")
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Check() error = %v, want nil (mock returns result on cancellation)", err)
	}

	if got == nil {
		t.Fatal("Check() returned nil result")
	}

	if got.Success {
		t.Error("Check() Success = true, want false (context cancelled)")
	}

	if got.PacketLoss != 1.0 {
		t.Errorf("Check() PacketLoss = %v, want 1.0", got.PacketLoss)
	}

	if got.ErrorMessage != "check cancelled" {
		t.Errorf("Check() ErrorMessage = %q, want %q", got.ErrorMessage, "check cancelled")
	}

	// Should return quickly after context cancellation, not wait for full delay.
	if elapsed > 200*time.Millisecond {
		t.Errorf("Check() took %v, want < 200ms (should respect context cancellation)", elapsed)
	}
}

func TestMockChecker_TargetValidation(t *testing.T) {
	tests := []struct {
		name       string
		targetWant string
		targetGot  string
		wantErr    bool
	}{
		{
			name:       "matching target",
			targetWant: "192.0.2.1",
			targetGot:  "192.0.2.1",
			wantErr:    false,
		},
		{
			name:       "mismatched target",
			targetWant: "192.0.2.1",
			targetGot:  "192.0.2.2",
			wantErr:    true,
		},
		{
			name:       "hostname validation",
			targetWant: "example.com",
			targetGot:  "example.com",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CheckResult{
				Success:   true,
				LatencyMs: 5.0,
				CheckedAt: time.Now().UTC(),
			}

			checker := newMockChecker(result, nil).withTargetValidation(tt.targetWant)
			got, err := checker.Check(context.Background(), tt.targetGot)

			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != result {
				t.Errorf("Check() = %+v, want %+v", got, result)
			}
		})
	}
}

func TestMockChecker_MultipleResults(t *testing.T) {
	// Test that the mock can be called multiple times and returns the same configured result.
	result := &CheckResult{
		Success:   true,
		LatencyMs: 8.5,
		CheckedAt: time.Now().UTC(),
	}

	checker := newMockChecker(result, nil)

	for i := 0; i < 5; i++ {
		got, err := checker.Check(context.Background(), "192.0.2.1")
		if err != nil {
			t.Errorf("iteration %d: Check() error = %v, want nil", i, err)
		}
		if got != result {
			t.Errorf("iteration %d: Check() = %+v, want %+v", i, got, result)
		}
	}
}

func TestMockChecker_InterfaceCompliance(t *testing.T) {
	// Verify that mockChecker implements the Checker interface.
	var _ Checker = (*mockChecker)(nil)

	// Test that mockChecker can be used as a Checker.
	var checker Checker = newMockChecker(
		&CheckResult{
			Success:   true,
			LatencyMs: 15.0,
			CheckedAt: time.Now().UTC(),
		},
		nil,
	)

	got, err := checker.Check(context.Background(), "192.0.2.1")
	if err != nil {
		t.Errorf("Check() error = %v, want nil", err)
	}

	if got == nil {
		t.Fatal("Check() returned nil result")
	}

	if !got.Success {
		t.Error("Check() Success = false, want true")
	}
}

func TestMockChecker_PartialPacketLoss(t *testing.T) {
	result := &CheckResult{
		CheckID:    "check-3",
		DeviceID:   "device-3",
		Success:    true,
		LatencyMs:  20.0,
		PacketLoss: 0.25, // 25% packet loss
		CheckedAt:  time.Now().UTC(),
	}

	checker := newMockChecker(result, nil)
	got, err := checker.Check(context.Background(), "192.0.2.1")

	if err != nil {
		t.Errorf("Check() error = %v, want nil", err)
	}

	if got.PacketLoss != 0.25 {
		t.Errorf("Check() PacketLoss = %v, want 0.25", got.PacketLoss)
	}

	// Success can be true even with partial packet loss.
	if !got.Success {
		t.Error("Check() Success = false, want true")
	}
}

func TestICMPChecker_InterfaceCompliance(t *testing.T) {
	// Verify that ICMPChecker implements the Checker interface.
	var _ Checker = (*ICMPChecker)(nil)

	// Test that ICMPChecker can be used as a Checker.
	var checker Checker = NewICMPChecker(5*time.Second, 3)

	if checker == nil {
		t.Fatal("NewICMPChecker() returned nil")
	}

	// We can't test actual ICMP functionality without network permissions,
	// but we can verify the interface contract is satisfied.
	_, ok := checker.(*ICMPChecker)
	if !ok {
		t.Error("type assertion to *ICMPChecker failed")
	}
}
