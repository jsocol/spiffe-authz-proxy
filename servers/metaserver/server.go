package metaserver

import (
	"net/http"
	"time"
)

type config struct {
	Addr          string
	HealthHandler http.Handler
	ReadTimeout   time.Duration
}

func defaultConfig() *config {
	return &config{
		Addr:        ":8081",
		ReadTimeout: time.Second,
	}
}

func New(opts ...Option) *http.Server {
	c := defaultConfig()
	for _, opt := range opts {
		opt.Apply(c)
	}

	mux := http.NewServeMux()
	mux.Handle("/health", http.StripPrefix("/health", c.HealthHandler))

	srv := &http.Server{
		Addr:        c.Addr,
		Handler:     mux,
		ReadTimeout: c.ReadTimeout,
	}

	return srv
}

type Option interface {
	Apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) Apply(c *config) {
	o(c)
}

func WithHealthHandler(h http.Handler) Option {
	return optionFunc(func(c *config) {
		c.HealthHandler = h
	})
}

func WithReadTimeout(t time.Duration) Option {
	return optionFunc(func(c *config) {
		c.ReadTimeout = t
	})
}

func WithAddr(a string) Option {
	return optionFunc(func(c *config) {
		c.Addr = a
	})
}
