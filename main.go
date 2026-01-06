package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"jsocol.io/middleware/logging"
	"jsocol.io/spiffe-authz-proxy/handlers"
	"jsocol.io/spiffe-authz-proxy/spiffeidutil"
)

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

	spid := spiffeidutil.FromContext(ctx)
	if !spid.IsZero() {
		req.Header.Set("SPIFFE-ID", spid.String())
	}

	c := &http.Client{
		Transport: t,
	}

	return c.Do(req)
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
	_ = upstream
	proxy := handlers.NewProxy()

	cfger := &SVIDTLSConfig{
		logger: logger,
		source: x509source,
	}

	bindAddr := envDefault("BIND_ADDR", ":8443")
	srv := &http.Server{
		Addr:                         bindAddr,
		DisableGeneralOptionsHandler: true,
		Handler:                      logging.Wrap(proxy, logging.WithLogger(logger)),
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
