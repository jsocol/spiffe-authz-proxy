package authorizer

import (
	"fmt"
	"os"
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

func FromFile(fileName string) (*MemoryAuthorizer, error) {
	src, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	cfg := &hclConfig{}
	err = decodeHCL(fileName, src, cfg)
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
