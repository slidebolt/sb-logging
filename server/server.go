package server

import (
	logging "github.com/slidebolt/sb-logging"
	"github.com/slidebolt/sb-logging/internal/memory"
)

// Service provides the importable sb-logging store surface. Backends are
// selected from the logging.Config contract and exposed via the same Store API.
type Service struct {
	store logging.Store
}

func New(cfg logging.Config) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	switch cfg.Target {
	case "memory":
		return &Service{store: memory.New()}, nil
	default:
		return nil, cfg.Validate()
	}
}

func NewFromEnv() (*Service, error) {
	return New(logging.DefaultConfig())
}

func (s *Service) Store() logging.Store {
	return s.store
}
