package testutil

import (
	"testing"

	"github.com/HerbHall/netvantage/internal/store"
)

// NewStore creates an in-memory SQLiteStore for testing.
// The store is automatically closed when the test completes.
func NewStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("testutil.NewStore: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
