package handlers_test

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jsocol.io/spiffe-authz-proxy/handlers"
)

//go:embed testdata/rootcert.pem
var rootCert []byte

//go:embed testdata/leafsvid.pem
var workloadSVID []byte

//go:embed testdata/leafkey.pem
var workloadKey []byte

//go:embed testdata/serversvid.pem
var serverSVID []byte

//go:embed testdata/serverkey.pem
var serverKey []byte

type mockAuthorizer func(context.Context, spiffeid.ID, string, string) error

func (m mockAuthorizer) Authorize(
	ctx context.Context,
	spid spiffeid.ID,
	method, path string,
) error {
	return m(ctx, spid, method, path)
}

type mockUpstream func(*http.Request) (*http.Response, error)

func (m mockUpstream) Do(r *http.Request) (*http.Response, error) {
	return m(r)
}

func TestProxy_Authorized(t *testing.T) {
	var authz mockAuthorizer = func(ctx context.Context, spid spiffeid.ID, method, path string) error {
		if spid.String() == "spiffe://example.org/workload" && method == http.MethodPatch &&
			path == "/my/path" {
			return nil
		}

		return errors.New("bad ID")
	}

	var upstream mockUpstream = func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Spiffe-Id") == "spiffe://example.org/workload" &&
			r.Method == http.MethodPatch &&
			r.URL.Path == "/my/path" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("it worked")),
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader("it did not work")),
		}, nil
	}

	proxy := handlers.NewProxy(handlers.WithAuthorizer(authz), handlers.WithUpstream(upstream))

	srv, client := newTestClientServer(t, proxy)
	srv.StartTLS()
	defer srv.Close()

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodPatch,
		srv.URL+"/my/path",
		http.NoBody,
	)

	resp, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respData, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, "it worked", string(respData))
}

func TestProxy_Unauthorized(t *testing.T) {
	var authz mockAuthorizer = func(ctx context.Context, spid spiffeid.ID, method, path string) error {
		if spid.String() == "spiffe://example.org/diff-workload" && method == http.MethodPatch &&
			path == "/my/path" {
			return nil
		}

		return errors.New("bad ID")
	}

	var upstream mockUpstream = func(r *http.Request) (*http.Response, error) {
		t.Fatalf("this should never be called")
		return nil, nil //nolint:nilnil,nlreturn
	}

	proxy := handlers.NewProxy(handlers.WithAuthorizer(authz), handlers.WithUpstream(upstream))

	srv, client := newTestClientServer(t, proxy)
	srv.StartTLS()
	defer srv.Close()

	req, _ := http.NewRequestWithContext(
		context.Background(),
		http.MethodPatch,
		srv.URL+"/my/path",
		http.NoBody,
	)

	resp, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func newTestClientServer(t *testing.T, handler http.Handler) (*httptest.Server, *http.Client) {
	t.Helper()

	bundle, err := x509bundle.Parse(
		spiffeid.RequireTrustDomainFromString("spiffe://example.org"),
		rootCert,
	)
	require.NoError(t, err)

	clientsvid, err := x509svid.Parse(append(workloadSVID, rootCert...), workloadKey)
	require.NoError(t, err)

	serversvid, err := x509svid.Parse(append(serverSVID, rootCert...), serverKey)
	require.NoError(t, err)

	clientTLS := tlsconfig.MTLSClientConfig(clientsvid, bundle, tlsconfig.AuthorizeAny())
	clientTLS.ServerName = "server.example.org"

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLS,
		},
	}

	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = tlsconfig.MTLSServerConfig(serversvid, bundle, tlsconfig.AuthorizeAny())
	srv.TLS.ServerName = "server.example.org"

	return srv, client
}
