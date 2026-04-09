package logging

func Open(cfg Config) (Store, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	switch cfg.Target {
	case "memory":
		return NewMemoryStore(), nil
	default:
		return nil, cfg.Validate()
	}
}

func OpenFromEnv() (Store, error) {
	return Open(DefaultConfig())
}
