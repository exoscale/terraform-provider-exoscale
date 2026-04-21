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

func TestDomainNameToUnicode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "example.com"},
		{"xn--n3h.ws", "☃.ws"},
		{"xn--domain-with--rcb.ch", "domain-with-ä.ch"},
		{"already-unicodeä.com", "already-unicodeä.com"},
		{"", ""},
	}

	for _, tt := range tests {
		got := domainNameToUnicode(tt.input)
		if got != tt.want {
			t.Errorf("domainNameToUnicode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDomainNameDiffSuppress(t *testing.T) {
	t.Parallel()

	// grab the live func from the schema so we're testing the real wiring
	suppress := resourceDomain().Schema["name"].DiffSuppressFunc

	tests := []struct {
		old, new string
		want     bool
	}{
		// same in both forms: should suppress
		{"xn--n3h.ws", "☃.ws", true},
		{"☃.ws", "xn--n3h.ws", true},
		{"example.com", "example.com", true},
		// different domains: must not suppress
		{"example.com", "other.com", false},
		{"xn--n3h.ws", "example.com", false},
	}

	for _, tt := range tests {
		got := suppress("name", tt.old, tt.new, nil)
		if got != tt.want {
			t.Errorf("DiffSuppressFunc(%q, %q) = %v, want %v", tt.old, tt.new, got, tt.want)
		}
	}
}

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
	t.Parallel()

	domain := v3.DNSDomain{}

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

func testAccCheckResourceDomainExists(n string, domain *v3.DNSDomain) resource.TestCheckFunc {
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
		d, err := client.GetDNSDomain(context.TODO(), v3.UUID(rs.Primary.ID))
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
	client, err := APIClientV3()
	if err != nil {
		return fmt.Errorf("unable to initialize Exoscale client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_domain" {
			continue
		}

		d, err := client.GetDNSDomain(context.TODO(), v3.UUID(rs.Primary.Attributes["id"]))
		if err != nil {
			if errors.Is(err, v3.ErrNotFound) {
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
