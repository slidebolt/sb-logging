package logging_test

import (
	"context"
	"errors"
	"testing"
	"time"

	logging "github.com/slidebolt/sb-logging"
)

func newMemoryStore(t *testing.T) logging.Store {
	t.Helper()
	store, err := logging.Open(logging.Config{Target: "memory"})
	if err != nil {
		t.Fatalf("Open(memory): %v", err)
	}
	return store
}

func TestMemoryStoreAppendGetAndList(t *testing.T) {
	store := newMemoryStore(t)
	ctx := context.Background()
	base := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)

	first := logging.Event{
		ID:      "evt-1",
		TS:      base,
		Source:  "plugin-automation",
		Kind:    "command.emit",
		Level:   "info",
		Message: "group turn_on",
		Plugin:  "plugin-automation",
		Entity:  "plugin-automation.group.basement",
		TraceID: "trace-1",
		Data:    map[string]any{"command": "light_turn_on"},
	}
	second := logging.Event{
		ID:      "evt-2",
		TS:      base.Add(2 * time.Second),
		Source:  "sb-script",
		Kind:    "automation.trigger",
		Level:   "info",
		Message: "automation fired",
		Plugin:  "plugin-esphome",
		Device:  "switch_main_basement",
		Entity:  "plugin-esphome.switch_main_basement.switch_main_basement_3558733165",
		TraceID: "trace-2",
	}

	if err := store.Append(ctx, first); err != nil {
		t.Fatalf("Append(first): %v", err)
	}
	if err := store.Append(ctx, second); err != nil {
		t.Fatalf("Append(second): %v", err)
	}

	got, err := store.Get(ctx, "evt-1")
	if err != nil {
		t.Fatalf("Get(evt-1): %v", err)
	}
	if got.ID != first.ID || got.Message != first.Message {
		t.Fatalf("Get(evt-1): got %+v want %+v", got, first)
	}

	events, err := store.List(ctx, logging.ListRequest{})
	if err != nil {
		t.Fatalf("List(all): %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("List(all): got %d want 2", len(events))
	}
	if events[0].ID != "evt-1" || events[1].ID != "evt-2" {
		t.Fatalf("List(all) order: got %q, %q", events[0].ID, events[1].ID)
	}
}

func TestMemoryStoreListFiltersLiterally(t *testing.T) {
	store := newMemoryStore(t)
	ctx := context.Background()
	base := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)

	events := []logging.Event{
		{
			ID:      "evt-1",
			TS:      base,
			Source:  "plugin-automation",
			Kind:    "command.emit",
			Level:   "info",
			Message: "basement on",
			Plugin:  "plugin-automation",
			Entity:  "plugin-automation.group.basement",
			TraceID: "trace-1",
		},
		{
			ID:      "evt-2",
			TS:      base.Add(time.Second),
			Source:  "plugin-automation",
			Kind:    "command.emit",
			Level:   "warn",
			Message: "bar on",
			Plugin:  "plugin-automation",
			Entity:  "plugin-automation.group.bar",
			TraceID: "trace-2",
		},
		{
			ID:      "evt-3",
			TS:      base.Add(2 * time.Second),
			Source:  "sb-script",
			Kind:    "automation.trigger",
			Level:   "info",
			Message: "main basement fired",
			Plugin:  "plugin-esphome",
			Device:  "switch_main_basement",
			Entity:  "plugin-esphome.switch_main_basement.switch_main_basement_3558733165",
			TraceID: "trace-1",
		},
	}

	for _, event := range events {
		if err := store.Append(ctx, event); err != nil {
			t.Fatalf("Append(%s): %v", event.ID, err)
		}
	}

	tests := []struct {
		name string
		req  logging.ListRequest
		want []string
	}{
		{
			name: "trace_id",
			req:  logging.ListRequest{TraceID: "trace-1"},
			want: []string{"evt-1", "evt-3"},
		},
		{
			name: "entity",
			req:  logging.ListRequest{Entity: "plugin-automation.group.basement"},
			want: []string{"evt-1"},
		},
		{
			name: "source and level",
			req:  logging.ListRequest{Source: "plugin-automation", Level: "warn"},
			want: []string{"evt-2"},
		},
		{
			name: "since until",
			req: logging.ListRequest{
				Since: base.Add(500 * time.Millisecond),
				Until: base.Add(1500 * time.Millisecond),
			},
			want: []string{"evt-2"},
		},
		{
			name: "limit",
			req:  logging.ListRequest{Source: "plugin-automation", Limit: 1},
			want: []string{"evt-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.List(ctx, tt.req)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("List len: got %d want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].ID != tt.want[i] {
					t.Fatalf("List[%d]: got %q want %q", i, got[i].ID, tt.want[i])
				}
			}
		})
	}
}

func TestMemoryStoreGetMissingReturnsErrNotFound(t *testing.T) {
	store := newMemoryStore(t)
	_, err := store.Get(context.Background(), "missing")
	if !errors.Is(err, logging.ErrNotFound) {
		t.Fatalf("Get(missing): got %v want ErrNotFound", err)
	}
}

func TestMemoryStoreReturnsClonedEvents(t *testing.T) {
	store := newMemoryStore(t)
	ctx := context.Background()

	original := logging.Event{
		ID:      "evt-1",
		TS:      time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		Source:  "plugin-automation",
		Kind:    "command.emit",
		Level:   "info",
		Message: "message",
		Data:    map[string]any{"key": "value"},
	}
	if err := store.Append(ctx, original); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := store.Get(ctx, "evt-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got.Data["key"] = "changed"

	again, err := store.Get(ctx, "evt-1")
	if err != nil {
		t.Fatalf("Get again: %v", err)
	}
	if again.Data["key"] != "value" {
		t.Fatalf("Get clone: got %v want %v", again.Data["key"], "value")
	}
}
