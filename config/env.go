package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

func FromEnv(ctx context.Context) *Config {
	cfg := &Config{}
	_ = envconfig.Process(ctx, cfg)
	return cfg
}
