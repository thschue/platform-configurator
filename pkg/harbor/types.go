package harbor

import (
	"github.com/thschue/platformer/pkg/helpers"
	"net/http"
)

type Config struct {
	Url           string                 `yaml:"url"`
	Configuration map[string]interface{} `yaml:"configuration"`
	Projects      []Project              `yaml:"projects"`
	Registries    []Registry             `yaml:"registries"`
	Replications  []ReplicationRule      `yaml:"replications"`
	Credentials   helpers.Credentials    `yaml:"credentials"`
	TLSConfig     helpers.TlsConfig      `yaml:"tlsConfig"`
	Client        http.Client
	RobotAccounts []RobotAccount `yaml:"robotAccounts"`
}

type Project struct {
	Name             string                 `yaml:"name"`
	Metadata         map[string]interface{} `yaml:"metadata"`
	ReplicationRules []ReplicationRule      `yaml:"replicationRules"`
}

type ReplicationRule struct {
	Repository           string `yaml:"repository"`
	Source               string `yaml:"sourceRegistry"`
	DestinationNamespace string `yaml:"destinationNamespace"`
	Crontab              string `yaml:"crontab"`
}

type RobotAccount struct {
	Name    string `yaml:"name"`
	Token   string `yaml:"token"`
	Project string `yaml:"project"`
}

type Registry struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Url         string              `yaml:"url"`
	Type        string              `yaml:"type"`
	Credentials RegistryCredentials `yaml:"credentials"`
}

type RegistryCredentials struct {
	AccessKey    string `yaml:"access_key"`
	AccessSecret string `yaml:"access_secret"`
}
