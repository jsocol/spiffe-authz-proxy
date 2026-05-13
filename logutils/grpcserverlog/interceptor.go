package grpcserverlog

import (
	"context"
	"log/slog"
	"net/netip"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func defaultLeveler(err error) slog.Level {
	st := status.Convert(err)
	switch st.Code() {
	case codes.Internal, codes.DataLoss:
		return slog.LevelError
	case codes.Unimplemented, codes.ResourceExhausted, codes.DeadlineExceeded, codes.FailedPrecondition, codes.Aborted, codes.Unavailable:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

type Interceptor struct {
	logger  *slog.Logger
	leveler func(error) slog.Level
}

func New(l *slog.Logger) *Interceptor {
	return &Interceptor{
		logger:  l,
		leveler: defaultLeveler,
	}
}

func (i *Interceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		client, ok := peer.FromContext(ctx)
		if !ok {
			client = &peer.Peer{}
		}

		clientAddr := netip.MustParseAddrPort(client.Addr.String())

		attrs := []slog.Attr{
			slog.String("rpc.method", info.FullMethod),
			slog.String("network.peer.address", clientAddr.Addr().String()),
			slog.Int("network.peer.port", int(clientAddr.Port())),
		}

		start := time.Now()

		resp, err = handler(ctx, req)

		duration := time.Since(start)
		level := i.leveler(err)
		st := status.Convert(err)
		attrs = append(attrs, slog.String(
			"rpc.response.status_code",
			st.Code().String(),
		), slog.Duration(
			"rpc.response.duration",
			duration,
		))
		if err != nil {
			attrs = append(attrs, slog.String(
				"error.type",
				st.Code().String(),
			))
		}

		slog.LogAttrs(ctx, level, info.FullMethod, attrs...)

		return
	}
}
