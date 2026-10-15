package exoscale

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/provider"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Common test environment information
const (
	testPrefix                     = "test-terraform-exoscale"
	testDescription                = "Created by the terraform-exoscale provider"
	testZoneName                   = "ch-gva-2"
	testInstanceTemplateName       = "Linux Ubuntu 22.04 LTS 64-bit"
	testInstanceTemplateUsername   = "ubuntu"
	testInstanceTemplateFilter     = "featured"
	testInstanceTemplateVisibility = "public"
)

// testAttrs represents a map of expected resource attributes during acceptance tests.
type testAttrs map[string]schema.SchemaValidateDiagFunc

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
		testEnvironment = DefaultEnvironment
	}
}

func TestProvider(t *testing.T) {
	t.Parallel()

	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
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

func TestCheckResourceAttributes(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"exoscale": func() (tfprotov6.ProviderServer, error) {
		ctx := context.Background()
		upgradedProvider, err := tf5to6server.UpgradeServer(
			ctx,
			Provider().GRPCProvider,
		)
		if err != nil {
			return nil, err
		}

		newProvider := providerserver.NewProtocol6(&provider.ExoscaleProvider{})

		providers := []func() tfprotov6.ProviderServer{
			func() tfprotov6.ProviderServer {
				return upgradedProvider
			},
			newProvider,
		}

		return tf6muxserver.NewMuxServer(ctx, providers...)
	},
}
