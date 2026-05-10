package envoyauthzserver

import (
	"log/slog"

	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc"
)

type Server struct {
	authv3.UnimplementedAuthorizationServer
	*config
}

func New(opts ...Option) *Server {
	cfg := &config{
		logger: slog.Default(),
	}

	for _, o := range opts {
		o.Apply(cfg)
	}

	s := &Server{
		config: cfg,
	}

	return s
}

func (s *Server) Register(srv grpc.ServiceRegistrar) {
	authv3.RegisterAuthorizationServer(srv, s)
}
