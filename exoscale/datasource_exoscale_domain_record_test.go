package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccDatasourceDomainRecord(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_domain_record" "test_record" {
  domain = "${exoscale_domain.exo.id}"
  filter {}
}`, testAccResourceDomainRecordConfigCreate),
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_domain_record" "test_record" {
  domain = "${exoscale_domain.exo.id}"
  filter {
    name   = "${exoscale_domain_record.mx.name}"
  }
}`, testAccResourceDomainRecordConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(testAttrs{
						"name":   ValidateString("mail1"),
						"domain": ValidateString(testDomain),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_domain_record" "test_record" {
  domain = "${exoscale_domain.exo.id}"
  filter {
    id = "${exoscale_domain_record.mx.id}"
  }
}`, testAccResourceDomainRecordConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(testAttrs{
						"name":   ValidateString("mail1"),
						"domain": ValidateString(testDomain),
					}),
				),
			},
		},
	})
}

func testAccDatasourceDomainRecordAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_domain_record" {
				continue
			}

			if rs.Primary.ID == "" {
				return fmt.Errorf("Snapshot records source ID not set")
			}

			println("test:")
			spew.Dump(rs.Primary.Attributes["records.#"])

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("exoscale_domain_record datasource not found in the state")
	}
}
