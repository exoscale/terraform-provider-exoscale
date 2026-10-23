package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// getKafkaACLConfig returns the ACL entries of a Kafka service.
// clearState is true when the service does not exist anymore.
func getKafkaACLConfig(ctx context.Context, client *exoscale.Client, service string, diagnostics *diag.Diagnostics) (acls *exoscale.DBAASKafkaAcls, clearState bool) {
	acls, err := client.GetDBAASKafkaAclConfig(ctx, service)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return nil, true
		}
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Kafka ACL configuration, got error: %s", err))
		return nil, false
	}

	return acls, false
}

// deleteKafkaACL runs an ACL entry deletion; an entry already gone is not an error.
func deleteKafkaACL(ctx context.Context, client *exoscale.Client, deleteFn func() (*exoscale.Operation, error), diagnostics *diag.Diagnostics) {
	op, err := deleteFn()
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete Kafka ACL entry, got error %s", err.Error()),
		)
		return
	}

	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete Kafka ACL entry, got error %s", err.Error()),
		)
	}
}

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
