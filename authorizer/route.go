package authorizer

import "strings"

type Route struct {
	Method string
	Path   string
}

func (r *Route) Match(method, path string) bool {
	if r.Method != method && r.Method != "*" {
		return false
	}
	parts := strings.Split(r.Path, "/")
	lastPart := len(parts) - 1

	if parts[0] == "*" {
		return true
	}

	for i, scope := range strings.Split(path, "/") {
		if i > lastPart {
			return parts[lastPart] == "*"
		}
		if parts[i] != "*" && parts[i] != scope {
			return false
		}
	}

	return true
}
