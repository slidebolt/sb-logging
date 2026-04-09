package logging_test

import (
	"os"
	"testing"

	logging "github.com/slidebolt/sb-logging"
	"github.com/slidebolt/sb-logging/server"
)

func TestDefaultConfigUsesMemoryWhenUnset(t *testing.T) {
	t.Setenv("SB_LOGGING_TARGET", "")
	cfg := logging.DefaultConfig()
	if cfg.Target != "memory" {
		t.Fatalf("Target: got %q want %q", cfg.Target, "memory")
	}
}

func TestDefaultConfigReadsEnv(t *testing.T) {
	t.Setenv("SB_LOGGING_TARGET", "memory")
	cfg := logging.DefaultConfig()
	if cfg.Target != "memory" {
		t.Fatalf("Target: got %q want %q", cfg.Target, "memory")
	}
}

func TestOpenFromEnvUsesMemory(t *testing.T) {
	t.Setenv("SB_LOGGING_TARGET", "memory")
	svc, err := server.NewFromEnv()
	if err != nil {
		t.Fatalf("OpenFromEnv: %v", err)
	}
	if svc == nil || svc.Store() == nil {
		t.Fatal("OpenFromEnv returned nil store")
	}
}

func TestOpenRejectsUnknownTarget(t *testing.T) {
	_, err := server.New(logging.Config{Target: "postgres"})
	if err == nil {
		t.Fatal("expected error for unknown target")
	}
}

func TestDefaultConfigTrimsEnv(t *testing.T) {
	t.Setenv("SB_LOGGING_TARGET", "  memory  ")
	cfg := logging.DefaultConfig()
	if cfg.Target != "memory" {
		t.Fatalf("Target: got %q want %q", cfg.Target, "memory")
	}
}

func TestDefaultConfigDoesNotReadProcessEnvDirectly(t *testing.T) {
	old, had := os.LookupEnv("SB_LOGGING_TARGET")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("SB_LOGGING_TARGET", old)
		} else {
			_ = os.Unsetenv("SB_LOGGING_TARGET")
		}
	})
	_ = os.Unsetenv("SB_LOGGING_TARGET")
	cfg := logging.DefaultConfig()
	if cfg.Target != "memory" {
		t.Fatalf("Target: got %q want %q", cfg.Target, "memory")
	}
}
