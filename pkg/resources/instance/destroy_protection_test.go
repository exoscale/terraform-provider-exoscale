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

{{ if .DeleteInstanceResource }}
resource "exoscale_compute_instance" "my_instance" {
  zone = "{{.Zone}}"
  name = "{{.Name}}"

  template_id = data.exoscale_template.my_template.id
  type        = "standard.micro"
  disk_size   = 10

  destroy_protected = {{.DestroyProtected}}
}
{{ end }}
`

func testDestroyProtection(t *testing.T) {
	tmpl := template.Must(template.New("compute_instance").Parse(computeInstanceResource))

	type TestData struct {
		Zone                   string
		DestroyProtected       bool
		Name                   string
		DeleteInstanceResource bool
	}

	buildTestConfig := func(testData TestData) string {
		var tmplBuf bytes.Buffer

		err := tmpl.Execute(&tmplBuf, testData)
		if err != nil {
			t.Fatal(err)
		}

		return tmplBuf.String()
	}

	instanceName := acctest.RandomWithPrefix(testutils.Prefix)

	checkDestroyProtection := func(expected string) func(s *terraform.State) error {
		return func(s *terraform.State) error {
			isDestroyProtected, err := testutils.AttrFromState(s, "resource.exoscale_compute_instance.my-instance", "destroy_protected")
			if err != nil {
				return err
			}

			if expected != isDestroyProtected {
				return fmt.Errorf("destroy_protected does not match expected value: %q; is %q", expected, isDestroyProtected)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		Steps: []resource.TestStep{
			{
				Config: buildTestConfig(TestData{
					Zone:             testutils.TestZoneName,
					DestroyProtected: true,
					Name:             instanceName,
				}),
				Check: checkDestroyProtection("true"),
			},
			{
				Config: buildTestConfig(TestData{
					Zone:                   testutils.TestZoneName,
					DestroyProtected:       false,
					Name:                   instanceName,
					DeleteInstanceResource: true,
				}),
				Check:       checkDestroyProtection("true"),
				ExpectError: regexp.MustCompile(`invalid request: Operation delete-instance on resource .* is forbidden - reason: manual instance protection`),
			},
			{
				Config: buildTestConfig(TestData{
					Zone:             testutils.TestZoneName,
					DestroyProtected: false,
					Name:             instanceName,
				}),
				Check: checkDestroyProtection("false"),
			},
			{
				Config: buildTestConfig(TestData{
					Zone:                   testutils.TestZoneName,
					DestroyProtected:       false,
					Name:                   instanceName,
					DeleteInstanceResource: true,
				}),
				Check: checkDestroyProtection("false"),
			},
		},
	})
}
