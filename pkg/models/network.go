package models

import "net"

// NetworkInterface represents a network interface on the server or agent host.
type NetworkInterface struct {
	Name       string   `json:"name"`
	Index      int      `json:"index"`
	MTU        int      `json:"mtu"`
	MACAddress string   `json:"mac_address"`
	Addresses  []string `json:"addresses"`
	IsUp       bool     `json:"is_up"`
	IsLoopback bool     `json:"is_loopback"`
}

// Subnet represents an IP subnet for scanning.
type Subnet struct {
	CIDR    string `json:"cidr"`
	Network net.IP `json:"-"`
	Mask    net.IP `json:"-"`
}

// ScanResult holds the result of a network scan.
type ScanResult struct {
	ID        string   `json:"id"`
	Subnet    string   `json:"subnet"`
	StartedAt string   `json:"started_at"`
	EndedAt   string   `json:"ended_at,omitempty"`
	Status    string   `json:"status"`
	Devices   []Device `json:"devices,omitempty"`
	Total     int      `json:"total"`
	Online    int      `json:"online"`
}

// AgentInfo represents the state of a connected Scout agent.
type AgentInfo struct {
	ID          string `json:"id"`
	DeviceID    string `json:"device_id"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	LastCheckIn string `json:"last_check_in"`
	EnrolledAt  string `json:"enrolled_at"`
	Platform    string `json:"platform"`
}
