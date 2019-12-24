package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceDomainRecordDomainName     = testPrefix + "-" + testRandomString() + ".net"
	testAccResourceDomainRecordName           = "mail1"
	testAccResourceDomainRecordNameUpdated    = "mail2"
	testAccResourceDomainRecordType           = "MX"
	testAccResourceDomainRecordContent        = "mta1"
	testAccResourceDomainRecordContentUpdated = "mta2"
	testAccResourceDomainRecordPrio           = 10
	testAccResourceDomainRecordPrioUpdated    = 20
	testAccResourceDomainRecordTTL            = 10
	testAccResourceDomainRecordTTLUpdated     = 20

	testAccResourceDomainRecordConfigCreate = fmt.Sprintf(`
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
		testAccResourceDomainRecordDomainName,
		testAccResourceDomainRecordName,
		testAccResourceDomainRecordType,
		testAccResourceDomainRecordContent,
		testAccResourceDomainRecordPrio,
		testAccResourceDomainRecordTTL,
	)

	testAccResourceDomainRecordConfigUpdate = fmt.Sprintf(`
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
		testAccResourceDomainRecordDomainName,
		testAccResourceDomainRecordNameUpdated,
		testAccResourceDomainRecordType,
		testAccResourceDomainRecordContentUpdated,
		testAccResourceDomainRecordPrioUpdated,
		testAccResourceDomainRecordTTLUpdated,
	)
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
						"name":        ValidateString(testAccResourceDomainRecordName),
						"record_type": ValidateString(testAccResourceDomainRecordType),
						"content":     ValidateString(testAccResourceDomainRecordContent),
						"prio":        ValidateString(fmt.Sprint(testAccResourceDomainRecordPrio)),
						"ttl":         ValidateString(fmt.Sprint(testAccResourceDomainRecordTTL)),
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
						"name":        ValidateString(testAccResourceDomainRecordNameUpdated),
						"record_type": ValidateString(testAccResourceDomainRecordType),
						"content":     ValidateString(testAccResourceDomainRecordContentUpdated),
						"prio":        ValidateString(fmt.Sprint(testAccResourceDomainRecordPrioUpdated)),
						"ttl":         ValidateString(fmt.Sprint(testAccResourceDomainRecordTTLUpdated)),
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
							"name":        ValidateString(testAccResourceDomainRecordNameUpdated),
							"record_type": ValidateString(testAccResourceDomainRecordType),
							"content":     ValidateString(testAccResourceDomainRecordContentUpdated),
							"prio":        ValidateString(fmt.Sprint(testAccResourceDomainRecordPrioUpdated)),
							"ttl":         ValidateString(fmt.Sprint(testAccResourceDomainRecordTTLUpdated)),
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
