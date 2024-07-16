package config

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
)

func New(filename string) (*Config, error) {
	file := filename
	yamlFile, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer yamlFile.Close()

	yamlBytes, err := io.ReadAll(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var config *Config

	err = yaml.Unmarshal(yamlBytes, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling yaml: %w", err)
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
