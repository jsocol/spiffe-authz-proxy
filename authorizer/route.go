package authorizer

import (
	"slices"
	"strings"
)

const (
	WildcardMethod   = "*"
	WildcardSegments = "**"
	WildcardSegment  = "*"
)

type Route struct {
	Pattern string   `hcl:"name,label"`
	Methods []string `hcl:"methods"`
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
