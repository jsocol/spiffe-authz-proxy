package upstream

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"jsocol.io/spiffe-authz-proxy/spiffeidutil"
)

type RoundTripperWrapper func(http.RoundTripper) http.RoundTripper

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Upstream struct {
	addr   net.Addr
	client httpDoer
}

func New(opts ...Option) (_ *Upstream, err error) {
	c := &config{}
	for _, opt := range opts {
		opt.Apply(c)
	}

	u := &Upstream{
		addr: c.addr,
	}

	if u.addr == nil {
		return nil, errors.New("an upstream address is required")
	}

	dialer := &net.Dialer{}
	var t http.RoundTripper = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, u.addr.Network(), u.addr.String())
			if err != nil {
				return nil, fmt.Errorf("could not dial upstream %s: %v", u.addr.String(), err)
			}

			return conn, nil
		},
	}

	for _, wrap := range c.wrappers {
		t = wrap(t)
	}

	if c.metrics != nil {
		reqCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "upstream_http_request_count",
			Help: "A counter of upstream requests from the proxy.",
		}, []string{"code", "method"})

		reqDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "upstream_http_request_duration_seconds",
			Help: "A histogram of latencies for upstream requests.",
		}, []string{"code", "method"})

		reqInFlight := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "upstream_http_requests_in_flight",
			Help: "A gauge of the number of upstream HTTP requests currently in flight.",
		})

		c.metrics.MustRegister(reqCounter, reqDuration, reqInFlight)

		t = promhttp.InstrumentRoundTripperCounter(reqCounter,
			promhttp.InstrumentRoundTripperDuration(reqDuration,
				promhttp.InstrumentRoundTripperInFlight(reqInFlight, t),
			),
		)
	}

	u.client = &http.Client{
		Transport: t,
	}

	return u, nil
}

func (u *Upstream) Proxy(r *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	req := r.WithContext(ctx)

	spid := spiffeidutil.FromContext(ctx)
	if !spid.IsZero() {
		req.Header.Set("SPIFFE-ID", spid.String())
	}

	return u.client.Do(req)
}

func (u *Upstream) Addr() net.Addr {
	return u.addr
}

type Option interface {
	Apply(*config)
}

type config struct {
	addr     net.Addr
	wrappers []RoundTripperWrapper
	metrics  prometheus.Registerer
}

type optionFunc func(*config)

func (o optionFunc) Apply(c *config) {
	o(c)
}

func WithAddr(a net.Addr) Option {
	return optionFunc(func(c *config) {
		c.addr = a
	})
}

func WithRoundTripperWrappers(wrappers ...RoundTripperWrapper) Option {
	return optionFunc(func(c *config) {
		c.wrappers = append(c.wrappers, wrappers...)
	})
}

func WithMetrics(r prometheus.Registerer) Option {
	return optionFunc(func(c *config) {
		c.metrics = r
	})
}
