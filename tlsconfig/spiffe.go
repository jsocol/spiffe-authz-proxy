package tlsconfig

import (
	"context"
	"crypto/tls"
	"log/slog"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

type sourcer interface {
	x509svid.Source
	x509bundle.Source
}

type SPIFFETLSConfig struct {
	logger *slog.Logger
	source sourcer
}

func New(opts ...Option) (*SPIFFETLSConfig, error) {
	s := &SPIFFETLSConfig{
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.source == nil {
		source, err := workloadapi.NewX509Source(context.Background())
		if err != nil {
			return nil, err
		}
		s.source = source
	}

	return s, nil
}

func (stc *SPIFFETLSConfig) GetConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			svid, err := stc.source.GetX509SVID()
			if err != nil {
				return nil, err
			}

			certBytes, keyBytes, err := svid.Marshal()
			if err != nil {
				return nil, err
			}

			cert, err := tls.X509KeyPair(certBytes, keyBytes)
			if err != nil {
				return nil, err
			}

			return &cert, nil
		},
		ClientAuth: tls.RequireAnyClientCert,
		VerifyConnection: func(cs tls.ConnectionState) error {
			certs := cs.PeerCertificates
			spid, _, err := x509svid.Verify(certs, stc.source)
			if err != nil {
				return err
			}
			stc.logger.Debug("verified connection", "spiffeid", spid)
			return err
		},
	}
}

type Option func(*SPIFFETLSConfig)

func WithSource(source sourcer) Option {
	return func(s *SPIFFETLSConfig) {
		s.source = source
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(s *SPIFFETLSConfig) {
		s.logger = l
	}
}
