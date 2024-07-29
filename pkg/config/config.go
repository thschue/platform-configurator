package config

import (
	"crypto/tls"
	"fmt"
	"github.com/spf13/viper"
	"net/http"
	"strings"
)

func New(filename string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(filename)
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	v.SetDefault("gitea.url", "")
	v.SetDefault("gitea.sshUrl", "")
	v.SetDefault("gitea.credentials.username", "")
	v.SetDefault("gitea.credentials.password", "")

	v.SetDefault("harbor.url", "")
	v.SetDefault("harbor.credentials.username", "")
	v.SetDefault("harbor.credentials.password", "")

	v.AutomaticEnv()

	var config *Config

	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

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
