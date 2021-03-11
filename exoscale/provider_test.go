package exoscale

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// Common test environment information
const (
	testPrefix               = "test-terraform-exoscale-provider"
	testDescription          = "Created by the terraform-exoscale provider"
	testZoneName             = "ch-gva-2"
	testInstanceTemplateName = "Linux Ubuntu 20.04 LTS 64-bit"

	/*
		Reference template used for tests: "Linux Ubuntu 20.04 LTS 64-bit" @ ch-gva-2 (featured)

		cs --region cloudstack listTemplates \
		    templatefilter=featured \
		    zoneid=1128bd56-b4d9-4ac6-a7b9-c715b187ce11 \
		    name="Linux Ubuntu 20.04 LTS 64-bit"
	*/
	testInstanceTemplateID = "23c0622f-34cd-44c3-b995-a56d436cff85"

	testInstanceTypeIDTiny  = "b6cd1ff5-3a2f-4e9d-a4d1-8988c1191fe8"
	testInstanceTypeIDSmall = "21624abb-764e-4def-81d7-9fc54b5957fb"
)

// testAttrs represents a map of expected resource attributes during acceptance tests.
type testAttrs map[string]schema.SchemaValidateFunc

// testResourceStateValidationFunc represents a resource state validation function.
type testResourceStateValidationFunc func(state *terraform.InstanceState) error

var (
	testAccProviders map[string]terraform.ResourceProvider
	testAccProvider  *schema.Provider
	testEnvironment  string
)

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"exoscale": testAccProvider,
	}

	testEnvironment = os.Getenv("EXOSCALE_API_ENVIRONMENT")
	if testEnvironment == "" {
		testEnvironment = defaultEnvironment
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
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
		} else if _, es := validateFunc(v, ""); len(es) > 0 {
			for i := range es {
				errors := make([]string, len(es))
				for _, e := range es {
					errors[i] = e.Error()
				}

				if len(errors) > 0 {
					return fmt.Errorf("invalid value for attribute %q:\n%s\n",
						attr, strings.Join(errors, "\n"))
				}
			}
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

func testRandomString() string {
	chars := "1234567890abcdefghijklmnopqrstuvwxyz"

	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 10)
	for i := range b {
		b[i] = chars[rand.Int63()%int64(len(chars))]
	}

	return string(b)
}
