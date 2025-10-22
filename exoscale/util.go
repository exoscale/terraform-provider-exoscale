package exoscale

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	egoscale "github.com/exoscale/egoscale/v2"
	v3 "github.com/exoscale/egoscale/v3"
)

// in returns true if v is found in list.
func in(list []string, v string) bool {
	for i := range list {
		if list[i] == v {
			return true
		}
	}

	return false
}

// defaultString returns the value of the string pointer v if not nil, otherwise the default value specified.
func defaultString(v *string, def string) string {
	if v != nil {
		return *v
	}

	return def
}

// defaultInt64 returns the value of the int64 pointer v if not nil, otherwise the default value specified.
func defaultInt64(v *int64, def int64) int64 {
	if v != nil {
		return *v
	}

	return def
}

// defaultBool returns the value of the bool pointer v if not nil, otherwise the default value specified.
func defaultBool(v *bool, def bool) bool {
	if v != nil {
		return *v
	}

	return def
}

// nonEmptyStringPtr returns a non-nil pointer to s if the string is not empty, otherwise nil.
func nonEmptyStringPtr(s string) *string {
	if s != "" {
		return &s
	}

	return nil
}

func schemaSetToStringArray(set *schema.Set) []string {
	array := make([]string, set.Len())
	for i, group := range set.List() {
		array[i] = group.(string)
	}

	return array
}

func unique(s []string) []string {
	inResult := map[string]struct{}{}
	var result []string
	for _, str := range s {
		if _, ok := inResult[str]; !ok {
			inResult[str] = struct{}{}
			result = append(result, str)
		}
	}
	return result
}

// DiffSuppressFunc https://www.terraform.io/plugin/sdkv2/schemas/schema-behaviors#diffsuppressfunc
// Do no show case differences between state and resource
func suppressCaseDiff(k, old, new string, d *schema.ResourceData) bool {
	return strings.EqualFold(old, new)
}

func parseIAMAccessKeyResource(v string) (*egoscale.IAMAccessKeyResource, error) {
	var iamAccessKeyResource egoscale.IAMAccessKeyResource

	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format")
	}
	iamAccessKeyResource.ResourceName = parts[1]

	parts = strings.SplitN(parts[0], "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format")
	}
	iamAccessKeyResource.Domain = parts[0]
	iamAccessKeyResource.ResourceType = parts[1]

	if iamAccessKeyResource.Domain == "" ||
		iamAccessKeyResource.ResourceType == "" ||
		iamAccessKeyResource.ResourceName == "" {
		return nil, fmt.Errorf("invalid format")
	}

	return &iamAccessKeyResource, nil
}

// validDNSNameRegex represents a valid DNS name pattern
// Based on RFC 1123 and RFC 952 standards
var validDNSNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

func isDNSName(i interface{}, k string) ([]string, []error) {
	v, ok := i.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}

	if err := validateDNSName(v); err != nil {
		return nil, []error{fmt.Errorf("expected %q to be a valid DNS name: %s", k, err)}
	}

	return nil, nil
}

// validateDNSName validates a DNS name according to RFC standards
func validateDNSName(name string) error {
	if name == "" {
		return fmt.Errorf("DNS name cannot be empty")
	}

	// Check maximum length (253 characters for FQDN)
	if len(name) > 253 {
		return fmt.Errorf("DNS name too long: maximum 253 characters allowed")
	}

	// Remove trailing dot if present (FQDN)
	name = strings.TrimSuffix(name, ".")

	// Check if it's a valid hostname using Go's net package
	if net.ParseIP(name) != nil {
		return fmt.Errorf("DNS name cannot be an IP address")
	}

	// Validate format using regex
	if !validDNSNameRegex.MatchString(name) {
		return fmt.Errorf("invalid DNS name format")
	}

	// Check each label individually
	labels := strings.Split(name, ".")
	for _, label := range labels {
		if err := validateDNSLabel(label); err != nil {
			return err
		}
	}

	return nil
}

// validateDNSLabel validates a single DNS label
func validateDNSLabel(label string) error {
	if label == "" {
		return fmt.Errorf("DNS label cannot be empty")
	}

	if len(label) > 63 {
		return fmt.Errorf("DNS label too long: maximum 63 characters allowed")
	}

	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return fmt.Errorf("DNS label cannot start or end with hyphen")
	}
	return nil
}

// This is an auxiliary function, aimed to extract a similar logic of detaching an instance
// before dropping a resource
func detachMatchingResource(
	ctx context.Context,
	client *v3.Client,
	resourceType string,
	resourceID v3.UUID,
	match func(*v3.Instance, v3.UUID) bool,
	detach func(ctx context.Context, client *v3.Client, id v3.UUID, instance *v3.Instance) (*v3.Operation, error),
) diag.Diagnostics {

	listInstancesResponse, err := client.ListInstances(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, listInst := range listInstancesResponse.Instances {
		inst, err := client.GetInstance(ctx, listInst.ID)
		if err != nil {
			return diag.FromErr(err)
		}

		if !match(inst, resourceID) {
			continue
		}

		tflog.Debug(ctx,
			fmt.Sprintf("Found instance with matching %s, detaching...", resourceType),
			map[string]interface{}{
				"instance_id": inst.ID,
				"resource_id": resourceID,
			},
		)

		op, err := detach(ctx, client, resourceID, inst)
		if err != nil {
			return diag.FromErr(err)
		}

		if _, err = client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
