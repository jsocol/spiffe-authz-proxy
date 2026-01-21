package authorizer_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/require"
	"jsocol.io/spiffe-authz-proxy/authorizer"
)

func TestMemory_Authorize(t *testing.T) {
	spid := spiffeid.RequireFromString("spiffe://example.org/foo")
	a := &authorizer.MemoryAuthorizer{}
	a.Update(map[spiffeid.ID][]authorizer.Route{
		spid: {
			{
				Pattern: "/foo/bar",
				Methods: []string{http.MethodGet, http.MethodPost},
			},
		},
	})

	err := a.Authorize(context.Background(), spid, http.MethodGet, "/foo/bar")
	require.NoError(t, err)
}

func TestMemory_Authorize_Unauthorized(t *testing.T) {
	spid := spiffeid.RequireFromString("spiffe://example.org/foo")
	a := &authorizer.MemoryAuthorizer{}
	a.Update(map[spiffeid.ID][]authorizer.Route{
		spiffeid.RequireFromString("spiffe://example.org/bar"): {
			{
				Pattern: "/foo/bar",
				Methods: []string{http.MethodGet, http.MethodPost},
			},
		},
	})

	err := a.Authorize(context.Background(), spid, http.MethodGet, "/foo/bar")
	require.Error(t, err)
}
