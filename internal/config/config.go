package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	Port     string `env:"PORT" envDefault:"8080"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	Ethereum EthereumConfig
	Request  RequestConfig
	Cache    CacheConfig
	Metrics  MetricsConfig
}

type EthereumConfig struct {
	RPCEndpoint string `env:"ETH_RPC_ENDPOINT" required:"true"`
	WSEndpoint  string `env:"ETH_WS_ENDPOINT"`
}

type RequestConfig struct {
	Timeout        time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	MaxRetries     int           `env:"MAX_RETRY_ATTEMPTS" envDefault:"3"`
	RetryDelay     time.Duration `env:"RETRY_DELAY" envDefault:"1s"`
	MaxConcurrency int           `env:"MAX_CONCURRENT_REQUESTS" envDefault:"10"`
}

type CacheConfig struct {
	TTL     time.Duration `env:"CACHE_TTL" envDefault:"5m"`
	MaxSize int           `env:"CACHE_MAX_SIZE" envDefault:"1000"`
}

type MetricsConfig struct {
	Enabled        bool `env:"METRICS_ENABLED" envDefault:"true"`
	TracingEnabled bool `env:"TRACING_ENABLED" envDefault:"false"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Request.Timeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}
	if c.Request.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	if c.Cache.MaxSize <= 0 {
		return fmt.Errorf("cache max size must be positive")
	}
	if c.Request.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive")
	}
	return nil
}
