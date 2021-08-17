package exoscale

import (
	"context"
	"fmt"
	"os"
	"reflect"
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

	testInstanceTypeIDTiny   = "b6cd1ff5-3a2f-4e9d-a4d1-8988c1191fe8"
	testInstanceTypeIDSmall  = "21624abb-764e-4def-81d7-9fc54b5957fb"
	testInstanceTypeIDMedium = "b6e9d1e8-89fc-4db3-aaa4-9b4c5b1d0844"
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

// resFromState returns the state of the resource r from the current global state s.
func resFromState(s *terraform.State, r string) (*terraform.InstanceState, error) {
	res, ok := s.RootModule().Resources[r]
	if !ok {
		return nil, fmt.Errorf("no resource %q found in state", r)
	}

	return res.Primary, nil
}

// attrFromState returns the value of the attribute a for the resource r from the current global state s.
func attrFromState(s *terraform.State, r, a string) (string, error) {
	res, err := resFromState(s, r)
	if err != nil {
		return "", err
	}

	v, ok := res.Attributes[a]
	if !ok {
		return "", fmt.Errorf("resource %q has no attribute %q", r, a)
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
			want:        testAttrs{"something": validateString("anything")},
			got:         nil,
			expectError: true,
		},
		{
			desc:        "attribute absent",
			want:        testAttrs{"something": validateString("this")},
			got:         map[string]string{"something else": "that"},
			expectError: true,
		},
		{
			desc:        "attribute with unexpected value",
			want:        testAttrs{"something": validateString("this")},
			got:         map[string]string{"something": "that"},
			expectError: true,
		},
		{
			desc: "attribute with expected value",
			want: testAttrs{"something": validateString("this")},
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

func Test_zonedStateContextFunc(t *testing.T) {
	type args struct {
		d *schema.ResourceData
	}

	testSchema := map[string]*schema.Schema{
		"zone": {
			Type: schema.TypeString,
		},
	}

	tests := []struct {
		name    string
		args    args
		want    []*schema.ResourceData
		wantErr bool
	}{
		{
			name: "missing zone",
			args: args{
				d: func() *schema.ResourceData {
					d := schema.TestResourceDataRaw(t, testSchema, nil)
					d.SetId("c01af84d-6ac6-4784-98bb-127c98be8258")
					return d
				}(),
			},
			wantErr: true,
		},
		{
			name: "ok",
			args: args{
				d: func() *schema.ResourceData {
					d := schema.TestResourceDataRaw(t, testSchema, nil)
					d.SetId("c01af84d-6ac6-4784-98bb-127c98be8258@ch-gva-2")
					return d
				}(),
			},
			want: []*schema.ResourceData{
				func() *schema.ResourceData {
					d := schema.TestResourceDataRaw(t, testSchema, nil)
					d.SetId("c01af84d-6ac6-4784-98bb-127c98be8258")
					_ = d.Set("zone", "ch-gva-2")
					return d
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := zonedStateContextFunc(context.Background(), tt.args.d, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("zonedStateContextFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("zonedStateContextFunc() got = %v, want %v", got, tt.want)
			}
		})
	}
}
