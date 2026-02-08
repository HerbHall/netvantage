package insight

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/analytics"
	"github.com/HerbHall/subnetree/pkg/llm"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
)

// -- Mock implementations --

// mockLLMProvider implements llm.Provider for testing.
type mockLLMProvider struct {
	chatFunc     func(ctx context.Context, messages []llm.Message, opts ...llm.CallOption) (*llm.Response, error)
	generateFunc func(ctx context.Context, prompt string, opts ...llm.CallOption) (*llm.Response, error)
}

func (m *mockLLMProvider) Generate(ctx context.Context, prompt string, opts ...llm.CallOption) (*llm.Response, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt, opts...)
	}
	return &llm.Response{Content: "mock answer", Model: "mock-model", Done: true}, nil
}

func (m *mockLLMProvider) Chat(ctx context.Context, messages []llm.Message, opts ...llm.CallOption) (*llm.Response, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, messages, opts...)
	}
	return &llm.Response{Content: `{"type":"list_anomalies"}`, Model: "mock-model", Done: true}, nil
}

// mockLLMPlugin implements plugin.Plugin + roles.LLMProvider.
type mockLLMPlugin struct {
	provider llm.Provider
}

func (m *mockLLMPlugin) Info() plugin.PluginInfo {
	return plugin.PluginInfo{Name: "mock-llm", Roles: []string{roles.RoleLLM}}
}

func (m *mockLLMPlugin) Init(_ context.Context, _ plugin.Dependencies) error { return nil }
func (m *mockLLMPlugin) Start(_ context.Context) error                       { return nil }
func (m *mockLLMPlugin) Stop(_ context.Context) error                        { return nil }
func (m *mockLLMPlugin) Provider() llm.Provider                              { return m.provider }

// mockPluginResolver implements plugin.PluginResolver.
type mockPluginResolver struct {
	byRole map[string][]plugin.Plugin
}

func (m *mockPluginResolver) Resolve(_ string) (plugin.Plugin, bool) { return nil, false }

func (m *mockPluginResolver) ResolveByRole(role string) []plugin.Plugin {
	if m.byRole == nil {
		return nil
	}
	return m.byRole[role]
}

// mockDiscoveryPlugin implements plugin.Plugin + roles.DiscoveryProvider.
type mockDiscoveryPlugin struct {
	devices []models.Device
}

func (m *mockDiscoveryPlugin) Info() plugin.PluginInfo {
	return plugin.PluginInfo{Name: "mock-discovery", Roles: []string{roles.RoleDiscovery}}
}

func (m *mockDiscoveryPlugin) Init(_ context.Context, _ plugin.Dependencies) error { return nil }
func (m *mockDiscoveryPlugin) Start(_ context.Context) error                       { return nil }
func (m *mockDiscoveryPlugin) Stop(_ context.Context) error                        { return nil }

func (m *mockDiscoveryPlugin) Devices(_ context.Context) ([]models.Device, error) {
	return m.devices, nil
}

func (m *mockDiscoveryPlugin) DeviceByID(_ context.Context, id string) (*models.Device, error) {
	for i := range m.devices {
		if m.devices[i].ID == id {
			return &m.devices[i], nil
		}
	}
	return nil, nil
}

// -- Constructor tests --

func TestNewNLQueryProcessor_NilPlugins(t *testing.T) {
	proc := newNLQueryProcessor(nil, nil)
	if proc != nil {
		t.Fatal("expected nil processor when plugins is nil")
	}
}

func TestNewNLQueryProcessor_NoLLMProvider(t *testing.T) {
	resolver := &mockPluginResolver{byRole: map[string][]plugin.Plugin{}}
	proc := newNLQueryProcessor(resolver, nil)
	if proc != nil {
		t.Fatal("expected nil processor when no LLM provider registered")
	}
}

func TestNewNLQueryProcessor_InvalidLLMType(t *testing.T) {
	// Register a plugin that doesn't implement roles.LLMProvider.
	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{
			roles.RoleLLM: {&mockDiscoveryPlugin{}},
		},
	}
	proc := newNLQueryProcessor(resolver, nil)
	if proc != nil {
		t.Fatal("expected nil processor when LLM plugin has wrong type")
	}
}

func TestNewNLQueryProcessor_Success(t *testing.T) {
	llmPlugin := &mockLLMPlugin{provider: &mockLLMProvider{}}
	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{
			roles.RoleLLM: {llmPlugin},
		},
	}
	proc := newNLQueryProcessor(resolver, nil)
	if proc == nil {
		t.Fatal("expected non-nil processor")
	}
	if proc.llmProvider == nil {
		t.Fatal("expected llmProvider to be set")
	}
}

// -- Process tests --

