package authorizer_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"jsocol.io/spiffe-authz-proxy/authorizer"
)

func TestFromFile_HCL(t *testing.T) {
	fileName := "testconfigs/basic.hcl"
	spidA := spiffeid.RequireFromString("spiffe://example.org/a/workload")
	spidB := spiffeid.RequireFromString("spiffe://example.org/b/other/app")

	authz, err := authorizer.FromFile(fileName)
	assert.NoError(t, err)
	assert.NotNil(t, authz)

	err = authz.Authorize(context.Background(), spidA, http.MethodGet, "/foo/bar")
	assert.NoError(t, err)

	err = authz.Authorize(context.Background(), spidB, http.MethodDelete, "/foo/bar")
	assert.NoError(t, err)
}

func TestFromFile_Conf(t *testing.T) {
	fileName := "testconfigs/basic.conf"
	spidA := spiffeid.RequireFromString("spiffe://example.org/a/workload")
	spidB := spiffeid.RequireFromString("spiffe://example.org/b/other/app")

	authz, err := authorizer.FromFile(fileName)
	assert.NoError(t, err)
	assert.NotNil(t, authz)

	err = authz.Authorize(context.Background(), spidA, http.MethodGet, "/foo/bar")
	assert.NoError(t, err)

	err = authz.Authorize(context.Background(), spidB, http.MethodDelete, "/foo/bar")
	assert.NoError(t, err)
}
