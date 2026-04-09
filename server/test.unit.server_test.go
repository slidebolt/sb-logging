package server

import (
	"context"
	"testing"
	"time"

	logging "github.com/slidebolt/sb-logging"
)

func TestNewMemoryServiceProvidesStore(t *testing.T) {
	svc, err := New(logging.Config{Target: "memory"})
	if err != nil {
		t.Fatalf("New(memory): %v", err)
	}
	if svc.Store() == nil {
		t.Fatal("Store returned nil")
	}
}

func TestServiceStoreAppendAndList(t *testing.T) {
	svc, err := New(logging.Config{Target: "memory"})
	if err != nil {
		t.Fatalf("New(memory): %v", err)
	}

	event := logging.Event{
		ID:      "evt-1",
		TS:      time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		Source:  "plugin-automation",
		Kind:    "command.received",
		Level:   "info",
		Message: "received command",
	}
	if err := svc.Store().Append(context.Background(), event); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := svc.Store().List(context.Background(), logging.ListRequest{Kind: "command.received"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].ID != "evt-1" {
		t.Fatalf("List: got %+v", got)
	}
}
