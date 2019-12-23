package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceAffinityName        = testPrefix + "-" + testRandomString()
	testAccResourceAffinityDescription = testDescription

	testAccResourceAffinityConfig = fmt.Sprintf(`
resource "exoscale_affinity" "ag" {
  name = "%s"
  description = "%s"
}
`,
		testAccResourceAffinityName,
		testAccResourceAffinityDescription,
	)
)

func TestAccResourceAffinity(t *testing.T) {
	ag := new(egoscale.AffinityGroup)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceAffinityDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceAffinityConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceAffinityExists("exoscale_affinity.ag", ag),
					testAccCheckResourceAffinity(ag),
					testAccCheckResourceAffinityAttributes(testAttrs{
						"name":        ValidateString(testAccResourceAffinityName),
						"description": ValidateString(testAccResourceAffinityDescription),
						"type":        ValidateString("host anti-affinity"),
					}),
				),
			},
			{
				ResourceName:      "exoscale_affinity.ag",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"name":        ValidateString(testAccResourceAffinityName),
							"description": ValidateString(testAccResourceAffinityDescription),
							"type":        ValidateString("host anti-affinity"),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceAffinityExists(n string, ag *egoscale.AffinityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := GetComputeClient(testAccProvider.Meta())

		ag.ID = id
		resp, err := client.Get(ag)
		if err != nil {
			return err
		}

		return Copy(ag, resp.(*egoscale.AffinityGroup))
	}
}

func testAccCheckResourceAffinity(ag *egoscale.AffinityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if ag.ID == nil {
			return errors.New("Affinity Group ID is nil")
		}

		return nil
	}
}

func testAccCheckResourceAffinityAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_affinity" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceAffinityDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_affinity" {
			continue
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		ag := &egoscale.AffinityGroup{ID: id}
		if _, err = client.Get(ag); err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
	}

	return errors.New("Affinity Group still exists")
}
