package main

import (
	"bytes"
	"code.gitea.io/sdk/gitea"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const configurationApi = "/api/v2.0/configurations"
const projectApi = "/api/v2.0/projects"
const registryApi = "/api/v2.0/registries"
const replicationPolicyApi = "/api/v2.0/replication/policies"
const robotAccountApi = "/api/v2.0/robots"
const replicationExecutionApi = "/api/v2.0/replication/executions"
const giteaSshUrl = "git@kds-deployment-stack-ssh-cluster:/"

type Config struct {
	Gitea    GiteaConfig     `yaml:"gitea"`
	Harbor   HarborConfig    `yaml:"harbor"`
	Projects []HarborProject `yaml:"projects"`
}

type HarborProject struct {
	Name             string                  `yaml:"name"`
	Metadata         map[string]interface{}  `yaml:"metadata"`
	ReplicationRules []HarborReplicationRule `yaml:"replicationRules"`
}

type HarborReplicationRule struct {
	Repository           string `yaml:"repository"`
	Source               string `yaml:"sourceRegistry"`
	DestinationNamespace string `yaml:"destinationNamespace"`
	Crontab              string `yaml:"crontab"`
}

type HarborRobotAccount struct {
	Name    string `yaml:"name"`
	Token   string `yaml:"token"`
	Project string `yaml:"project"`
}

type GiteaConfig struct {
	Url          string              `yaml:"url"`
	Credentials  Credentials         `yaml:"credentials"`
	Orgs         []GiteaOrganization `yaml:"orgs"`
	Repositories []GiteaRepository   `yaml:"repositories"`
	TLSConfig    TlsConfig           `yaml:"tlsConfig"`
}

type GiteaOrganization struct {
	Name         string            `yaml:"name"`
	Visibility   gitea.VisibleType `yaml:"visibility"`
	Repositories []GiteaRepository `yaml:"repositories"`
}

type GiteaRepository struct {
	Name         string        `yaml:"name"`
	Organization string        `yaml:"org"`
	Description  string        `yaml:"description"`
	Private      bool          `yaml:"private"`
	Stages       []GiteaStages `yaml:"stages"`
}

type GiteaStages struct {
	Name string `yaml:"name"`
}

type HarborConfig struct {
	Url           string                  `yaml:"url"`
	Configuration map[string]interface{}  `yaml:"configuration"`
	Projects      []HarborProject         `yaml:"projects"`
	Registries    []HarborRegistry        `yaml:"registries"`
	Replications  []HarborReplicationRule `yaml:"replications"`
	Credentials   Credentials             `yaml:"credentials"`
	TLSConfig     TlsConfig               `yaml:"tlsConfig"`
	Client        http.Client
	RobotAccounts []HarborRobotAccount `yaml:"robotAccounts"`
}

type HarborRegistry struct {
	Name        string                    `yaml:"name"`
	Description string                    `yaml:"description"`
	Url         string                    `yaml:"url"`
	Type        string                    `yaml:"type"`
	Credentials HarborRegistryCredentials `yaml:"credentials"`
}

type HarborRegistryCredentials struct {
	AccessKey    string `yaml:"access_key"`
	AccessSecret string `yaml:"access_secret"`
}

type TlsConfig struct {
	InsecureSkipVerify bool `yaml:"insecureSkipVerify"`
}

type Credentials struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Gitea struct {
	credentials Credentials
}

type Harbor struct {
	credentials Credentials
}

type TemplateData struct {
	Stage       string
	GiteaSshUrl string
	GitOrg      string
	GitRepo     string
}

const appSetTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: platform-{{ .Stage }}
spec:
  generators:
    - git:
        repoURL: {{ .GiteaSshUrl }}/{{ .GitOrg }}/{{ .GitRepo }}.git
        revision: main
        files:
          - path: {{ .Stage }}/**/config.yaml
  template:
    metadata:
      name: {{ "'{{path.basename}}'" }}
    spec:
      project: default
      sources:
      - repoURL: {{ "'{{repoURL}}'" }}
        targetRevision: {{ "'{{targetRevision}}'" }}
        chart: {{ "'{{chart}}'" }}
        helm:
          valueFiles:
          - $values/{{ .Stage }}/{{ "{{path.basename}}" }}/values.yaml
      - repoURL: {{ .GiteaSshUrl }}/{{ .GitOrg }}/{{ .GitRepo }}.git
        targetRevision: main
        path: {{ .Stage }}/{{ "{{path.basename}}" }}
      - repoURL: {{ .GiteaSshUrl }}/{{ .GitOrg }}/{{ .GitRepo }}.git
        targetRevision: main
        ref: values
      destination:
        server: https://kubernetes.default.svc
        namespace: {{ "'{{path.basename}}'" }}
      syncPolicy:
        automated: 
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
          - ServerSideApply={{ "'{{serverSideApply}}'" }}

`

func (g *GiteaConfig) createOrganization(organization GiteaOrganization) error {

	client, err := gitea.NewClient(g.Url, gitea.SetBasicAuth(g.Credentials.Username, g.Credentials.Password), gitea.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.TLSConfig.InsecureSkipVerify,
			},
		},
	}))

	if err != nil {
		fmt.Println(err)
		fmt.Println("Error creating Gitea client")
	}

	orgOption := gitea.CreateOrgOption{
		Name:       organization.Name,
		Visibility: organization.Visibility,
	}

	org, _, err := client.CreateOrg(orgOption)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error creating organization")
	}

	fmt.Println(fmt.Sprintf("Organization %s created", org.UserName))
	return nil
}

func (g *GiteaConfig) createRepository(organization string, repo GiteaRepository) error {
	client, err := gitea.NewClient(g.Url, gitea.SetBasicAuth(g.Credentials.Username, g.Credentials.Password), gitea.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.TLSConfig.InsecureSkipVerify,
			},
		},
	}))
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error creating Gitea client")
	}

	repoOption := gitea.CreateRepoOption{
		Name:          repo.Name,
		Private:       repo.Private,
		AutoInit:      true,
		DefaultBranch: "main",
	}

	_, _, err = client.CreateOrgRepo(organization, repoOption)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error creating repository")
	}

	err = g.createDeployKey(repo)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error creating deploy key")
	}

	for _, stage := range repo.Stages {
		err = g.commitAppSet(stage.Name, organization, repo.Name)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Error committing appset")
		}
	}

	return nil
}

func generateSSHKeyPair() (privateKey, publicKey string, err error) {
	// Generate a new private key.
	privateKeyObj, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate the corresponding public key.
	publicKeyBytes, err := ssh.NewPublicKey(&privateKeyObj.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %w", err)
	}
	publicKey = string(ssh.MarshalAuthorizedKey(publicKeyBytes))

	// Encode the private key to PEM format.
	privateKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKeyObj),
		},
	)

	privateKey = string(privateKeyPEM)

	return privateKey, publicKey, nil
}

func (g *GiteaConfig) createDeployKey(repository GiteaRepository) error {
	// Example usage
	privateKey, publicKey, err := generateSSHKeyPair()
	if err != nil {
		fmt.Println("Error generating SSH key pair:", err)
		return err
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
	_, _, err = client.CreateDeployKey(repository.Organization, repository.Name, deployKeyOption)
	if err != nil {
		return fmt.Errorf("failed to create deploy key: %w", err)
	}

	err = createKubernetesSecretForArgoCD("default", repository.Name+"-deploy-key", privateKey, repository)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes secret: %w", err)
	}

	return nil
}

func createKubernetesSecretForArgoCD(namespace, secretName, privateKey string, repo GiteaRepository) error {
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to build kubeconfig: %w", err)
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

	fmt.Printf("Secret %s created successfully in namespace %s\n", secretName, namespace)
	return nil
}

func (g *GiteaConfig) createStageTemplate(stage string, gitOrg string, gitRepo string) (string, error) {
	// Create a new file
	data := TemplateData{
		Stage:       stage,
		GiteaSshUrl: giteaSshUrl,
		GitOrg:      gitOrg,
		GitRepo:     gitRepo,
	}

	tmpl, err := template.New("appSetTemplate").Parse(appSetTemplate)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error parsing template")
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		fmt.Println("Error executing template")
	}
	return tpl.String(), nil
}

func (g *GiteaConfig) commitAppSet(stage string, gitOrg string, gitRepo string) error {
	client, err := gitea.NewClient(g.Url, gitea.SetBasicAuth(g.Credentials.Username, g.Credentials.Password), gitea.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.TLSConfig.InsecureSkipVerify,
			},
		},
	}))
	if err != nil {
		fmt.Println("Error creating Gitea client")
	}
	content, err := g.createStageTemplate(stage, gitOrg, gitRepo)
	if err != nil {
		fmt.Println("Error creating stage template")
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
func (h *HarborConfig) createRegistry(registry HarborRegistry) {
	jsonData := map[string]interface{}{
		"name":        registry.Name,
		"description": registry.Description,
		"url":         registry.Url,
		"type":        registry.Type,
		"credential": map[string]interface{}{
			"access_key":    registry.Credentials.AccessKey,
			"access_secret": registry.Credentials.AccessSecret,
		},
	}

	_, errorCode, err := h.queryHarborApi("POST", h.Url+registryApi, jsonData)
	if err != nil && errorCode != 409 {
		fmt.Println("Error creating registry")
	}

	if errorCode == 409 {
		fmt.Println("Registry already exists, updating registry")
		_, errorCode, err = h.queryHarborApi("PUT", h.Url+registryApi+"/"+registry.Name, jsonData)
		if err != nil {
			fmt.Println("Error updating registry")
		}
		fmt.Println(fmt.Sprintf("Registry %s updated", registry.Name))
	} else {
		fmt.Println(fmt.Sprintf("Registry %s created", registry.Name))
	}
}

func (h *HarborConfig) getRegistryId(name string) interface{} {
	registries, _, err := h.queryHarborApi("GET", h.Url+registryApi, nil)
	if err != nil {
		fmt.Println("Error getting registries")
	}

	var registryList []map[string]interface{}

	json.NewDecoder(registries).Decode(&registryList)

	for _, registry := range registryList {
		if registry["name"] == name {
			return registry["id"]
		}
	}
	return 999
}

func (h *HarborConfig) createConfiguration(config map[string]interface{}) error {
	_, _, err := h.queryHarborApi("PUT", h.Url+configurationApi, config)
	if err != nil {
		fmt.Println("Error creating configuration")
	}

	return nil

}
func (h *HarborConfig) createProject(project HarborProject) error {
	jsonData := map[string]interface{}{
		"project_name": project.Name,
		"metadata":     project.Metadata,
	}

	_, errorCode, err := h.queryHarborApi("POST", h.Url+projectApi, jsonData)
	if err != nil && errorCode != 409 {
		fmt.Println("Error creating project")
	}

	if errorCode == 409 {
		fmt.Println("Project already exists, updating project")
		_, errorCode, err = h.queryHarborApi("PUT", h.Url+projectApi+"/"+project.Name, jsonData)
		if err != nil {
			fmt.Println("Error updating project")
		}
		fmt.Println(fmt.Sprintf("Project %s updated", project.Name))
	} else {
		fmt.Println(fmt.Sprintf("Project %s created", project.Name))
	}

	return nil
}

func (h *HarborConfig) createReplicationRule(rule HarborReplicationRule) error {
	src := h.getRegistryId(rule.Source)

	jsonData := map[string]interface{}{
		"dest_namespace": rule.DestinationNamespace,
		"enabled":        true,
		"name":           strings.Replace(rule.Repository, "/", "-", -1),
		"override":       true,
		"src_registry": map[string]interface{}{
			"id": src,
		},
		"filters": []map[string]interface{}{
			{
				"type":  "name",
				"value": rule.Repository,
			},
		},
		"trigger": map[string]interface{}{
			"type": "scheduled",
			"trigger_settings": map[string]string{
				"cron": rule.Crontab,
			},
		},
	}

	_, errorCode, err := h.queryHarborApi("POST", h.Url+replicationPolicyApi, jsonData)
	if err != nil && errorCode != 409 {
		fmt.Println("Error creating replication rule")
	}

	if errorCode == 201 {
		id := h.getReplicationRuleId(strings.Replace(rule.Repository, "/", "-", -1))
		h.runReplicationRule(id)
		fmt.Println(fmt.Sprintf("Replication rule %s started", rule.Repository))
	}

	return err
}

func (h *HarborConfig) getReplicationRuleId(name string) interface{} {
	replicationRules, _, err := h.queryHarborApi("GET", h.Url+replicationPolicyApi, nil)
	if err != nil {
		fmt.Println("Error getting replication rules")
	}

	var replicationRuleList []map[string]interface{}

	json.NewDecoder(replicationRules).Decode(&replicationRuleList)

	for _, rule := range replicationRuleList {
		if rule["name"] == name {
			return rule["id"]
		}
	}
	return 999
}

func (h *HarborConfig) runReplicationRule(ruleId interface{}) {
	jsonData := map[string]interface{}{
		"policy_id": ruleId,
	}

	_, _, err := h.queryHarborApi("POST", h.Url+replicationExecutionApi, jsonData)
	if err != nil {
		fmt.Println("Error running replication rule")
	}
}

func (h *HarborConfig) createRobotAccount(account HarborRobotAccount) error {
	data := map[string]interface{}{
		"disable":    false,
		"duration":   -1,
		"editable":   false,
		"expires_at": -1,
		"level":      "system",
		"name":       account.Name,
		"permissions": []map[string]interface{}{
			{
				"access": []map[string]string{
					{
						"action":   "list",
						"resource": "artifact",
					},
					{
						"action":   "read",
						"resource": "artifact",
					},
					{
						"action":   "list",
						"resource": "repository",
					},
					{
						"action":   "pull",
						"resource": "repository",
					},
					{
						"action":   "read",
						"resource": "repository",
					},
					{
						"action":   "list",
						"resource": "tag",
					},
				},
				"kind":      "project",
				"namespace": account.Project,
			},
		},
	}

	_, errorCode, err := h.queryHarborApi("POST", h.Url+robotAccountApi, data)
	if err != nil && errorCode != 409 {
		fmt.Println("Error creating robot")
	}

	if errorCode == 409 {
		fmt.Println("Robot already exists, updating robot")
		_, errorCode, err = h.queryHarborApi("PUT", h.Url+robotAccountApi+"/"+account.Name, data)
		if err != nil {
			fmt.Println("Error updating project")
		}
		fmt.Println(fmt.Sprintf("Robot %s updated", account.Name))
	} else {
		fmt.Println(fmt.Sprintf("Robot %s created", account.Name))
	}
	return nil
}

func (h *HarborConfig) queryHarborApi(method string, endpoint string, data map[string]interface{}) (io.ReadCloser, int, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(h.Credentials.Username, h.Credentials.Password)

	fmt.Println(req)
	resp, err := h.Client.Do(req)
	if err != nil {
		fmt.Println(err)
		return resp.Body, resp.StatusCode, err
	}
	return resp.Body, resp.StatusCode, nil

}

func newConfig(filename string) (*Config, error) {
	file := filename
	yamlFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer yamlFile.Close()

	yamlBytes, _ := io.ReadAll(yamlFile)

	var config *Config

	_ = yaml.Unmarshal(yamlBytes, &config)

	if config.Harbor.Url != "" {
		config.Harbor.Client = http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: config.Harbor.TLSConfig.InsecureSkipVerify,
				},
			},
		}
	}

	return config, nil
}

func main() {
	config, err := newConfig("./config.yaml")
	if err != nil {
		fmt.Println("Error reading config file")
	}

	for k, v := range config.Harbor.Configuration {
		data := map[string]interface{}{
			k: v,
		}
		err = config.Harbor.createConfiguration(data)
		if err != nil {
			fmt.Println("Error creating configuration")
		}
	}

	for _, project := range config.Harbor.Projects {
		err = config.Harbor.createProject(project)
		if err != nil {
			fmt.Println("Error creating project")
		}
	}

	for _, registry := range config.Harbor.Registries {
		config.Harbor.createRegistry(registry)
	}

	for _, rule := range config.Harbor.Replications {
		err = config.Harbor.createReplicationRule(rule)
		if err != nil {
			fmt.Println("Error creating replication rule")
		}
	}

	for _, account := range config.Harbor.RobotAccounts {
		err = config.Harbor.createRobotAccount(account)
		if err != nil {
			fmt.Println("Error creating robot account")
		}
	}

	for _, org := range config.Gitea.Orgs {
		err := config.Gitea.createOrganization(org)
		if err != nil {
			fmt.Println("Error creating organization")
		}
	}

	for _, repo := range config.Gitea.Repositories {
		err := config.Gitea.createRepository(repo.Organization, repo)
		if err != nil {
			fmt.Println("Error creating repository")
		}
	}
}
