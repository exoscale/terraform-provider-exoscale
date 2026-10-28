package database

import (
	"fmt"
	"strings"
)

// parseKafkaACLImportID splits a "service/acl_id@zone" import identifier.
func parseKafkaACLImportID(id string) (service, aclID, zone string, err error) {
	resourceID, zone, err := parseZonedImportID(id)
	if err != nil {
		return "", "", "", fmt.Errorf("expected service/acl_id@zone, got %q", id)
	}

	parts := strings.Split(resourceID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("expected service/acl_id@zone, got %q", id)
	}

	return parts[0], parts[1], zone, nil
}
