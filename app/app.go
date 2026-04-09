package app

import (
	"encoding/json"
	"fmt"
	"log"

	contract "github.com/slidebolt/sb-contract"
	logging "github.com/slidebolt/sb-logging"
	"github.com/slidebolt/sb-logging/server"
)

const ServiceID = "sb-logging"

type App struct {
	svc *server.Service
}

func New() *App {
	return &App{}
}

func (a *App) Hello() contract.HelloResponse {
	return contract.HelloResponse{
		ID:              ServiceID,
		Kind:            contract.KindService,
		ContractVersion: contract.ContractVersion,
		DependsOn:       nil,
	}
}

func (a *App) OnStart(_ map[string]json.RawMessage) (json.RawMessage, error) {
	svc, err := server.New(logging.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("open logging service: %w", err)
	}
	a.svc = svc
	log.Println("sb-logging: started")
	return nil, nil
}

func (a *App) OnShutdown() error {
	a.svc = nil
	return nil
}
