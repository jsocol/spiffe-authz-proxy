package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"jsocol.io/middleware/logging"

	"jsocol.io/spiffe-authz-proxy/authorizer"
	"jsocol.io/spiffe-authz-proxy/handlers"
	"jsocol.io/spiffe-authz-proxy/tlsconfig"
	"jsocol.io/spiffe-authz-proxy/upstream"
)

func main() {
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	shutdownCh := make(chan struct{})
	shutdownOnce := sync.OnceFunc(func() {
		close(shutdownCh)
	})

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		shutdownOnce()
	}()

	startupCtx, startupCancel := context.WithTimeout(ctx, 10*time.Second)

	x509source, err := workloadapi.NewX509Source(startupCtx)
	if err != nil {
		logger.Error("could not get x509 source", "error", err)
		os.Exit(1)
	}

	var upstreamOptions []upstream.Option
	upstreamOptions = append(upstreamOptions, upstream.WithTCP("127.0.0.1", "5005"))
	up, err := upstream.New(upstreamOptions...)
	if err != nil {
		logger.Error("could not create upstream", "error", err)
		os.Exit(2)
	}

	authz := authorizer.NewMemoryAuthorizer()

	proxy := handlers.NewProxy(
		handlers.WithUpstream(up),
		handlers.WithLogger(logger),
		handlers.WithAuthorizer(authz),
	)

	cfger, err := tlsconfig.New(
		tlsconfig.WithLogger(logger),
		tlsconfig.WithSource(x509source),
	)
	if err != nil {
		logger.Error("could not create tls config", "error", err)
		os.Exit(3)
	}

	bindAddr := envDefault("BIND_ADDR", ":8443")
	srv := &http.Server{
		Addr:                         bindAddr,
		DisableGeneralOptionsHandler: true,
		Handler:                      logging.Wrap(proxy, logging.WithLogger(logger)),
		TLSConfig:                    cfger.GetConfig(),
	}

	startupCancel()

	go func() {
		<-shutdownCh
		srv.Shutdown(ctx)
		cancel()
	}()

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
