package testutils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// ValidateString validates that the given field is a string and matches the expected value.
func ValidateString(str string) schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(func(i interface{}, k string) (s []string, es []error) {
		value, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		if value != str {
			es = append(es, fmt.Errorf("string %q doesn't match expected value %q", value, str))
			return
		}

		return
	})
}

// ValidateLowercaseString validates that the given fields contains only lowercase characters
func validateLowercaseString(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if strings.ContainsAny(v, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		errs = append(errs, fmt.Errorf("%q must be lowercase, got: %q", key, v))
	}
	return
}

// ValidatePortRange validates that the given field contains a port range.
func ValidatePortRange(i interface{}, k string) (s []string, es []error) {
	value, ok := i.(string)
	if !ok {
		es = append(es, fmt.Errorf("expected type of %s to be string", k))
		return
	}

	ports := strings.Split(value, "-")
	if len(ports) > 2 {
		es = append(es, fmt.Errorf("expected a range of at most two values, got %d", len(ports)))
		return
	}

	ps := make([]uint16, len(ports))
	for i, port := range ports {
		p, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			es = append(es, err)
			return
		}

		if p == 0 {
			es = append(es, fmt.Errorf("port expected to be between 1 and 65535, got %d", p))
			return
		}

		ps[i] = uint16(p)
	}

	if len(ports) == 2 {
		if ps[0] >= ps[1] {
			es = append(es, fmt.Errorf("port range should be ordered, %s < %s", ports[0], ports[1]))
		}
	}

	return
}

// ValidateComputeInstanceType validates that the given field contains a valid Exoscale Compute instance type.
func ValidateComputeInstanceType(v interface{}, _ cty.Path) diag.Diagnostics {
	value, ok := v.(string)
	if !ok {
		return diag.Errorf("expected field %q type to be string", v)
	}

	if !strings.Contains(value, ".") {
		return diag.Errorf(`invalid value %q, expected format "FAMILY.SIZE"`, value)
	}

	return nil
}

// ValidateComputeUserData validates that the given field contains a valid data.
func ValidateComputeUserData(v interface{}, _ cty.Path) diag.Diagnostics {
	value, ok := v.(string)
	if !ok {
		return diag.Errorf("expected field %q type to be string", v)
	}

	_, _, err := utils.EncodeUserData(value)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
