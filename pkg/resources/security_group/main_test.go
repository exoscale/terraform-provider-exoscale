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
	sgDS := "data.exoscale_security_group.test_sg"
	sgDSByName := "data.exoscale_security_group.test_sg_by_name"

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
			// 2 Update SG description (with single single ds by name)
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/002.sg_update_description.tf.tmpl",
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
				),
			},
			// 3 Add SG external sources
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/003.sg_add_external_sources.tf.tmpl",
					&testdataSpec,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
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
			// 4 Remove SG external source
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/004.sg_remove_external_source.tf.tmpl",
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
			// Import
			{
				ResourceName: sg,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return fmt.Sprintf(
							"%s",
							s.RootModule().Resources[sg].Primary.ID,
						), nil
					}
				}(),
				ImportState: true,
			},
		},
	})
}
