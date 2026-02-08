package models

import "testing"

func TestDeviceIconCoverage(t *testing.T) {
	knownTypes := []DeviceType{
		DeviceTypeServer, DeviceTypeDesktop, DeviceTypeLaptop,
		DeviceTypeMobile, DeviceTypeRouter, DeviceTypeSwitch,
		DeviceTypePrinter, DeviceTypeIoT, DeviceTypeAccessPoint,
		DeviceTypeFirewall, DeviceTypeNAS, DeviceTypePhone,
		DeviceTypeTablet, DeviceTypeCamera, DeviceTypeUnknown,
	}
	for _, dt := range knownTypes {
		icon := dt.Icon()
		if icon == "" {
			t.Errorf("DeviceType %q has empty icon", dt)
		}
	}
}

func TestDeviceIconUnknownFallback(t *testing.T) {
	got := DeviceType("nonexistent").Icon()
	want := "help-circle"
	if got != want {
		t.Errorf("unknown device type icon = %q, want %q", got, want)
	}
}
