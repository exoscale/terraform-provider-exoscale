package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

var (
	testAccResourceSecurityGroupRule1Description                        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSecurityGroupRule1EndPort                     uint16 = 2
	testAccResourceSecurityGroupRule1FlowDirection                      = "INGRESS"
	testAccResourceSecurityGroupRule1Network                            = "1.2.3.4/32"
	testAccResourceSecurityGroupRule1Protocol                           = "TCP"
	testAccResourceSecurityGroupRule1StartPort                   uint16 = 1
	testAccResourceSecurityGroupRule2Description                        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSecurityGroupRule2FlowDirection                      = "EGRESS"
	testAccResourceSecurityGroupRule2ICMPCode                    int64  = 0
	testAccResourceSecurityGroupRule2ICMPType                    int64  = 8
	testAccResourceSecurityGroupRule2Protocol                           = "ICMP"
	testAccResourceSecurityGroupRule3Description                        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSecurityGroupRule3EndPort                     uint16 = 2
	testAccResourceSecurityGroupRule3Protocol                           = "TCP"
	testAccResourceSecurityGroupRule3ProtocolPublicSecurityGroup        = "public-nlb-healthcheck-sources"
	testAccResourceSecurityGroupRule3StartPort                   uint16 = 1
	testAccResourceSecurityGroupRule3FlowDirection                      = "INGRESS"

	testAccResourceSecurityGroupRule1ConfigCreate = fmt.Sprintf(`
resource "exoscale_security_group" "test" {
  name = "%s"
}

resource "exoscale_security_group_rule" "test" {
  security_group_id = exoscale_security_group.test.id
  cidr              = "%s"
  description       = "%s"
  end_port          = %d
  protocol          = "%s"
  start_port        = %d
  type              = "%s"
}
`,
		testAccResourceSecurityGroupName,
		testAccResourceSecurityGroupRule1Network,
		testAccResourceSecurityGroupRule1Description,
		testAccResourceSecurityGroupRule1EndPort,
		testAccResourceSecurityGroupRule1Protocol,
		testAccResourceSecurityGroupRule1StartPort,
		testAccResourceSecurityGroupRule1FlowDirection,
	)

	testAccResourceSecurityGroupRule2ConfigCreate = fmt.Sprintf(`
resource "exoscale_security_group" "test" {
  name = "%s"
}

resource "exoscale_security_group_rule" "test" {
  security_group         = exoscale_security_group.test.name
  description            = "%s"
  icmp_code              = %d
  icmp_type              = %d
  protocol               = "%s"
  type                   = "%s"
  user_security_group_id = exoscale_security_group.test.id
}
`,
		testAccResourceSecurityGroupName,
		testAccResourceSecurityGroupRule2Description,
		testAccResourceSecurityGroupRule2ICMPCode,
		testAccResourceSecurityGroupRule2ICMPType,
		testAccResourceSecurityGroupRule2Protocol,
		testAccResourceSecurityGroupRule2FlowDirection,
	)

	testAccResourceSecurityGroupRule3ConfigCreate = fmt.Sprintf(`
resource "exoscale_security_group" "test" {
  name = "%s"
}

resource "exoscale_security_group_rule" "test" {
  security_group        = exoscale_security_group.test.name
  description           = "%s"
  end_port              = %d
  protocol              = "%s"
  public_security_group = "%s"
  start_port            = %d
  type                  = "%s"
}
`,
		testAccResourceSecurityGroupName,
		testAccResourceSecurityGroupRule3Description,
		testAccResourceSecurityGroupRule3EndPort,
		testAccResourceSecurityGroupRule3Protocol,
		testAccResourceSecurityGroupRule3ProtocolPublicSecurityGroup,
		testAccResourceSecurityGroupRule3StartPort,
		testAccResourceSecurityGroupRule3FlowDirection,
	)
)

