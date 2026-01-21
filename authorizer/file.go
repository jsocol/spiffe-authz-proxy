package authorizer

import (
	"os"
)

func FromFile(fileName string) (*MemoryAuthorizer, error) {
	// ignore gosec G304, this is on purpose
	src, err := os.ReadFile(fileName) //nolint:gosec
	if err != nil {
		return nil, err
	}

	cfg := &hclConfig{}
	err = decodeHCL(fileName, src, cfg)
	if err != nil {
		return nil, err
	}

	return cfg.toAuthorizer()
}
