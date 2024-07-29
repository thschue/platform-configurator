package harbor

import (
	"encoding/json"
	"fmt"
	"log"
)

type RobotResponse struct {
	Name   string `json:"name"`
	Secret string `json:"secret"`
}

func (h *Config) CreateRobotAccount(account RobotAccount) error {
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

	resp, errorCode, err := h.queryApi("POST", h.Url+robotAccountApi, data)
	if err != nil && errorCode != 409 {
		return fmt.Errorf("error creating robot: %w", err)
	}

	if errorCode == 409 {
		log.Println("Robot already exists, updating robot")
		resp, errorCode, err = h.queryApi("PUT", h.Url+robotAccountApi+"/"+account.Name, data)
		if err != nil {
			return fmt.Errorf("error updating robot: %w", err)
		}
		log.Println(fmt.Sprintf("Robot %s updated", account.Name))
	} else {
		log.Println(fmt.Sprintf("Robot %s created", account.Name))
	}

	var response RobotResponse
	err = json.NewDecoder(resp).Decode(&response)
	if err != nil {
		log.Println("Could not decode response")
	}

	fmt.Println(response)

	err = h.createKubernetesSecretForArgoCD("argocd", RobotAccount{Name: response.Name, Token: response.Secret}, "helm-"+account.Name)
	if err != nil {
		log.Println("Could not create secret")
	}

	return nil
}
