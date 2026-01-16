package authorizer_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"jsocol.io/spiffe-authz-proxy/authorizer"
)

func TestRoute_Match(t *testing.T) {
	t.Parallel()

	t.Run("routes", func(t *testing.T) {
		t.Parallel()
		t.Run("** is any route", func(t *testing.T) {
			tok := &authorizer.Route{
				Methods: []string{"*"},
				Pattern: "**",
			}

			assert.True(t, tok.Match(http.MethodGet, "/buckets/create"))
			assert.True(t, tok.Match(http.MethodPost, "/teams"))
			assert.True(t, tok.Match("WORDLE", "/slaps"))
			assert.True(t, tok.Match(http.MethodOptions, "/really/123/long/fake/path"))
		})

		t.Run("path can be limited", func(t *testing.T) {
			tok := &authorizer.Route{
				Methods: []string{"*"},
				Pattern: "/buckets/create",
			}

			assert.True(t, tok.Match(http.MethodGet, "/buckets/create"))
			assert.True(t, tok.Match(http.MethodPatch, "/buckets/create"))
			assert.False(t, tok.Match(http.MethodPost, "/teams"))
		})

		t.Run("path allows wildcards in the middle", func(t *testing.T) {
			tok := &authorizer.Route{
				Methods: []string{"*"},
				Pattern: "/bucket/*/123",
			}

			assert.True(t, tok.Match(http.MethodGet, "/bucket/read/123"))
			assert.True(t, tok.Match(http.MethodPost, "/bucket/write/123"))
			assert.False(t, tok.Match(http.MethodGet, "/bucket/read/432"))
		})

		t.Run("path allows wildcards at the end", func(t *testing.T) {
			tok := &authorizer.Route{
				Methods: []string{"*"},
				Pattern: "/bucket/**",
			}

			assert.True(t, tok.Match(http.MethodGet, "/bucket/read/123"))
			assert.True(t, tok.Match(http.MethodPatch, "/bucket/write/432"))
			assert.False(t, tok.Match(http.MethodPost, "/buckets/create"))
		})

		t.Run("path allows wildcards further down", func(t *testing.T) {
			tok := &authorizer.Route{
				Methods: []string{"*"},
				Pattern: "/bucket/read/*",
			}

			assert.True(t, tok.Match(http.MethodGet, "/bucket/read/123"))
			assert.False(t, tok.Match(http.MethodGet, "/bucket/write/213"))
		})

		t.Run("long paths are too specific", func(t *testing.T) {
			tok := &authorizer.Route{
				Methods: []string{"*"},
				Pattern: "/bucket/foo/bar",
			}

			assert.True(t, tok.Match(http.MethodPost, "/bucket/foo/bar"))
			assert.False(t, tok.Match(http.MethodPatch, "/bucket/foo/bar/baz"))
		})
	})
}
