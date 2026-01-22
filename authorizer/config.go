package authorizer

import "log/slog"

type config struct {
	logger *slog.Logger
}

func defaultConfig() *config {
	return &config{
		logger: slog.Default(),
	}
}

type Option interface {
	Apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) Apply(c *config) {
	o(c)
}

func WithLogger(l *slog.Logger) Option {
	return optionFunc(func(c *config) {
		c.logger = l
	})
}
