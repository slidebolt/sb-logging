package logging

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Target                string
	MemoryMaxEvents       int
	SQLitePath            string
	SQLiteMaxEvents       int
	SQLitePruneThreshold int
}

func DefaultConfig() Config {
	target := strings.TrimSpace(os.Getenv("SB_LOGGING_TARGET"))
	if target == "" {
		target = "memory"
	}
	memoryMax := 1000
	if raw := strings.TrimSpace(os.Getenv("SB_LOGGING_MEMORY_MAX_EVENTS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			memoryMax = n
		}
	}

	sqliteMax := 100000
	if raw := strings.TrimSpace(os.Getenv("SB_LOGGING_SQLITE_MAX_LOGS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			sqliteMax = n
		}
	}

	sqlitePrune := 150000
	if raw := strings.TrimSpace(os.Getenv("SB_LOGGING_SQLITE_PRUNE_LOGS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			sqlitePrune = n
		}
	}

	return Config{
		Target:                target,
		MemoryMaxEvents:       memoryMax,
		SQLitePath:            strings.TrimSpace(os.Getenv("SB_LOGGING_SQLITE_PATH")),
		SQLiteMaxEvents:       sqliteMax,
		SQLitePruneThreshold: sqlitePrune,
	}
}

func (c Config) Validate() error {
	c = c.Normalize()
	switch c.Target {
	case "memory":
		if c.MemoryMaxEvents <= 0 {
			return fmt.Errorf("memory max events must be > 0")
		}
		return nil
	case "sqlite":
		if c.SQLitePath == "" {
			return fmt.Errorf("sqlite path is required")
		}
		return nil
	default:
		return fmt.Errorf("unsupported logging target: %s", c.Target)
	}
}

func (c Config) Normalize() Config {
	c.Target = strings.TrimSpace(c.Target)
	c.SQLitePath = strings.TrimSpace(c.SQLitePath)
	if c.Target == "" {
		c.Target = "memory"
	}
	if c.Target == "memory" && c.MemoryMaxEvents <= 0 {
		c.MemoryMaxEvents = 1000
	}
	if c.SQLiteMaxEvents <= 0 {
		c.SQLiteMaxEvents = 100000
	}
	if c.SQLitePruneThreshold <= 0 {
		c.SQLitePruneThreshold = 150000
	}
	if c.SQLitePruneThreshold < c.SQLiteMaxEvents {
		c.SQLitePruneThreshold = c.SQLiteMaxEvents
	}
	return c
}
