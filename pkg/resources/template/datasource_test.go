package template_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestDataSourceTemplate(t *testing.T) {
	t.Parallel()

	dsByName := "data.exoscale_template.by_name"
	dsByID := "data.exoscale_template.by_id"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testutils.ParseTestdataConfig("./testdata/001.datasource_missing_lookup.tf.tmpl", &testdataSpec),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.datasource.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(dsByName, "id", regexp.MustCompile(`^[0-9a-f-]{36}$`)),
					resource.TestCheckResourceAttr(dsByName, "name", testutils.TestInstanceTemplateName),
					resource.TestCheckResourceAttr(dsByName, "default_user", testutils.TestInstanceTemplateUsername),
					resource.TestCheckResourceAttr(dsByName, "visibility", "public"),
					resource.TestCheckResourceAttr(dsByName, "zone", testutils.TestZoneName),

					resource.TestCheckResourceAttrPair(dsByID, "id", dsByName, "id"),
					resource.TestCheckResourceAttrPair(dsByID, "name", dsByName, "name"),
					resource.TestCheckResourceAttrPair(dsByID, "default_user", dsByName, "default_user"),
					resource.TestCheckResourceAttr(dsByID, "visibility", "public"),
				),
			},
		},
	})
}
