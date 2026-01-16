package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

func main() {
	ctx := context.Background()
	x509s, _ := workloadapi.NewX509Source(ctx)
	authz := tlsconfig.AuthorizeMemberOf(spiffeid.RequireTrustDomainFromString("spiffe://example.org"))
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsconfig.MTLSClientConfig(x509s, x509s, authz),
		},
	}

	addr := os.Getenv("SERVER_ADDR")

	resp, err := client.Get(fmt.Sprintf("https://%s/hi/Name", addr))
	if err != nil {
		panic(err)
	}

	fmt.Printf("got response: %v\n", resp)
}