func TestAccResourceSecurityGroupRule(t *testing.T) {
	var (
		r                 = "exoscale_security_group_rule.test"
		securityGroup     egoscale.SecurityGroup
		securityGroupRule egoscale.SecurityGroupRule
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSecurityGroupRuleDestroy(r),
		Steps: []resource.TestStep{
			{
				// Create - rule #1
				Config: testAccResourceSecurityGroupRule1ConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.test", &securityGroup),
					testAccCheckResourceSecurityGroupRuleExists(r, &securityGroupRule),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(testAccResourceSecurityGroupRule1Description, *securityGroupRule.Description)
						a.Equal(testAccResourceSecurityGroupRule1EndPort, *securityGroupRule.EndPort)
						a.Equal(testAccResourceSecurityGroupRule1FlowDirection, strings.ToUpper(*securityGroupRule.FlowDirection))
						a.Equal(testAccResourceSecurityGroupRule1Network, securityGroupRule.Network.String())
						a.Equal(testAccResourceSecurityGroupRule1Protocol, strings.ToUpper(*securityGroupRule.Protocol))
						a.Equal(testAccResourceSecurityGroupRule1StartPort, *securityGroupRule.StartPort)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSecurityGroupRuleAttrDescription:   validateString(testAccResourceSecurityGroupRule1Description),
						resSecurityGroupRuleAttrEndPort:       validateString(fmt.Sprint(testAccResourceSecurityGroupRule1EndPort)),
						resSecurityGroupRuleAttrFlowDirection: validateString(testAccResourceSecurityGroupRule1FlowDirection),
						resSecurityGroupRuleAttrNetwork:       validateString(testAccResourceSecurityGroupRule1Network),
						resSecurityGroupRuleAttrProtocol:      validateString(testAccResourceSecurityGroupRule1Protocol),
						resSecurityGroupRuleAttrStartPort:     validateString(fmt.Sprint(testAccResourceSecurityGroupRule1StartPort)),
					})),
				),
			},
			{
				// Import - rule #1
				ResourceName: r,
				ImportStateIdFunc: func(
					securityGroup *egoscale.SecurityGroup,
					securityGroupRule *egoscale.SecurityGroupRule,
				) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s", *securityGroup.ID, *securityGroupRule.ID), nil
					}
				}(&securityGroup, &securityGroupRule),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resSecurityGroupRuleAttrDescription:   validateString(testAccResourceSecurityGroupRule1Description),
							resSecurityGroupRuleAttrEndPort:       validateString(fmt.Sprint(testAccResourceSecurityGroupRule1EndPort)),
							resSecurityGroupRuleAttrFlowDirection: validateString(testAccResourceSecurityGroupRule1FlowDirection),
							resSecurityGroupRuleAttrNetwork:       validateString(testAccResourceSecurityGroupRule1Network),
							resSecurityGroupRuleAttrProtocol:      validateString(testAccResourceSecurityGroupRule1Protocol),
							resSecurityGroupRuleAttrStartPort:     validateString(fmt.Sprint(testAccResourceSecurityGroupRule1StartPort)),
						},
						s[0].Attributes)
				},
			},
			{
				// Create - rule #2
				Config: testAccResourceSecurityGroupRule2ConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.test", &securityGroup),
					testAccCheckResourceSecurityGroupRuleExists(r, &securityGroupRule),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(testAccResourceSecurityGroupRule2Description, *securityGroupRule.Description)
						a.Equal(testAccResourceSecurityGroupRule2FlowDirection, strings.ToUpper(*securityGroupRule.FlowDirection))
						a.Equal(testAccResourceSecurityGroupRule2ICMPCode, *securityGroupRule.ICMPCode)
						a.Equal(testAccResourceSecurityGroupRule2ICMPType, *securityGroupRule.ICMPType)
						a.Equal(testAccResourceSecurityGroupRule2Protocol, strings.ToUpper(*securityGroupRule.Protocol))
						a.Equal(*securityGroup.ID, *securityGroupRule.SecurityGroupID)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSecurityGroupRuleAttrDescription:           validateString(testAccResourceSecurityGroupRule2Description),
						resSecurityGroupRuleAttrFlowDirection:         validateString(testAccResourceSecurityGroupRule2FlowDirection),
						resSecurityGroupRuleAttrICMPCode:              validateString(fmt.Sprint(testAccResourceSecurityGroupRule2ICMPCode)),
						resSecurityGroupRuleAttrICMPType:              validateString(fmt.Sprint(testAccResourceSecurityGroupRule2ICMPType)),
						resSecurityGroupRuleAttrProtocol:              validateString(testAccResourceSecurityGroupRule2Protocol),
						resSecurityGroupRuleAttrUserSecurityGroupID:   validation.ToDiagFunc(validation.IsUUID),
						resSecurityGroupRuleAttrUserSecurityGroupName: validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Import - rule #2
				ResourceName: r,
				ImportStateIdFunc: func(
					securityGroup *egoscale.SecurityGroup,
					securityGroupRule *egoscale.SecurityGroupRule,
				) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s", *securityGroup.ID, *securityGroupRule.ID), nil
					}
				}(&securityGroup, &securityGroupRule),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resSecurityGroupRuleAttrDescription:           validateString(testAccResourceSecurityGroupRule2Description),
							resSecurityGroupRuleAttrFlowDirection:         validateString(testAccResourceSecurityGroupRule2FlowDirection),
							resSecurityGroupRuleAttrICMPCode:              validateString(fmt.Sprint(testAccResourceSecurityGroupRule2ICMPCode)),
							resSecurityGroupRuleAttrICMPType:              validateString(fmt.Sprint(testAccResourceSecurityGroupRule2ICMPType)),
							resSecurityGroupRuleAttrProtocol:              validateString(testAccResourceSecurityGroupRule2Protocol),
							resSecurityGroupRuleAttrUserSecurityGroupID:   validation.ToDiagFunc(validation.IsUUID),
							resSecurityGroupRuleAttrUserSecurityGroupName: validation.ToDiagFunc(validation.NoZeroValues),
						},
						s[0].Attributes)
				},
			},
			{
				// Create - rule #3
				Config: testAccResourceSecurityGroupRule3ConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.test", &securityGroup),
					testAccCheckResourceSecurityGroupRuleExists(r, &securityGroupRule),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(testAccResourceSecurityGroupRule3Description, *securityGroupRule.Description)
						a.Equal(testAccResourceSecurityGroupRule3EndPort, *securityGroupRule.EndPort)
						a.Equal(testAccResourceSecurityGroupRule3FlowDirection, strings.ToUpper(*securityGroupRule.FlowDirection))
						a.Equal(testAccResourceSecurityGroupRule3ProtocolPublicSecurityGroup, *securityGroupRule.SecurityGroupName)
						a.Equal(testAccResourceSecurityGroupRule3Protocol, strings.ToUpper(*securityGroupRule.Protocol))
						a.Equal(testAccResourceSecurityGroupRule3StartPort, *securityGroupRule.StartPort)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSecurityGroupRuleAttrDescription:         validateString(testAccResourceSecurityGroupRule3Description),
						resSecurityGroupRuleAttrEndPort:             validateString(fmt.Sprint(testAccResourceSecurityGroupRule3EndPort)),
						resSecurityGroupRuleAttrFlowDirection:       validateString(testAccResourceSecurityGroupRule3FlowDirection),
						resSecurityGroupRuleAttrPublicSecurityGroup: validateString(testAccResourceSecurityGroupRule3ProtocolPublicSecurityGroup),
						resSecurityGroupRuleAttrProtocol:            validateString(testAccResourceSecurityGroupRule3Protocol),
						resSecurityGroupRuleAttrStartPort:           validateString(fmt.Sprint(testAccResourceSecurityGroupRule3StartPort)),
					})),
				),
			},
			{
				// Import - rule #3
				ResourceName: r,
				ImportStateIdFunc: func(
					securityGroup *egoscale.SecurityGroup,
					securityGroupRule *egoscale.SecurityGroupRule,
				) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s", *securityGroup.ID, *securityGroupRule.ID), nil
					}
				}(&securityGroup, &securityGroupRule),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resSecurityGroupRuleAttrDescription:         validateString(testAccResourceSecurityGroupRule3Description),
							resSecurityGroupRuleAttrEndPort:             validateString(fmt.Sprint(testAccResourceSecurityGroupRule3EndPort)),
							resSecurityGroupRuleAttrFlowDirection:       validateString(testAccResourceSecurityGroupRule3FlowDirection),
							resSecurityGroupRuleAttrPublicSecurityGroup: validateString(testAccResourceSecurityGroupRule3ProtocolPublicSecurityGroup),
							resSecurityGroupRuleAttrProtocol:            validateString(testAccResourceSecurityGroupRule3Protocol),
							resSecurityGroupRuleAttrStartPort:           validateString(fmt.Sprint(testAccResourceSecurityGroupRule3StartPort)),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSecurityGroupRuleExists(
	r string,
	securityGroupRule *egoscale.SecurityGroupRule,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		securityGroupID, ok := rs.Primary.Attributes[resSecurityGroupRuleAttrSecurityGroupID]
		if !ok {
			return fmt.Errorf("resource attribute %q not set", resSecurityGroupRuleAttrSecurityGroupID)
		}

		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

		securityGroup, err := client.GetSecurityGroup(ctx, testZoneName, securityGroupID)
		if err != nil {
			return err
		}

		for _, r := range securityGroup.Rules {
			if *r.ID == rs.Primary.ID {
				*securityGroupRule = *r
				return nil
			}
		}

		return fmt.Errorf("resource Security Group rule %q not found", rs.Primary.ID)
	}
}

func testAccCheckResourceSecurityGroupRuleDestroy(r string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		securityGroupID, ok := rs.Primary.Attributes[resSecurityGroupRuleAttrSecurityGroupID]
		if !ok {
			return fmt.Errorf("resource attribute %q not set", resSecurityGroupRuleAttrUserSecurityGroupID)
		}

		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

		securityGroup, err := client.GetSecurityGroup(ctx, testZoneName, securityGroupID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		for _, r := range securityGroup.Rules {
			if *r.ID == rs.Primary.ID {
				return errors.New("Security Group rule still exists")
			}
		}

		return nil
	}
}
