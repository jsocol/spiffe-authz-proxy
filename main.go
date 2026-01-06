package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"jsocol.io/middleware/logging"
)

type ctxKey string

const spidKey ctxKey = "spiffeid"

func WithSPIFFEID(ctx context.Context, spID spiffeid.ID) context.Context {
	return context.WithValue(ctx, spidKey, spID)
}

func FromContext(ctx context.Context) spiffeid.ID {
	val := ctx.Value(spidKey)
	if spID, ok := val.(spiffeid.ID); ok {
		return spID
	}
	return spiffeid.ID{}
}

type Upstream struct {
	// scheme string
	// host   string
	// port   string
}

// TODO: Handle different types of upstreams, TCP or Unix
func (u *Upstream) GetTransport() *http.Transport {
	dialer := &net.Dialer{}
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", "/some/path")
		},
	}
}

func (u *Upstream) Do(r *http.Request) (*http.Response, error) {
	t := u.GetTransport()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	req := r.WithContext(ctx)

	spid := FromContext(ctx)
	req.Header.Set("SPIFFE-ID", spid.String())

	c := &http.Client{
		Transport: t,
	}

	return c.Do(req)
}

type Route struct {
	Method string
	Path   string
}

func (r *Route) Match(method, path string) bool {
	if r.Method != method {
		return false
	}
	parts := strings.Split(r.Path, "/")
	lastPart := len(parts) - 1

	if parts[0] == "*" {
		return true
	}

	for i, scope := range strings.Split(path, "/") {
		if i > lastPart {
			return parts[lastPart] == "*"
		}
		if !(parts[i] == "*" || parts[i] == scope) {
			return false
		}
	}

	return true
}

type MemoryAuthorizer struct {
	// TODO: This is terribly inefficient and probably needs improvement
	routes map[spiffeid.ID][]Route
}

func (a *MemoryAuthorizer) Authorize(_ context.Context, spid spiffeid.ID, method, path string) error {
	routes, ok := a.routes[spid]
	if !ok {
		return fmt.Errorf("spiffeid %s is not authorized on any routes", spid)
	}

	for _, r := range routes {
		if r.Match(method, path) {
			return nil
		}
	}
	return fmt.Errorf("spiffeid %s is not authorized for method %s on path %s", spid, method, path)
}

type proxyAuthorizer interface {
	Authorize(ctx context.Context, spid spiffeid.ID, method, path string) error
}

type Proxy struct {
	// authz is a map
	authz    proxyAuthorizer
	logger   *slog.Logger
	upstream *Upstream
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clientCert := r.TLS.PeerCertificates[0]
	if len(clientCert.URIs) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		p.logger.DebugContext(ctx, "could not identify spiffeid")
		return
	}

	san := clientCert.URIs[0]
	spID, err := spiffeid.FromURI(san)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		p.logger.DebugContext(ctx, "could not parse uri into spiffeid", "uri", san)
		return
	}

	ctx = WithSPIFFEID(ctx, spID)

	err = p.authz.Authorize(ctx, spID, r.Method, r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		p.logger.DebugContext(ctx, "unauthorized", "spid", spID, "error", err)
		return
	}

	resp, err := p.upstream.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		p.logger.ErrorContext(ctx, "error connecting to upstream", "error", err)
		return
	}

	for header, values := range resp.Header {
		for _, v := range values {
			r.Header.Add(header, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

type sourcer interface {
	x509svid.Source
	x509bundle.Source
}

type SVIDTLSConfig struct {
	logger *slog.Logger
	source sourcer
}

func (stc *SVIDTLSConfig) GetConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			svid, err := stc.source.GetX509SVID()
			if err != nil {
				return nil, err
			}

			certBytes, keyBytes, err := svid.Marshal()
			if err != nil {
				return nil, err
			}

			cert, err := tls.X509KeyPair(certBytes, keyBytes)
			if err != nil {
				return nil, err
			}

			return &cert, nil
		},
		ClientAuth: tls.RequireAnyClientCert,
		VerifyConnection: func(cs tls.ConnectionState) error {
			certs := cs.PeerCertificates
			spid, _, err := x509svid.Verify(certs, stc.source)
			if err != nil {
				return err
			}
			stc.logger.Debug("verified connection", "spiffeid", spid)
			return err
		},
	}
}

func main() {
	// shutdownCh := make(chan struct{})
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startupCtx, startupCancel := context.WithTimeout(ctx, 10*time.Second)
	defer startupCancel()

	x509source, err := workloadapi.NewX509Source(startupCtx)
	if err != nil {
		logger.Error("could not get x509 source", "error", err)
		os.Exit(1)
	}

	upstream := &Upstream{}
	proxy := &Proxy{
		logger:   logger,
		upstream: upstream,
	}

	mux := http.NewServeMux()
	mux.Handle("/", proxy)

	cfger := &SVIDTLSConfig{
		logger: logger,
		source: x509source,
	}

	bindAddr := envDefault("BIND_ADDR", ":8443")
	srv := &http.Server{
		Addr:                         bindAddr,
		DisableGeneralOptionsHandler: true,
		Handler:                      logging.Wrap(mux, logging.WithLogger(logger)),
		TLSConfig:                    cfger.GetConfig(),
	}

	if err := srv.ListenAndServeTLS("", ""); err != nil {
		if err != http.ErrServerClosed {
			logger.Error("error starting server", "error", err)
			os.Exit(10)
		}
	}
}

func envDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
