package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceDomainRecord(t *testing.T) {
	domain := new(egoscale.DNSDomain)
	record := new(egoscale.DNSRecord)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDomainRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDomainRecordConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDomainExists("exoscale_domain.exo", domain),
					testAccCheckResourceDomainRecordExists("exoscale_domain_record.mx", domain, record),
					testAccCheckResourceDomainRecord(record),
					testAccCheckResourceDomainRecordAttributes(testAttrs{
						"name":        ValidateString("mail1"),
						"record_type": ValidateString("MX"),
						"content":     ValidateString("mta1"),
						"prio":        ValidateString("10"),
						"ttl":         ValidateString("10"),
					}),
				),
			},
			{
				Config: testAccResourceDomainRecordConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDomainExists("exoscale_domain.exo", domain),
					testAccCheckResourceDomainRecordExists("exoscale_domain_record.mx", domain, record),
					testAccCheckResourceDomainRecord(record),
					testAccCheckResourceDomainRecordAttributes(testAttrs{
						"name":        ValidateString("mail2"),
						"record_type": ValidateString("MX"),
						"content":     ValidateString("mta2"),
						"prio":        ValidateString("20"),
						"ttl":         ValidateString("20"),
					}),
				),
			},
			{
				ResourceName:      "exoscale_domain_record.mx",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"name":        ValidateString("mail2"),
							"record_type": ValidateString("MX"),
							"content":     ValidateString("mta2"),
							"prio":        ValidateString("20"),
							"ttl":         ValidateString("20"),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceDomainRecordExists(n string, domain *egoscale.DNSDomain, record *egoscale.DNSRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
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

func testAccCheckResourceDomainRecord(record *egoscale.DNSRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if record.TTL == 0 {
			return errors.New("TTL is zero")
		}

		return nil
	}
}

func testAccCheckResourceDomainRecordAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_domain_record" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceDomainRecordDestroy(s *terraform.State) error {
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
		return errors.New("Domain Record still exists")
	}
	return nil
}

var testAccResourceDomainRecordConfigCreate = fmt.Sprintf(`
resource "exoscale_domain" "exo" {
  name = "%s"
}

resource "exoscale_domain_record" "mx" {
  domain      = "${exoscale_domain.exo.id}"
  name        = "mail1"
  record_type = "MX"
  content     = "mta1"
  prio        = 10
  ttl         = 10
}
`,
	testDomain)

var testAccResourceDomainRecordConfigUpdate = fmt.Sprintf(`
resource "exoscale_domain" "exo" {
  name = "%s"
}

resource "exoscale_domain_record" "mx" {
  domain      = "${exoscale_domain.exo.id}"
  name        = "mail2"
  record_type = "MX"
  content     = "mta2"
  ttl         = 20
  prio        = 20
}
`,
	testDomain)
