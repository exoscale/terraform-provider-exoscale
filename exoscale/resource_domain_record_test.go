package exoscale

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDomainRecord(t *testing.T) {
	domain := new(egoscale.DNSDomain)
	record := new(egoscale.DNSRecord)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDNSRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDNSRecordCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDNSDomainExists("exoscale_domain.exo", domain),
					testAccCheckDNSRecordExists("exoscale_domain_record.www", domain, record),
					testAccCheckDNSRecordAttributes(record),
					testAccCheckDNSRecordCreateAttributes("www", "1.2.3.4"),
				),
			},
		},
	})
}

func testAccCheckDNSRecordExists(n string, domain *egoscale.DNSDomain, record *egoscale.DNSRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No domain ID is set")
		}

		id, _ := strconv.ParseInt(rs.Primary.ID, 10, 64)

		client := GetDNSClient(testAccProvider.Meta())
		r, err := client.GetRecord(context.TODO(), domain.Name, id)
		if err != nil {
			return err
		}

		*record = *r

		return nil
	}
}

func testAccCheckDNSRecordAttributes(record *egoscale.DNSRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if record.TTL == 0 {
			return fmt.Errorf("DNS Domain Record: ttl is zero")
		}

		return nil
	}
}

func testAccCheckDNSRecordCreateAttributes(name string, content string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_domain_record" {
				continue
			}

			if rs.Primary.Attributes["name"] != name {
				continue
			}

			if rs.Primary.Attributes["content"] != content {
				return fmt.Errorf("DNS DomainRecord: bad content, want %s", content)
			}

			return nil
		}

		return fmt.Errorf("Could not find domain record %s", name)
	}
}

func testAccCheckDNSRecordDestroy(s *terraform.State) error {
	client := GetDNSClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_domain_record" {
			continue
		}

		id, _ := strconv.ParseInt(rs.Primary.ID, 10, 64)
		d, err := client.GetRecord(context.TODO(), rs.Primary.Attributes["domain"], id)
		if err != nil {
			if _, ok := err.(*egoscale.DNSErrorResponse); ok {
				return nil
			}
			return err
		}
		if d == nil {
			return nil
		}
		return fmt.Errorf("DNS DomainRecord: still exists")
	}
	return nil
}

var testAccDNSRecordCreate = `
resource "exoscale_domain" "exo" {
  name = "acceptance.exo"
}

resource "exoscale_domain_record" "www" {
  domain = "${exoscale_domain.exo.id}"
  name = "www"
  record_type = "A"
  content = "1.2.3.4"
}
`
