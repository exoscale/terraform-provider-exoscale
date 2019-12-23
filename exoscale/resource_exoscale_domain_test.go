package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceDomainName = testPrefix + "-" + testRandomString() + ".net"

	testAccDNSDomainCreate = fmt.Sprintf(`
resource "exoscale_domain" "exo" {
  name = "%s"
}
`,
		testAccResourceDomainName,
	)
)

func TestAccResourceDomain(t *testing.T) {
	domain := new(egoscale.DNSDomain)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSDomainCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDomainExists("exoscale_domain.exo", domain),
					testAccCheckResourceDomain(domain),
					testAccCheckResourceDomainAttributes(testAttrs{
						"name":       ValidateString(testAccResourceDomainName),
						"state":      ValidateString("hosted"),
						"auto_renew": ValidateString("false"),
						"expires_on": ValidateString(""),
						"token":      ValidateRegexp("^[0-9a-f]+$"),
					}),
				),
			},
			{
				ResourceName:      "exoscale_domain.exo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"name":       ValidateString(testAccResourceDomainName),
							"state":      ValidateString("hosted"),
							"auto_renew": ValidateString("false"),
							"expires_on": ValidateString(""),
							"token":      ValidateRegexp("^[0-9a-f]+$"),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceDomainExists(n string, domain *egoscale.DNSDomain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetDNSClient(testAccProvider.Meta())
		d, err := client.GetDomain(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		*domain = *d

		return nil
	}
}

func testAccCheckResourceDomain(domain *egoscale.DNSDomain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(domain.Token) != 32 {
			return fmt.Errorf("expected token length %d, got %d", 32, len(domain.Token))
		}

		return nil
	}
}

func testAccCheckResourceDomainAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_domain" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceDomainDestroy(s *terraform.State) error {
	client := GetDNSClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_domain" {
			continue
		}

		d, err := client.GetDomain(context.TODO(), rs.Primary.Attributes["name"])
		if err != nil {
			if _, ok := err.(*egoscale.DNSErrorResponse); ok {
				return nil
			}
			return err
		}
		if d == nil {
			return nil
		}
		return errors.New("Domain still exists")
	}
	return nil
}
