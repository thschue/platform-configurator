package harbor

import (
	"fmt"
	"log"
)

func (h *Config) CreateProject(project Project) error {
	jsonData := map[string]interface{}{
		"project_name": project.Name,
		"metadata":     project.Metadata,
	}

	_, errorCode, err := h.queryApi("POST", h.Url+projectApi, jsonData)
	if err != nil && errorCode != 409 {
		return fmt.Errorf("error creating project: %w", err)
	}

	if errorCode == 409 {
		log.Println("Project already exists, updating project")
		_, errorCode, err = h.queryApi("PUT", h.Url+projectApi+"/"+project.Name, jsonData)
		if err != nil {
			return fmt.Errorf("error updating project: %w", err)
		}
		log.Println(fmt.Sprintf("Project %s updated", project.Name))
	} else {
		log.Println(fmt.Sprintf("Project %s created", project.Name))
	}

	return nil
}
