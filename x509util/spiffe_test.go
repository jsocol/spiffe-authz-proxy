package x509util_test

import (
	"context"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jsocol.io/spiffe-authz-proxy/x509util"
)

//go:embed testdata/cert.pem
var testCertBytes []byte

func TestSPIFFEIDFromCert(t *testing.T) {
	certDER, _ := pem.Decode(testCertBytes)
	cert, err := x509.ParseCertificate(certDER.Bytes)
	require.NoError(t, err)

	spID, err := x509util.SPIFFEIDFromCert(context.Background(), cert)
	require.NoError(t, err)

	assert.Equal(t, "spiffe://example.org/workload", spID.String())
}
