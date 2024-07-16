package harbor

import "fmt"

func (h *Config) CreateConfiguration(config map[string]interface{}) error {
	_, _, err := h.queryApi("PUT", h.Url+configurationApi, config)
	if err != nil {
		fmt.Println("Error creating configuration")
	}

	return nil

}
