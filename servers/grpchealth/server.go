package grpchealth

import (
	"context"

	"google.golang.org/grpc"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	healthv1.UnimplementedHealthServer
}

func New() *Server {
	s := &Server{}

	return s
}

func (s *Server) Check(ctx context.Context, hcr *healthv1.HealthCheckRequest) (*healthv1.HealthCheckResponse, error) {
	return &healthv1.HealthCheckResponse{
		Status: healthv1.HealthCheckResponse_SERVING,
	}, nil
}

func (s *Server) Register(srv grpc.ServiceRegistrar) {
	healthv1.RegisterHealthServer(srv, s)
}
