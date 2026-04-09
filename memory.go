package logging

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type memoryRecord struct {
	seq   int
	event Event
}

type MemoryStore struct {
	mu      sync.RWMutex
	nextSeq int
	records []memoryRecord
	byID    map[string]Event
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byID: map[string]Event{}}
}

func (s *MemoryStore) Append(_ context.Context, event Event) error {
	event.Normalize()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSeq++
	s.records = append(s.records, memoryRecord{seq: s.nextSeq, event: cloneEvent(event)})
	s.byID[event.ID] = cloneEvent(event)
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Event, error) {
	id = strings.TrimSpace(id)

	s.mu.RLock()
	defer s.mu.RUnlock()

	event, ok := s.byID[id]
	if !ok {
		return Event{}, ErrNotFound
	}
	return cloneEvent(event), nil
}

func (s *MemoryStore) List(_ context.Context, req ListRequest) ([]Event, error) {
	req.Normalize()

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]memoryRecord, 0, len(s.records))
	for _, rec := range s.records {
		if matchesEvent(rec.event, req) {
			out = append(out, rec)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].event.TS.Equal(out[j].event.TS) {
			return out[i].seq < out[j].seq
		}
		return out[i].event.TS.Before(out[j].event.TS)
	})

	if req.Limit > 0 && len(out) > req.Limit {
		out = out[:req.Limit]
	}

	events := make([]Event, 0, len(out))
	for _, rec := range out {
		events = append(events, cloneEvent(rec.event))
	}
	return events, nil
}

func matchesEvent(event Event, req ListRequest) bool {
	if !req.Since.IsZero() && event.TS.Before(req.Since) {
		return false
	}
	if !req.Until.IsZero() && event.TS.After(req.Until) {
		return false
	}
	if req.Source != "" && event.Source != req.Source {
		return false
	}
	if req.Kind != "" && event.Kind != req.Kind {
		return false
	}
	if req.Level != "" && event.Level != req.Level {
		return false
	}
	if req.Plugin != "" && event.Plugin != req.Plugin {
		return false
	}
	if req.Device != "" && event.Device != req.Device {
		return false
	}
	if req.Entity != "" && event.Entity != req.Entity {
		return false
	}
	if req.TraceID != "" && event.TraceID != req.TraceID {
		return false
	}
	return true
}

func cloneEvent(event Event) Event {
	cloned := event
	if event.Data != nil {
		cloned.Data = make(map[string]any, len(event.Data))
		for k, v := range event.Data {
			cloned.Data[k] = v
		}
	}
	return cloned
}
