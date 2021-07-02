package exoscale

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Common test environment information
const (
	testPrefix                   = "test-terraform-exoscale"
	testDescription              = "Created by the terraform-exoscale provider"
	testZoneName                 = "de-muc-1"
	testInstanceTemplateName     = "Linux Ubuntu 20.04 LTS 64-bit"
	testInstanceTemplateUsername = "ubuntu"
	testInstanceTemplateFilter   = "featured"

	/*
		Reference template used for tests: "Linux Ubuntu 20.04 LTS 64-bit" @ de-muc-1 (featured)

		cs --region cloudstack listTemplates \
		    templatefilter=featured \
		    zoneid=85664334-0fd5-47bd-94a1-b4f40b1d2eb7 \
		    name="Linux Ubuntu 20.04 LTS 64-bit"
	*/
	testInstanceTemplateID = "a5ddefa9-7e98-40cb-94b3-e20348b878fa"

	testInstanceTypeIDTiny  = "b6cd1ff5-3a2f-4e9d-a4d1-8988c1191fe8"
	testInstanceTypeIDSmall = "21624abb-764e-4def-81d7-9fc54b5957fb"
)

// testAttrs represents a map of expected resource attributes during acceptance tests.
type testAttrs map[string]schema.SchemaValidateDiagFunc

// testResourceStateValidationFunc represents a resource state validation function.
type testResourceStateValidationFunc func(state *terraform.InstanceState) error

var (
	testAccProviders map[string]func() (*schema.Provider, error)
	testAccProvider  *schema.Provider
	testEnvironment  string
)

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]func() (*schema.Provider, error){
		"exoscale": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}

	testEnvironment = os.Getenv("EXOSCALE_API_ENVIRONMENT")
	if testEnvironment == "" {
		testEnvironment = defaultEnvironment
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	key := os.Getenv("EXOSCALE_API_KEY")
	secret := os.Getenv("EXOSCALE_API_SECRET")
	if key == "" || secret == "" {
		t.Fatal("EXOSCALE_API_KEY and EXOSCALE_API_SECRET must be set for acceptance tests")
	}
}

// checkResourceAttributes compares a map of resource attributes against a map
// of expected resource attributes and performs validation on the values.
func checkResourceAttributes(want testAttrs, got map[string]string) error {
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

// checkResourceStateValidateAttributes compares a map of resource attributes against
// a map of expected resource attributes and performs validation on the values.
func checkResourceStateValidateAttributes(want testAttrs) testResourceStateValidationFunc {
	return func(s *terraform.InstanceState) error {
		return checkResourceAttributes(want, s.Attributes)
	}
}

// checkResourceState executes the specified testResourceStateValidationFunc
// functions against the state of the resource matching the specified
// identifier r (e.g. "exoscale_compute.test"), and returns an error if any
// test function returns an error.
func checkResourceState(r string, tests ...testResourceStateValidationFunc) resource.TestCheckFunc {
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

func attrFromState(s *terraform.State, r, key string) (string, error) {
	res, ok := s.RootModule().Resources[r]
	if !ok {
		return "", fmt.Errorf("no resource %q found in state", r)
	}

	v, ok := res.Primary.Attributes[key]
	if !ok {
		return "", fmt.Errorf("no attribute %q found in resource %q", key, r)
	}

	return v, nil
}

func TestCheckResourceAttributes(t *testing.T) {
	type testCase struct {
		desc        string
		want        testAttrs
		got         map[string]string
		expectError bool
	}

	for _, tc := range []testCase{
		{
			desc:        "empty attributes map",
			want:        testAttrs{"something": ValidateString("anything")},
			got:         nil,
			expectError: true,
		},
		{
			desc:        "attribute absent",
			want:        testAttrs{"something": ValidateString("this")},
			got:         map[string]string{"something else": "that"},
			expectError: true,
		},
		{
			desc:        "attribute with unexpected value",
			want:        testAttrs{"something": ValidateString("this")},
			got:         map[string]string{"something": "that"},
			expectError: true,
		},
		{
			desc: "attribute with expected value",
			want: testAttrs{"something": ValidateString("this")},
			got:  map[string]string{"something": "this"},
		},
	} {
		err := checkResourceAttributes(tc.want, tc.got)
		if err != nil && !tc.expectError {
			t.Errorf("test case %q failed: expected no error but got: %s", tc.desc, err)
		} else if err == nil && tc.expectError {
			t.Errorf("test case failed: %s: expected an error but got none", tc.desc)
		}
	}
}
