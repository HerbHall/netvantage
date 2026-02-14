package pulse

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

// -- Store tests: QueryMetrics --

func seedMetricsData(t *testing.T, s *PulseStore, deviceID string, count int, start time.Time, interval time.Duration) {
	t.Helper()
	ctx := context.Background()

	// Insert a check for the device.
	check := &Check{
		ID:              "chk-" + deviceID,
		DeviceID:        deviceID,
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       start,
		UpdatedAt:       start,
	}
	if err := s.InsertCheck(ctx, check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	for i := 0; i < count; i++ {
		ts := start.Add(time.Duration(i) * interval)
		success := i%5 != 0 // every 5th check fails
		successInt := true
		if !success {
			successInt = false
		}
		r := &CheckResult{
			CheckID:    "chk-" + deviceID,
			DeviceID:   deviceID,
			Success:    successInt,
			LatencyMs:  10.0 + float64(i%20)*5.0, // 10-105ms range
			PacketLoss: float64(i%10) * 0.05,      // 0.0-0.45 range
			CheckedAt:  ts,
		}
		if !success {
			r.ErrorMessage = "timeout"
		}
		if err := s.InsertResult(ctx, r); err != nil {
			t.Fatalf("insert result %d: %v", i, err)
		}
	}
}

func TestQueryMetrics_LatencyRanges(t *testing.T) {
	s := testStore(t)

	// Seed 120 results over 2 hours (1 per minute).
	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(-2 * time.Hour)
	seedMetricsData(t, s, "dev-1", 120, start, time.Minute)

	tests := []struct {
		name       string
		timeRange  string
		wantBucket int // expected bucket size in seconds
		wantMin    int // minimum expected points
	}{
		{
			name:       "1h range uses 1-min buckets",
			timeRange:  "1h",
			wantBucket: 60,
			wantMin:    30, // at least 30 points in last hour
		},
		{
			name:       "6h range uses 1-min buckets",
			timeRange:  "6h",
			wantBucket: 60,
			wantMin:    60, // all 120 results fall within 6h
		},
		{
			name:       "24h range uses 1-min buckets",
			timeRange:  "24h",
			wantBucket: 60,
			wantMin:    60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series, err := s.QueryMetrics(context.Background(), "dev-1", "latency", tt.timeRange)
			if err != nil {
				t.Fatalf("QueryMetrics: %v", err)
			}
			if series.DeviceID != "dev-1" {
				t.Errorf("DeviceID = %q, want %q", series.DeviceID, "dev-1")
			}
			if series.Metric != "latency" {
				t.Errorf("Metric = %q, want %q", series.Metric, "latency")
			}
			if series.Range != tt.timeRange {
				t.Errorf("Range = %q, want %q", series.Range, tt.timeRange)
			}
			if len(series.Points) < tt.wantMin {
				t.Errorf("len(Points) = %d, want >= %d", len(series.Points), tt.wantMin)
			}

			// Verify points are ordered by timestamp ASC.
			for i := 1; i < len(series.Points); i++ {
				if !series.Points[i].Timestamp.After(series.Points[i-1].Timestamp) {
					t.Errorf("points not in ASC order at index %d: %v <= %v",
						i, series.Points[i].Timestamp, series.Points[i-1].Timestamp)
					break
				}
			}

			// Verify values are reasonable averages (between 10 and 105).
			for i, p := range series.Points {
				if p.Value < 10.0 || p.Value > 105.0 {
					t.Errorf("points[%d].Value = %f, want between 10 and 105", i, p.Value)
					break
				}
			}
		})
	}
}

func TestQueryMetrics_LargerRanges(t *testing.T) {
	s := testStore(t)

	// Seed 1000 results over 10 days (every ~14.4 minutes).
	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(-10 * 24 * time.Hour)
	seedMetricsData(t, s, "dev-2", 1000, start, 864*time.Second)

	tests := []struct {
		name       string
		timeRange  string
		wantBucket int
	}{
		{
			name:       "7d range uses 5-min buckets",
			timeRange:  "7d",
			wantBucket: 300,
		},
		{
			name:       "30d range uses 1-hour buckets",
			timeRange:  "30d",
			wantBucket: 3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series, err := s.QueryMetrics(context.Background(), "dev-2", "latency", tt.timeRange)
			if err != nil {
				t.Fatalf("QueryMetrics: %v", err)
			}
			if len(series.Points) == 0 {
				t.Fatal("expected at least 1 point")
			}

			// Verify bucket alignment: timestamps should be divisible by bucket size.
			for i, p := range series.Points {
				unix := p.Timestamp.Unix()
				if unix%int64(tt.wantBucket) != 0 {
					t.Errorf("points[%d].Timestamp unix=%d not aligned to %d-second bucket",
						i, unix, tt.wantBucket)
					break
				}
			}
		})
	}
}

