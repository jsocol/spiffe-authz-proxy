package authorizer

import (
	"os"
)

func FromFile(fileName string, opts ...Option) (*MemoryAuthorizer, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o.Apply(cfg)
	}

	// ignore gosec G304, this is on purpose
	src, err := os.ReadFile(fileName) //nolint:gosec
	if err != nil {
		return nil, err
	}

	hCfg := &hclConfig{}
	err = decodeHCL(fileName, src, hCfg)
	if err != nil {
		return nil, err
	}

	return hCfg.toAuthorizer(cfg)
}
