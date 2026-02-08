package models

// DeviceIcon maps a DeviceType to its icon identifier.
// Identifiers use Lucide icon names (https://lucide.dev) for
// compatibility with the React dashboard.
var DeviceIcon = map[DeviceType]string{
	DeviceTypeServer:      "server",
	DeviceTypeDesktop:     "monitor",
	DeviceTypeLaptop:      "laptop",
	DeviceTypeMobile:      "smartphone",
	DeviceTypeRouter:      "router",
	DeviceTypeSwitch:      "network",
	DeviceTypePrinter:     "printer",
	DeviceTypeIoT:         "cpu",
	DeviceTypeAccessPoint: "wifi",
	DeviceTypeFirewall:    "shield",
	DeviceTypeNAS:         "hard-drive",
	DeviceTypePhone:       "phone",
	DeviceTypeTablet:      "tablet",
	DeviceTypeCamera:      "camera",
	DeviceTypeUnknown:     "help-circle",
}

// Icon returns the icon identifier for a DeviceType.
// Returns "help-circle" for unrecognised types.
func (dt DeviceType) Icon() string {
	if icon, ok := DeviceIcon[dt]; ok {
		return icon
	}
	return DeviceIcon[DeviceTypeUnknown]
}
