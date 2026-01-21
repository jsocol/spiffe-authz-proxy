package upstream

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"jsocol.io/spiffe-authz-proxy/spiffeidutil"
)

type RoundTripperWrapper func(http.RoundTripper) http.RoundTripper

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Upstream struct {
	addr   net.Addr
	client httpDoer
}

func New(opts ...Option) (_ *Upstream, err error) {
	c := &config{}
	for _, opt := range opts {
		opt.Apply(c)
	}

	u := &Upstream{
		addr: c.addr,
	}

	if u.addr == nil {
		return nil, errors.New("an upstream address is required")
	}

	dialer := &net.Dialer{}
	var t http.RoundTripper = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, u.addr.Network(), u.addr.String())
			if err != nil {
				return nil, fmt.Errorf("could not dial upstream %s: %v", u.addr.String(), err)
			}

			return conn, nil
		},
	}

	for _, wrap := range c.wrappers {
		t = wrap(t)
	}

	u.client = &http.Client{
		Transport: t,
	}

	return u, nil
}

func (u *Upstream) Do(r *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	req := r.WithContext(ctx)

	spid := spiffeidutil.FromContext(ctx)
	if !spid.IsZero() {
		req.Header.Set("SPIFFE-ID", spid.String())
	}

	return u.client.Do(req)
}

func (u *Upstream) Addr() net.Addr {
	return u.addr
}

type Option interface {
	Apply(*config)
}

type config struct {
	addr     net.Addr
	wrappers []RoundTripperWrapper
}

type optionFunc func(*config)

func (o optionFunc) Apply(c *config) {
	o(c)
}

func WithAddr(a net.Addr) Option {
	return optionFunc(func(c *config) {
		c.addr = a
	})
}

func WithRoundTripperWrappers(wrappers ...RoundTripperWrapper) Option {
	return optionFunc(func(c *config) {
		c.wrappers = append(c.wrappers, wrappers...)
	})
}
