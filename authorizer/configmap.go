package authorizer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func FromConfigMap(
	ctx context.Context,
	cmName, fileName string,
	opts ...Option,
) (*MemoryAuthorizer, error) {
	cfg := &config{
		logger: slog.Default(),
	}
	for _, o := range opts {
		o.Apply(cfg)
	}

	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	namespaceBytes, err := os.ReadFile(namespaceFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("configmap config only works within kubernetes: %w", err)
		}

		return nil, err
	}
	namespace := string(namespaceBytes)

	cm, err := clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	routes, err := configMapToRoutes(cm, fileName)
	if err != nil {
		return nil, err
	}

	authz := &MemoryAuthorizer{
		cfg:    cfg,
		routes: routes,
	}
	authz.watcher = watchConfigMap(authz, clientSet, namespace, cmName, fileName)

	return authz, nil
}

func configMapToRoutes(cm *corev1.ConfigMap, fileName string) (map[spiffeid.ID][]Route, error) {
	src, ok := cm.Data[fileName]
	if !ok {
		return nil, fmt.Errorf("could not find file %s in configmap %s", fileName, cm.GetName())
	}

	cfg := &hclConfig{}
	err := decodeHCL(fileName, []byte(src), cfg)
	if err != nil {
		return nil, err
	}

	return cfg.toRouteMap()
}

// TODO: Simplify/restructure this.
//
//nolint:gocognit
func watchConfigMap(
	ma *MemoryAuthorizer,
	clientSet *kubernetes.Clientset,
	namespace, cmName, fileName string,
) func(context.Context) error {
	logger := ma.cfg.logger.With("configMap", cmName, "fileName", fileName, "namespace", namespace)

	return func(ctx context.Context) error {
		objMeta := metav1.ObjectMeta{
			Namespace: namespace,
			Name:      cmName,
		}

		watcher, err := clientSet.CoreV1().
			ConfigMaps(namespace).
			Watch(ctx, metav1.SingleObject(objMeta))
		if err != nil {
			if apierrors.IsUnauthorized(err) {
				logger.WarnContext(
					ctx,
					"could not start configmap watcher; need 'watch' permission",
				)

				return nil
			}

			return err
		}

		resultChan := watcher.ResultChan()

	resultLoop:
		for {
			select {
			case <-ctx.Done():
				watcher.Stop()

				break resultLoop
			case event := <-resultChan:
				switch event.Type {
				case watch.Added:
					fallthrough
				case watch.Modified:
					updatedMap, ok := event.Object.(*corev1.ConfigMap)
					if !ok {
						logger.WarnContext(ctx, "error from k8s api, not a configmap")

						continue
					}

					routes, err := configMapToRoutes(updatedMap, fileName)
					if err != nil {
						logger.WarnContext(ctx, "error reading new configmap data", "error", err)

						continue
					}

					ma.Update(routes)
					logger.InfoContext(ctx, "updated authz rules from configmap")
				case watch.Error:
					apiErr, ok := event.Object.(*metav1.Status)
					if ok {
						logger.ErrorContext(ctx, "error from k8s api, stopping watcher", "error", apiErr)
					} else {
						logger.ErrorContext(ctx, "unknown error from k8s api, stopping watcher")
					}

					watcher.Stop()

					break resultLoop
				case watch.Deleted:
					logger.WarnContext(ctx, "configmap has been deleted, stopping watcher")
					watcher.Stop()

					break resultLoop
				case watch.Bookmark:
				}
			}
		}

		return nil
	}
}
