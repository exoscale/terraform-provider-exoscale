package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var (
	testAccDataSourceElasticIPAddressFamily4                = "inet4"
	testAccDataSourceElasticIPAddressFamily6                = "inet6"
	testAccDataSourceElasticIPDescription                   = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceElasticIPHealthcheckInterval    int64  = 5
	testAccDataSourceElasticIPHealthcheckMode               = "https"
	testAccDataSourceElasticIPHealthcheckPort        uint16 = 443
	testAccDataSourceElasticIPHealthcheckStrikesFail int64  = 1
	testAccDataSourceElasticIPHealthcheckStrikesOK   int64  = 2
	testAccDataSourceElasticIPHealthcheckTLSSNI             = "example.net"
	testAccDataSourceElasticIPHealthcheckTimeout     int64  = 3
	testAccDataSourceElasticIPHealthcheckURI                = "/health"
	testAccDataSourceElasticIPLabelValue                    = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceElasticIPReverseDNS                    = "tf-provider-rdns-test.exoscale.com"
	testAccDataSourceElasticIPLookupLabelKey                = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceElasticIPLookupLabelValue              = acctest.RandomWithPrefix(testPrefix)

	testAccDataSourceElasticIPConfig4 = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_elastic_ip" "test4" {
  zone        = local.zone
  description = "%s"

  reverse_dns = "%s"

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

  labels = {
    test = "%s"
  }
}
`,
		testZoneName,
		testAccDataSourceElasticIPDescription,
		testAccDataSourceElasticIPReverseDNS,
		testAccDataSourceElasticIPHealthcheckMode,
		testAccDataSourceElasticIPHealthcheckPort,
		testAccDataSourceElasticIPHealthcheckURI,
		testAccDataSourceElasticIPHealthcheckInterval,
		testAccDataSourceElasticIPHealthcheckTimeout,
		testAccDataSourceElasticIPHealthcheckStrikesOK,
		testAccDataSourceElasticIPHealthcheckStrikesFail,
		testAccDataSourceElasticIPHealthcheckTLSSNI,
		testAccDataSourceElasticIPLabelValue,
	)

	testAccDataSourceElasticIPConfig6 = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_elastic_ip" "test6" {
  zone           = local.zone
  description    = "%s"
  address_family = "%s"

	reverse_dns     = "%s"

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

  labels = {
    test = "%s"
	%s   = "%s"
  }
}
`,
		testZoneName,
		testAccDataSourceElasticIPDescription,
		testAccDataSourceElasticIPAddressFamily6,
		testAccDataSourceElasticIPReverseDNS,
		testAccDataSourceElasticIPHealthcheckMode,
		testAccDataSourceElasticIPHealthcheckPort,
		testAccDataSourceElasticIPHealthcheckURI,
		testAccDataSourceElasticIPHealthcheckInterval,
		testAccDataSourceElasticIPHealthcheckTimeout,
		testAccDataSourceElasticIPHealthcheckStrikesOK,
		testAccDataSourceElasticIPHealthcheckStrikesFail,
		testAccDataSourceElasticIPHealthcheckTLSSNI,
		testAccDataSourceElasticIPLabelValue,
		testAccDataSourceElasticIPLookupLabelKey,
		testAccDataSourceElasticIPLookupLabelValue,
	)
)

func TestAccDataSourceElasticIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      ` data "exoscale_elastic_ip" "test" { zone = "lolnope" }`,
				ExpectError: regexp.MustCompile("one of ip_address, id or labels must be specified"),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_elastic_ip" "by-ip4-address" {
  zone       = local.zone
  ip_address = exoscale_elastic_ip.test4.ip_address
}`,
					testAccDataSourceElasticIPConfig4,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceElasticIPAttributes("data.exoscale_elastic_ip.by-ip4-address", testAttrs{
						dsElasticIPAttrAddressFamily: validateString(testAccDataSourceElasticIPAddressFamily4),
						dsElasticIPAttrCIDR:          validation.ToDiagFunc(validation.IsCIDR),
						dsElasticIPAttrDescription:   validateString(testAccDataSourceElasticIPDescription),
						dsElasticIPAttrID:            validation.ToDiagFunc(validation.IsUUID),
						dsElasticIPAttrReverseDNS:    validateString(testAccDataSourceElasticIPReverseDNS),
						resElasticIPAttrIPAddress:    validation.ToDiagFunc(validation.IsIPv4Address),
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
  id = exoscale_elastic_ip.test4.id
}`,
					testAccDataSourceElasticIPConfig4,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceElasticIPAttributes("data.exoscale_elastic_ip.by-id", testAttrs{
						dsElasticIPAttrAddressFamily: validateString(testAccDataSourceElasticIPAddressFamily4),
						dsElasticIPAttrCIDR:          validation.ToDiagFunc(validation.IsCIDR),
						dsElasticIPAttrDescription:   validateString(testAccDataSourceElasticIPDescription),
						dsElasticIPAttrID:            validation.ToDiagFunc(validation.IsUUID),
						dsElasticIPAttrReverseDNS:    validateString(testAccDataSourceElasticIPReverseDNS),
						resElasticIPAttrIPAddress:    validation.ToDiagFunc(validation.IsIPv4Address),
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

data "exoscale_elastic_ip" "by-ip6-address" {
  zone       = local.zone
  ip_address = exoscale_elastic_ip.test6.ip_address
}`,
					testAccDataSourceElasticIPConfig6,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceElasticIPAttributes("data.exoscale_elastic_ip.by-ip6-address", testAttrs{
						dsElasticIPAttrAddressFamily: validateString(testAccDataSourceElasticIPAddressFamily6),
						dsElasticIPAttrCIDR:          validation.ToDiagFunc(validation.IsCIDR),
						dsElasticIPAttrDescription:   validateString(testAccDataSourceElasticIPDescription),
						dsElasticIPAttrID:            validation.ToDiagFunc(validation.IsUUID),
						dsElasticIPAttrReverseDNS:    validateString(testAccDataSourceElasticIPReverseDNS),
						resElasticIPAttrIPAddress:    validation.ToDiagFunc(validation.IsIPv6Address),
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

# Warning: This datasource requires 'exoscale_elastic_ip.test6'
data "exoscale_elastic_ip" "by-labels" {
  zone   = local.zone
  labels = { %s = "%s" }
}`,
					testAccDataSourceElasticIPConfig6,
					testAccDataSourceElasticIPLookupLabelKey,
					testAccDataSourceElasticIPLookupLabelValue,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceElasticIPAttributes("data.exoscale_elastic_ip.by-labels", testAttrs{
						dsElasticIPAttrAddressFamily: validateString(testAccDataSourceElasticIPAddressFamily6),
						dsElasticIPAttrCIDR:          validation.ToDiagFunc(validation.IsCIDR),
						dsElasticIPAttrDescription:   validateString(testAccDataSourceElasticIPDescription),
						dsElasticIPAttrID:            validation.ToDiagFunc(validation.IsUUID),
						resElasticIPAttrIPAddress:    validation.ToDiagFunc(validation.IsIPv6Address),
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
