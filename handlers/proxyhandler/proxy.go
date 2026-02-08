package proxyhandler

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"

	"jsocol.io/spiffe-authz-proxy/spiffeidutil"
)

type proxyAuthorizer interface {
	Authorize(ctx context.Context, spid spiffeid.ID, method, path string) error
}

type upstreamer interface {
	Do(*http.Request) (*http.Response, error)
}

type Proxy struct {
	logger   *slog.Logger
	authz    proxyAuthorizer
	upstream upstreamer
	metrics  *proxyMetrics
}

func New(opts ...Option) *Proxy {
	c := &config{
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt.Apply(c)
	}

	p := &Proxy{
		logger:   c.logger,
		authz:    c.authz,
		upstream: c.upstream,
	}

	if c.metrics != nil {
		m := &proxyMetrics{
			errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "proxy_authz_error_count",
				Help: "A counter of errors that occur during AuthN/AuthZ.",
			}, []string{"reason"}),
			results: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "proxy_authz_result_count",
				Help: "A counter of AuthZ results.",
			}, []string{"result"}),
		}

		c.metrics.MustRegister(m.errors, m.results)

		p.metrics = m
	}

	return p
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clientCert := r.TLS.PeerCertificates[0]
	spID, err := x509svid.IDFromCert(clientCert)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		p.logger.DebugContext(ctx, "could not parse uri into spiffeid")
		p.metrics.Error("no_spiffeid")

		return
	}

	logger := p.logger.With("spiffeid", spID.String())

	ctx = spiffeidutil.WithSPIFFEID(ctx, spID)
	err = p.authz.Authorize(ctx, spID, r.Method, r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		logger.DebugContext(ctx, "unauthorized", "error", err)
		p.metrics.Result("unauthorized")

		return
	}

	p.metrics.Result("authorized")

	upstreamURL := &url.URL{}
	*upstreamURL = *r.URL
	upstreamURL.Scheme = "http"
	upstreamURL.Host = r.Host

	logger = logger.With("upstreamURL", upstreamURL)
	req, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL.String(), r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.ErrorContext(ctx, "error creating upstream request", "error", err)

		return
	}

	maps.Copy(req.Header, r.Header)
	req.Header.Add("Host", r.Host)
	req.Header.Add("Spiffe-Id", spID.String())
	resp, err := p.upstream.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.ErrorContext(ctx, "error connecting to upstream", "error", err)

		return
	}

	respHeader := w.Header()
	maps.Copy(respHeader, resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		logger.ErrorContext(ctx, "failed to write response", "error", err)
	}
}

type config struct {
	logger   *slog.Logger
	upstream upstreamer
	authz    proxyAuthorizer
	metrics  prometheus.Registerer
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

func WithAuthorizer(a proxyAuthorizer) Option {
	return optionFunc(func(c *config) {
		c.authz = a
	})
}

func WithUpstream(u upstreamer) Option {
	return optionFunc(func(c *config) {
		c.upstream = u
	})
}

func WithMetrics(r prometheus.Registerer) Option {
	return optionFunc(func(c *config) {
		c.metrics = r
	})
}

type proxyMetrics struct {
	errors  *prometheus.CounterVec
	results *prometheus.CounterVec
}

func (pm *proxyMetrics) Error(reason string) {
	if pm != nil {
		pm.errors.With(prometheus.Labels{"reason": reason}).Inc()
	}
}

func (pm *proxyMetrics) Result(result string) {
	if pm != nil {
		pm.results.With(prometheus.Labels{"result": result}).Inc()
	}
}
