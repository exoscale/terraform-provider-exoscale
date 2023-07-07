package exoscale

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	egoscale "github.com/exoscale/egoscale/v2"
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
