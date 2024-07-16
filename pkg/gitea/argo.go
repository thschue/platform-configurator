package gitea

import (
	"bytes"
	"code.gitea.io/sdk/gitea"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/thschue/platformer/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"text/template"
)

func (g *Config) createDeployKey(repository Repository) (*gitea.Response, error) {
	// Example usage
	privateKey, publicKey, err := helpers.GenerateSSHKeyPair()
	if err != nil {
		return &gitea.Response{}, fmt.Errorf("failed to generate SSH key pair: %w", err)
	}

	client, err := gitea.NewClient(g.Url, gitea.SetBasicAuth(g.Credentials.Username, g.Credentials.Password), gitea.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.TLSConfig.InsecureSkipVerify,
			},
		},
	}))

	deployKeyOption := gitea.CreateKeyOption{
		Title:    "GitOps Deployment Key",
		Key:      publicKey,
		ReadOnly: true,
	}
	_, resp, err := client.CreateDeployKey(repository.Organization, repository.Name, deployKeyOption)
	if err != nil {
		return resp, fmt.Errorf("failed to create deploy key: %w", err)
	}

	err = createKubernetesSecretForArgoCD("default", repository.Name+"-deploy-key", privateKey, repository)
	if err != nil {
		return resp, fmt.Errorf("failed to create kubernetes secret: %w", err)
	}

	return resp, nil
}

func createKubernetesSecretForArgoCD(namespace, secretName, privateKey string, repo Repository) error {
	config, err := helpers.BuildKubeConfig()
	if err != nil {
		return fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name: secretName,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
			},
		},
		Data: map[string][]byte{
			"url":           []byte("git@kds-deployment-stack-ssh-cluster:/" + repo.Organization + "/" + repo.Name + ".git"),
			"sshPrivateKey": []byte(privateKey),
			"type":          []byte("git"),
			"insecure":      []byte("true"),
		},
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, v1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	log.Printf("Secret %s created successfully in namespace %s\n", secretName, namespace)
	return nil
}

func (g *Config) createStageTemplate(stage string, gitOrg string, gitRepo string) (string, error) {
	// Create a new file
	data := AppSetTemplateData{
		Stage:       stage,
		GiteaSshUrl: g.SSHUrl,
		GitOrg:      gitOrg,
		GitRepo:     gitRepo,
	}

	tmpl, err := template.New("appSetTemplate").Parse(appSetTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return tpl.String(), nil
}

func (g *Config) commitAppSet(stage string, gitOrg string, gitRepo string) error {
	client, err := gitea.NewClient(g.Url, gitea.SetBasicAuth(g.Credentials.Username, g.Credentials.Password), gitea.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.TLSConfig.InsecureSkipVerify,
			},
		},
	}))
	if err != nil {
		return fmt.Errorf("failed to create gitea client: %w", err)
	}
	content, err := g.createStageTemplate(stage, gitOrg, gitRepo)
	if err != nil {
		return fmt.Errorf("failed to create stage template: %w", err)
	}

	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

	// Check if the file exists
	fileDetail, resp, _ := client.GetContents(gitOrg, gitRepo, "main", stage+"/appset.yaml")
	if resp.StatusCode == 404 {
		// File does not exist, create it
		opts := gitea.CreateFileOptions{
			FileOptions: gitea.FileOptions{
				BranchName: "main",
				Message:    "Initial commit of AppSet " + stage,
				Author: gitea.Identity{
					Name:  "Deployer",
					Email: "deploy@on-clouds.at",
				},
				Committer: gitea.Identity{
					Name:  "Deployer",
					Email: "deploy@on-clouds.at",
				},
			},
			Content: encodedContent,
		}
		_, _, err := client.CreateFile(gitOrg, gitRepo, stage+"/appset.yaml", opts)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
	} else {
		// File exists, update it
		opts := gitea.UpdateFileOptions{
			FileOptions: gitea.FileOptions{
				BranchName: "main",
				Message:    "Initial commit of AppSet " + stage,
				Author: gitea.Identity{
					Name:  "Deployer",
					Email: "deploy@on-clouds.at",
				},
				Committer: gitea.Identity{
					Name:  "Deployer",
					Email: "deploy@on-clouds.at",
				},
			},
			Content: encodedContent,
			SHA:     fileDetail.SHA,
		}
		_, _, err := client.UpdateFile(gitOrg, gitRepo, stage+"/appset.yaml", opts)
		if err != nil {
			return fmt.Errorf("failed to update file: %w", err)
		}
	}

	return nil
}
