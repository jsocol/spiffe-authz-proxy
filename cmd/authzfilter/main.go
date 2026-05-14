package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	"jsocol.io/spiffe-authz-proxy/authorizer"
	"jsocol.io/spiffe-authz-proxy/config"
	"jsocol.io/spiffe-authz-proxy/logutils/grpcserverlog"
	"jsocol.io/spiffe-authz-proxy/servers/envoyauthzserver"
	"jsocol.io/spiffe-authz-proxy/servers/metaserver"
)

const (
	exitCodeNoConfig = iota
	exitCodeX509Source
	exitCodeBadConfig
	exitCodeServerError
)

func main() {
	ctx := context.Background()

	cfg, err := config.FromEnv(ctx)
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

	promRegistry := prometheus.NewRegistry()

	logger.DebugContext(ctx, "configured", "config", cfg)

	runExtAuthz(ctx, logger, cfg, promRegistry)
}

func makeAuthorizer(ctx context.Context, logger *slog.Logger, cfg *config.Config) (*authorizer.MemoryAuthorizer, error) {
	authzURL, err := cfg.AuthzConfigURL()
	if err != nil {
		return nil, fmt.Errorf("could not interpret authz config source: %s", cfg.AuthzConfig)
	}

	logger.DebugContext(
		ctx,
		"loading authz data",
		"sourceKind",
		authzURL.Scheme,
		"sourcePath",
		authzURL.Path,
	)
	var authz *authorizer.MemoryAuthorizer
	switch authzURL.Scheme {
	case "file":
		authz, err = authorizer.FromFile(authzURL.Path)
		if err != nil {
			return nil, err
		}
	case "configmap":
		authz, err = authorizer.FromConfigMap(
			ctx,
			authzURL.Host,
			strings.TrimPrefix(authzURL.Path, "/"),
			authorizer.WithLogger(logger.With("logger", "authorizer")),
		)
		if err != nil {
			return nil, err
		}

		go func() {
			if err := authz.Watch(ctx); err != nil {
				logger.InfoContext(ctx, "error watching configmap", "error", err)
			}
		}()
	default:
		return nil, fmt.Errorf("unsupported authz config source: %s", cfg.AuthzConfig)
	}

	logger.InfoContext(
		ctx,
		"loaded authorization config",
		"filePath", cfg.AuthzConfig,
		"ruleCount", authz.Length(),
	)

	return authz, nil
}

func runExtAuthz(ctx context.Context, logger *slog.Logger, cfg *config.Config, promRegistry *prometheus.Registry) {
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

	logInt := grpcserverlog.New(logger.With("logger", "grpc"))

	metaSrv := metaserver.New(
		metaserver.WithAddr(cfg.MetaAddr),
	)

	go func() {
		logger.InfoContext(ctx, "starting meta endpoints server", "addr", metaSrv.Addr)
		if err := metaSrv.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				logger.ErrorContext(ctx, "error starting meta server", "error", err)
				os.Exit(exitCodeServerError)
			}
		}
	}()

	go func() {
		gracePeriod := 10 * time.Second //nolint:mnd
		<-shutdownCh

		logger.InfoContext(ctx, "shutting down healthcheck server", "gracePeriod", gracePeriod)

		ctx, cancel := context.WithTimeout(context.Background(), gracePeriod)
		defer cancel()

		if err := metaSrv.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "error shutting down healthcheck server", "error", err)
		}
	}()

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logInt.UnaryInterceptor()),
	)

	authz, err := makeAuthorizer(ctx, logger, cfg)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"could not read authz config",
			"error", err,
			"authzConfig", cfg.AuthzConfig,
		)
		os.Exit(exitCodeBadConfig)
	}

	s := envoyauthzserver.New(
		envoyauthzserver.WithAuthorizer(authz),
		envoyauthzserver.WithLogger(logger.With("logger", "envoyauthzserver")),
	)

	s.Register(srv)

	listenConfig := &net.ListenConfig{}
	lis, err := listenConfig.Listen(ctx, "tcp", cfg.GRPCBindAddr)
	if err != nil {
		logger.ErrorContext(ctx, "could bind listen address", "error", err)
		os.Exit(exitCodeServerError)
	}

	go func() {
		<-shutdownCh
		srv.GracefulStop()
	}()

	if err := srv.Serve(lis); err != nil {
		logger.ErrorContext(ctx, "could not start grpc server", "error", err)
		os.Exit(exitCodeServerError)
	}
}
