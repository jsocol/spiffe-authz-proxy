package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"jsocol.io/middleware/logging"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	x509source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		logger.Error("could not get x509 source", "error", err)
		os.Exit(1)
	}

	svid, err := x509source.GetX509SVID()
	if err != nil {
		logger.Error("error fetching svid", "error", err)
		os.Exit(2)
	}

	pemCert, pemKey, err := svid.Marshal()
	if err != nil {
		logger.Error("error decoding svid", "error", err)
		os.Exit(3)
	}

	cert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		logger.Error("error parsing key pair", "error", err)
		os.Exit(4)
	}

	rawTrustDomain := os.Getenv("TRUST_DOMAIN")
	trustDomain, err := spiffeid.TrustDomainFromString(rawTrustDomain)
	if err != nil {
		logger.Error("could not parse trust domain", "error", err, "trust_domain", rawTrustDomain)
		os.Exit(5)
	}

	bundle, err := x509source.GetX509BundleForTrustDomain(trustDomain)
	if err != nil {
		logger.Error("could not fetch bundle for trust domain", "error", err, "trust_domain", trustDomain)
		os.Exit(6)
	}

	pemBundle, err := bundle.Marshal()
	if err != nil {
		logger.Error("could not marshal bundle for trust domain", "error", err, "trust_domain", trustDomain)
		os.Exit(7)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pemBundle)

	upstream := &http.Client{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		clientCert := r.TLS.PeerCertificates[0]
		subj := clientCert.Subject
		svid, err := spiffeid.FromString(subj.String())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.DebugContext(ctx, "could not parse subject into spiffeid", "subject", subj)
			return
		}

		path := r.URL.Path

		// TODO: Check path/route vs svid
		_, _ = svid, path

		resp, err := upstream.Get(r.URL.String())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.ErrorContext(ctx, "error connecting to upstream", "error", err)
			return
		}

		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})

	bindAddr := envDefault("BIND_ADDR", ":8443")
	srv := &http.Server{
		Addr:    bindAddr,
		Handler: logging.Wrap(mux, logging.WithLogger(logger)),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    pool,
		},
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
