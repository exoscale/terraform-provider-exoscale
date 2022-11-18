package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

var (
	testAccResourceElasticIPAddressFamily4                       = "inet4"
	testAccResourceElasticIPAddressFamily6                       = "inet6"
	testAccResourceElasticIPDescription                          = acctest.RandString(10)
	testAccResourceElasticIPDescriptionUpdated                   = testAccResourceElasticIPDescription + "-updated"
	testAccResourceElasticIPHealthcheckInterval           int64  = 5
	testAccResourceElasticIPHealthcheckIntervalUpdated           = testAccResourceElasticIPHealthcheckInterval + 1
	testAccResourceElasticIPHealthcheckMode                      = "http"
	testAccResourceElasticIPHealthcheckModeUpdated               = "https"
	testAccResourceElasticIPHealthcheckPort               uint16 = 80
	testAccResourceElasticIPHealthcheckPortUpdated        uint16 = 443
	testAccResourceElasticIPHealthcheckStrikesFail        int64  = 1
	testAccResourceElasticIPHealthcheckStrikesFailUpdated        = testAccResourceElasticIPHealthcheckStrikesFail + 1
	testAccResourceElasticIPHealthcheckStrikesOK          int64  = 2
	testAccResourceElasticIPHealthcheckStrikesOKUpdated          = testAccResourceElasticIPHealthcheckStrikesOK + 1
	testAccResourceElasticIPHealthcheckTLSSNI                    = "example.net"
	testAccResourceElasticIPHealthcheckTimeout            int64  = 3
	testAccResourceElasticIPHealthcheckTimeoutUpdated            = testAccResourceElasticIPHealthcheckTimeout + 1
	testAccResourceElasticIPHealthcheckURI                       = "/health"
	testAccResourceElasticIPHealthcheckURIUpdated                = testAccResourceElasticIPHealthcheckURI + "-updated"
	testAccResourceElasticIPLabelValue                           = acctest.RandomWithPrefix(testPrefix)
	testAccResourceElasticIPLabelValueUpdated                    = testAccResourceElasticIPLabelValue + "-updated"

	testAccResourceElasticIP4ConfigCreate = fmt.Sprintf(`
resource "exoscale_elastic_ip" "test4" {
  zone        = "%s"
  description = "%s"

  healthcheck {
    mode         = "%s"
    port         = %d
    uri          = "%s"
    interval     = %d
    timeout      = %d
    strikes_ok   = %d
    strikes_fail = %d
  }

  labels = {
    test = "%s"
  }
}
`,
		testZoneName,
		testAccResourceElasticIPDescription,
		testAccResourceElasticIPHealthcheckMode,
		testAccResourceElasticIPHealthcheckPort,
		testAccResourceElasticIPHealthcheckURI,
		testAccResourceElasticIPHealthcheckInterval,
		testAccResourceElasticIPHealthcheckTimeout,
		testAccResourceElasticIPHealthcheckStrikesOK,
		testAccResourceElasticIPHealthcheckStrikesFail,
		testAccResourceElasticIPLabelValue,
	)

	testAccResourceElasticIP4ConfigUpdate = fmt.Sprintf(`
resource "exoscale_elastic_ip" "test4" {
  zone        = "%s"
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

  labels = {
    test = "%s"
  }
}
`,
		testZoneName,
		testAccResourceElasticIPDescriptionUpdated,
		testAccResourceElasticIPHealthcheckModeUpdated,
		testAccResourceElasticIPHealthcheckPortUpdated,
		testAccResourceElasticIPHealthcheckURIUpdated,
		testAccResourceElasticIPHealthcheckIntervalUpdated,
		testAccResourceElasticIPHealthcheckTimeoutUpdated,
		testAccResourceElasticIPHealthcheckStrikesOKUpdated,
		testAccResourceElasticIPHealthcheckStrikesFailUpdated,
		testAccResourceElasticIPHealthcheckTLSSNI,
		testAccResourceElasticIPLabelValueUpdated,
	)

	testAccResourceElasticIP6ConfigCreate = fmt.Sprintf(`
resource "exoscale_elastic_ip" "test6" {
  zone        = "%s"
  description = "%s"
	address_family = "%s"

  healthcheck {
    mode         = "%s"
    port         = %d
    uri          = "%s"
    interval     = %d
    timeout      = %d
    strikes_ok   = %d
    strikes_fail = %d
  }

  labels = {
    test = "%s"
  }
}
`,
		testZoneName,
		testAccResourceElasticIPDescription,
		testAccResourceElasticIPAddressFamily6,
		testAccResourceElasticIPHealthcheckMode,
		testAccResourceElasticIPHealthcheckPort,
		testAccResourceElasticIPHealthcheckURI,
		testAccResourceElasticIPHealthcheckInterval,
		testAccResourceElasticIPHealthcheckTimeout,
		testAccResourceElasticIPHealthcheckStrikesOK,
		testAccResourceElasticIPHealthcheckStrikesFail,
		testAccResourceElasticIPLabelValue,
	)

	testAccResourceElasticIP6ConfigUpdate = fmt.Sprintf(`
resource "exoscale_elastic_ip" "test6" {
  zone        = "%s"
  description = "%s"
	address_family = "%s"

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
		testAccResourceElasticIPDescriptionUpdated,
		testAccResourceElasticIPAddressFamily6,
		testAccResourceElasticIPHealthcheckModeUpdated,
		testAccResourceElasticIPHealthcheckPortUpdated,
		testAccResourceElasticIPHealthcheckURIUpdated,
		testAccResourceElasticIPHealthcheckIntervalUpdated,
		testAccResourceElasticIPHealthcheckTimeoutUpdated,
		testAccResourceElasticIPHealthcheckStrikesOKUpdated,
		testAccResourceElasticIPHealthcheckStrikesFailUpdated,
		testAccResourceElasticIPHealthcheckTLSSNI,
		testAccResourceElasticIPLabelValueUpdated,
	)
)

func TestAccResourceElasticIP(t *testing.T) {
	var (
		r4        = "exoscale_elastic_ip.test4"
		r6        = "exoscale_elastic_ip.test6"
		elasticIP egoscale.ElasticIP
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceElasticIPDestroy(&elasticIP),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceElasticIP4ConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceElasticIPExists(r4, &elasticIP),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceElasticIPAddressFamily4, *elasticIP.AddressFamily)
						a.Equal(testAccResourceElasticIPDescription, *elasticIP.Description)
						a.NotNil(elasticIP.Healthcheck)
						a.Equal(testAccResourceElasticIPHealthcheckInterval, int64(elasticIP.Healthcheck.Interval.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckMode, *elasticIP.Healthcheck.Mode)
						a.Equal(testAccResourceElasticIPHealthcheckPort, *elasticIP.Healthcheck.Port)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesFail, *elasticIP.Healthcheck.StrikesFail)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesOK, *elasticIP.Healthcheck.StrikesOK)
						a.Equal(testAccResourceElasticIPHealthcheckTimeout, int64(elasticIP.Healthcheck.Timeout.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckURI, *elasticIP.Healthcheck.URI)

						return nil
					},
					checkResourceState(r4, checkResourceStateValidateAttributes(testAttrs{
						resElasticIPAttrDescription:                                         validateString(testAccResourceElasticIPDescription),
						resElasticIPAttrAddressFamily:                                       validateString(testAccResourceElasticIPAddressFamily4),
						resElasticIPAttrCIDR:                                                validation.ToDiagFunc(validation.IsCIDR),
						resElasticIPAttrIPAddress:                                           validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrLabels + ".test":                                    validateString(testAccResourceElasticIPLabelValue),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckInterval):    validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckInterval)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckMode):        validateString(testAccResourceElasticIPHealthcheckMode),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckPort):        validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckPort)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesFail): validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesFail)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesOK):   validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesOK)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTimeout):     validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckTimeout)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckURI):         validateString(testAccResourceElasticIPHealthcheckURI),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceElasticIP4ConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceElasticIPExists(r4, &elasticIP),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceElasticIPDescriptionUpdated, *elasticIP.Description)
						a.NotNil(elasticIP.Healthcheck)
						a.Equal(testAccResourceElasticIPHealthcheckIntervalUpdated, int64(elasticIP.Healthcheck.Interval.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckModeUpdated, *elasticIP.Healthcheck.Mode)
						a.Equal(testAccResourceElasticIPHealthcheckPortUpdated, *elasticIP.Healthcheck.Port)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesFailUpdated, *elasticIP.Healthcheck.StrikesFail)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesOKUpdated, *elasticIP.Healthcheck.StrikesOK)
						a.Equal(testAccResourceElasticIPHealthcheckTLSSNI, *elasticIP.Healthcheck.TLSSNI)
						a.True(*elasticIP.Healthcheck.TLSSkipVerify)
						a.Equal(testAccResourceElasticIPHealthcheckTimeoutUpdated, int64(elasticIP.Healthcheck.Timeout.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckURIUpdated, *elasticIP.Healthcheck.URI)

						return nil
					},
					checkResourceState(r4, checkResourceStateValidateAttributes(testAttrs{
						resElasticIPAttrDescription:                                           validateString(testAccResourceElasticIPDescriptionUpdated),
						resElasticIPAttrAddressFamily:                                         validateString(testAccResourceElasticIPAddressFamily4),
						resElasticIPAttrCIDR:                                                  validation.ToDiagFunc(validation.IsCIDR),
						resElasticIPAttrIPAddress:                                             validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrLabels + ".test":                                      validateString(testAccResourceElasticIPLabelValueUpdated),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckInterval):      validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckIntervalUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckMode):          validateString(testAccResourceElasticIPHealthcheckModeUpdated),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckPort):          validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckPortUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesFail):   validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesFailUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesOK):     validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesOKUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSNI):        validateString(testAccResourceElasticIPHealthcheckTLSSNI),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSkipVerify): validateString("true"),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTimeout):       validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckTimeoutUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckURI):           validateString(testAccResourceElasticIPHealthcheckURIUpdated),
					})),
				),
			},
			{
				// Import
				ResourceName:      r4,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(elasticIP *egoscale.ElasticIP) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *elasticIP.ID, testZoneName), nil
					}
				}(&elasticIP),
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resElasticIPAttrDescription:                                           validateString(testAccResourceElasticIPDescriptionUpdated),
							resElasticIPAttrAddressFamily:                                         validateString(testAccResourceElasticIPAddressFamily4),
							resElasticIPAttrCIDR:                                                  validation.ToDiagFunc(validation.IsCIDR),
							resElasticIPAttrIPAddress:                                             validation.ToDiagFunc(validation.IsIPAddress),
							resElasticIPAttrLabels + ".test":                                      validateString(testAccResourceElasticIPLabelValueUpdated),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckInterval):      validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckIntervalUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckMode):          validateString(testAccResourceElasticIPHealthcheckModeUpdated),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckPort):          validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckPortUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesFail):   validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesFailUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesOK):     validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesOKUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSNI):        validateString(testAccResourceElasticIPHealthcheckTLSSNI),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSkipVerify): validateString("true"),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTimeout):       validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckTimeoutUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckURI):           validateString(testAccResourceElasticIPHealthcheckURIUpdated),
						},
						s[0].Attributes)
				},
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceElasticIPDestroy(&elasticIP),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceElasticIP6ConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceElasticIPExists(r6, &elasticIP),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceElasticIPAddressFamily6, *elasticIP.AddressFamily)
						a.Equal(testAccResourceElasticIPDescription, *elasticIP.Description)
						a.NotNil(elasticIP.Healthcheck)
						a.Equal(testAccResourceElasticIPHealthcheckInterval, int64(elasticIP.Healthcheck.Interval.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckMode, *elasticIP.Healthcheck.Mode)
						a.Equal(testAccResourceElasticIPHealthcheckPort, *elasticIP.Healthcheck.Port)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesFail, *elasticIP.Healthcheck.StrikesFail)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesOK, *elasticIP.Healthcheck.StrikesOK)
						a.Equal(testAccResourceElasticIPHealthcheckTimeout, int64(elasticIP.Healthcheck.Timeout.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckURI, *elasticIP.Healthcheck.URI)

						return nil
					},
					checkResourceState(r6, checkResourceStateValidateAttributes(testAttrs{
						resElasticIPAttrDescription:                                         validateString(testAccResourceElasticIPDescription),
						resElasticIPAttrAddressFamily:                                       validateString(testAccResourceElasticIPAddressFamily6),
						resElasticIPAttrCIDR:                                                validation.ToDiagFunc(validation.IsCIDR),
						resElasticIPAttrIPAddress:                                           validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrLabels + ".test":                                    validateString(testAccResourceElasticIPLabelValue),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckInterval):    validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckInterval)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckMode):        validateString(testAccResourceElasticIPHealthcheckMode),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckPort):        validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckPort)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesFail): validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesFail)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesOK):   validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesOK)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTimeout):     validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckTimeout)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckURI):         validateString(testAccResourceElasticIPHealthcheckURI),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceElasticIP6ConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceElasticIPExists(r6, &elasticIP),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceElasticIPDescriptionUpdated, *elasticIP.Description)
						a.NotNil(elasticIP.Healthcheck)
						a.Equal(testAccResourceElasticIPHealthcheckIntervalUpdated, int64(elasticIP.Healthcheck.Interval.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckModeUpdated, *elasticIP.Healthcheck.Mode)
						a.Equal(testAccResourceElasticIPHealthcheckPortUpdated, *elasticIP.Healthcheck.Port)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesFailUpdated, *elasticIP.Healthcheck.StrikesFail)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesOKUpdated, *elasticIP.Healthcheck.StrikesOK)
						a.Equal(testAccResourceElasticIPHealthcheckTLSSNI, *elasticIP.Healthcheck.TLSSNI)
						a.True(*elasticIP.Healthcheck.TLSSkipVerify)
						a.Equal(testAccResourceElasticIPHealthcheckTimeoutUpdated, int64(elasticIP.Healthcheck.Timeout.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckURIUpdated, *elasticIP.Healthcheck.URI)

						return nil
					},
					checkResourceState(r6, checkResourceStateValidateAttributes(testAttrs{
						resElasticIPAttrDescription:                                           validateString(testAccResourceElasticIPDescriptionUpdated),
						resElasticIPAttrAddressFamily:                                         validateString(testAccResourceElasticIPAddressFamily6),
						resElasticIPAttrCIDR:                                                  validation.ToDiagFunc(validation.IsCIDR),
						resElasticIPAttrIPAddress:                                             validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrLabels + ".test":                                      validateString(testAccResourceElasticIPLabelValueUpdated),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckInterval):      validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckIntervalUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckMode):          validateString(testAccResourceElasticIPHealthcheckModeUpdated),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckPort):          validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckPortUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesFail):   validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesFailUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesOK):     validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesOKUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSNI):        validateString(testAccResourceElasticIPHealthcheckTLSSNI),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSkipVerify): validateString("true"),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTimeout):       validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckTimeoutUpdated)),
						resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckURI):           validateString(testAccResourceElasticIPHealthcheckURIUpdated),
					})),
				),
			},
			{
				// Import
				ResourceName:      r6,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(elasticIP *egoscale.ElasticIP) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *elasticIP.ID, testZoneName), nil
					}
				}(&elasticIP),
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resElasticIPAttrDescription:                                           validateString(testAccResourceElasticIPDescriptionUpdated),
							resElasticIPAttrAddressFamily:                                         validateString(testAccResourceElasticIPAddressFamily6),
							resElasticIPAttrCIDR:                                                  validation.ToDiagFunc(validation.IsCIDR),
							resElasticIPAttrIPAddress:                                             validation.ToDiagFunc(validation.IsIPAddress),
							resElasticIPAttrLabels + ".test":                                      validateString(testAccResourceElasticIPLabelValueUpdated),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckInterval):      validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckIntervalUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckMode):          validateString(testAccResourceElasticIPHealthcheckModeUpdated),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckPort):          validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckPortUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesFail):   validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesFailUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesOK):     validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckStrikesOKUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSNI):        validateString(testAccResourceElasticIPHealthcheckTLSSNI),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSkipVerify): validateString("true"),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTimeout):       validateString(fmt.Sprint(testAccResourceElasticIPHealthcheckTimeoutUpdated)),
							resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckURI):           validateString(testAccResourceElasticIPHealthcheckURIUpdated),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceElasticIPExists(r string, elasticIP *egoscale.ElasticIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))
		res, err := client.GetElasticIP(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*elasticIP = *res
		return nil
	}
}

func testAccCheckResourceElasticIPDestroy(elasticIP *egoscale.ElasticIP) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

		_, err := client.GetElasticIP(ctx, testZoneName, *elasticIP.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Elastic IP still exists")
	}
}
