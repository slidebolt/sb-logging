package app

import (
	"encoding/json"
	"fmt"
	"log"

	contract "github.com/slidebolt/sb-contract"
	logcfg "github.com/slidebolt/sb-logging"
	messenger "github.com/slidebolt/sb-messenger-sdk"
	"github.com/slidebolt/sb-logging/server"
)

const ServiceID = "sb-logging"

type App struct {
	svc *server.Service
	msg messenger.Messenger
}

func New() *App {
	return &App{}
}

func (a *App) Hello() contract.HelloResponse {
	return contract.HelloResponse{
		ID:              ServiceID,
		Kind:            contract.KindService,
		ContractVersion: contract.ContractVersion,
		DependsOn:       []string{"messenger"},
	}
}

func (a *App) OnStart(deps map[string]json.RawMessage) (json.RawMessage, error) {
	msg, err := messenger.Connect(deps)
	if err != nil {
		return nil, fmt.Errorf("connect messenger: %w", err)
	}
	a.msg = msg

	svc, err := server.New(logcfg.DefaultConfig())
	if err != nil {
		msg.Close()
		a.msg = nil
		return nil, fmt.Errorf("open logging service: %w", err)
	}
	if err := svc.Register(msg); err != nil {
		_ = svc.Close()
		msg.Close()
		a.msg = nil
		return nil, fmt.Errorf("register logging service: %w", err)
	}
	a.svc = svc
	log.Println("sb-logging: started")
	return nil, nil
}

func (a *App) OnShutdown() error {
	if a.svc != nil {
		if err := a.svc.Close(); err != nil {
			return err
		}
	}
	a.svc = nil
	if a.msg != nil {
		a.msg.Close()
		a.msg = nil
	}
	return nil
}
