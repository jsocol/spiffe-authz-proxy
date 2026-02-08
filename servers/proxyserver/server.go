package proxyserver

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type config struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	ProxyHandler      http.Handler
	TLSConfig         *tls.Config
	Metrics           prometheus.Registerer
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
		TLSConfig:                    c.TLSConfig,
	}

	if c.Metrics != nil {
		reqCount := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "inbound_http_request_count",
			Help: "A counter of the number of HTTP requests handled by the proxy server.",
		}, []string{"code", "method"})

		reqDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "inbound_http_request_duration_seconds",
			Help:    "A histogram of latencies for inbound requests.",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 4.0, 8.0, 16.0, 32.0},
		}, []string{"handler", "method"})

		reqInFlight := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "inbound_http_requests_in_flight",
			Help: "A gauge of the number of inbound HTTP requests currently in flight.",
		})

		c.Metrics.MustRegister(reqCount, reqDuration, reqInFlight)

		h.Handler = promhttp.InstrumentHandlerCounter(reqCount,
			promhttp.InstrumentHandlerDuration(reqDuration,
				promhttp.InstrumentHandlerInFlight(reqInFlight, h.Handler),
			),
		)
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

func WithMetrics(r prometheus.Registerer) Option {
	return optionFunc(func(c *config) {
		c.Metrics = r
	})
}