func TestProcess_HappyPath(t *testing.T) {
	s := testStore(t)

	// Seed an anomaly.
	ctx := context.Background()
	_ = s.InsertAnomaly(ctx, &analytics.Anomaly{
		ID: "a1", DeviceID: "dev-1", MetricName: "cpu",
		Severity: "warning", Type: "zscore", Value: 95, Expected: 50,
		DetectedAt: time.Now(), Description: "high cpu",
	})

	mockLLM := &mockLLMProvider{
		chatFunc: func(_ context.Context, _ []llm.Message, _ ...llm.CallOption) (*llm.Response, error) {
			return &llm.Response{
				Content: `{"type":"list_anomalies","limit":10}`,
				Model:   "test-model",
				Done:    true,
			}, nil
		},
		generateFunc: func(_ context.Context, _ string, _ ...llm.CallOption) (*llm.Response, error) {
			return &llm.Response{
				Content: "Found 1 anomaly on dev-1.",
				Model:   "test-model",
				Done:    true,
			}, nil
		},
	}

	llmPlugin := &mockLLMPlugin{provider: mockLLM}
	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{
			roles.RoleLLM: {llmPlugin},
		},
	}

	proc := newNLQueryProcessor(resolver, s)
	resp, err := proc.Process(ctx, "show anomalies")
	if err != nil {
		t.Fatalf("Process() error: %v", err)
	}

	if resp.Query != "show anomalies" {
		t.Errorf("Query = %q, want %q", resp.Query, "show anomalies")
	}
	if resp.Model != "test-model" {
		t.Errorf("Model = %q, want %q", resp.Model, "test-model")
	}
	if resp.Answer != "Found 1 anomaly on dev-1." {
		t.Errorf("Answer = %q", resp.Answer)
	}
	if resp.Structured == nil {
		t.Error("Structured should not be nil")
	}
}

