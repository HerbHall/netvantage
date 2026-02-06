package services_test

import (
	"testing"

	"github.com/HerbHall/subnetree/internal/services"
)

func TestInterfaceService_ListNetworkInterfaces(t *testing.T) {
	svc := services.NewInterfaceService()

	interfaces, err := svc.ListNetworkInterfaces()
	if err != nil {
		t.Fatalf("ListNetworkInterfaces: %v", err)
	}

	// We should have at least one interface on any system
	// (though in CI/containers this may not always be true)
	if len(interfaces) == 0 {
		t.Log("No interfaces found (may be expected in some environments)")
		return
	}

	// Verify interface structure
	for i := range interfaces {
		iface := &interfaces[i]
		if iface.Name == "" {
			t.Errorf("Interface %d has empty name", i)
		}
		if iface.IPAddress == "" {
			t.Errorf("Interface %q has empty IP address", iface.Name)
		}
		if iface.Subnet == "" {
			t.Errorf("Interface %q has empty subnet", iface.Name)
		}
		if iface.Status != "up" && iface.Status != "down" {
			t.Errorf("Interface %q has invalid status %q", iface.Name, iface.Status)
		}
	}
}

func TestInterfaceService_ListNetworkInterfaces_NoLoopback(t *testing.T) {
	svc := services.NewInterfaceService()

	interfaces, err := svc.ListNetworkInterfaces()
	if err != nil {
		t.Fatalf("ListNetworkInterfaces: %v", err)
	}

	// Verify no loopback interfaces are returned
	for i := range interfaces {
		iface := &interfaces[i]
		// Common loopback names: lo, lo0, Loopback, etc.
		if iface.IPAddress == "127.0.0.1" {
			t.Errorf("Loopback interface %q should be filtered out", iface.Name)
		}
	}
}

func TestNetworkInterface_MACFormat(t *testing.T) {
	svc := services.NewInterfaceService()

	interfaces, err := svc.ListNetworkInterfaces()
	if err != nil {
		t.Fatalf("ListNetworkInterfaces: %v", err)
	}

	// Verify MAC address format (if present)
	for i := range interfaces {
		iface := &interfaces[i]
		if iface.MAC == "" {
			continue // MAC may be empty for some virtual interfaces
		}
		// MAC should be colon-separated, e.g., "aa:bb:cc:dd:ee:ff"
		if len(iface.MAC) != 17 {
			t.Errorf("Interface %q has invalid MAC length: %q", iface.Name, iface.MAC)
		}
	}
}
