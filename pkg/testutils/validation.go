package testutils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

// testAttrs represents a map of expected resource attributes during acceptance tests.
type TestAttrs map[string]schema.SchemaValidateDiagFunc

func AccPreCheck(t *testing.T) {
	key := os.Getenv("EXOSCALE_API_KEY")
	secret := os.Getenv("EXOSCALE_API_SECRET")
	if key == "" || secret == "" {
		t.Fatal("EXOSCALE_API_KEY and EXOSCALE_API_SECRET must be set for acceptance tests")
	}
}

// testResourceStateValidationFunc represents a resource state validation function.
type TestResourceStateValidationFunc func(state *terraform.InstanceState) error

// checkResourceAttributes compares a map of resource attributes against a map
// of expected resource attributes and performs validation on the values.
func CheckResourceAttributes(want TestAttrs, got map[string]string) error {
	for attr, validateFunc := range want {
		v, ok := got[attr]
		if !ok {
			return fmt.Errorf("expected attribute %q not found in map", attr)
		} else if diags := validateFunc(v, cty.GetAttrPath(attr)); diags.HasError() {
			errors := make([]string, 0)
			for _, d := range diags {
				if d.Severity == diag.Error {
					errors = append(errors, d.Summary)
				}
			}

			return fmt.Errorf("invalid value for attribute %q:\n%s\n", // nolint:revive
				attr, strings.Join(errors, "\n"))
		}
	}

	return nil
}

// checkResourceState executes the specified TestResourceStateValidationFunc
// functions against the state of the resource matching the specified
// identifier r (e.g. "exoscale_compute.test"), and returns an error if any
// test function returns an error.
func CheckResourceState(r string, tests ...TestResourceStateValidationFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res, ok := s.RootModule().Resources[r]
		if !ok {
			return fmt.Errorf("resource %q not found in the state", r)
		}

		for _, t := range tests {
			if err := t(res.Primary); err != nil {
				return err
			}
		}

		return nil
	}
}

// CheckResourceStateValidateAttributes compares a map of resource attributes against
// a map of expected resource attributes and performs validation on the values.
func CheckResourceStateValidateAttributes(want TestAttrs) TestResourceStateValidationFunc {
	return func(s *terraform.InstanceState) error {
		return CheckResourceAttributes(want, s.Attributes)
	}
}

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
