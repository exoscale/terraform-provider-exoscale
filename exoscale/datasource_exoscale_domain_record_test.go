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
	testAccDataSourceDomainRecordDomainName = testPrefix + "-" + testRandomString() + ".net"
	testAccDataSourceDomainRecordName1      = "mail1"
	testAccDataSourceDomainRecordName2      = "mail2"
	testAccDataSourceDomainRecordType       = "MX"
	testAccDataSourceDomainRecordContent1   = "mta1"
	testAccDataSourceDomainRecordContent2   = "mta2"
	testAccDataSourceDomainRecordPrio       = 10
	testAccDataSourceDomainRecordTTL        = 10

	testAccDataSourceDomainRecordConfigCreate1 = fmt.Sprintf(`
resource "exoscale_domain" "exo" {
  name = "%s"
}

resource "exoscale_domain_record" "mx" {
  domain      = exoscale_domain.exo.id
  name        = "%s"
  record_type = "%s"
  content     = "%s"
  prio        = %d
  ttl         = %d
}
`,
		testAccDataSourceDomainRecordDomainName,
		testAccDataSourceDomainRecordName1,
		testAccDataSourceDomainRecordType,
		testAccDataSourceDomainRecordContent1,
		testAccDataSourceDomainRecordPrio,
		testAccDataSourceDomainRecordTTL,
	)

	testAccDataSourceDomainRecordConfigCreate2 = fmt.Sprintf(`
resource "exoscale_domain_record" "mx2" {
  domain      = exoscale_domain.exo.id
  name        = "%s"
  record_type = "%s"
  content     = "%s"
  prio        = %d
  ttl         = %d
}
`,
		testAccDataSourceDomainRecordName2,
		testAccDataSourceDomainRecordType,
		testAccDataSourceDomainRecordContent2,
		testAccDataSourceDomainRecordPrio,
		testAccDataSourceDomainRecordTTL,
	)
)

func TestAccDataSourceDomainRecord(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

%s

data "exoscale_domain_record" "test_record" {
  domain = exoscale_domain.exo.id
  filter {}
}`, testAccDataSourceDomainRecordConfigCreate1, testAccDataSourceDomainRecordConfigCreate2),
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
%s

%s

data "exoscale_domain_record" "test_record" {
  domain = exoscale_domain.exo.id
  filter {
    name   = exoscale_domain_record.mx.name
  }
}`, testAccDataSourceDomainRecordConfigCreate1, testAccDataSourceDomainRecordConfigCreate2),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.name":   ValidateString(testAccDataSourceDomainRecordName1),
							"records.0.domain": ValidateString(testAccDataSourceDomainRecordDomainName),
						},
					),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

%s

			data "exoscale_domain_record" "test_record" {
			  domain = exoscale_domain.exo.id
			  filter {
			    id = exoscale_domain_record.mx.id
			  }
			}`, testAccDataSourceDomainRecordConfigCreate1, testAccDataSourceDomainRecordConfigCreate2),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.name":   ValidateString(testAccDataSourceDomainRecordName1),
							"records.0.domain": ValidateString(testAccDataSourceDomainRecordDomainName),
						},
					),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

%s

			data "exoscale_domain_record" "test_record" {
			  domain = exoscale_domain.exo.id
			  filter {
			    record_type = "MX"
			  }
			}`, testAccDataSourceDomainRecordConfigCreate1, testAccDataSourceDomainRecordConfigCreate2),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.domain": ValidateString(testAccDataSourceDomainRecordDomainName),
							"records.1.domain": ValidateString(testAccDataSourceDomainRecordDomainName),
						},
					),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

%s

			data "exoscale_domain_record" "test_record" {
			  domain = exoscale_domain.exo.id
			  filter {
			    content_regex = "mta.*"
			  }
			}`, testAccDataSourceDomainRecordConfigCreate1, testAccDataSourceDomainRecordConfigCreate2),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDomainRecordAttributes(
						"data.exoscale_domain_record.test_record",
						testAttrs{
							"records.0.domain": ValidateString(testAccDataSourceDomainRecordDomainName),
							"records.1.domain": ValidateString(testAccDataSourceDomainRecordDomainName),
						},
					),
				),
			},
		},
	})
}

func testAccDataSourceDomainRecordAttributes(rsName string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		all := s.RootModule().Resources
		rs, ok := all[rsName]
		if !ok {
			return errors.New("exoscale_domain_record data source not found in the state")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Snapshot records source ID not set")
		}

		return checkResourceAttributes(expected, rs.Primary.Attributes)
	}

}
