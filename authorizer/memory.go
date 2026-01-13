package authorizer

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

const (
	WildcardMethod   = "*"
	WildcardSegments = "**"
	WildcardSegment  = "*"
)

type Route struct {
	Pattern string
	Methods []string
}

func (r *Route) Match(method, path string) bool {
	if !slices.Contains(r.Methods, method) && !slices.Contains(r.Methods, "*") {
		return false
	}

	rPath := strings.TrimRight(r.Pattern, "/")
	parts := strings.Split(rPath, "/")
	lastPart := len(parts) - 1

	if parts[0] == WildcardSegments {
		return true
	}

	for i, scope := range strings.Split(path, "/") {
		if i > lastPart {
			return parts[lastPart] == WildcardSegments
		}
		if parts[i] != WildcardSegment && parts[i] != WildcardSegments && parts[i] != scope {
			return false
		}
	}

	return true
}

type MemoryAuthorizer struct {
	// TODO: This is terribly inefficient and probably needs improvement
	routes map[spiffeid.ID][]Route
}

func (a *MemoryAuthorizer) Authorize(_ context.Context, spid spiffeid.ID, method, path string) error {
	routes, ok := a.routes[spid]
	if !ok {
		return fmt.Errorf("unknown spiffeid %s", spid)
	}

	for _, r := range routes {
		if r.Match(method, path) {
			return nil
		}
	}
	return fmt.Errorf("spiffeid %s is not authorized for method %s on path %s", spid, method, path)
}