func TestQueryMetrics_PacketLoss(t *testing.T) {
	s := testStore(t)

	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(-1 * time.Hour)
	seedMetricsData(t, s, "dev-pl", 60, start, time.Minute)

	series, err := s.QueryMetrics(context.Background(), "dev-pl", "packet_loss", "1h")
	if err != nil {
		t.Fatalf("QueryMetrics: %v", err)
	}
	if series.Metric != "packet_loss" {
		t.Errorf("Metric = %q, want %q", series.Metric, "packet_loss")
	}
	if len(series.Points) == 0 {
		t.Fatal("expected at least 1 point")
	}

	// Packet loss values should be between 0.0 and 0.45 (our seed data range).
	for i, p := range series.Points {
		if p.Value < 0.0 || p.Value > 0.5 {
			t.Errorf("points[%d].Value = %f, want between 0.0 and 0.5", i, p.Value)
			break
		}
	}
}

func TestQueryMetrics_SuccessRate(t *testing.T) {
	s := testStore(t)

	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(-1 * time.Hour)
	seedMetricsData(t, s, "dev-sr", 60, start, time.Minute)

	series, err := s.QueryMetrics(context.Background(), "dev-sr", "success_rate", "1h")
	if err != nil {
		t.Fatalf("QueryMetrics: %v", err)
	}
	if series.Metric != "success_rate" {
		t.Errorf("Metric = %q, want %q", series.Metric, "success_rate")
	}
	if len(series.Points) == 0 {
		t.Fatal("expected at least 1 point")
	}

	// Success rate should be between 0 and 100 (percentage).
	for i, p := range series.Points {
		if p.Value < 0.0 || p.Value > 100.0 {
			t.Errorf("points[%d].Value = %f, want between 0 and 100", i, p.Value)
			break
		}
	}
}

func TestQueryMetrics_InvalidMetric(t *testing.T) {
	s := testStore(t)

	_, err := s.QueryMetrics(context.Background(), "dev-1", "invalid_metric", "24h")
	if err == nil {
		t.Fatal("expected error for invalid metric, got nil")
	}
}

func TestQueryMetrics_InvalidRange(t *testing.T) {
	s := testStore(t)

	_, err := s.QueryMetrics(context.Background(), "dev-1", "latency", "invalid_range")
	if err == nil {
		t.Fatal("expected error for invalid range, got nil")
	}
}

func TestQueryMetrics_NoData(t *testing.T) {
	s := testStore(t)

	series, err := s.QueryMetrics(context.Background(), "nonexistent-device", "latency", "24h")
	if err != nil {
		t.Fatalf("QueryMetrics: %v", err)
	}
	if series == nil {
		t.Fatal("expected non-nil series")
	}
	if len(series.Points) != 0 {
		t.Errorf("len(Points) = %d, want 0 for no data", len(series.Points))
	}
}

func TestQueryMetrics_AggregationAccuracy(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Insert a check.
	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID:              "chk-agg",
		DeviceID:        "dev-agg",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now.Add(-10 * time.Minute),
		UpdatedAt:       now.Add(-10 * time.Minute),
	}
	if err := s.InsertCheck(ctx, check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	// Insert 3 results in the same 1-minute bucket with known values.
	// Bucket will be aligned to the minute boundary.
	bucketStart := now.Add(-5 * time.Minute).Truncate(time.Minute)
	results := []struct {
		latency    float64
		packetLoss float64
		success    bool
		offset     time.Duration
	}{
		{latency: 10.0, packetLoss: 0.0, success: true, offset: 0},
		{latency: 20.0, packetLoss: 0.1, success: true, offset: 10 * time.Second},
		{latency: 30.0, packetLoss: 0.2, success: false, offset: 20 * time.Second},
	}

	for _, r := range results {
		cr := &CheckResult{
			CheckID:    "chk-agg",
			DeviceID:   "dev-agg",
			Success:    r.success,
			LatencyMs:  r.latency,
			PacketLoss: r.packetLoss,
			CheckedAt:  bucketStart.Add(r.offset),
		}
		if err := s.InsertResult(ctx, cr); err != nil {
			t.Fatalf("insert result: %v", err)
		}
	}

	// Test latency aggregation: AVG(10, 20, 30) = 20.
	series, err := s.QueryMetrics(ctx, "dev-agg", "latency", "1h")
	if err != nil {
		t.Fatalf("QueryMetrics latency: %v", err)
	}
	if len(series.Points) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(series.Points))
	}
	if math.Abs(series.Points[0].Value-20.0) > 0.01 {
		t.Errorf("latency AVG = %f, want 20.0", series.Points[0].Value)
	}

	// Test packet_loss aggregation: AVG(0.0, 0.1, 0.2) = 0.1.
	series, err = s.QueryMetrics(ctx, "dev-agg", "packet_loss", "1h")
	if err != nil {
		t.Fatalf("QueryMetrics packet_loss: %v", err)
	}
	if len(series.Points) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(series.Points))
	}
	if math.Abs(series.Points[0].Value-0.1) > 0.01 {
		t.Errorf("packet_loss AVG = %f, want 0.1", series.Points[0].Value)
	}

	// Test success_rate aggregation: (2/3) * 100 = 66.67%.
	series, err = s.QueryMetrics(ctx, "dev-agg", "success_rate", "1h")
	if err != nil {
		t.Fatalf("QueryMetrics success_rate: %v", err)
	}
	if len(series.Points) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(series.Points))
	}
	expected := 2.0 / 3.0 * 100.0
	if math.Abs(series.Points[0].Value-expected) > 0.1 {
		t.Errorf("success_rate = %f, want ~%f", series.Points[0].Value, expected)
	}
}

