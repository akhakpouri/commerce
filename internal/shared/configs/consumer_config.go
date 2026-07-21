package configs

type ConsumerConfig struct {
	Url      string
	Max      int
	Timeout  int
	WaitTime int
	Count    int
}

func (cfg ConsumerConfig) Validate() *ConsumerConfig {
	if cfg.Url == "" {
		panic("consumer queue URL is required")
	}
	if cfg.Count <= 0 {
		cfg.Count = 5
	}
	if cfg.Max <= 0 || cfg.Max > 10 {
		cfg.Max = 10
	}
	// Consumer.process() gives the handler (timeout - 5) seconds, reserving 5s
	// to call delete() before the message becomes visible again. Timeout <= 5
	// would make that deadline zero or negative, so every message would fail
	// before the handler ever runs.
	if cfg.Timeout <= 5 {
		cfg.Timeout = 30
	}
	if cfg.WaitTime <= 0 {
		cfg.WaitTime = 20
	}
	return &cfg
}
