package gitea

import (
	"code.gitea.io/sdk/gitea"
	"github.com/thschue/platformer/pkg/helpers"
)

type Config struct {
	Url          string              `yaml:"url"`
	SSHUrl       string              `yaml:"sshUrl"`
	Credentials  helpers.Credentials `yaml:"credentials"`
	Orgs         []Organization      `yaml:"orgs"`
	Repositories []Repository        `yaml:"repositories"`
	TLSConfig    helpers.TlsConfig   `yaml:"tlsConfig"`
}

type Organization struct {
	Name         string            `yaml:"name"`
	Visibility   gitea.VisibleType `yaml:"visibility"`
	Repositories []Repository      `yaml:"repositories"`
}

type Repository struct {
	Name         string   `yaml:"name"`
	Organization string   `yaml:"org"`
	Description  string   `yaml:"description"`
	Private      bool     `yaml:"private"`
	Stages       []Stages `yaml:"stages"`
}

type Stages struct {
	Name string `yaml:"name"`
}
