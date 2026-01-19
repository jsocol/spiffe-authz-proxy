package x509util

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

func SPIFFEIDFromCert(_ context.Context, cert *x509.Certificate) (spiffeid.ID, error) {
	if len(cert.URIs) != 1 {
		return spiffeid.ID{}, fmt.Errorf("expected 1 uri san, got %d", len(cert.URIs))
	}

	uriSAN := cert.URIs[0]

	return spiffeid.FromURI(uriSAN)
}
