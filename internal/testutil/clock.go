package testutil

import (
	"sync"
	"time"
)

// Clock provides a controllable time source for tests.
type Clock struct {
	mu  sync.Mutex
	now time.Time
}

// NewClock returns a Clock initialized to the given time.
// If no time is provided, it defaults to a fixed point:
// 2025-01-01 00:00:00 UTC.
func NewClock(now ...time.Time) *Clock {
	t := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if len(now) > 0 {
		t = now[0]
	}
	return &Clock{now: t}
}

// Now returns the clock's current time.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by d.
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// Set overrides the clock's current time.
func (c *Clock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}
