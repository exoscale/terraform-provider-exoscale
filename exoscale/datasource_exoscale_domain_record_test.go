package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccDataSourceDomainRecordConfigCreate = fmt.Sprintf(`
	resource "exoscale_domain_record" "mx2" {
	  domain      = "${exoscale_domain.exo.id}"
	  name        = "%s"
	  record_type = "%s"
	  content     = "%s"
	  prio        = %d
	  ttl         = %d
	}
	`,
		testAccResourceDomainRecordNameUpdated,
		testAccResourceDomainRecordType,
		testAccResourceDomainRecordContentUpdated,
		testAccResourceDomainRecordPrio,
		testAccResourceDomainRecordTTL,
	)
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
}`, testAccResourceDomainRecordConfigCreate, testAccDataSourceDomainRecordConfigCreate),
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
}`, testAccResourceDomainRecordConfigCreate, testAccDataSourceDomainRecordConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.name":   ValidateString("mail1"),
							"records.0.domain": ValidateString(testAccResourceDomainName),
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
			}`, testAccResourceDomainRecordConfigCreate, testAccDataSourceDomainRecordConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.name":   ValidateString("mail1"),
							"records.0.domain": ValidateString(testAccResourceDomainName),
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
			}`, testAccResourceDomainRecordConfigCreate, testAccDataSourceDomainRecordConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.domain": ValidateString(testAccResourceDomainName),
							"records.1.domain": ValidateString(testAccResourceDomainName),
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
			    content = "mta.*"
			  }
			}`, testAccResourceDomainRecordConfigCreate, testAccDataSourceDomainRecordConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.domain": ValidateString(testAccResourceDomainName),
							"records.1.domain": ValidateString(testAccResourceDomainName),
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
