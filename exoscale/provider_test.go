package exoscale

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// Common test environment information
const (
	testPrefix               = "test-terraform-exoscale-provider"
	testDescription          = "Created by the terraform-exoscale provider"
	testZoneName             = "ch-gva-2"
	testInstanceTemplateName = "Linux Ubuntu 20.04 LTS 64-bit"
	testInstanceTemplateID   = "4843c915-b8ce-47e8-add7-d7a8de6d8bdc" // "Linux Ubuntu 20.04 LTS 64-bit" @ ch-gva-2
)

// testAttrs represents a map of expected resource attributes during acceptance tests.
type testAttrs map[string]schema.SchemaValidateFunc

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

var testEnvironment string

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
					return fmt.Errorf("invalid value for attribute %q:\n%s\n", attr, strings.Join(errors, "\n"))
				}
			}
		}
	}

	return nil
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
