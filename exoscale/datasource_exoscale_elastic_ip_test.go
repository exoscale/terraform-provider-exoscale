package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccDataSourceElasticIPDescription                   = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceElasticIPHealthcheckInterval    int64  = 5
	testAccDataSourceElasticIPHealthcheckMode               = "https"
	testAccDataSourceElasticIPHealthcheckPort        uint16 = 443
	testAccDataSourceElasticIPHealthcheckStrikesFail int64  = 1
	testAccDataSourceElasticIPHealthcheckStrikesOK   int64  = 2
	testAccDataSourceElasticIPHealthcheckTLSSNI             = "example.net"
	testAccDataSourceElasticIPHealthcheckTimeout     int64  = 3
	testAccDataSourceElasticIPHealthcheckURI                = "/health"

	testAccDataSourceElasticIPConfig = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_elastic_ip" "test" {
  zone        = local.zone
  description = "%s"

  healthcheck {
    mode            = "%s"
    port            = %d
    uri             = "%s"
    interval        = %d
    timeout         = %d
    strikes_ok      = %d
    strikes_fail    = %d
    tls_sni         = "%s"
    tls_skip_verify = true
  }
}
`,
		testZoneName,
		testAccDataSourceElasticIPDescription,
		testAccDataSourceElasticIPHealthcheckMode,
		testAccDataSourceElasticIPHealthcheckPort,
		testAccDataSourceElasticIPHealthcheckURI,
		testAccDataSourceElasticIPHealthcheckInterval,
		testAccDataSourceElasticIPHealthcheckTimeout,
		testAccDataSourceElasticIPHealthcheckStrikesOK,
		testAccDataSourceElasticIPHealthcheckStrikesFail,
		testAccDataSourceElasticIPHealthcheckTLSSNI,
	)
)

func TestAccDataSourceElasticIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      ` data "exoscale_elastic_ip" "test" { zone = "lolnope" }`,
				ExpectError: regexp.MustCompile("either ip_address or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_elastic_ip" "by-ip-address" {
  zone       = local.zone
  ip_address = exoscale_elastic_ip.test.ip_address
}`,
					testAccDataSourceElasticIPConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceElasticIPAttributes("data.exoscale_elastic_ip.by-ip-address", testAttrs{
						dsElasticIPAttrDescription: validateString(testAccDataSourceElasticIPDescription),
						dsElasticIPAttrID:          validation.ToDiagFunc(validation.IsUUID),
						resElasticIPAttrIPAddress:  validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckInterval):    validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckInterval)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckMode):        validateString(testAccDataSourceElasticIPHealthcheckMode),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckPort):        validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckPort)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckStrikesFail): validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckStrikesFail)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckStrikesOK):   validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckStrikesOK)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckTimeout):     validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckTimeout)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckURI):         validateString(testAccDataSourceElasticIPHealthcheckURI),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_elastic_ip" "by-id" {
  zone = local.zone
  id = exoscale_elastic_ip.test.id
}`,
					testAccDataSourceElasticIPConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceElasticIPAttributes("data.exoscale_elastic_ip.by-id", testAttrs{
						dsElasticIPAttrDescription: validateString(testAccDataSourceElasticIPDescription),
						dsElasticIPAttrID:          validation.ToDiagFunc(validation.IsUUID),
						resElasticIPAttrIPAddress:  validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckInterval):    validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckInterval)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckMode):        validateString(testAccDataSourceElasticIPHealthcheckMode),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckPort):        validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckPort)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckStrikesFail): validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckStrikesFail)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckStrikesOK):   validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckStrikesOK)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckTimeout):     validateString(fmt.Sprint(testAccDataSourceElasticIPHealthcheckTimeout)),
						resElasticIPAttrHealthcheck(dsElasticIPAttrHealthcheckURI):         validateString(testAccDataSourceElasticIPHealthcheckURI),
					}),
				),
			},
		},
	})
}

func testAccDataSourceElasticIPAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_elastic_ip data source not found in the state")
	}
}
