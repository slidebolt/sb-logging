package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	logging "github.com/slidebolt/sb-logging-sdk"
)

func TestStore_Pruning(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-pruning.db")

	// Set low thresholds for testing: Prune at 10, keep 5.
	maxLogs := 5
	pruneThreshold := 10
	store, err := New(dbPath, maxLogs, pruneThreshold)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// 1. Insert 9 logs (just below threshold)
	for i := 1; i <= 9; i++ {
		err := store.Append(ctx, logging.Event{
			ID:      fmt.Sprintf("log-%d", i),
			TS:      time.Now().Add(time.Duration(i) * time.Second),
			Source:  "test",
			Message: fmt.Sprintf("message %d", i),
		})
		if err != nil {
			t.Fatalf("failed to append log %d: %v", i, err)
		}
	}

	// Verify count is 9
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 9 {
		t.Fatalf("expected 9 logs, got %d", count)
	}

	// 2. Insert 1 more log (total 10, hits threshold)
	// Note: pruning happens every 100 appends normally, but we triggered it manually 
	// or we can just call the internal prune method directly for the test.
	err = store.Append(ctx, logging.Event{
		ID:      "log-10",
		TS:      time.Now().Add(10 * time.Second),
		Source:  "test",
		Message: "message 10",
	})
	if err != nil {
		t.Fatal(err)
	}

	// The background goroutine might take a millisecond, let's trigger it manually to be deterministic
	store.prune()

	// 3. Verify count is now maxLogs (5)
	err = store.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("expected pruning to 5 logs, got %d", count)
	}

	// 4. Verify we kept the NEWEST logs (IDs 6-10)
	var minID string
	err = store.db.QueryRow("SELECT id FROM events ORDER BY ts_unix_ns ASC LIMIT 1").Scan(&minID)
	if err != nil {
		t.Fatal(err)
	}
	if minID != "log-6" {
		t.Fatalf("expected oldest log to be log-6, got %s", minID)
	}
}
