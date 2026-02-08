package insight

import (
	"context"
	"fmt"

	"github.com/HerbHall/subnetree/pkg/analytics"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
)

// Supported intent types for the NL query parser.
const (
	IntentListAnomalies    = "list_anomalies"
	IntentListBaselines    = "list_baselines"
	IntentListForecasts    = "list_forecasts"
	IntentListCorrelations = "list_correlations"
	IntentListDevices      = "list_devices"
	IntentDeviceStatus     = "device_status"
)

// queryIntent represents the structured output from the LLM intent parser.
type queryIntent struct {
	Type     string `json:"type"`
	DeviceID string `json:"device_id,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// deviceStatusResult is the composite response for a device_status intent.
type deviceStatusResult struct {
	Device    *models.Device       `json:"device,omitempty"`
	Anomalies []analytics.Anomaly  `json:"anomalies"`
	Baselines []analytics.Baseline `json:"baselines"`
	Forecasts []analytics.Forecast `json:"forecasts"`
}

// execute dispatches the intent to the appropriate handler.
func (i *queryIntent) execute(ctx context.Context, store *InsightStore, plugins plugin.PluginResolver) (any, error) {
	switch i.Type {
	case IntentListAnomalies:
		return i.executeListAnomalies(ctx, store)
	case IntentListBaselines:
		return i.executeListBaselines(ctx, store)
	case IntentListForecasts:
		return i.executeListForecasts(ctx, store)
	case IntentListCorrelations:
		return i.executeListCorrelations(ctx, store)
	case IntentListDevices:
		return i.executeListDevices(ctx, plugins)
	case IntentDeviceStatus:
		return i.executeDeviceStatus(ctx, store, plugins)
	default:
		return nil, fmt.Errorf("unsupported intent type: %s", i.Type)
	}
}

func (i *queryIntent) executeListAnomalies(ctx context.Context, store *InsightStore) ([]analytics.Anomaly, error) {
	if store == nil {
		return []analytics.Anomaly{}, nil
	}
	limit := i.Limit
	if limit <= 0 {
		limit = 50
	}
	anomalies, err := store.ListAnomalies(ctx, i.DeviceID, limit)
	if err != nil {
		return nil, err
	}
	if anomalies == nil {
		return []analytics.Anomaly{}, nil
	}
	return anomalies, nil
}

func (i *queryIntent) executeListBaselines(ctx context.Context, store *InsightStore) ([]analytics.Baseline, error) {
	if store == nil {
		return []analytics.Baseline{}, nil
	}
	baselines, err := store.GetBaselines(ctx, i.DeviceID)
	if err != nil {
		return nil, err
	}
	if baselines == nil {
		return []analytics.Baseline{}, nil
	}
	return baselines, nil
}

func (i *queryIntent) executeListForecasts(ctx context.Context, store *InsightStore) ([]analytics.Forecast, error) {
	if store == nil {
		return []analytics.Forecast{}, nil
	}
	forecasts, err := store.GetForecasts(ctx, i.DeviceID)
	if err != nil {
		return nil, err
	}
	if forecasts == nil {
		return []analytics.Forecast{}, nil
	}
	return forecasts, nil
}

func (i *queryIntent) executeListCorrelations(ctx context.Context, store *InsightStore) ([]analytics.AlertGroup, error) {
	if store == nil {
		return []analytics.AlertGroup{}, nil
	}
	groups, err := store.ListActiveCorrelations(ctx)
	if err != nil {
		return nil, err
	}
	if groups == nil {
		return []analytics.AlertGroup{}, nil
	}
	return groups, nil
}

func (i *queryIntent) executeListDevices(ctx context.Context, plugins plugin.PluginResolver) ([]models.Device, error) {
	if plugins == nil {
		return []models.Device{}, nil
	}
	discoveryPlugins := plugins.ResolveByRole(roles.RoleDiscovery)
	if len(discoveryPlugins) == 0 {
		return []models.Device{}, nil
	}
	dp, ok := discoveryPlugins[0].(roles.DiscoveryProvider)
	if !ok {
		return []models.Device{}, nil
	}
	devices, err := dp.Devices(ctx)
	if err != nil {
		return nil, err
	}
	if devices == nil {
		return []models.Device{}, nil
	}
	return devices, nil
}

func (i *queryIntent) executeDeviceStatus(ctx context.Context, store *InsightStore, plugins plugin.PluginResolver) (*deviceStatusResult, error) {
	if i.DeviceID == "" {
		return nil, fmt.Errorf("device_id required for device_status intent")
	}

	result := &deviceStatusResult{
		Anomalies: []analytics.Anomaly{},
		Baselines: []analytics.Baseline{},
		Forecasts: []analytics.Forecast{},
	}

	// Resolve device info from Discovery plugin.
	if plugins != nil {
		discoveryPlugins := plugins.ResolveByRole(roles.RoleDiscovery)
		if len(discoveryPlugins) > 0 {
			if dp, ok := discoveryPlugins[0].(roles.DiscoveryProvider); ok {
				if device, err := dp.DeviceByID(ctx, i.DeviceID); err == nil && device != nil {
					result.Device = device
				}
			}
		}
	}

	// Gather analytics data from store.
	if store != nil {
		if anomalies, err := store.ListAnomalies(ctx, i.DeviceID, 10); err == nil && anomalies != nil {
			result.Anomalies = anomalies
		}
		if baselines, err := store.GetBaselines(ctx, i.DeviceID); err == nil && baselines != nil {
			result.Baselines = baselines
		}
		if forecasts, err := store.GetForecasts(ctx, i.DeviceID); err == nil && forecasts != nil {
			result.Forecasts = forecasts
		}
	}

	return result, nil
}
