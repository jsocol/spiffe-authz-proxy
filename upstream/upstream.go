package upstream

import (
	"context"
	"errors"
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
		err := opt(u)
		if err != nil {
			return nil, err
		}
	}

	if u.addr == nil {
		return nil, errors.New("an upstream address is required")
	}

	dialer := &net.Dialer{}
	var t http.RoundTripper = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, u.addr.Network(), u.addr.String())
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

type Option func(*Upstream) error

func WithTCP(host, port string) Option {
	return func(u *Upstream) error {
		addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, port))
		if err != nil {
			return err
		}
		u.addr = addr
		return nil
	}
}

func WithUnix(path string) Option {
	return func(u *Upstream) error {
		u.addr = &net.UnixAddr{
			Net:  "unix",
			Name: path,
		}
		return nil
	}
}

func WithRoundTripperWrappers(wrappers ...RoundTripperWrapper) Option {
	return func(u *Upstream) error {
		u.wrappers = append(u.wrappers, wrappers...)
		return nil
	}
}
