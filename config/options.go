package config

type Option func(*Config)

func WithDefaults() Option {
	defaultMetric := MetricConfig{Enabled: true}
	return func(cfg *Config) {
		cfg.Temperature = "C"
		cfg.CPU = CPUConfig{
			MetricConfig: defaultMetric,
		}
		cfg.Memory = MemoryConfig{
			MetricConfig: defaultMetric,
			IncludeSwap:  true,
		}
	}
}

func WithSizeFormat(fmt func(int, bool) string) Option {
	return func(cfg *Config) {
		cfg.FormatSize = fmt
	}
}
