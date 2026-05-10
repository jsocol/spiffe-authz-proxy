package envoyauthzserver_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jsocol.io/spiffe-authz-proxy/servers/envoyauthzserver"
)

func TestCheck(t *testing.T) {
	testcases := []struct {
		name     string
		authz    *mockAuthorizer
		spiffeid string
		path     string
		method   string
		response *authv3.CheckResponse
	}{
		{
			name:     "no error",
			authz:    &mockAuthorizer{},
			spiffeid: "spiffe://example.org/workload",
			method:   http.MethodGet,
			path:     "/foo/bar/baz",
			response: &authv3.CheckResponse{
				HttpResponse: &authv3.CheckResponse_OkResponse{},
			},
		},
		{
			name:     "bad spiffeid",
			spiffeid: "not-spiffe://oops",
			method:   http.MethodPatch,
			path:     "/x/y/z",
			response: &authv3.CheckResponse{
				HttpResponse: &authv3.CheckResponse_ErrorResponse{
					ErrorResponse: &authv3.DeniedHttpResponse{
						Body: "source principal was not a spiffeid: scheme is missing or invalid",
					},
				},
			},
		},
		{
			name: "unauthorized",
			authz: &mockAuthorizer{
				err: errors.New("unauthorized"),
			},
			spiffeid: "spiffe://example.org/workload",
			method:   http.MethodPost,
			path:     "/a/b/123",
			response: &authv3.CheckResponse{
				HttpResponse: &authv3.CheckResponse_DeniedResponse{
					DeniedResponse: &authv3.DeniedHttpResponse{
						Status: &typev3.HttpStatus{
							Code: typev3.StatusCode_Forbidden,
						},
						Body: "unauthorized",
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s := envoyauthzserver.New(envoyauthzserver.WithAuthorizer(tc.authz))
			req := &authv3.CheckRequest{
				Attributes: &authv3.AttributeContext{
					Source: &authv3.AttributeContext_Peer{
						Principal: tc.spiffeid,
					},
					Request: &authv3.AttributeContext_Request{
						Http: &authv3.AttributeContext_HttpRequest{
							Method: tc.method,
							Path:   tc.path,
						},
					},
				},
			}
			ctx := context.Background()

			res, err := s.Check(ctx, req)
			require.NoError(t, err)

			assert.Equal(t, tc.response, res)

			if tc.authz != nil {
				spid := spiffeid.RequireFromString(tc.spiffeid)
				assert.Equal(t, spid, tc.authz.spid)
				assert.Equal(t, tc.method, tc.authz.method)
				assert.Equal(t, tc.path, tc.authz.path)
			}
		})
	}
}

type mockAuthorizer struct {
	spid         spiffeid.ID
	path, method string
	err          error
}

func (m *mockAuthorizer) Authorize(_ context.Context, spid spiffeid.ID, method, path string) error {
	m.spid = spid
	m.path = path
	m.method = method

	return m.err
}
