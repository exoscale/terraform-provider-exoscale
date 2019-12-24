package exoscale

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceSecurityGroupRulesSecurityGroupName = testPrefix + "-" + testRandomString()

	testAccResourceSecurityGroupRulesConfigCreate = fmt.Sprintf(`
resource "exoscale_security_group" "sg" {
  name = "%s"
}

resource "exoscale_security_group_rules" "rules" {
  security_group_id = exoscale_security_group.sg.id

  ingress {
    protocol = "ICMP"
    icmp_type = 8
    icmp_code = 0
    cidr_list = ["0.0.0.0/0"]
  }

  ingress {
    protocol = "ICMPv6"
    icmp_type = 128
    icmp_code = 0
    cidr_list = ["::/0"]
  }

  ingress {
    protocol = "TCP"
    cidr_list = ["10.0.0.0/24", "::/0"]
    ports = ["22", "8000-8888"]
    user_security_group_list = [exoscale_security_group.sg.name, "default"]
  }

  egress {
    protocol = "UDP"
    cidr_list = ["192.168.0.0/24", "::/0"]
    ports = ["44", "2375-2377"]
    user_security_group_list = ["default"]
  }
}
`,
		testAccResourceSecurityGroupRulesSecurityGroupName,
	)

	testAccResourceSecurityGroupRulesConfigUpdate = fmt.Sprintf(`
resource "exoscale_security_group" "sg" {
  name = "%s"
}

resource "exoscale_security_group_rules" "rules" {
  security_group_id = exoscale_security_group.sg.id

  ingress {
    protocol = "ICMP"
    icmp_type = 8
    icmp_code = 0
    cidr_list = ["0.0.0.0/0"]
  }

  ingress {
    protocol = "ICMPv6"
    icmp_type = 128
    icmp_code = 0
    cidr_list = ["::/0"]
  }

  ingress {
    protocol = "TCP"
    cidr_list = ["10.0.0.0/24", "::/0"]
    ports = ["2222", "8000-8888"]
    user_security_group_list = [exoscale_security_group.sg.name, "default"]
  }

  egress {
    protocol = "UDP"
    cidr_list = ["192.168.0.0/24", "::/0"]
    ports = ["4444", "2375-2377"]
    user_security_group_list = ["default"]
  }
}
`,
		testAccResourceSecurityGroupRulesSecurityGroupName,
	)
)

func TestPreparePorts(t *testing.T) {
	ports := preparePorts(schema.NewSet(schema.HashString, []interface{}{"22", "10-20"}))

	for _, portRange := range ports {
		if portRange[0] == 22 && portRange[1] != 22 {
			t.Errorf("bad port, wanted 22-22, got %#v", portRange)
		}

		if portRange[0] == 10 && portRange[1] != 20 {
			t.Errorf("bad port, wanted 10-20, got %#v", ports[1])
		}
	}
}

