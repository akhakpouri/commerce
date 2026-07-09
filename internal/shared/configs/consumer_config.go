package configs

type ConsumerConfig struct {
	Url      string
	Max      int32
	Timeout  int32
	WaitTime int32
	Count    int
}

func (cfg ConsumerConfig) Validate() *ConsumerConfig {
	if cfg.Count <= 0 {
		cfg.Count = 5
	}
	if cfg.Max <= 0 || cfg.Max > 10 {
		cfg.Max = 10
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}
	if cfg.WaitTime <= 0 {
		cfg.WaitTime = 20
	}
	return &cfg
}
