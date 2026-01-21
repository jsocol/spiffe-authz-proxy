package authorizer

import (
	"context"
	"errors"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func FromConfigMap(ctx context.Context, cmName, fileName string) (*MemoryAuthorizer, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	namespace, err := os.ReadFile(namespaceFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("configmap config only works within kubernetes: %w", err)
		}
		return nil, err
	}

	cm, err := clientSet.CoreV1().ConfigMaps(string(namespace)).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	src, ok := cm.Data[fileName]
	if !ok {
		return nil, fmt.Errorf("could not find file %s in configmap %s", fileName, cmName)
	}

	cfg := &hclConfig{}
	err = decodeHCL(fileName, []byte(src), cfg)
	if err != nil {
		return nil, err
	}

	return cfg.toAuthorizer()
}
