package harbor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const configurationApi = "/api/v2.0/configurations"
const projectApi = "/api/v2.0/projects"
const registryApi = "/api/v2.0/registries"
const replicationPolicyApi = "/api/v2.0/replication/policies"
const robotAccountApi = "/api/v2.0/robots"
const replicationExecutionApi = "/api/v2.0/replication/executions"

func (h *Config) IsAvailable() (bool, error) {
	_, err := http.NewRequest("GET", h.Url, nil)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	return true, nil
}

func (h *Config) queryApi(method string, endpoint string, data map[string]interface{}) (io.ReadCloser, int, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, 0, fmt.Errorf("error marshalling json: %w", err)
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(h.Credentials.Username, h.Credentials.Password)

	resp, err := h.Client.Do(req)
	if err != nil {
		return resp.Body, resp.StatusCode, fmt.Errorf("error querying api: %w", err)
	}
	return resp.Body, resp.StatusCode, nil

}
