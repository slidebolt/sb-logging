package logging_test

import (
	"os"
	"testing"

	logging "github.com/slidebolt/sb-logging"
	"github.com/slidebolt/sb-logging/server"
)

func TestDefaultConfigUsesMemoryWhenUnset(t *testing.T) {
	t.Setenv("SB_LOGGING_TARGET", "")
	t.Setenv("SB_LOGGING_SQLITE_PATH", "")
	cfg := logging.DefaultConfig()
	if cfg.Target != "memory" {
		t.Fatalf("Target: got %q want %q", cfg.Target, "memory")
	}
	if cfg.MemoryMaxEvents != 1000 {
		t.Fatalf("MemoryMaxEvents: got %d want %d", cfg.MemoryMaxEvents, 1000)
	}
}

func TestDefaultConfigReadsEnv(t *testing.T) {
	t.Setenv("SB_LOGGING_TARGET", "sqlite")
	t.Setenv("SB_LOGGING_MEMORY_MAX_EVENTS", "250")
	t.Setenv("SB_LOGGING_SQLITE_PATH", " /tmp/slidebolt-logs.db ")
	cfg := logging.DefaultConfig()
	if cfg.Target != "sqlite" {
		t.Fatalf("Target: got %q want %q", cfg.Target, "sqlite")
	}
	if cfg.MemoryMaxEvents != 250 {
		t.Fatalf("MemoryMaxEvents: got %d want %d", cfg.MemoryMaxEvents, 250)
	}
	if cfg.SQLitePath != "/tmp/slidebolt-logs.db" {
		t.Fatalf("SQLitePath: got %q want %q", cfg.SQLitePath, "/tmp/slidebolt-logs.db")
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
	t.Setenv("SB_LOGGING_MEMORY_MAX_EVENTS", "  1500  ")
	t.Setenv("SB_LOGGING_SQLITE_PATH", "  /tmp/ignored.db  ")
	cfg := logging.DefaultConfig()
	if cfg.Target != "memory" {
		t.Fatalf("Target: got %q want %q", cfg.Target, "memory")
	}
	if cfg.MemoryMaxEvents != 1500 {
		t.Fatalf("MemoryMaxEvents: got %d want %d", cfg.MemoryMaxEvents, 1500)
	}
	if cfg.SQLitePath != "/tmp/ignored.db" {
		t.Fatalf("SQLitePath: got %q want %q", cfg.SQLitePath, "/tmp/ignored.db")
	}
}

func TestOpenSQLiteRequiresPath(t *testing.T) {
	_, err := server.New(logging.Config{Target: "sqlite"})
	if err == nil {
		t.Fatal("expected error for sqlite without path")
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
	_ = os.Unsetenv("SB_LOGGING_MEMORY_MAX_EVENTS")
	_ = os.Unsetenv("SB_LOGGING_SQLITE_PATH")
	cfg := logging.DefaultConfig()
	if cfg.Target != "memory" {
		t.Fatalf("Target: got %q want %q", cfg.Target, "memory")
	}
	if cfg.MemoryMaxEvents != 1000 {
		t.Fatalf("MemoryMaxEvents: got %d want %d", cfg.MemoryMaxEvents, 1000)
	}
}
