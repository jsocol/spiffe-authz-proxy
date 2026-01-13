package authorizer

import (
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type hclPath struct {
	Pattern string   `hcl:"name,label"`
	Methods []string `hcl:"methods"`
}

type hclEntry struct {
	SPIFFEID string    `hcl:"name,label"`
	Paths    []hclPath `hcl:"path,block"`
}

type hclConfig struct {
	Entries []hclEntry `hcl:"spiffeid,block"`
}

func FromFile(path string) (*MemoryAuthorizer, error) {
	cfg := &hclConfig{}
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
			a.routes[id] = append(a.routes[id], Route(path))
		}
	}

	return a, nil
}
