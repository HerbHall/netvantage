//go:build windows

package metrics

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestWindowsCollect_CPURange(t *testing.T) {
	logger := zap.NewNop()
	c := NewCollector(logger)

	m, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}

	if m.CpuPercent < 0 || m.CpuPercent > 100 {
		t.Errorf("CPU percent out of range [0,100]: %f", m.CpuPercent)
	}
}

func TestWindowsCollect_MemoryValid(t *testing.T) {
	logger := zap.NewNop()
	c := NewCollector(logger)

	m, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}

	if m.MemoryTotalBytes <= 0 {
		t.Errorf("expected positive total memory, got %f", m.MemoryTotalBytes)
	}
	if m.MemoryUsedBytes < 0 {
		t.Errorf("expected non-negative used memory, got %f", m.MemoryUsedBytes)
	}
	if m.MemoryUsedBytes > m.MemoryTotalBytes {
		t.Errorf("used memory (%f) exceeds total (%f)", m.MemoryUsedBytes, m.MemoryTotalBytes)
	}
	if m.MemoryPercent < 0 || m.MemoryPercent > 100 {
		t.Errorf("memory percent out of range [0,100]: %f", m.MemoryPercent)
	}
}

func TestWindowsCollect_DisksPresent(t *testing.T) {
	logger := zap.NewNop()
	c := NewCollector(logger)

	m, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}

	// Windows should always have at least one disk (C:\).
	if len(m.Disks) == 0 {
		t.Fatal("expected at least one disk metric on Windows")
	}

	for _, d := range m.Disks {
		if d.TotalBytes <= 0 {
			t.Errorf("disk %s: expected positive total bytes, got %f", d.MountPoint, d.TotalBytes)
		}
		if d.FreeBytes < 0 {
			t.Errorf("disk %s: negative free bytes: %f", d.MountPoint, d.FreeBytes)
		}
		if d.UsedBytes < 0 {
			t.Errorf("disk %s: negative used bytes: %f", d.MountPoint, d.UsedBytes)
		}
	}
}

func TestWindowsCollect_NetworkPresent(t *testing.T) {
	logger := zap.NewNop()
	c := NewCollector(logger)

	m, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}

	// Most Windows machines have at least one operational network interface.
	// This is not guaranteed in CI, so we just check for non-negative values.
	for _, n := range m.Networks {
		if n.BytesSent < 0 {
			t.Errorf("interface %s: negative bytes sent: %f", n.InterfaceName, n.BytesSent)
		}
		if n.BytesRecv < 0 {
			t.Errorf("interface %s: negative bytes recv: %f", n.InterfaceName, n.BytesRecv)
		}
	}
}
