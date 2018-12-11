package exoscale

import (
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccSecurityGroup(t *testing.T) {
	sg := new(egoscale.SecurityGroup)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityGroupExists("exoscale_security_group.sg", sg),
					testAccCheckSecurityGroupAttributes(sg),
					testAccCheckSecurityGroupCreateAttributes("terraform-test-security-group"),
				),
			},
			{
				Config: testAccSecurityGroupUpdateTags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityGroupExists("exoscale_security_group.sg", sg),
					testAccCheckSecurityGroupAttributes(sg),
					testAccCheckSecurityGroupCreateAttributes("terraform-test-security-group"),
				),
			},
		},
	})
}

func testAccCheckSecurityGroupExists(n string, sg *egoscale.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group ID is set")
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

func testAccCheckSecurityGroupAttributes(sg *egoscale.SecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if sg.ID == nil {
			return fmt.Errorf("security group is nil")
		}

		return nil
	}
}

func testAccCheckSecurityGroupCreateAttributes(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_security_group" {
				continue
			}

			if rs.Primary.Attributes["name"] != name {
				continue
			}

			if rs.Primary.Attributes["description"] == "" {
				return fmt.Errorf("Security Groups: expected description to be set")
			}

			return nil
		}

		return fmt.Errorf("Could not find security group name: %s", name)
	}
}

func testAccCheckSecurityGroupDestroy(s *terraform.State) error {
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
	return fmt.Errorf("SecurityGroup: still exists")
}

var testAccSecurityGroupCreate = `
resource "exoscale_security_group" "sg" {
  name = "terraform-test-security-group"
  description = "Terraform Security Group Test"
}
`

var testAccSecurityGroupUpdateTags = `
resource "exoscale_security_group" "sg" {
  name = "terraform-test-security-group"
  description = "Terraform Security Group Test"
}
`
