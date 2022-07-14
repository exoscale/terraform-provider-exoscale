package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	exo "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccResourceDomainName = acctest.RandomWithPrefix(testPrefix) + ".net"

	testAccDNSDomainCreate = fmt.Sprintf(`
resource "exoscale_domain" "exo" {
  name = "%s"
}
`,
		testAccResourceDomainName,
	)
)

func TestAccResourceDomain(t *testing.T) {
	domain := exo.DNSDomain{}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceDomainDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSDomainCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDomainExists("exoscale_domain.exo", &domain),
					testAccCheckResourceDomainAttributes(testAttrs{
						"name": validateString(testAccResourceDomainName),
					}),
					testAccCheckResourceDomainStateUpgradeV1("exoscale_domain.exo"),
				),
			},
			{
				ResourceName:      "exoscale_domain.exo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"name": validateString(testAccResourceDomainName),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceDomainExists(n string, domain *exo.DNSDomain) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())
		d, err := client.GetDNSDomain(context.TODO(), defaultZone, rs.Primary.ID)
		if err != nil {
			return err
		}

		*domain = *d

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
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_domain" {
			continue
		}

		d, err := client.GetDNSDomain(context.TODO(), defaultZone, rs.Primary.Attributes["id"])
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
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

func testAccCheckResourceDomainStateUpgradeV1(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		upgraded, err := resourceDomainStateUpgradeV0(
			context.TODO(),
			map[string]interface{}{
				"id":   testAccResourceDomainName,
				"name": testAccResourceDomainName,
			},
			testAccProvider.Meta(),
		)
		if err != nil {
			return fmt.Errorf("error migrating state: %s", err)
		}

		if upgraded["id"].(string) != rs.Primary.ID {
			return fmt.Errorf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", upgraded["id"].(string), rs.Primary.ID)
		}

		return nil
	}
}
