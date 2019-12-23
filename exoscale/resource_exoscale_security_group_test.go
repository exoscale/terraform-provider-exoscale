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
	testAccResourceSecurityGroupName        = testPrefix + "-" + testRandomString()
	testAccResourceSecurityGroupDescription = testDescription

	testAccResourceSecurityGroupConfig = fmt.Sprintf(`
resource "exoscale_security_group" "sg" {
  name = "%s"
  description = "%s"
}
`,
		testAccResourceSecurityGroupName,
		testAccResourceSecurityGroupDescription)
)

func TestAccResourceSecurityGroup(t *testing.T) {
	sg := new(egoscale.SecurityGroup)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSecurityGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.sg", sg),
					testAccCheckResourceSecurityGroup(sg),
					testAccCheckResourceSecurityGroupAttributes(testAttrs{
						"name":        ValidateString(testAccResourceSecurityGroupName),
						"description": ValidateString(testAccResourceSecurityGroupDescription),
					}),
				),
			},
			{
				ResourceName:      "exoscale_security_group.sg",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"name":        ValidateString(testAccResourceSecurityGroupName),
							"description": ValidateString(testAccResourceSecurityGroupDescription),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSecurityGroupExists(n string, sg *egoscale.SecurityGroup) resource.TestCheckFunc {
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
		resp, err := client.Get(&egoscale.SecurityGroup{
			ID: id,
		})
		if err != nil {
			return err
		}

		return Copy(sg, resp.(*egoscale.SecurityGroup))
	}
}

func testAccCheckResourceSecurityGroup(sg *egoscale.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if sg.ID == nil {
			return errors.New("Security Group ID is nil")
		}

		return nil
	}
}

func testAccCheckResourceSecurityGroupAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_security_group" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceSecurityGroupDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_security_group" {
			continue
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		key := &egoscale.SecurityGroup{ID: id}
		_, err = client.Get(key)
		if err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
	}
	return errors.New("Security Group still exists")
}
