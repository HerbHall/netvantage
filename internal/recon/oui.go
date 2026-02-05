package recon

import (
	"bufio"
	"bytes"
	_ "embed"
	"strings"
	"sync"
)

//go:embed oui_data.txt
var ouiRawData []byte

// OUITable provides MAC address prefix to manufacturer lookup.
type OUITable struct {
	once  sync.Once
	table map[string]string
}

// NewOUITable creates a new OUI lookup table.
func NewOUITable() *OUITable {
	return &OUITable{}
}

// Lookup returns the manufacturer for a given MAC address.
// The MAC can be in any common format (AA:BB:CC:DD:EE:FF, AA-BB-CC-DD-EE-FF, AABBCCDDEEFF).
// Returns empty string if not found.
func (o *OUITable) Lookup(mac string) string {
	o.once.Do(o.load)

	prefix := normalizeMAC(mac)
	if prefix == "" {
		return ""
	}
	return o.table[prefix]
}

// load parses the embedded OUI data into the lookup table.
func (o *OUITable) load() {
	o.table = make(map[string]string, 40000)
	scanner := bufio.NewScanner(bytes.NewReader(ouiRawData))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		prefix := strings.ToUpper(strings.TrimSpace(parts[0]))
		vendor := strings.TrimSpace(parts[1])
		if prefix != "" && vendor != "" {
			o.table[prefix] = vendor
		}
	}
}

// normalizeMAC extracts the first 3 octets from a MAC address and returns
// them in uppercase colon-separated format (e.g., "AA:BB:CC").
func normalizeMAC(mac string) string {
	// Remove separators to get raw hex.
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")

	if len(mac) < 6 {
		return ""
	}

	// First 3 octets as colon-separated uppercase.
	return mac[0:2] + ":" + mac[2:4] + ":" + mac[4:6]
}
