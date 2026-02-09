package security_group_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestSecurityGroup(t *testing.T) {
	t.Parallel()

	sg := "exoscale_security_group.test_sg"
	sgAux := "exoscale_security_group.test_sg_aux"
	sgDS := "data.exoscale_security_group.test_sg"
	sgDSByName := "data.exoscale_security_group.test_sg_by_name"
	rule1 := "exoscale_security_group_rule.test_rule_1"
	rule2 := "exoscale_security_group_rule.test_rule_2"
	rule3 := "exoscale_security_group_rule.test_rule_3"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create SG and 2 data sources (match by id and name)
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/001.sg_create.tf.tmpl",
					&testdataSpec,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						sg,
						"name",
						testutils.ResourceName(testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(
						sg,
						"description",
						"test security group",
					),
					resource.TestCheckNoResourceAttr(sg, "external_sources"),
					resource.TestCheckResourceAttrPair(
						sg,
						"description",
						sgDS,
						"description",
					),
					resource.TestCheckNoResourceAttr(sgDS, "external_sources"),
					resource.TestCheckResourceAttrPair(
						sg,
						"description",
						sgDSByName,
						"description",
					),
					resource.TestCheckNoResourceAttr(sgDSByName, "external_sources"),
				),
			},
			// 2 Update SG, remove DS by name
			// - update description
			// - add external sources
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/002.sg_update.tf.tmpl",
					&testdataSpec,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						sg,
						"description",
						"updated description",
					),
					resource.TestCheckResourceAttrPair(
						sg,
						"description",
						sgDS,
						"description",
					),
					resource.TestCheckResourceAttr(
						sg,
						"external_sources.#",
						"2",
					),
					resource.TestCheckTypeSetElemAttr(
						sg,
						"external_sources.*",
						"192.168.1.1/32",
					),
					resource.TestCheckTypeSetElemAttr(
						sg,
						"external_sources.*",
						"192.168.2.1/32",
					),
					resource.TestCheckResourceAttrPair(
						sg,
						"external_sources",
						sgDS,
						"external_sources",
					),
				),
			},
			// 3 Remove SG external source
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/003.remove_external_source.tf.tmpl",
					&testdataSpec,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						sg,
						"external_sources.#",
						"1",
					),
					resource.TestCheckTypeSetElemAttr(
						sg,
						"external_sources.*",
						"192.168.1.1/32",
					),
					resource.TestCheckResourceAttrPair(
						sg,
						"external_sources",
						sgDS,
						"external_sources",
					),
				),
			},
			// 4 Add rules
			// - rule 1 (cidr & ports)
			// - rule 2 (public sg & icmp)
			// - add aux SG and rule 3 with user sg and default flow direction
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/004.add_rules.tf.tmpl",
					&testdataSpec,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						rule1,
						"cidr",
						"1.1.1.1/32",
					),
					resource.TestCheckResourceAttr(
						rule1,
						"description",
						"test",
					),
					resource.TestCheckResourceAttr(
						rule1,
						"type",
						"INGRESS",
					),
					resource.TestCheckResourceAttr(
						rule1,
						"protocol",
						"UDP",
					),
					resource.TestCheckResourceAttr(
						rule1,
						"start_port",
						"8080",
					),
					resource.TestCheckResourceAttr(
						rule1,
						"end_port",
						"8081",
					),
					resource.TestCheckResourceAttr(
						rule2,
						"public_security_group",
						"public-sks-apiservers",
					),
					resource.TestCheckResourceAttr(
						rule2,
						"type",
						"EGRESS",
					),
					resource.TestCheckResourceAttr(
						rule2,
						"protocol",
						"ICMP",
					),
					resource.TestCheckResourceAttr(
						rule2,
						"icmp_type",
						"8",
					),
					resource.TestCheckResourceAttr(
						rule2,
						"icmp_code",
						"0",
					),
					resource.TestCheckResourceAttrPair(
						sgAux,
						"id",
						rule3,
						"user_security_group_id",
					),
					resource.TestCheckResourceAttr(
						rule3,
						"type",
						"INGRESS",
					),
					resource.TestCheckResourceAttr(
						rule3,
						"protocol",
						"TCP",
					),
					resource.TestCheckResourceAttr(
						rule3,
						"start_port",
						"8080",
					),
					resource.TestCheckResourceAttr(
						rule3,
						"end_port",
						"8081",
					),
				),
			},
			// Import SG
			{
				ResourceName: rule1,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return fmt.Sprintf(
							"%s@%s",
							s.RootModule().Resources[sg].Primary.ID,
							s.RootModule().Resources[rule1].Primary.ID,
						), nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					// type and protocol are case insensitive,
					// refresh after import cannot ignore case as state is empty
					"type",
					"protocol",
				},
			},
		},
	})
}
