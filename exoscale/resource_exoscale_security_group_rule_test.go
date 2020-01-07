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
	testAccResourceSecurityGroupRuleSecurityGroupName = testPrefix + "-" + testRandomString()
	testAccResourceSecurityGroupRuleWithCIDRProtocol  = "TCP"
	testAccResourceSecurityGroupRuleWithCIDRType      = "EGRESS"
	testAccResourceSecurityGroupRuleWithCIDRCIDR      = "::/0"
	testAccResourceSecurityGroupRuleWithCIDRStartPort = 2
	testAccResourceSecurityGroupRuleWithCIDREndPort   = 1024
	testAccResourceSecurityGroupRuleWithUSGProtocol   = "ICMPv6"
	testAccResourceSecurityGroupRuleWithUSGType       = "INGRESS"
	testAccResourceSecurityGroupRuleWithUSGICMPType   = 128
	testAccResourceSecurityGroupRuleWithUSGICMPCode   = 0

	testAccResourceSecurityGroupRuleConfigWithCIDR = fmt.Sprintf(`
resource "exoscale_security_group" "sg" {
  name = "%s"
}

resource "exoscale_security_group_rule" "cidr" {
  security_group_id = exoscale_security_group.sg.id
  protocol = "%s"
  type = "%s"
  cidr = "%s"
  start_port = %d
  end_port = %d
}
`,
		testAccResourceSecurityGroupRuleSecurityGroupName,
		testAccResourceSecurityGroupRuleWithCIDRProtocol,
		testAccResourceSecurityGroupRuleWithCIDRType,
		testAccResourceSecurityGroupRuleWithCIDRCIDR,
		testAccResourceSecurityGroupRuleWithCIDRStartPort,
		testAccResourceSecurityGroupRuleWithCIDREndPort,
	)

	testAccResourceSecurityGroupRuleConfigWithUSG = fmt.Sprintf(`
resource "exoscale_security_group" "sg" {
  name = "%s"
}

resource "exoscale_security_group_rule" "usg" {
  security_group = exoscale_security_group.sg.name
  protocol = "%s"
  type = "%s"
  icmp_type = %d
  icmp_code = %d
  user_security_group = exoscale_security_group.sg.name
}
`,
		testAccResourceSecurityGroupRuleSecurityGroupName,
		testAccResourceSecurityGroupRuleWithUSGProtocol,
		testAccResourceSecurityGroupRuleWithUSGType,
		testAccResourceSecurityGroupRuleWithUSGICMPType,
		testAccResourceSecurityGroupRuleWithUSGICMPCode,
	)
)

func TestAccResourceSecurityGroupRule(t *testing.T) {
	sg := new(egoscale.SecurityGroup)
	cidr := new(egoscale.EgressRule)
	usg := new(egoscale.IngressRule)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceSecurityGroupRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSecurityGroupRuleConfigWithCIDR,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.sg", sg),
					testAccCheckEgressRuleExists("exoscale_security_group_rule.cidr", sg, cidr),
					testAccCheckResourceSecurityGroupRule(cidr),
					testAccCheckResourceSecurityGroupRule((*egoscale.EgressRule)(usg)),
					testAccCheckResourceSecurityGroupRuleAttributes(testAttrs{
						"security_group": ValidateString(testAccResourceSecurityGroupRuleSecurityGroupName),
						"protocol":       ValidateString(testAccResourceSecurityGroupRuleWithCIDRProtocol),
						"type":           ValidateString(testAccResourceSecurityGroupRuleWithCIDRType),
						"cidr":           ValidateString(testAccResourceSecurityGroupRuleWithCIDRCIDR),
						"start_port":     ValidateString(fmt.Sprint(testAccResourceSecurityGroupRuleWithCIDRStartPort)),
						"end_port":       ValidateString(fmt.Sprint(testAccResourceSecurityGroupRuleWithCIDREndPort)),
					}),
				),
			},
			{
				ResourceName:      "exoscale_security_group_rule.cidr",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"security_group": ValidateString(testAccResourceSecurityGroupRuleSecurityGroupName),
							"protocol":       ValidateString(testAccResourceSecurityGroupRuleWithCIDRProtocol),
							"type":           ValidateString(testAccResourceSecurityGroupRuleWithCIDRType),
							"cidr":           ValidateString(testAccResourceSecurityGroupRuleWithCIDRCIDR),
							"start_port":     ValidateString(fmt.Sprint(testAccResourceSecurityGroupRuleWithCIDRStartPort)),
							"end_port":       ValidateString(fmt.Sprint(testAccResourceSecurityGroupRuleWithCIDREndPort)),
						},
						s[0].Attributes)
				},
			},
			{
				Config: testAccResourceSecurityGroupRuleConfigWithUSG,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.sg", sg),
					testAccCheckIngressRuleExists("exoscale_security_group_rule.usg", sg, usg),
					testAccCheckResourceSecurityGroupRule(usg),
					testAccCheckResourceSecurityGroupRule((*egoscale.EgressRule)(usg)),
					testAccCheckResourceSecurityGroupRuleAttributes(testAttrs{
						"security_group":      ValidateString(testAccResourceSecurityGroupRuleSecurityGroupName),
						"protocol":            ValidateString(testAccResourceSecurityGroupRuleWithUSGProtocol),
						"type":                ValidateString(testAccResourceSecurityGroupRuleWithUSGType),
						"icmp_type":           ValidateString(fmt.Sprint(testAccResourceSecurityGroupRuleWithUSGICMPType)),
						"icmp_code":           ValidateString(fmt.Sprint(testAccResourceSecurityGroupRuleWithUSGICMPCode)),
						"user_security_group": ValidateString(testAccResourceSecurityGroupRuleSecurityGroupName),
					}),
				),
			},
			{
				ResourceName:      "exoscale_security_group_rule.usg",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"security_group":      ValidateString(testAccResourceSecurityGroupRuleSecurityGroupName),
							"protocol":            ValidateString(testAccResourceSecurityGroupRuleWithUSGProtocol),
							"type":                ValidateString(testAccResourceSecurityGroupRuleWithUSGType),
							"icmp_type":           ValidateString(fmt.Sprint(testAccResourceSecurityGroupRuleWithUSGICMPType)),
							"icmp_code":           ValidateString(fmt.Sprint(testAccResourceSecurityGroupRuleWithUSGICMPCode)),
							"user_security_group": ValidateString(testAccResourceSecurityGroupRuleSecurityGroupName),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckEgressRuleExists(n string, sg *egoscale.SecurityGroup, rule *egoscale.EgressRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		if len(sg.EgressRule) == 0 {
			return errors.New("no egress rules found")
		}

		return Copy(rule, sg.EgressRule[0])
	}
}

func testAccCheckIngressRuleExists(n string, sg *egoscale.SecurityGroup, rule *egoscale.IngressRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		if len(sg.IngressRule) == 0 {
			return errors.New("no Ingress rules found")
		}

		return Copy(rule, sg.IngressRule[0])
	}
}

func testAccCheckResourceSecurityGroupRule(v interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		switch v.(type) {
		case egoscale.IngressRule, egoscale.EgressRule:
			r, _ := v.(egoscale.IngressRule)
			if r.RuleID == nil {
				return errors.New("security group rule id is nil")
			}
		}

		return nil
	}
}

func testAccCheckResourceSecurityGroupRuleAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_security_group_rule" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("security_group_rule resource not found in the state")
	}
}

func testAccCheckResourceSecurityGroupRuleDestroy(s *terraform.State) error {
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

	return errors.New("Security Group Rule still exists")
}
