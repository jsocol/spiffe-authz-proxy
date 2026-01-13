package config

import (
	"fmt"
	"net"
	"net/url"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type Config struct {
	LogLevel    string   `env:"LOG_LEVEL, default=info"`
	LogFormat   string   `env:"LOG_FORMAT, default=json"`
	BindAddr    string   `env:"BIND_ADDR, default=0:8443"`
	WorkloadAPI string   `env:"WORKLOAD_API, default=unix:/tmp/spire-agent/public/api.sock"`
	TrustDomain *url.URL `env:"TRUST_DOMAIN, required"`
	AuthzConfig string   `env:"AUTHZ_CONFIG, required"`
	Upstream    *url.URL `env:"UPSTREAM_ADDR, default=tcp://127.0.0.1:8000"`
}

func (c *Config) UpstreamAddr() (net.Addr, error) {
	switch c.Upstream.Scheme {
	case "tcp":
		return net.ResolveTCPAddr("tcp", c.Upstream.Host)
	case "tcp4":
		return net.ResolveTCPAddr("tcp4", c.Upstream.Host)
	case "tcp6":
		return net.ResolveTCPAddr("tcp6", c.Upstream.Host)
	case "unix":
		return &net.UnixAddr{
			Net:  "unix",
			Name: c.Upstream.Path,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported upstream scheme: %s", c.Upstream.Scheme)
	}
}

func (c *Config) TD() (spiffeid.TrustDomain, error) {
	return spiffeid.TrustDomainFromURI(c.TrustDomain)
}
