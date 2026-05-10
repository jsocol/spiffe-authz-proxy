package envoyauthzserver

import (
	"context"
	"log/slog"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type authorizer interface {
	Authorize(ctx context.Context, spid spiffeid.ID, method, path string) error
}

type config struct {
	logger *slog.Logger
	authz  authorizer
}

type Option interface {
	Apply(*config)
}

type optFunc func(*config)

func (o optFunc) Apply(c *config) {
	o(c)
}

func WithLogger(l *slog.Logger) Option {
	return optFunc(func(c *config) {
		c.logger = l
	})
}

func WithAuthorizer(a authorizer) Option {
	return optFunc(func(c *config) {
		c.authz = a
	})
}
