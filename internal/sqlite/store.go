package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"

	logging "github.com/slidebolt/sb-logging-sdk"
	"github.com/slidebolt/sb-logging/internal/storeutil"
)

type Store struct {
	db             *sql.DB
	maxEvents      int
	pruneThreshold int
	appendCount    uint64
}

func New(path string, maxEvents, pruneThreshold int) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	if _, err := db.Exec(`
PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS events (
	seq INTEGER PRIMARY KEY AUTOINCREMENT,
	id TEXT NOT NULL,
	ts TEXT NOT NULL,
	ts_unix_ns INTEGER NOT NULL,
	source TEXT NOT NULL,
	kind TEXT NOT NULL,
	level TEXT NOT NULL,
	message TEXT NOT NULL,
	plugin TEXT NOT NULL DEFAULT '',
	device TEXT NOT NULL DEFAULT '',
	entity TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL DEFAULT '',
	trace_id TEXT NOT NULL DEFAULT '',
	data TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_events_id ON events(id);
CREATE INDEX IF NOT EXISTS idx_events_ts_seq ON events(ts_unix_ns, seq);
CREATE INDEX IF NOT EXISTS idx_events_source ON events(source);
CREATE INDEX IF NOT EXISTS idx_events_kind ON events(kind);
CREATE INDEX IF NOT EXISTS idx_events_level ON events(level);
CREATE INDEX IF NOT EXISTS idx_events_plugin ON events(plugin);
CREATE INDEX IF NOT EXISTS idx_events_device ON events(device);
CREATE INDEX IF NOT EXISTS idx_events_entity ON events(entity);
CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);
`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init sqlite schema: %w", err)
	}
	return &Store{
		db:             db,
		maxEvents:      maxEvents,
		pruneThreshold: pruneThreshold,
	}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Append(ctx context.Context, event logging.Event) error {
	event.Normalize()
	data := "{}"
	if event.Data != nil {
		raw, err := json.Marshal(event.Data)
		if err != nil {
			return fmt.Errorf("marshal event data: %w", err)
		}
		data = string(raw)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO events (id, ts, ts_unix_ns, source, kind, level, message, plugin, device, entity, action, trace_id, data)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, event.ID, event.TS.Format(timeFormat), event.TS.UnixNano(), event.Source, event.Kind, event.Level, event.Message, event.Plugin, event.Device, event.Entity, event.Action, event.TraceID, data)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	// Check for pruning every 100 appends to minimize overhead.
	if atomic.AddUint64(&s.appendCount, 1)%100 == 0 {
		go s.prune()
	}

	return nil
}

func (s *Store) prune() {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil {
		log.Printf("logging: prune count error: %v", err)
		return
	}

	if count < s.pruneThreshold {
		return
	}

	// Delete oldest records to reach maxEvents.
	toDelete := count - s.maxEvents
	if toDelete <= 0 {
		return
	}

	_, err = s.db.Exec(`
DELETE FROM events
WHERE seq IN (
	SELECT seq FROM events
	ORDER BY ts_unix_ns ASC, seq ASC
	LIMIT ?
)
`, toDelete)
	if err != nil {
		log.Printf("logging: prune delete error: %v", err)
		return
	}

	log.Printf("logging: pruned %d old records from sqlite", toDelete)
}

func (s *Store) Get(ctx context.Context, id string) (logging.Event, error) {
	id = strings.TrimSpace(id)
	var event logging.Event
	var ts string
	var data string
	row := s.db.QueryRowContext(ctx, `
SELECT id, ts, source, kind, level, message, plugin, device, entity, action, trace_id, data
FROM events
WHERE id = ?
ORDER BY seq DESC
LIMIT 1
`, id)
	if err := row.Scan(&event.ID, &ts, &event.Source, &event.Kind, &event.Level, &event.Message, &event.Plugin, &event.Device, &event.Entity, &event.Action, &event.TraceID, &data); err != nil {
		if err == sql.ErrNoRows {
			return logging.Event{}, logging.ErrNotFound
		}
		return logging.Event{}, fmt.Errorf("get event: %w", err)
	}
	parsedTS, err := time.Parse(timeFormat, ts)
	if err != nil {
		return logging.Event{}, fmt.Errorf("parse event ts: %w", err)
	}
	event.TS = parsedTS
	if err := decodeData(&event, data); err != nil {
		return logging.Event{}, err
	}
	return storeutil.CloneEvent(event), nil
}

func (s *Store) List(ctx context.Context, req logging.ListRequest) ([]logging.Event, error) {
	req.Normalize()
	args := make([]any, 0, 9)
	clauses := make([]string, 0, 9)
	if !req.Since.IsZero() {
		clauses = append(clauses, "ts_unix_ns >= ?")
		args = append(args, req.Since.UnixNano())
	}
	if !req.Until.IsZero() {
		clauses = append(clauses, "ts_unix_ns <= ?")
		args = append(args, req.Until.UnixNano())
	}
	for _, field := range []struct {
		value string
		sql   string
	}{
		{req.Source, "source = ?"},
		{req.Kind, "kind = ?"},
		{req.Level, "level = ?"},
		{req.Plugin, "plugin = ?"},
		{req.Device, "device = ?"},
		{req.Entity, "entity = ?"},
		{req.TraceID, "trace_id = ?"},
	} {
		if field.value != "" {
			clauses = append(clauses, field.sql)
			args = append(args, field.value)
		}
	}
	query := `
SELECT id, ts, source, kind, level, message, plugin, device, entity, action, trace_id, data
FROM events`
	if len(clauses) > 0 {
		query += "\nWHERE " + strings.Join(clauses, " AND ")
	}
	query += "\nORDER BY ts_unix_ns ASC, seq ASC"
	if req.Limit > 0 {
		query += "\nLIMIT ?"
		args = append(args, req.Limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	events := []logging.Event{}
	for rows.Next() {
		var event logging.Event
		var ts string
		var data string
		if err := rows.Scan(&event.ID, &ts, &event.Source, &event.Kind, &event.Level, &event.Message, &event.Plugin, &event.Device, &event.Entity, &event.Action, &event.TraceID, &data); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		parsedTS, err := time.Parse(timeFormat, ts)
		if err != nil {
			return nil, fmt.Errorf("parse event ts: %w", err)
		}
		event.TS = parsedTS
		if err := decodeData(&event, data); err != nil {
			return nil, err
		}
		events = append(events, storeutil.CloneEvent(event))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return events, nil
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

func decodeData(event *logging.Event, data string) error {
	event.Normalize()
	if strings.TrimSpace(data) == "" || strings.TrimSpace(data) == "{}" {
		event.Data = nil
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(data), &decoded); err != nil {
		return fmt.Errorf("decode event data: %w", err)
	}
	event.Data = decoded
	return nil
}
