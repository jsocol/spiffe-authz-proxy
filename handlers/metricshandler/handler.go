package metricshandler

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var _ http.Handler = (*Metrics)(nil)

type Metrics struct {
	*http.ServeMux

	Gatherer   prometheus.Gatherer
	Registerer prometheus.Registerer
	Logger     *slog.Logger
}

func New(opts ...Option) *Metrics {
	c := &config{}
	for _, o := range opts {
		o.Apply(c)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(c.Gatherer, promhttp.HandlerOpts{
		Registry: c.Registerer,
		ErrorLog: slog.NewLogLogger(c.Logger.Handler(), slog.LevelInfo),
	}))

	m := &Metrics{
		ServeMux:   mux,
		Gatherer:   c.Gatherer,
		Registerer: c.Registerer,
		Logger:     c.Logger,
	}

	return m
}

type config struct {
	Gatherer   prometheus.Gatherer
	Registerer prometheus.Registerer
	Logger     *slog.Logger
}

type Option interface {
	Apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) Apply(c *config) {
	o(c)
}

func WithRegistry(r *prometheus.Registry) Option {
	return optionFunc(func(c *config) {
		c.Gatherer = r
		c.Registerer = r
	})
}

func WithGatherer(g prometheus.Gatherer) Option {
	return optionFunc(func(c *config) {
		c.Gatherer = g
	})
}

func WithRegisterer(r prometheus.Registerer) Option {
	return optionFunc(func(c *config) {
		c.Registerer = r
	})
}

func WithLogger(l *slog.Logger) Option {
	return optionFunc(func(c *config) {
		c.Logger = l
	})
}
