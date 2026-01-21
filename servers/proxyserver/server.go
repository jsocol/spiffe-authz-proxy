package proxyserver

import (
	"crypto/tls"
	"net/http"
	"time"
)

type config struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	ProxyHandler      http.Handler
	TLSConfig         *tls.Config
}

func defaultConfig() *config {
	return &config{
		Addr:              ":8443",
		ReadTimeout:       time.Second,
		ReadHeaderTimeout: time.Second,
	}
}

func New(opts ...Option) *http.Server {
	c := defaultConfig()
	for _, opt := range opts {
		opt.Apply(c)
	}

	h := &http.Server{
		Addr:                         c.Addr,
		Handler:                      c.ProxyHandler,
		DisableGeneralOptionsHandler: true,
		ReadTimeout:                  c.ReadTimeout,
		ReadHeaderTimeout:            c.ReadHeaderTimeout,
	}

	return h
}

type Option interface {
	Apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) Apply(c *config) {
	o(c)
}

func WithAddr(a string) Option {
	return optionFunc(func(c *config) {
		c.Addr = a
	})
}

func WithProxyHandler(p http.Handler) Option {
	return optionFunc(func(c *config) {
		c.ProxyHandler = p
	})
}

func WithReadTimeout(t time.Duration) Option {
	return optionFunc(func(c *config) {
		c.ReadTimeout = t
	})
}

func WithReadHeaderTimeout(t time.Duration) Option {
	return optionFunc(func(c *config) {
		c.ReadHeaderTimeout = t
	})
}

func WithTLSConfig(t *tls.Config) Option {
	return optionFunc(func(c *config) {
		c.TLSConfig = t
	})
}
