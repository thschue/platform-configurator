package harbor

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func (h *Config) CreateReplicationRule(rule ReplicationRule) error {
	src := h.getRegistryId(rule.Source)

	jsonData := map[string]interface{}{
		"dest_namespace": rule.DestinationNamespace,
		"enabled":        true,
		"name":           strings.Replace(rule.Repository, "/", "-", -1),
		"override":       true,
		"src_registry": map[string]interface{}{
			"id": src,
		},
		"filters": []map[string]interface{}{
			{
				"type":  "name",
				"value": rule.Repository,
			},
		},
		"trigger": map[string]interface{}{
			"type": "scheduled",
			"trigger_settings": map[string]string{
				"cron": rule.Crontab,
			},
		},
	}

	_, errorCode, err := h.queryApi("POST", h.Url+replicationPolicyApi, jsonData)
	if err != nil && errorCode != 409 {
		return fmt.Errorf("error creating replication rule: %w", err)
	}

	if errorCode == 201 {
		id, err := h.getReplicationRuleId(strings.Replace(rule.Repository, "/", "-", -1))
		err = h.runReplicationRule(id)
		if err != nil {
			return fmt.Errorf("error running replication rule: %w", err)
		}
		log.Println(fmt.Sprintf("Replication rule %s started", rule.Repository))
	}
	return nil
}

func (h *Config) getReplicationRuleId(name string) (interface{}, error) {
	replicationRules, _, err := h.queryApi("GET", h.Url+replicationPolicyApi, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting replication rules: %w", err)
	}

	var replicationRuleList []map[string]interface{}

	json.NewDecoder(replicationRules).Decode(&replicationRuleList)

	for _, rule := range replicationRuleList {
		if rule["name"] == name {
			return rule["id"], nil
		}
	}
	return 999, fmt.Errorf("replication rule not found")
}

func (h *Config) runReplicationRule(ruleId interface{}) error {
	jsonData := map[string]interface{}{
		"policy_id": ruleId,
	}

	_, _, err := h.queryApi("POST", h.Url+replicationExecutionApi, jsonData)
	if err != nil {
		return fmt.Errorf("error running replication rule: %w", err)
	}
	return nil
}
