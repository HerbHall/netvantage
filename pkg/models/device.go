package models

import "time"

// DeviceType categorizes a network device.
type DeviceType string

const (
	DeviceTypeServer  DeviceType = "server"
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeLaptop  DeviceType = "laptop"
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypeRouter  DeviceType = "router"
	DeviceTypeSwitch  DeviceType = "switch"
	DeviceTypePrinter DeviceType = "printer"
	DeviceTypeIoT     DeviceType = "iot"
	DeviceTypeUnknown DeviceType = "unknown"
)

// DeviceStatus represents the current state of a device.
type DeviceStatus string

const (
	DeviceStatusOnline   DeviceStatus = "online"
	DeviceStatusOffline  DeviceStatus = "offline"
	DeviceStatusDegraded DeviceStatus = "degraded"
	DeviceStatusUnknown  DeviceStatus = "unknown"
)

// DiscoveryMethod indicates how a device was discovered.
type DiscoveryMethod string

const (
	DiscoveryAgent  DiscoveryMethod = "agent"
	DiscoveryICMP   DiscoveryMethod = "icmp"
	DiscoveryARP    DiscoveryMethod = "arp"
	DiscoverySNMP   DiscoveryMethod = "snmp"
	DiscoverymDNS   DiscoveryMethod = "mdns"
	DiscoveryUPnP   DiscoveryMethod = "upnp"
	DiscoveryMQTT   DiscoveryMethod = "mqtt"
	DiscoveryManual DiscoveryMethod = "manual"
)

// Device represents a network device tracked by SubNetree.
type Device struct {
	ID              string            `json:"id"`
	Hostname        string            `json:"hostname"`
	IPAddresses     []string          `json:"ip_addresses"`
	MACAddress      string            `json:"mac_address,omitempty"`
	Manufacturer    string            `json:"manufacturer,omitempty"`
	DeviceType      DeviceType        `json:"device_type"`
	OS              string            `json:"os,omitempty"`
	Status          DeviceStatus      `json:"status"`
	DiscoveryMethod DiscoveryMethod   `json:"discovery_method"`
	AgentID         string            `json:"agent_id,omitempty"`
	LastSeen        time.Time         `json:"last_seen"`
	FirstSeen       time.Time         `json:"first_seen"`
	Notes           string            `json:"notes,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	CustomFields    map[string]string `json:"custom_fields,omitempty"`
}
