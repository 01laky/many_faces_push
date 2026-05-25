package fcmclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

type serviceAccount struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
}

const maxServiceAccountJSONBytes = 32 * 1024

// ValidateServiceAccountJSON parses and validates a Firebase service account JSON blob.
func ValidateServiceAccountJSON(raw string) (projectID string, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("service account JSON is empty")
	}
	if len(trimmed) > maxServiceAccountJSONBytes {
		return "", fmt.Errorf("service account JSON exceeds size limit")
	}

	var sa serviceAccount
	if err := json.Unmarshal([]byte(trimmed), &sa); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	if sa.Type != "service_account" {
		return "", fmt.Errorf("type must be service_account")
	}
	if strings.TrimSpace(sa.ProjectID) == "" {
		return "", fmt.Errorf("project_id is required")
	}
	if strings.TrimSpace(sa.PrivateKey) == "" || !strings.Contains(sa.PrivateKey, "BEGIN") {
		return "", fmt.Errorf("private_key is invalid or truncated")
	}
	if strings.TrimSpace(sa.ClientEmail) == "" {
		return "", fmt.Errorf("client_email is required")
	}
	return sa.ProjectID, nil
}
