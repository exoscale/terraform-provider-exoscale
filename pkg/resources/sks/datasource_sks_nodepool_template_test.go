package sks_test

import (
	"testing"
	"time"

	tftest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestAccDataSourceSKSNodepoolTemplate(t *testing.T) {
	t.Parallel()

	var (
		r           = "exoscale_sks_nodepool.test"
		ds          = "data.exoscale_sks_nodepool_template.test"
		dsList      = "data.exoscale_sks_nodepool_template_list.test"
		sksNodepool egoscale.SKSNodepool
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceSKSNodepoolDestroy(r),
		Steps: []tftest.TestStep{
			{
				// Create a nodepool pinned on the latest template.
				Config: parseSKSConfig(t, "./testdata/024.sksnp_template_datasource.tf.tmpl", td),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestMatchResourceAttr(ds, "id", testUUIDRegex),
					tftest.TestCheckResourceAttrSet(ds, "name"),
					tftest.TestCheckResourceAttrSet(ds, "created_at"),
					tftest.TestCheckResourceAttr(ds, "variant", "standard"),

					tftest.TestCheckTypeSetElemAttrPair(dsList, "templates.*.id", ds, "id"),
					tftest.TestCheckResourceAttr(dsList, "templates.0.variant", "standard"),

					testAccCheckResourceSKSNodepoolExists(r, &sksNodepool),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(egoscale.SKSNodepoolTemplateSelectionPinned, sksNodepool.TemplateSelection)
						if a.NotNil(sksNodepool.Template) {
							a.Equal(s.RootModule().Resources[ds].Primary.ID, sksNodepool.Template.ID.String())
						}
						a.Equal(int64(1), sksNodepool.Size)
						return nil
					},
					tftest.TestCheckResourceAttrPair(r, "template_id", ds, "id"),
					tftest.TestCheckResourceAttr(r, "template_selection", "pinned"),
					tftest.TestCheckResourceAttr(r, "size", "1"),
				),
			},
		},
	})
}
