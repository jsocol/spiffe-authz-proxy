package healthhandler

import (
	"log/slog"
	"net/http"
)

type Updater interface {
	Update() <-chan struct{}
}

type Health struct {
	*http.ServeMux
	sourceUpdater Updater
	logger        *slog.Logger
}

var _ http.Handler = (*Health)(nil)

func New(opts ...Option) *Health {
	c := &config{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt.Apply(c)
	}

	mux := http.NewServeMux()
	h := &Health{
		ServeMux:      mux,
		logger:        c.logger,
		sourceUpdater: c.updater,
	}

	mux.HandleFunc("/ready", h.serveReady)
	mux.HandleFunc("/live", h.serveLive)
	mux.HandleFunc("/startup", h.serveStartup)

	return h
}

func (*Health) serveReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (*Health) serveLive(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (*Health) serveStartup(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type config struct {
	logger  *slog.Logger
	updater Updater
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

func WithUpdater(u Updater) Option {
	return optionFunc(func(c *config) {
		c.updater = u
	})
}
