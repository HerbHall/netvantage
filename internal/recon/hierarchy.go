package recon

import (
	"context"
	"fmt"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// HierarchyInferrer builds a layer-aware device tree from topology data.
type HierarchyInferrer struct {
	store  *ReconStore
	logger *zap.Logger
}

// NewHierarchyInferrer creates a new hierarchy inference engine.
func NewHierarchyInferrer(store *ReconStore, logger *zap.Logger) *HierarchyInferrer {
	return &HierarchyInferrer{store: store, logger: logger}
}

// InferHierarchy runs the full hierarchy inference pipeline.
// It assigns network layers and parent device IDs based on device types
// and topology links.
func (h *HierarchyInferrer) InferHierarchy(ctx context.Context) error {
	devices, err := h.store.ListAllDevices(ctx)
	if err != nil {
		return fmt.Errorf("load devices: %w", err)
	}
	if len(devices) == 0 {
		return nil
	}

	links, err := h.store.GetTopologyLinks(ctx)
	if err != nil {
		return fmt.Errorf("load topology links: %w", err)
	}

	result := InferHierarchyFromData(devices, links)

	var updatedCount int
	for _, a := range result {
		if updateErr := h.store.UpdateDeviceHierarchy(ctx, a.DeviceID, a.ParentDeviceID, a.NetworkLayer); updateErr != nil {
			h.logger.Error("failed to update device hierarchy",
				zap.String("device_id", a.DeviceID),
				zap.Error(updateErr),
			)
			continue
		}
		updatedCount++
	}

	h.logger.Info("hierarchy inference completed",
		zap.Int("devices", len(devices)),
		zap.Int("updated", updatedCount),
	)

	return nil
}

// HierarchyAssignment represents the computed hierarchy for a single device.
type HierarchyAssignment struct {
	DeviceID       string
	ParentDeviceID string
	NetworkLayer   int
}

// InferHierarchyFromData is the pure-logic hierarchy inference function,
// separated from I/O for testability.
func InferHierarchyFromData(devices []models.Device, links []TopologyLink) []HierarchyAssignment {
	if len(devices) == 0 {
		return nil
	}

	// Build lookup maps.
	deviceByID := make(map[string]*models.Device, len(devices))
	for i := range devices {
		deviceByID[devices[i].ID] = &devices[i]
	}

	// Build adjacency from topology links.
	// linksFromSource[sourceID] = list of target device IDs
	linksFromSource := make(map[string][]string)
	// linksToTarget[targetID] = list of source device IDs
	linksToTarget := make(map[string][]string)
	// fdbLinksFromSwitch[switchID] = list of target device IDs (FDB only)
	fdbLinksFromSwitch := make(map[string][]string)

	for i := range links {
		src := links[i].SourceDeviceID
		tgt := links[i].TargetDeviceID
		linksFromSource[src] = append(linksFromSource[src], tgt)
		linksToTarget[tgt] = append(linksToTarget[tgt], src)

		if links[i].LinkType == "fdb" {
			fdbLinksFromSwitch[src] = append(fdbLinksFromSwitch[src], tgt)
		}
	}

	assignments := make(map[string]*HierarchyAssignment, len(devices))
	for _, d := range devices {
		assignments[d.ID] = &HierarchyAssignment{DeviceID: d.ID}
	}

	// Step 1: Identify gateway devices (routers and firewalls).
	var gatewayID string
	for _, d := range devices {
		if d.DeviceType == models.DeviceTypeRouter || d.DeviceType == models.DeviceTypeFirewall {
			assignments[d.ID].NetworkLayer = models.NetworkLayerGateway
			if gatewayID == "" {
				gatewayID = d.ID
			}
		}
	}

	// If we found multiple routers/firewalls, link them: firewalls -> first router
	// or if no router, the first gateway is the root.
	var firstRouterID string
	for _, d := range devices {
		if d.DeviceType == models.DeviceTypeRouter {
			if firstRouterID == "" {
				firstRouterID = d.ID
			}
			break
		}
	}
	if firstRouterID == "" {
		firstRouterID = gatewayID
	}

	// Set firewall -> router parent if both exist.
	if firstRouterID != "" {
		for _, d := range devices {
			if d.DeviceType == models.DeviceTypeFirewall && d.ID != firstRouterID {
				// Check if there's a direct link between firewall and router.
				assignments[d.ID].ParentDeviceID = firstRouterID
			}
		}
	}

	// Step 2: Classify switches.
	for _, d := range devices {
		if d.DeviceType != models.DeviceTypeSwitch {
			continue
		}
		// Check if the switch has a direct link to the gateway.
		isDistribution := false
		if firstRouterID != "" {
			for _, tgt := range linksFromSource[d.ID] {
				if tgt == firstRouterID {
					isDistribution = true
					break
				}
			}
			if !isDistribution {
				for _, src := range linksToTarget[d.ID] {
					if src == firstRouterID {
						isDistribution = true
						break
					}
				}
			}
		}

		if isDistribution {
			assignments[d.ID].NetworkLayer = models.NetworkLayerDistribution
			assignments[d.ID].ParentDeviceID = firstRouterID
		} else {
			assignments[d.ID].NetworkLayer = models.NetworkLayerAccess
		}
	}

	// Step 3: Assign parents for access-layer switches.
	// An access switch's parent is the distribution switch it connects to.
	for _, d := range devices {
		if d.DeviceType != models.DeviceTypeSwitch {
			continue
		}
		if assignments[d.ID].NetworkLayer != models.NetworkLayerAccess {
			continue
		}
		// Look for an upstream switch (distribution layer).
		for _, tgt := range linksFromSource[d.ID] {
			if tgtDev, ok := deviceByID[tgt]; ok && tgtDev.DeviceType == models.DeviceTypeSwitch {
				if assignments[tgt].NetworkLayer == models.NetworkLayerDistribution {
					assignments[d.ID].ParentDeviceID = tgt
					break
				}
			}
		}
		for _, src := range linksToTarget[d.ID] {
			if assignments[d.ID].ParentDeviceID != "" {
				break
			}
			if srcDev, ok := deviceByID[src]; ok && srcDev.DeviceType == models.DeviceTypeSwitch {
				if assignments[src].NetworkLayer == models.NetworkLayerDistribution {
					assignments[d.ID].ParentDeviceID = src
					break
				}
			}
		}
		// If still no parent, assign to gateway.
		if assignments[d.ID].ParentDeviceID == "" && firstRouterID != "" {
			assignments[d.ID].ParentDeviceID = firstRouterID
		}
	}

	// Step 4: Access points get layer 3 (access) with parent = router or nearest switch.
	for _, d := range devices {
		if d.DeviceType != models.DeviceTypeAccessPoint {
			continue
		}
		assignments[d.ID].NetworkLayer = models.NetworkLayerAccess
		// Find parent from topology links.
		for _, src := range linksToTarget[d.ID] {
			if srcDev, ok := deviceByID[src]; ok {
				if srcDev.DeviceType == models.DeviceTypeSwitch || srcDev.DeviceType == models.DeviceTypeRouter {
					assignments[d.ID].ParentDeviceID = src
					break
				}
			}
		}
		if assignments[d.ID].ParentDeviceID == "" && firstRouterID != "" {
			assignments[d.ID].ParentDeviceID = firstRouterID
		}
	}

	// Step 5: Assign parents from FDB links (switch -> device).
	for switchID, targets := range fdbLinksFromSwitch {
		for _, tgtID := range targets {
			a := assignments[tgtID]
			if a == nil {
				continue
			}
			// Don't override already-assigned infrastructure parents.
			tgtDev := deviceByID[tgtID]
			if tgtDev == nil {
				continue
			}
			if tgtDev.DeviceType == models.DeviceTypeRouter ||
				tgtDev.DeviceType == models.DeviceTypeFirewall {
				continue
			}
			if a.ParentDeviceID == "" {
				a.ParentDeviceID = switchID
			}
		}
	}

	// Step 6: Assign remaining devices with no parent to the gateway.
	// Also set their layer to endpoint.
	for _, d := range devices {
		a := assignments[d.ID]
		if a.NetworkLayer != models.NetworkLayerUnknown {
			continue
		}
		// Everything else is an endpoint.
		a.NetworkLayer = models.NetworkLayerEndpoint
	}

	// Assign parentless non-infrastructure devices to the gateway.
	for _, d := range devices {
		a := assignments[d.ID]
		if a.ParentDeviceID != "" {
			continue
		}
		if a.NetworkLayer == models.NetworkLayerGateway {
			continue
		}
		if firstRouterID != "" && d.ID != firstRouterID {
			a.ParentDeviceID = firstRouterID
		}
	}

	// Convert to slice.
	result := make([]HierarchyAssignment, 0, len(assignments))
	for _, a := range assignments {
		result = append(result, *a)
	}

	return result
}
