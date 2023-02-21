package testutils

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/exoscale/terraform-provider-exoscale/exoscale"
)

const (
	Prefix                         = "test-terraform-exoscale"
	TestDescription                = "Created by the terraform-exoscale provider"
	TestZoneName                   = "ch-dk-2"
	TestInstanceTemplateName       = "Linux Ubuntu 20.04 LTS 64-bit"
	TestInstanceTemplateUsername   = "ubuntu"
	TestInstanceTemplateFilter     = "featured"
	TestInstanceTemplateVisibility = "public"

	TestInstanceTypeIDTiny   = "b6cd1ff5-3a2f-4e9d-a4d1-8988c1191fe8"
	TestInstanceTypeIDSmall  = "21624abb-764e-4def-81d7-9fc54b5957fb"
	TestInstanceTypeIDMedium = "b6e9d1e8-89fc-4db3-aaa4-9b4c5b1d0844"
)

func AccPreCheck(t *testing.T) {
	key := os.Getenv("EXOSCALE_API_KEY")
	secret := os.Getenv("EXOSCALE_API_SECRET")
	if key == "" || secret == "" {
		t.Fatal("EXOSCALE_API_KEY and EXOSCALE_API_SECRET must be set for acceptance tests")
	}
}

// Providers returns all providers used during acceptance testing.
func Providers() map[string]func() (*schema.Provider, error) {
	testAccProvider := exoscale.Provider()
	return map[string]func() (*schema.Provider, error){
		"exoscale": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
}

// testAttrs represents a map of expected resource attributes during acceptance tests.
type TestAttrs map[string]schema.SchemaValidateDiagFunc

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
