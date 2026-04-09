package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	logging "github.com/slidebolt/sb-logging"
)

type record struct {
	seq   int
	event logging.Event
}

type Store struct {
	mu      sync.RWMutex
	nextSeq int
	records []record
	byID    map[string]logging.Event
}

func New() *Store {
	return &Store{byID: map[string]logging.Event{}}
}

func (s *Store) Append(_ context.Context, event logging.Event) error {
	event.Normalize()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSeq++
	s.records = append(s.records, record{seq: s.nextSeq, event: cloneEvent(event)})
	s.byID[event.ID] = cloneEvent(event)
	return nil
}

func (s *Store) Get(_ context.Context, id string) (logging.Event, error) {
	id = strings.TrimSpace(id)

	s.mu.RLock()
	defer s.mu.RUnlock()

	event, ok := s.byID[id]
	if !ok {
		return logging.Event{}, logging.ErrNotFound
	}
	return cloneEvent(event), nil
}

func (s *Store) List(_ context.Context, req logging.ListRequest) ([]logging.Event, error) {
	req.Normalize()

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]record, 0, len(s.records))
	for _, rec := range s.records {
		if matches(rec.event, req) {
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

	events := make([]logging.Event, 0, len(out))
	for _, rec := range out {
		events = append(events, cloneEvent(rec.event))
	}
	return events, nil
}

func matches(event logging.Event, req logging.ListRequest) bool {
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

func cloneEvent(event logging.Event) logging.Event {
	cloned := event
	if event.Data != nil {
		cloned.Data = make(map[string]any, len(event.Data))
		for k, v := range event.Data {
			cloned.Data[k] = v
		}
	}
	return cloned
}