func TestAccResourceSecurityGroupRules(t *testing.T) {
	sg := new(egoscale.SecurityGroup)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceSecurityGroupRulesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSecurityGroupRulesConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.sg", sg),
					testAccCheckSecurityGroupHasManyRules(16),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						CIDR:     egoscale.MustParseCIDR("0.0.0.0/0"),
						Protocol: "ICMP",
						IcmpType: 8,
						IcmpCode: 0,
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						CIDR:     egoscale.MustParseCIDR("::/0"),
						Protocol: "ICMPv6",
						IcmpType: 128,
						IcmpCode: 0,
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						CIDR:      egoscale.MustParseCIDR("10.0.0.0/24"),
						StartPort: 22,
						EndPort:   22,
						Protocol:  "TCP",
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						CIDR:      egoscale.MustParseCIDR("::/0"),
						StartPort: 22,
						EndPort:   22,
						Protocol:  "TCP",
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						CIDR:      egoscale.MustParseCIDR("10.0.0.0/24"),
						StartPort: 8000,
						EndPort:   8888,
						Protocol:  "TCP",
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						CIDR:      egoscale.MustParseCIDR("::/0"),
						StartPort: 8000,
						EndPort:   8888,
						Protocol:  "TCP",
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						SecurityGroupName: testAccResourceSecurityGroupRulesSecurityGroupName,
						StartPort:         22,
						EndPort:           22,
						Protocol:          "TCP",
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						SecurityGroupName: "default",
						StartPort:         22,
						EndPort:           22,
						Protocol:          "TCP",
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						SecurityGroupName: testAccResourceSecurityGroupRulesSecurityGroupName,
						StartPort:         8000,
						EndPort:           8888,
						Protocol:          "TCP",
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						SecurityGroupName: "default",
						StartPort:         8000,
						EndPort:           8888,
						Protocol:          "TCP",
					}),
					testAccCheckSecurityGroupEgressRuleExists(sg, &egoscale.EgressRule{
						CIDR:      egoscale.MustParseCIDR("192.168.0.0/24"),
						StartPort: 44,
						EndPort:   44,
						Protocol:  "UDP",
					}),
					testAccCheckSecurityGroupEgressRuleExists(sg, &egoscale.EgressRule{
						CIDR:      egoscale.MustParseCIDR("::/0"),
						StartPort: 44,
						EndPort:   44,
						Protocol:  "UDP",
					}),
					testAccCheckSecurityGroupEgressRuleExists(sg, &egoscale.EgressRule{
						CIDR:      egoscale.MustParseCIDR("192.168.0.0/24"),
						StartPort: 2375,
						EndPort:   2377,
						Protocol:  "UDP",
					}),
					testAccCheckSecurityGroupEgressRuleExists(sg, &egoscale.EgressRule{
						CIDR:      egoscale.MustParseCIDR("::/0"),
						StartPort: 2375,
						EndPort:   2377,
						Protocol:  "UDP",
					}),
					testAccCheckSecurityGroupEgressRuleExists(sg, &egoscale.EgressRule{
						SecurityGroupName: "default",
						StartPort:         44,
						EndPort:           44,
						Protocol:          "UDP",
					}),
					testAccCheckSecurityGroupEgressRuleExists(sg, &egoscale.EgressRule{
						SecurityGroupName: "default",
						StartPort:         2375,
						EndPort:           2377,
						Protocol:          "UDP",
					}),
				),
			},
			{
				Config: testAccResourceSecurityGroupRulesConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.sg", sg),
					testAccCheckSecurityGroupHasManyRules(16),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						SecurityGroupName: testAccResourceSecurityGroupRulesSecurityGroupName,
						StartPort:         2222,
						EndPort:           2222,
						Protocol:          "TCP",
					}),
					testAccCheckSecurityGroupIngressRuleExists(sg, &egoscale.IngressRule{
						SecurityGroupName: "default",
						StartPort:         2222,
						EndPort:           2222,
						Protocol:          "TCP",
					}),
					testAccCheckSecurityGroupEgressRuleExists(sg, &egoscale.EgressRule{
						CIDR:      egoscale.MustParseCIDR("192.168.0.0/24"),
						StartPort: 4444,
						EndPort:   4444,
						Protocol:  "UDP",
					}),
					testAccCheckSecurityGroupEgressRuleExists(sg, &egoscale.EgressRule{
						CIDR:      egoscale.MustParseCIDR("::/0"),
						StartPort: 4444,
						EndPort:   4444,
						Protocol:  "UDP",
					}),
					testAccCheckSecurityGroupEgressRuleExists(sg, &egoscale.EgressRule{
						SecurityGroupName: "default",
						StartPort:         4444,
						EndPort:           4444,
						Protocol:          "UDP",
					}),
				),
			},
		},
	})
}

func testAccCheckSecurityGroupHasManyRules(quantity int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_security_group_rules" {
				continue
			}

			total := 0
			for k, id := range rs.Primary.Attributes {
				if strings.HasSuffix(k, ".ids.#") {
					count, _ := strconv.Atoi(id)
					total += count
				}
			}

			if total != quantity {
				return fmt.Errorf("number of rules doesn't match, want %d != has %d", quantity, total)
			}

			return nil
		}

		return errors.New("Could not find any security group rules")
	}
}

func testAccCheckSecurityGroupIngressRuleExists(sg *egoscale.SecurityGroup, rule *egoscale.IngressRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, r := range sg.IngressRule {
			if strings.EqualFold(r.Protocol, rule.Protocol) && r.StartPort == rule.StartPort && r.EndPort == rule.EndPort && r.IcmpCode == rule.IcmpCode && r.IcmpType == rule.IcmpType {
				if r.CIDR != nil && rule.CIDR != nil && r.CIDR.Equal(*rule.CIDR) {
					return nil
				}
				if r.SecurityGroupName != "" && r.SecurityGroupName == rule.SecurityGroupName {
					return nil
				}
			}
		}

		return fmt.Errorf("rule not found %#v", rule)
	}
}

func testAccCheckSecurityGroupEgressRuleExists(sg *egoscale.SecurityGroup, rule *egoscale.EgressRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, r := range sg.EgressRule {
			if strings.EqualFold(r.Protocol, rule.Protocol) && r.StartPort == rule.StartPort && r.EndPort == rule.EndPort && r.IcmpCode == rule.IcmpCode && r.IcmpType == rule.IcmpType {
				if r.CIDR != nil && rule.CIDR != nil && r.CIDR.Equal(*rule.CIDR) {
					return nil
				}
				if r.SecurityGroupName != "" && r.SecurityGroupName == rule.SecurityGroupName {
					return nil
				}
			}
		}

		return fmt.Errorf("rule not found %#v", rule)
	}
}

func testAccCheckResourceSecurityGroupRulesDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_security_group_rules" {
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

	return errors.New("Security Group Rules still exist")
}
