package server

import (
	"context"
	"encoding/json"
	"fmt"

	logcfg "github.com/slidebolt/sb-logging"
	logging "github.com/slidebolt/sb-logging-sdk"
	"github.com/slidebolt/sb-logging/internal/memory"
	"github.com/slidebolt/sb-logging/internal/sqlite"
	messenger "github.com/slidebolt/sb-messenger-sdk"
)

type Service struct {
	store logging.Store
	close func() error
}

func New(cfg logcfg.Config) (*Service, error) {
	cfg = cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	switch cfg.Target {
	case "memory":
		return &Service{store: memory.New(cfg.MemoryMaxEvents)}, nil
	case "sqlite":
		store, err := sqlite.New(cfg.SQLitePath, cfg.SQLiteMaxEvents, cfg.SQLitePruneThreshold)
		if err != nil {
			return nil, err
		}
		return &Service{store: store, close: store.Close}, nil
	default:
		return nil, cfg.Validate()
	}
}

func NewFromEnv() (*Service, error) {
	return New(logcfg.DefaultConfig())
}

func (s *Service) Store() logging.Store {
	return s.store
}

func (s *Service) Register(msg messenger.Messenger) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("logging service not initialized")
	}
	_, err := msg.Subscribe("logging.>", func(m *messenger.Message) {
		var resp logging.Response
		switch m.Subject {
		case "logging.append":
			var req logging.AppendRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp.Error = "bad request: " + err.Error()
				respond(m, resp)
				return
			}
			if err := s.store.Append(context.Background(), req.Event); err != nil {
				resp.Error = err.Error()
				respond(m, resp)
				return
			}
			resp.OK = true
		case "logging.get":
			var req logging.GetRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp.Error = "bad request: " + err.Error()
				respond(m, resp)
				return
			}
			event, err := s.store.Get(context.Background(), req.ID)
			if err != nil {
				resp.Error = err.Error()
				respond(m, resp)
				return
			}
			resp.OK = true
			resp.Event = &event
		case "logging.list":
			var req logging.ListLogsRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp.Error = "bad request: " + err.Error()
				respond(m, resp)
				return
			}
			events, err := s.store.List(context.Background(), req.Request)
			if err != nil {
				resp.Error = err.Error()
				respond(m, resp)
				return
			}
			resp.OK = true
			resp.Events = events
		default:
			resp.Error = "unknown subject"
		}
		respond(m, resp)
	})
	if err != nil {
		return err
	}
	return msg.Flush()
}

func (s *Service) Close() error {
	if s == nil || s.close == nil {
		return nil
	}
	return s.close()
}

func respond(m *messenger.Message, resp logging.Response) {
	if m == nil {
		return
	}
	data, _ := json.Marshal(resp)
	m.Respond(data)
}
