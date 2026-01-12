package config_test

import (
	"net"
	"net/netip"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"jsocol.io/spiffe-authz-proxy/config"
)

func TestConfig_UpstreamAddr(t *testing.T) {
	t.Run("tcp/ipv4", func(t *testing.T) {
		tcp, _ := url.Parse("tcp://127.0.0.1:5001")
		cfg := &config.Config{
			Upstream: tcp,
		}
		expected := net.TCPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:5001"))

		actual, err := cfg.UpstreamAddr()
		assert.NoError(t, err)

		actualTCP, ok := actual.(*net.TCPAddr)
		assert.True(t, ok, "UpstreamAddr() returns a *net.TCPAddr")
		assert.Equal(t, expected.String(), actualTCP.String())
	})

	t.Run("tcp/hostname", func(t *testing.T) {
		tcp, _ := url.Parse("tcp4://localhost:5002")
		cfg := &config.Config{
			Upstream: tcp,
		}
		expected := net.TCPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:5002"))

		actual, err := cfg.UpstreamAddr()
		assert.NoError(t, err)

		assert.Equal(t, expected, actual)
	})

	t.Run("unix", func(t *testing.T) {
		unix, _ := url.Parse("unix:/tmp/my/socket")
		cfg := &config.Config{
			Upstream: unix,
		}
		expected := &net.UnixAddr{
			Name: "/tmp/my/socket",
			Net:  "unix",
		}

		actual, err := cfg.UpstreamAddr()
		assert.NoError(t, err)

		assert.Equal(t, expected, actual)
	})
}
