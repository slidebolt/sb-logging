package logging

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Target string
}

func DefaultConfig() Config {
	target := strings.TrimSpace(os.Getenv("SB_LOGGING_TARGET"))
	if target == "" {
		target = "memory"
	}
	return Config{Target: target}
}

func (c Config) Validate() error {
	switch c.Target {
	case "memory":
		return nil
	default:
		return fmt.Errorf("unsupported logging target: %s", c.Target)
	}
}