func TestProcess_ParseIntentError(t *testing.T) {
	mockLLM := &mockLLMProvider{
		chatFunc: func(_ context.Context, _ []llm.Message, _ ...llm.CallOption) (*llm.Response, error) {
			return &llm.Response{Content: "not valid json", Model: "m", Done: true}, nil
		},
	}

	llmPlugin := &mockLLMPlugin{provider: mockLLM}
	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{roles.RoleLLM: {llmPlugin}},
	}

	proc := newNLQueryProcessor(resolver, nil)
	_, err := proc.Process(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestProcess_LLMChatError(t *testing.T) {
	mockLLM := &mockLLMProvider{
		chatFunc: func(_ context.Context, _ []llm.Message, _ ...llm.CallOption) (*llm.Response, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	llmPlugin := &mockLLMPlugin{provider: mockLLM}
	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{roles.RoleLLM: {llmPlugin}},
	}

	proc := newNLQueryProcessor(resolver, nil)
	_, err := proc.Process(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error when Chat() fails")
	}
}

func TestProcess_FormatResponseError(t *testing.T) {
	mockLLM := &mockLLMProvider{
		chatFunc: func(_ context.Context, _ []llm.Message, _ ...llm.CallOption) (*llm.Response, error) {
			return &llm.Response{Content: `{"type":"list_anomalies"}`, Model: "m", Done: true}, nil
		},
		generateFunc: func(_ context.Context, _ string, _ ...llm.CallOption) (*llm.Response, error) {
			return nil, fmt.Errorf("model overloaded")
		},
	}

	llmPlugin := &mockLLMPlugin{provider: mockLLM}
	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{roles.RoleLLM: {llmPlugin}},
	}

	proc := newNLQueryProcessor(resolver, nil)
	_, err := proc.Process(context.Background(), "anomalies")
	if err == nil {
		t.Fatal("expected error when Generate() fails")
	}
}

// -- Intent execution tests --

func TestExecuteListAnomalies_WithStore(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_ = s.InsertAnomaly(ctx, &analytics.Anomaly{
		ID: "a1", DeviceID: "dev-1", MetricName: "cpu",
		Severity: "warning", Type: "zscore", Value: 95, Expected: 50,
		DetectedAt: time.Now(), Description: "test",
	})

	intent := &queryIntent{Type: IntentListAnomalies, Limit: 10}
	result, err := intent.execute(ctx, s, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	anomalies, ok := result.([]analytics.Anomaly)
	if !ok {
		t.Fatalf("expected []analytics.Anomaly, got %T", result)
	}
	if len(anomalies) != 1 {
		t.Errorf("got %d anomalies, want 1", len(anomalies))
	}
}

func TestExecuteListAnomalies_NilStore(t *testing.T) {
	intent := &queryIntent{Type: IntentListAnomalies}
	result, err := intent.execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	anomalies, ok := result.([]analytics.Anomaly)
	if !ok {
		t.Fatalf("expected []analytics.Anomaly, got %T", result)
	}
	if len(anomalies) != 0 {
		t.Errorf("expected empty slice, got %d", len(anomalies))
	}
}

func TestExecuteListDevices_WithDiscovery(t *testing.T) {
	discovery := &mockDiscoveryPlugin{
		devices: []models.Device{
			{ID: "dev-1", Hostname: "web-01", Status: models.DeviceStatusOnline},
			{ID: "dev-2", Hostname: "db-01", Status: models.DeviceStatusOnline},
		},
	}
	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{
			roles.RoleDiscovery: {discovery},
		},
	}

	intent := &queryIntent{Type: IntentListDevices}
	result, err := intent.execute(context.Background(), nil, resolver)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	devices, ok := result.([]models.Device)
	if !ok {
		t.Fatalf("expected []models.Device, got %T", result)
	}
	if len(devices) != 2 {
		t.Errorf("got %d devices, want 2", len(devices))
	}
}

func TestExecuteListDevices_NilPlugins(t *testing.T) {
	intent := &queryIntent{Type: IntentListDevices}
	result, err := intent.execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	devices, ok := result.([]models.Device)
	if !ok {
		t.Fatalf("expected []models.Device, got %T", result)
	}
	if len(devices) != 0 {
		t.Errorf("expected empty slice, got %d", len(devices))
	}
}

func TestExecuteDeviceStatus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Seed data for dev-1.
	_ = s.InsertAnomaly(ctx, &analytics.Anomaly{
		ID: "a1", DeviceID: "dev-1", MetricName: "cpu",
		Severity: "warning", Type: "zscore", Value: 95, Expected: 50,
		DetectedAt: time.Now(), Description: "high cpu",
	})
	_ = s.UpsertBaseline(ctx, &analytics.Baseline{
		DeviceID: "dev-1", MetricName: "cpu", Algorithm: "ewma",
		Mean: 50, StdDev: 5, Samples: 200, Stable: true, UpdatedAt: time.Now(),
	})

	discovery := &mockDiscoveryPlugin{
		devices: []models.Device{
			{ID: "dev-1", Hostname: "web-01", Status: models.DeviceStatusOnline},
		},
	}
	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{
			roles.RoleDiscovery: {discovery},
		},
	}

	intent := &queryIntent{Type: IntentDeviceStatus, DeviceID: "dev-1"}
	result, err := intent.execute(ctx, s, resolver)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	status, ok := result.(*deviceStatusResult)
	if !ok {
		t.Fatalf("expected *deviceStatusResult, got %T", result)
	}
	if status.Device == nil {
		t.Error("expected Device to be set")
	}
	if status.Device != nil && status.Device.Hostname != "web-01" {
		t.Errorf("Device.Hostname = %q, want %q", status.Device.Hostname, "web-01")
	}
	if len(status.Anomalies) != 1 {
		t.Errorf("got %d anomalies, want 1", len(status.Anomalies))
	}
	if len(status.Baselines) != 1 {
		t.Errorf("got %d baselines, want 1", len(status.Baselines))
	}
}

func TestExecuteDeviceStatus_MissingDeviceID(t *testing.T) {
	intent := &queryIntent{Type: IntentDeviceStatus}
	_, err := intent.execute(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for missing device_id")
	}
}

func TestExecuteUnsupportedIntent(t *testing.T) {
	intent := &queryIntent{Type: "unknown_type"}
	_, err := intent.execute(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for unsupported intent type")
	}
}

func TestExecuteListBaselines_NilStore(t *testing.T) {
	intent := &queryIntent{Type: IntentListBaselines, DeviceID: "dev-1"}
	result, err := intent.execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	baselines, ok := result.([]analytics.Baseline)
	if !ok {
		t.Fatalf("expected []analytics.Baseline, got %T", result)
	}
	if len(baselines) != 0 {
		t.Errorf("expected empty, got %d", len(baselines))
	}
}

func TestExecuteListForecasts_NilStore(t *testing.T) {
	intent := &queryIntent{Type: IntentListForecasts, DeviceID: "dev-1"}
	result, err := intent.execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	forecasts, ok := result.([]analytics.Forecast)
	if !ok {
		t.Fatalf("expected []analytics.Forecast, got %T", result)
	}
	if len(forecasts) != 0 {
		t.Errorf("expected empty, got %d", len(forecasts))
	}
}

func TestExecuteListCorrelations_NilStore(t *testing.T) {
	intent := &queryIntent{Type: IntentListCorrelations}
	result, err := intent.execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	groups, ok := result.([]analytics.AlertGroup)
	if !ok {
		t.Fatalf("expected []analytics.AlertGroup, got %T", result)
	}
	if len(groups) != 0 {
		t.Errorf("expected empty, got %d", len(groups))
	}
}
