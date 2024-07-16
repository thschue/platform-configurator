package helpers

import (
	"fmt"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

func BuildKubeConfig() (*rest.Config, error) {
	var config *rest.Config

	// Path to the ServiceAccount token
	serviceAccountTokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// Check if running inside a Kubernetes cluster
	if _, err := os.ReadFile(serviceAccountTokenPath); err == nil {
		// Use InClusterConfig if the ServiceAccount token exists
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		// Fallback to kubeconfig for outside cluster usage
		kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
	}
	return config, nil
}
