package health

import (
	"log/slog"
	"net/http"
)

type updater interface {
	Update() <-chan struct{}
}

type Health struct {
	*http.ServeMux
	sourceUpdater updater
	logger        *slog.Logger
}

func NewHealth() *Health {
	mux := http.NewServeMux()
	h := &Health{
		ServeMux: mux,
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

type HealthOption func(*Health)
