package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var (
	testAccResourceDomainRecordDomainName           = acctest.RandomWithPrefix(testPrefix) + ".net"
	testAccResourceDomainRecordName                 = "mail1"
	testAccResourceDomainRecordNameUpdated          = "mail2"
	testAccResourceDomainRecordType                 = "MX"
	testAccResourceDomainRecordContent              = "mta1." + testAccResourceDomainRecordDomainName
	testAccResourceDomainRecordContentUpdated       = "mta2." + testAccResourceDomainRecordDomainName
	testAccResourceDomainRecordTXTContent           = "test value for TXT record"
	testAccResourceDomainRecordTXTContentNormalized = "\"test value for TXT record\""
	testAccResourceDomainRecordPrio                 = 10
	testAccResourceDomainRecordPrioUpdated          = 20
	testAccResourceDomainRecordTTL                  = 10
	testAccResourceDomainRecordTTLUpdated           = 20

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

resource "exoscale_domain_record" "a" {
  domain      = exoscale_domain.exo.id
  name        = ""
  record_type = "A"
  content     = "1.2.3.4"
}

resource "exoscale_domain_record" "txt" {
  domain      = exoscale_domain.exo.id
  name        = "test"
  record_type = "TXT"
  content     = "%s"
}
`,
		testAccResourceDomainRecordDomainName,
		testAccResourceDomainRecordName,
		testAccResourceDomainRecordType,
		testAccResourceDomainRecordContent,
		testAccResourceDomainRecordPrio,
		testAccResourceDomainRecordTTL,
		testAccResourceDomainRecordTXTContent,
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

resource "exoscale_domain_record" "a" {
  domain      = exoscale_domain.exo.id
  name        = ""
  record_type = "A"
  content     = "1.2.3.4"
}

resource "exoscale_domain_record" "txt" {
  domain      = exoscale_domain.exo.id
  name        = "test"
  record_type = "TXT"
  content     = "\"test value for TXT record\""
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
	dr := v3.DNSDomainRecord{}
	domain := v3.DNSDomain{}
	record := v3.DNSDomainRecord{}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceDomainRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDomainRecordConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDomainExists("exoscale_domain.exo", &domain),
					testAccCheckResourceDomainRecordExists("exoscale_domain_record.a", &domain, &dr),
					testAccCheckResourceDomainRecordExists("exoscale_domain_record.mx", &domain, &record),
					testAccCheckResourceDomainRecord(&record),
					testAccCheckResourceDomainRecordAttributes("exoscale_domain_record.mx", testAttrs{
						"name":        validateString(testAccResourceDomainRecordName),
						"record_type": validateString(testAccResourceDomainRecordType),
						"content":     validateString(testAccResourceDomainRecordContent),
						"prio":        validateString(fmt.Sprint(testAccResourceDomainRecordPrio)),
						"ttl":         validateString(fmt.Sprint(testAccResourceDomainRecordTTL)),
					}),
					testAccCheckResourceDomainRecordAttributes("exoscale_domain_record.txt", testAttrs{
						"content":            validateString(testAccResourceDomainRecordTXTContent),
						"content_normalized": validateString(testAccResourceDomainRecordTXTContentNormalized),
					}),
					testAccCheckResourceDomainRecordStateUpgradeV1("exoscale_domain.exo", "exoscale_domain_record.mx"),
				),
			},
			{
				Config: testAccResourceDomainRecordConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDomainExists("exoscale_domain.exo", &domain),
					testAccCheckResourceDomainRecordExists("exoscale_domain_record.mx", &domain, &record),
					testAccCheckResourceDomainRecord(&record),
					testAccCheckResourceDomainRecordAttributes("exoscale_domain_record.mx", testAttrs{
						"name":        validateString(testAccResourceDomainRecordNameUpdated),
						"record_type": validateString(testAccResourceDomainRecordType),
						"content":     validateString(testAccResourceDomainRecordContentUpdated),
						"prio":        validateString(fmt.Sprint(testAccResourceDomainRecordPrioUpdated)),
						"ttl":         validateString(fmt.Sprint(testAccResourceDomainRecordTTLUpdated)),
					}),
					testAccCheckResourceDomainRecordAttributes("exoscale_domain_record.txt", testAttrs{
						"content":            validateString(testAccResourceDomainRecordTXTContentNormalized),
						"content_normalized": validateString(testAccResourceDomainRecordTXTContentNormalized),
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
							"name":        validateString(testAccResourceDomainRecordNameUpdated),
							"record_type": validateString(testAccResourceDomainRecordType),
							"content":     validateString(testAccResourceDomainRecordContentUpdated),
							"prio":        validateString(fmt.Sprint(testAccResourceDomainRecordPrioUpdated)),
							"ttl":         validateString(fmt.Sprint(testAccResourceDomainRecordTTLUpdated)),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceDomainRecordExists(n string, domain *v3.DNSDomain, record *v3.DNSDomainRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client, err := APIClientV3()
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}
		r, err := client.GetDNSDomainRecord(context.TODO(), domain.ID, v3.UUID(rs.Primary.ID))
		if err != nil {
			return err
		}

		*record = *r

		return nil
	}
}

func testAccCheckResourceDomainRecord(record *v3.DNSDomainRecord) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if record.Ttl == 0 {
			return errors.New("TTL is zero")
		}

		return nil
	}
}

func testAccCheckResourceDomainRecordAttributes(n string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		return checkResourceAttributes(expected, rs.Primary.Attributes)
	}
}

func testAccCheckResourceDomainRecordDestroy(s *terraform.State) error {
	client, err := APIClientV3()
	if err != nil {
		return fmt.Errorf("unable to initialize Exoscale client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_domain_record" {
			continue
		}

		d, err := client.GetDNSDomainRecord(context.TODO(), v3.UUID(rs.Primary.Attributes["id"]), v3.UUID(rs.Primary.ID))
		if err != nil {
			if errors.Is(err, v3.ErrNotFound) {
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

func testAccCheckResourceDomainRecordStateUpgradeV1(nd, nr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rsd, ok := s.RootModule().Resources[nd]
		if !ok {
			return errors.New("resource not found in the state")
		}
		rsr, ok := s.RootModule().Resources[nr]
		if !ok {
			return errors.New("resource not found in the state")
		}

		upgraded, err := resourceDomainRecordStateUpgradeV0(
			context.TODO(),
			map[string]interface{}{
				"id":          "123456",
				"domain":      testAccResourceDomainRecordDomainName,
				"record_type": testAccResourceDomainRecordType,
				"name":        testAccResourceDomainRecordName,
				"content":     testAccResourceDomainRecordContent,
			},
			testAccProvider.Meta(),
		)
		if err != nil {
			return fmt.Errorf("error migrating state: %s", err)
		}

		if upgraded["domain"].(string) != rsd.Primary.ID {
			return fmt.Errorf("state migrate: expected domain:%q, got:%q", upgraded["domain"].(string), rsd.Primary.ID)
		}
		if upgraded["id"].(string) != rsr.Primary.ID {
			return fmt.Errorf("state migrate: expected id:%q, got:%q", upgraded["id"].(string), rsr.Primary.ID)
		}

		return nil
	}
}
