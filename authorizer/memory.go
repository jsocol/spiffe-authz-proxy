package authorizer

import (
	"context"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type MemoryAuthorizer struct {
	// TODO: This is terribly inefficient and probably needs improvement
	routes map[spiffeid.ID][]Route
}

func (a *MemoryAuthorizer) Authorize(_ context.Context, spid spiffeid.ID, method, path string) error {
	routes, ok := a.routes[spid]
	if !ok {
		return fmt.Errorf("spiffeid %s is not authorized on any routes", spid)
	}

	for _, r := range routes {
		if r.Match(method, path) {
			return nil
		}
	}
	return fmt.Errorf("spiffeid %s is not authorized for method %s on path %s", spid, method, path)
}
