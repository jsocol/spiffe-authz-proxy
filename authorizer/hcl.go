package authorizer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/json"
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

// this is borrowed from hcl/v2/hclsimple.DecodeFile, and allows us to accept
// more extensions, like the super-common ".conf"
func decodeHCL(fileName string, src []byte, target any) error {
	var file *hcl.File
	var diags hcl.Diagnostics

	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".hcl", ".conf":
		file, diags = hclsyntax.ParseConfig(src, fileName, hcl.InitialPos)
	case ".json":
		file, diags = json.Parse(src, fileName)
	default:
		return fmt.Errorf("could not read config, unsupported format %s", ext)
	}
	if diags.HasErrors() {
		return diags
	}

	diags = gohcl.DecodeBody(file.Body, nil, target)
	if diags.HasErrors() {
		return diags
	}

	return nil
}

func (h *hclConfig) toRouteMap() (map[spiffeid.ID][]Route, error) {
	routes := make(map[spiffeid.ID][]Route, len(h.Entries))

	for _, entry := range h.Entries {
		id, err := spiffeid.FromString(entry.SPIFFEID)
		if err != nil {
			return nil, err
		}
		routes[id] = make([]Route, 0, len(entry.Paths))

		for _, path := range entry.Paths {
			routes[id] = append(routes[id], Route(path))
		}
	}

	return routes, nil
}

func (h *hclConfig) toAuthorizer(cfg *config) (*MemoryAuthorizer, error) {
	routes, err := h.toRouteMap()
	if err != nil {
		return nil, err
	}

	a := &MemoryAuthorizer{
		cfg:    cfg,
		routes: routes,
	}

	return a, nil
}
