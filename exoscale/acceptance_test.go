package exoscale

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

type importStateCheckFunc func([]*terraform.InstanceState) error

func composeImportStateCheckFunc(fs ...importStateCheckFunc) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		for _, f := range fs {
			if err := f(states); err != nil {
				return err
			}
		}

		return nil
	}
}

func testAccCheckResourceImportedAttributes(resourceType string, expected testAttrs) importStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		attrs, err := testAccFetchImportedAttributes(s, resourceType)
		if err != nil {
			return err
		}

		return checkResourceAttributes(expected, attrs)
	}
}

func testAccFetchImportedAttributes(states []*terraform.InstanceState, resourceType string) (map[string]string, error) {
	for _, s := range states {
		if s.Ephemeral.Type == resourceType {
			return s.Attributes, nil
		}
	}

	return nil, fmt.Errorf("imported resource %q not found", resourceType)
}
