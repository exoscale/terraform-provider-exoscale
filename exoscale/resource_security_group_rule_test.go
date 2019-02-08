package exoscale

import (
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccSecurityGroupRule(t *testing.T) {
	sg := new(egoscale.SecurityGroup)
	cidr := new(egoscale.EgressRule)
	usg := new(egoscale.IngressRule)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSecurityGroupRuleCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecurityGroupExists("exoscale_security_group.sg", sg),
					testAccCheckEgressRuleExists("exoscale_security_group_rule.cidr", sg, cidr),
					testAccCheckIngressRuleExists("exoscale_security_group_rule.usg", sg, usg),
					testAccCheckSecurityGroupRuleAttributes(cidr),
					testAccCheckSecurityGroupRuleAttributes((*egoscale.EgressRule)(usg)),
					testAccCheckSecurityGroupRuleCreateAttributes("EGRESS", "TCP"),
					testAccCheckSecurityGroupRuleCreateAttributes("INGRESS", "ICMPv6"),
				),
			},
		},
	})
}

func testAccCheckEgressRuleExists(n string, sg *egoscale.SecurityGroup, rule *egoscale.EgressRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Security Group Rule ID is set")
		}

		if len(sg.EgressRule) == 0 {
			return fmt.Errorf("no egress rules found")
		}

		return Copy(rule, sg.EgressRule[0])
	}
}

func testAccCheckIngressRuleExists(n string, sg *egoscale.SecurityGroup, rule *egoscale.IngressRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Security Group Rule ID is set")
		}

		if len(sg.IngressRule) == 0 {
			return fmt.Errorf("no Ingress rules found")
		}

		return Copy(rule, sg.IngressRule[0])
	}
}
func testAccCheckSecurityGroupRuleAttributes(r *egoscale.EgressRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if r.RuleID == nil {
			return fmt.Errorf("security group rule id is nil")
		}

		return nil
	}
}

func testAccCheckSecurityGroupRuleCreateAttributes(typ, protocol string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_security_group_rule" {
				continue
			}

			if rs.Primary.Attributes["type"] != typ {
				continue
			}

			p := rs.Primary.Attributes["protocol"]
			if p != protocol {
				return fmt.Errorf("Security Groups Rule: bad protocol wanted %s, got %s", protocol, p)
			}

			return nil
		}

		return fmt.Errorf("Could not find security group rule: %s", typ)
	}
}

func testAccCheckSecurityGroupRuleDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_security_group_rule" {
			continue
		}

		sgID, err := egoscale.ParseUUID(rs.Primary.Attributes["security_group_id"])
		if err != nil {
			return err
		}

		sg := &egoscale.SecurityGroup{ID: sgID}
		_, err = client.Get(sg)
		if err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
	}

	return fmt.Errorf("security group rule still exists")
}

var testAccSecurityGroupRuleCreate = `
resource "exoscale_security_group" "sg" {
  name = "terraform-test-security-group"
  description = "Terraform Security Group Test"
}

resource "exoscale_security_group_rule" "cidr" {
  security_group_id = "${exoscale_security_group.sg.id}"
  protocol = "TCP"
  type = "EGRESS"
  cidr = "::/0"
  start_port = 2
  end_port = 1024
}

resource "exoscale_security_group_rule" "usg" {
  security_group = "${exoscale_security_group.sg.name}"
  protocol = "ICMPv6"
  type = "INGRESS"
  icmp_type = 128
  icmp_code = 0
  user_security_group = "${exoscale_security_group.sg.name}"
}
`
