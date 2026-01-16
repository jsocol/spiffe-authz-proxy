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
	addr     net.Addr
	client   httpDoer
	wrappers []RoundTripperWrapper
}

func New(opts ...Option) (_ *Upstream, err error) {
	u := &Upstream{}
	for _, opt := range opts {
		opt(u)
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

	for _, wrap := range u.wrappers {
		t = wrap(t)
	}

	c := &http.Client{
		Transport: t,
	}
	u.client = c
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

type Option func(*Upstream)

func WithAddr(a net.Addr) Option {
	return func(u *Upstream) {
		u.addr = a
	}
}

func WithRoundTripperWrappers(wrappers ...RoundTripperWrapper) Option {
	return func(u *Upstream) {
		u.wrappers = append(u.wrappers, wrappers...)
	}
}