// -- Handler tests: handleDeviceMetrics --

func TestHandleDeviceMetrics_Success(t *testing.T) {
	m, _ := newTestModule(t)

	// Seed data.
	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(-30 * time.Minute)
	check := &Check{
		ID:              "chk-hm",
		DeviceID:        "dev-hm",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       start,
		UpdatedAt:       start,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}
	for i := 0; i < 30; i++ {
		r := &CheckResult{
			CheckID:   "chk-hm",
			DeviceID:  "dev-hm",
			Success:   true,
			LatencyMs: 15.0 + float64(i),
			CheckedAt: start.Add(time.Duration(i) * time.Minute),
		}
		if err := m.store.InsertResult(context.Background(), r); err != nil {
			t.Fatalf("insert result %d: %v", i, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics/dev-hm?metric=latency&range=1h", http.NoBody)
	req.SetPathValue("device_id", "dev-hm")
	w := httptest.NewRecorder()

	m.handleDeviceMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var series MetricSeries
	if err := json.NewDecoder(w.Body).Decode(&series); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if series.DeviceID != "dev-hm" {
		t.Errorf("DeviceID = %q, want %q", series.DeviceID, "dev-hm")
	}
	if series.Metric != "latency" {
		t.Errorf("Metric = %q, want %q", series.Metric, "latency")
	}
	if series.Range != "1h" {
		t.Errorf("Range = %q, want %q", series.Range, "1h")
	}
	if len(series.Points) == 0 {
		t.Error("expected at least 1 data point")
	}
}

func TestHandleDeviceMetrics_DefaultRange(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics/dev-1?metric=latency", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var series MetricSeries
	if err := json.NewDecoder(w.Body).Decode(&series); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if series.Range != "24h" {
		t.Errorf("Range = %q, want %q (default)", series.Range, "24h")
	}
}

func TestHandleDeviceMetrics_MissingMetric(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceMetrics(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceMetrics_InvalidMetric(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics/dev-1?metric=bogus", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceMetrics(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceMetrics_InvalidRange(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics/dev-1?metric=latency&range=99d", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceMetrics(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceMetrics_EmptyDeviceID(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics/?metric=latency", http.NoBody)
	w := httptest.NewRecorder()

	m.handleDeviceMetrics(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceMetrics_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}

	req := httptest.NewRequest(http.MethodGet, "/metrics/dev-1?metric=latency", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceMetrics(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleDeviceMetrics_EmptyResult(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics/nonexistent?metric=latency&range=1h", http.NoBody)
	req.SetPathValue("device_id", "nonexistent")
	w := httptest.NewRecorder()

	m.handleDeviceMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var series MetricSeries
	if err := json.NewDecoder(w.Body).Decode(&series); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if series.Points == nil {
		t.Error("Points is nil, want empty array")
	}
	if len(series.Points) != 0 {
		t.Errorf("len(Points) = %d, want 0", len(series.Points))
	}
}

func TestHandleDeviceMetrics_AllMetricTypes(t *testing.T) {
	m, _ := newTestModule(t)

	// Seed data.
	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(-30 * time.Minute)
	check := &Check{
		ID:              "chk-all",
		DeviceID:        "dev-all",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       start,
		UpdatedAt:       start,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}
	for i := 0; i < 10; i++ {
		r := &CheckResult{
			CheckID:    "chk-all",
			DeviceID:   "dev-all",
			Success:    i%3 != 0,
			LatencyMs:  20.0,
			PacketLoss: 0.1,
			CheckedAt:  start.Add(time.Duration(i) * time.Minute),
		}
		if err := m.store.InsertResult(context.Background(), r); err != nil {
			t.Fatalf("insert result %d: %v", i, err)
		}
	}

	tests := []struct {
		metric string
	}{
		{metric: "latency"},
		{metric: "packet_loss"},
		{metric: "success_rate"},
	}

	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics/dev-all?metric="+tt.metric+"&range=1h", http.NoBody)
			req.SetPathValue("device_id", "dev-all")
			w := httptest.NewRecorder()

			m.handleDeviceMetrics(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}

			var series MetricSeries
			if err := json.NewDecoder(w.Body).Decode(&series); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if series.Metric != tt.metric {
				t.Errorf("Metric = %q, want %q", series.Metric, tt.metric)
			}
			if len(series.Points) == 0 {
				t.Error("expected at least 1 data point")
			}
		})
	}
}
