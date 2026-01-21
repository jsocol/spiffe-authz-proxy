package proxyhandler

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"

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
}

func NewProxy(opts ...ProxyOption) *Proxy {
	p := &Proxy{
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(p)
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

		return
	}

	logger := p.logger.With("spiffeid", spID.String())

	ctx = spiffeidutil.WithSPIFFEID(ctx, spID)
	err = p.authz.Authorize(ctx, spID, r.Method, r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		logger.DebugContext(ctx, "unauthorized", "error", err)

		return
	}

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

type ProxyOption func(*Proxy)

func WithLogger(l *slog.Logger) ProxyOption {
	return func(p *Proxy) {
		p.logger = l
	}
}

func WithAuthorizer(a proxyAuthorizer) ProxyOption {
	return func(p *Proxy) {
		p.authz = a
	}
}

func WithUpstream(u upstreamer) ProxyOption {
	return func(p *Proxy) {
		p.upstream = u
	}
}
