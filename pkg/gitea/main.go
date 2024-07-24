package gitea

import (
	"code.gitea.io/sdk/gitea"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
)

func (h *Config) IsAvailable() (bool, error) {
	_, err := http.NewRequest("GET", h.Url, nil)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	return true, nil
}

func (g *Config) organizationExists(orgName string) bool {
	client, err := gitea.NewClient(g.Url, gitea.SetBasicAuth(g.Credentials.Username, g.Credentials.Password), gitea.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.TLSConfig.InsecureSkipVerify,
			},
		},
	}))

	_, resp, err := client.GetOrg(orgName)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false // Organization does not exist
		}
		log.Fatalf("Failed to get organization: %v", err)
	}

	return true // Organization exists
}

func (g *Config) CreateOrganization(organization Organization) error {

	client, err := gitea.NewClient(g.Url, gitea.SetBasicAuth(g.Credentials.Username, g.Credentials.Password), gitea.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.TLSConfig.InsecureSkipVerify,
			},
		},
	}))

	if err != nil {
		return fmt.Errorf("error creating Gitea client: %w", err)
	}

	orgOption := gitea.CreateOrgOption{
		Name:       organization.Name,
		Visibility: organization.Visibility,
	}

	org, _, err := client.CreateOrg(orgOption)
	if err != nil {
		if !g.organizationExists(organization.Name) {
			return fmt.Errorf("error creating organization: %w", err)
		}
		log.Println(fmt.Sprintf("Organization %s already exists", organization.Name))
		return nil
	}
	log.Println(fmt.Sprintf("Organization %s created", org.UserName))
	return nil
}

func (g *Config) repositoryExists(repo Repository) bool {
	client, err := gitea.NewClient(g.Url, gitea.SetBasicAuth(g.Credentials.Username, g.Credentials.Password), gitea.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.TLSConfig.InsecureSkipVerify,
			},
		},
	}))

	_, resp, err := client.GetRepo(repo.Organization, repo.Name)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false // Repository does not exist
		}
		return false
	}

	return true // Repository exists
}

func (g *Config) CreateRepository(organization string, repo Repository) error {
	client, err := gitea.NewClient(g.Url, gitea.SetBasicAuth(g.Credentials.Username, g.Credentials.Password), gitea.SetHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: g.TLSConfig.InsecureSkipVerify,
			},
		},
	}))
	if err != nil {
		return fmt.Errorf("error creating Gitea client: %w", err)
	}

	repoOption := gitea.CreateRepoOption{
		Name:          repo.Name,
		Private:       repo.Private,
		AutoInit:      true,
		DefaultBranch: "main",
	}

	_, resp, err := client.CreateOrgRepo(organization, repoOption)
	if err != nil && resp.StatusCode != 409 {
		if !g.repositoryExists(repo) {
			return fmt.Errorf("error creating repository: %w", err)
		}
		log.Println(fmt.Sprintf("Repository %s already exists", repo.Name))
	}

	for _, stage := range repo.Stages {
		if stage.ArgoProject == "" {
			stage.ArgoProject = "default"
		}

		if stage.ArgoCluster == "" {
			stage.ArgoCluster = "https://kubernetes.default.svc"
		}
		err = g.commitAppSet(stage, organization, repo.Name)
		if err != nil {
			return fmt.Errorf("error committing appset: %w", err)
		}
	}

	resp, err = g.createDeployKey(repo)
	fmt.Println(err)
	if err != nil && !errors.Is(err, fmt.Errorf("failed to create deploy key: A key with the same name already exists")) {
		return fmt.Errorf("error creating deploy key: %w", err)
	}

	return nil
}
