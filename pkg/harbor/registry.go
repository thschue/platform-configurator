package harbor

import (
	"encoding/json"
	"fmt"
	"log"
)

func (h *Config) CreateRegistry(registry Registry) error {
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

	_, errorCode, err := h.queryApi("POST", h.Url+registryApi, jsonData)
	if err != nil && errorCode != 409 {
		return fmt.Errorf("error creating registry: %w", err)
	}

	if errorCode == 409 {
		log.Println("Registry already exists, updating registry")
		_, errorCode, err = h.queryApi("PUT", h.Url+registryApi+"/"+registry.Name, jsonData)
		if err != nil {
			return fmt.Errorf("error updating registry: %w", err)
		}
		log.Println(fmt.Sprintf("Registry %s updated", registry.Name))
	} else {
		log.Println(fmt.Sprintf("Registry %s created", registry.Name))
	}
	return nil
}

func (h *Config) getRegistryId(name string) interface{} {
	registries, _, err := h.queryApi("GET", h.Url+registryApi, nil)
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
