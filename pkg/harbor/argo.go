package harbor

import (
	"context"
	"fmt"
	"github.com/thschue/platformer/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"strings"
)

func (h *Config) createKubernetesSecretForArgoCD(namespace string, account RobotAccount, secretName string) error {
	config, err := helpers.BuildKubeConfig()
	if err != nil {
		return fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	cleanUrl := strings.ReplaceAll(h.Url, "https://", "")
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name: secretName,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
			},
		},
		Data: map[string][]byte{
			"enableOCI": []byte("true"),
			"url":       []byte(cleanUrl),
			"project":   []byte("default"),
			"type":      []byte("helm"),
			"username":  []byte(account.Name),
			"password":  []byte(account.Token),
			"insecure":  []byte("true"),
		},
	}

	if namespace == "" {
		namespace = "argocd"
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, v1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	log.Printf("Secret %s created successfully in namespace %s\n", secretName, namespace)
	return nil
}
