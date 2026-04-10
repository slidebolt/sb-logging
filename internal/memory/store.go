package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	logging "github.com/slidebolt/sb-logging-sdk"
	"github.com/slidebolt/sb-logging/internal/storeutil"
)

type record struct {
	seq   int
	event logging.Event
}

type Store struct {
	maxEvents int
	mu        sync.RWMutex
	nextSeq   int
	records   []record
	byID      map[string]logging.Event
}

func New(maxEvents int) *Store {
	if maxEvents <= 0 {
		maxEvents = 1000
	}
	return &Store{
		maxEvents: maxEvents,
		byID:      map[string]logging.Event{},
	}
}

func (s *Store) Append(_ context.Context, event logging.Event) error {
	event.Normalize()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSeq++
	s.records = append(s.records, record{seq: s.nextSeq, event: storeutil.CloneEvent(event)})
	s.byID[event.ID] = storeutil.CloneEvent(event)
	s.pruneLocked()
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
	return storeutil.CloneEvent(event), nil
}

func (s *Store) List(_ context.Context, req logging.ListRequest) ([]logging.Event, error) {
	req.Normalize()

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]record, 0, len(s.records))
	for _, rec := range s.records {
		if storeutil.Matches(rec.event, req) {
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
		events = append(events, storeutil.CloneEvent(rec.event))
	}
	return events, nil
}

func (s *Store) pruneLocked() {
	if s.maxEvents <= 0 || len(s.records) <= s.maxEvents {
		return
	}
	drop := len(s.records) - s.maxEvents
	for _, rec := range s.records[:drop] {
		delete(s.byID, rec.event.ID)
	}
	remaining := make([]record, len(s.records[drop:]))
	copy(remaining, s.records[drop:])
	s.records = remaining
}
