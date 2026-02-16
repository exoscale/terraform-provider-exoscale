package instance_test

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var computeInstanceResource = `
data "exoscale_template" "my_template" {
  zone = "{{.Zone}}"
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}

data "exoscale_security_group" "default" {
  name = "default"
}

{{ if not .DeleteInstanceResource }}
resource "exoscale_compute_instance" "my_instance" {
  zone = "{{.Zone}}"
  name = "{{.Name}}"

  security_group_ids      = [
    data.exoscale_security_group.default.id,
  ]

  template_id = data.exoscale_template.my_template.id
  type        = "standard.micro"
  disk_size   = 10

{{ if .SetDestroyProtected }}
  destroy_protected = {{.DestroyProtected}}
{{ end }}
}
{{ end }}
`

var (
	destroyProtectionTmpl  = template.Must(template.New("compute_instance").Parse(computeInstanceResource))
	destroyProtectionError = regexp.MustCompile(`Forbidden: Operation delete-instance on resource .* is forbidden - reason: manual instance protection`)
)

type destroyProtectionTestData struct {
	Zone                   string
	SetDestroyProtected    bool
	DestroyProtected       bool
	Name                   string
	DeleteInstanceResource bool
}

func buildTestConfig(t *testing.T, testData destroyProtectionTestData) string {
	var tmplBuf bytes.Buffer

	err := destroyProtectionTmpl.Execute(&tmplBuf, testData)
	if err != nil {
		t.Fatal(err)
	}

	return tmplBuf.String()
}

func checkDestroyProtection(expected string) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		isDestroyProtected, err := testutils.AttrFromState(s, "exoscale_compute_instance.my_instance", "destroy_protected")
		if err != nil {
			return err
		}

		if expected != isDestroyProtected {
			return fmt.Errorf("destroy_protected does not match expected value: %q; is %q", expected, isDestroyProtected)
		}

		return nil
	}
}

func checkResourceDoesNotExist(name string) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		if _, ok := s.RootModule().Resources[name]; ok {
			return fmt.Errorf("compute instance was not deleted after destroy protection was removed")
		}

		return nil
	}
}

func testExplicitDestroyProtection(t *testing.T) {
	instanceName := acctest.RandomWithPrefix(testutils.Prefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// test instance creation with the destroy_protected field
				Config: buildTestConfig(t, destroyProtectionTestData{
					Zone:                testutils.TestZoneName,
					SetDestroyProtected: true,
					DestroyProtected:    true,
					Name:                instanceName,
				}),
				Check: checkDestroyProtection("true"),
			},
			{
				// test that the API returns an error if we try to delete the protected instance
				Config: buildTestConfig(t, destroyProtectionTestData{
					Zone:                   testutils.TestZoneName,
					SetDestroyProtected:    true,
					DestroyProtected:       true,
					Name:                   instanceName,
					DeleteInstanceResource: true,
				}),
				Check:       checkDestroyProtection("true"),
				ExpectError: destroyProtectionError,
			},
			{
				// test that we can remove the destroy protection
				Config: buildTestConfig(t, destroyProtectionTestData{
					Zone:                testutils.TestZoneName,
					SetDestroyProtected: true,
					DestroyProtected:    false,
					Name:                instanceName,
				}),
				Check: checkDestroyProtection("false"),
			},
			{
				// test that we can delete the instance after removing the destroy protection
				Config: buildTestConfig(t, destroyProtectionTestData{
					Zone:                   testutils.TestZoneName,
					SetDestroyProtected:    true,
					DestroyProtected:       false,
					Name:                   instanceName,
					DeleteInstanceResource: true,
				}),
				Check: checkResourceDoesNotExist("exoscale_compute_instance.my-instance"),
			},
		},
	})
}

func testDefaultDestroyProtection(t *testing.T) {
	instanceName := acctest.RandomWithPrefix(testutils.Prefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// test instance creation without the destroy_protected field
				Config: buildTestConfig(t, destroyProtectionTestData{
					Zone: testutils.TestZoneName,
					Name: instanceName,
				}),
			},

			// test updating an instance to set the destroy_protected field
			{
				Config: buildTestConfig(t, destroyProtectionTestData{
					Zone:                testutils.TestZoneName,
					SetDestroyProtected: true,
					DestroyProtected:    true,
					Name:                instanceName,
				}),
				Check: checkDestroyProtection("true"),
			},
			{
				Config: buildTestConfig(t, destroyProtectionTestData{
					Zone:                   testutils.TestZoneName,
					SetDestroyProtected:    true,
					DestroyProtected:       true,
					Name:                   instanceName,
					DeleteInstanceResource: true,
				}),
				Check:       checkDestroyProtection("true"),
				ExpectError: destroyProtectionError,
			},

			// test that removing the `destroy_protected` field removes the destroy protection
			// behaving as if false were the default value.
			{
				Config: buildTestConfig(t, destroyProtectionTestData{
					Zone:                testutils.TestZoneName,
					SetDestroyProtected: false,
					Name:                instanceName,
				}),
			},
			{
				Config: buildTestConfig(t, destroyProtectionTestData{
					Zone:                   testutils.TestZoneName,
					SetDestroyProtected:    false,
					Name:                   instanceName,
					DeleteInstanceResource: true,
				}),
			},
		},
	})
}
