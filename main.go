package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"jsocol.io/middleware/logging"

	"jsocol.io/spiffe-authz-proxy/authorizer"
	"jsocol.io/spiffe-authz-proxy/config"
	"jsocol.io/spiffe-authz-proxy/handlers"
	"jsocol.io/spiffe-authz-proxy/logutils"
	"jsocol.io/spiffe-authz-proxy/upstream"
	"jsocol.io/spiffe-authz-proxy/x509util"
)

const (
	exitCodeNoConfig = iota
	exitCodeX509Source
	exitCodeBadConfig
	exitCodeServerError
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	startupTimeout := 15 * time.Second //nolint:mnd
	startupCtx, startupCancel := context.WithTimeout(ctx, startupTimeout)
	cfg, err := config.FromEnv(startupCtx)
	if err != nil {
		fmt.Printf("could not parse configuration: %v\n", err)
		os.Exit(exitCodeNoConfig)
	}

	var logLevel slog.Level
	err = logLevel.UnmarshalText([]byte(cfg.LogLevel))
	if err != nil {
		fmt.Printf("%s, defaulting to %s\n", err, logLevel.Level())
	}

	logOptions := &slog.HandlerOptions{
		Level: logLevel,
	}
	var logHandler slog.Handler
	switch cfg.LogFormat {
	case "json":
		logHandler = slog.NewJSONHandler(os.Stdout, logOptions)
	case "text":
		logHandler = slog.NewTextHandler(os.Stdout, logOptions)
	default:
		fmt.Printf("unknown log format: %s. supported values are [json, text]\n", cfg.LogFormat)
		os.Exit(exitCodeNoConfig)
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.DebugContext(startupCtx, "configured", "config", cfg)

	shutdownCh := make(chan struct{})
	shutdownOnce := sync.OnceFunc(func() {
		close(shutdownCh)
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		shutdownOnce()
	}()

	logger.InfoContext(startupCtx, "x509 source connected", "workloadAddr", cfg.WorkloadAPI)

	upstreamAddr, err := cfg.UpstreamAddr()
	if err != nil {
		logger.ErrorContext(startupCtx, "no upstream address", "error", err)
		os.Exit(exitCodeBadConfig)
	}

	upstreamOptions := []upstream.Option{
		upstream.WithAddr(upstreamAddr),
	}

	up, err := upstream.New(upstreamOptions...)
	if err != nil {
		logger.ErrorContext(
			startupCtx,
			"could not create upstream",
			"error", err,
			"upstreamAddr", cfg.Upstream.String(),
		)
		os.Exit(exitCodeBadConfig)
	}

	logger.InfoContext(startupCtx, "created upstream", "upstreamAddr", cfg.Upstream.String())

	authz, err := authorizer.FromFile(cfg.AuthzConfig)
	if err != nil {
		logger.ErrorContext(
			startupCtx,
			"could not read authz config",
			"error", err,
			"filePath", cfg.AuthzConfig,
		)
		os.Exit(exitCodeBadConfig)
	}

	logger.InfoContext(
		startupCtx,
		"loaded authorization config",
		"filePath", cfg.AuthzConfig,
		"ruleCount", authz.Length(),
	)

	proxy := handlers.NewProxy(
		handlers.WithUpstream(up),
		handlers.WithLogger(logger.With("logger", "proxy")),
		handlers.WithAuthorizer(authz),
	)

	x509source, err := workloadapi.NewX509Source(startupCtx, workloadapi.WithClientOptions(
		workloadapi.WithLogger(logutils.NewSPIFFEAdapter(ctx, logger.With("logger", "x509source"))),
		workloadapi.WithAddr(cfg.WorkloadAPI),
	))
	if err != nil {
		logger.ErrorContext(
			startupCtx,
			"could not get x509 source",
			"error", err,
			"workloadAddr", cfg.WorkloadAPI,
		)
		os.Exit(exitCodeX509Source)
	}

	svid, err := x509source.GetX509SVID()
	if err != nil {
		logger.ErrorContext(
			startupCtx,
			"could not get x509 svid",
			"error", err,
		)
		os.Exit(exitCodeX509Source)
	}

	spID, err := x509util.SPIFFEIDFromCert(startupCtx, svid.Certificates[0])
	if err != nil {
		logger.ErrorContext(
			startupCtx,
			"could not get spiffe ID from svid",
			"error", err,
		)
	}

	logger.InfoContext(
		startupCtx,
		"got server svid",
		"spiffeid", spID.String(),
	)

	tlsConfig := tlsconfig.MTLSServerConfig(x509source, x509source, tlsconfig.AuthorizeAny())

	srv := &http.Server{
		Addr:                         cfg.BindAddr,
		DisableGeneralOptionsHandler: true,
		Handler:                      logging.Wrap(proxy, logging.WithLogger(logger)),
		ReadHeaderTimeout:            time.Second,
		TLSConfig:                    tlsConfig,
	}

	startupCancel()

	go func() {
		<-shutdownCh
		err := srv.Shutdown(ctx)
		if err != nil {
			logger.ErrorContext(ctx, "error shutting down http server", "error", err)
		}
		cancel()
	}()

	logger.InfoContext(ctx, "starting http server", "addr", srv.Addr)
	if err := srv.ListenAndServeTLS("", ""); err != nil {
		if err != http.ErrServerClosed {
			logger.Error("error starting server", "error", err)
			os.Exit(exitCodeServerError)
		}
	}
}
