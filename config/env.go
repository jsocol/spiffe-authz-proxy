package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

func FromEnv(ctx context.Context) (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
