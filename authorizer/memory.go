package authorizer

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type Entry struct {
	SPIFFEID string  `hcl:"name,label"`
	Paths    []Route `hcl:"path,block"`
}

type Config struct {
	Entries []Entry `hcl:"spiffeid,block"`
}

type MemoryAuthorizer struct {
	// TODO: This is terribly inefficient and probably needs improvement
	routes map[spiffeid.ID][]Route
}

func FromFile(path string) (*MemoryAuthorizer, error) {
	cfg := &Config{}
	err := hclsimple.DecodeFile(path, nil, cfg)
	if err != nil {
		return nil, err
	}

	a := &MemoryAuthorizer{
		routes: make(map[spiffeid.ID][]Route, len(cfg.Entries)),
	}
	for _, entry := range cfg.Entries {
		id, err := spiffeid.FromString(entry.SPIFFEID)
		if err != nil {
			return nil, err
		}
		a.routes[id] = make([]Route, 0, len(entry.Paths))

		for _, path := range entry.Paths {
			a.routes[id] = append(a.routes[id], Route{
				Pattern: path.Pattern,
				Methods: path.Methods,
			})
		}
	}

	return a, nil
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
