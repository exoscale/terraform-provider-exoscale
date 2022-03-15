package exoscale

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

// DiffSuppressFunc https://www.terraform.io/plugin/sdkv2/schemas/schema-behaviors#diffsuppressfunc
// Do no show case differences between state and resource
func suppressCaseDiff(k, old, new string, d *schema.ResourceData) bool {
	return strings.EqualFold(old, new)
}
