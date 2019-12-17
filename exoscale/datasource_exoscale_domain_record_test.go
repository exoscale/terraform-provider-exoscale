package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

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

%s

data "exoscale_domain_record" "test_record" {
  domain = "${exoscale_domain.exo.id}"
  filter {}
}`, testAccResourceDomainRecordConfigCreate, testAccResourceDomainRecordConfigCreate2),
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
%s

%s

data "exoscale_domain_record" "test_record" {
  domain = "${exoscale_domain.exo.id}"
  filter {
    name   = "${exoscale_domain_record.mx.name}"
  }
}`, testAccResourceDomainRecordConfigCreate, testAccResourceDomainRecordConfigCreate2),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.name":   ValidateString("mail1"),
							"records.0.domain": ValidateString(testDomain),
						},
					),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

%s

			data "exoscale_domain_record" "test_record" {
			  domain = "${exoscale_domain.exo.id}"
			  filter {
			    id = "${exoscale_domain_record.mx.id}"
			  }
			}`, testAccResourceDomainRecordConfigCreate, testAccResourceDomainRecordConfigCreate2),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.name":   ValidateString("mail1"),
							"records.0.domain": ValidateString(testDomain),
						},
					),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

%s

			data "exoscale_domain_record" "test_record" {
			  domain = "${exoscale_domain.exo.id}"
			  filter {
			    record_type = "MX"
			  }
			}`, testAccResourceDomainRecordConfigCreate, testAccResourceDomainRecordConfigCreate2),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.domain": ValidateString(testDomain),
							"records.1.domain": ValidateString(testDomain),
						},
					),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

%s

			data "exoscale_domain_record" "test_record" {
			  domain = "${exoscale_domain.exo.id}"
			  filter {
			    content = "mta*"
			  }
			}`, testAccResourceDomainRecordConfigCreate, testAccResourceDomainRecordConfigCreate2),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.domain": ValidateString(testDomain),
							"records.1.domain": ValidateString(testDomain),
						},
					),
				),
			},
		},
	})
}

func testAccDatasourceDomainRecordAttributes(rsName string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		all := s.RootModule().Resources
		rs, ok := all[rsName]
		if !ok {
			return errors.New("exoscale_domain_record datasource not found in the state")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Snapshot records source ID not set")
		}

		return checkResourceAttributes(expected, rs.Primary.Attributes)
	}

}

var testAccResourceDomainRecordConfigCreate2 = `
resource "exoscale_domain_record" "mx2" {
  domain      = "${exoscale_domain.exo.id}"
  name        = "mail2"
  record_type = "MX"
  content     = "mta2"
  prio        = 10
  ttl         = 10
}
`
