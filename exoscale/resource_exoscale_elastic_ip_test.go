package exoscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
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
	testAccResourceElasticIPReverseDNS                           = "tf-provider-test.exoscale.com"
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

	reverse_dns = "%s"

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
		testAccResourceElasticIPReverseDNS,
		testAccResourceElasticIPLabelValueUpdated,
	)
	// Change address_family (replace the existing resource)
	testAccResourceElasticIP4ConfigUpdate2 = fmt.Sprintf(`
resource "exoscale_elastic_ip" "test4" {
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

	reverse_dns = "%s"

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
		testAccResourceElasticIPReverseDNS,
		testAccResourceElasticIPLabelValueUpdated,
	)
	testAccResourceElasticIP6ConfigCreate = fmt.Sprintf(`
resource "exoscale_elastic_ip" "test6" {
  zone        = "%s"
  description = "%s"
	address_family = "%s"

	reverse_dns = "%s"

  labels = {
    test = "%s"
  }
}
`,
		testZoneName,
		testAccResourceElasticIPDescription,
		testAccResourceElasticIPAddressFamily6,
		testAccResourceElasticIPReverseDNS,
		testAccResourceElasticIPLabelValue,
	)

	testAccResourceElasticIP6ConfigUpdate = fmt.Sprintf(`
resource "exoscale_elastic_ip" "test6" {
  zone        = "%s"
  description = "%s"
	address_family = "%s"

	reverse_dns = ""

  labels = {
    test = "%s"
  }
}
`,
		testZoneName,
		testAccResourceElasticIPDescriptionUpdated,
		testAccResourceElasticIPAddressFamily6,
		testAccResourceElasticIPLabelValueUpdated,
	)
)

func TestAccResourceElasticIP(t *testing.T) {
	var (
		r4          = "exoscale_elastic_ip.test4"
		r6          = "exoscale_elastic_ip.test6"
		elasticIP4  egoscale.ElasticIP
		elasticIP6  egoscale.ElasticIP
		ElasticIPID string // After the update of healthcheck, ID must be the same
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceElasticIPDestroy(&elasticIP4),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceElasticIP4ConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceElasticIPExists(r4, &elasticIP4),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceElasticIPAddressFamily4, *elasticIP4.AddressFamily)
						a.Equal(testAccResourceElasticIPDescription, *elasticIP4.Description)
						a.NotNil(elasticIP4.Healthcheck)
						a.Equal(testAccResourceElasticIPHealthcheckInterval, int64(elasticIP4.Healthcheck.Interval.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckMode, *elasticIP4.Healthcheck.Mode)
						a.Equal(testAccResourceElasticIPHealthcheckPort, *elasticIP4.Healthcheck.Port)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesFail, *elasticIP4.Healthcheck.StrikesFail)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesOK, *elasticIP4.Healthcheck.StrikesOK)
						a.Equal(testAccResourceElasticIPHealthcheckTimeout, int64(elasticIP4.Healthcheck.Timeout.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckURI, *elasticIP4.Healthcheck.URI)
						ElasticIPID = *elasticIP4.ID

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
				// Update Healthcheck and description (update the resource)
				Config: testAccResourceElasticIP4ConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceElasticIPExists(r4, &elasticIP4),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceElasticIPDescriptionUpdated, *elasticIP4.Description)
						a.Equal(ElasticIPID, *elasticIP4.ID)
						a.NotNil(elasticIP4.Healthcheck)
						a.Equal(testAccResourceElasticIPHealthcheckIntervalUpdated, int64(elasticIP4.Healthcheck.Interval.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckModeUpdated, *elasticIP4.Healthcheck.Mode)
						a.Equal(testAccResourceElasticIPHealthcheckPortUpdated, *elasticIP4.Healthcheck.Port)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesFailUpdated, *elasticIP4.Healthcheck.StrikesFail)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesOKUpdated, *elasticIP4.Healthcheck.StrikesOK)
						a.Equal(testAccResourceElasticIPHealthcheckTLSSNI, *elasticIP4.Healthcheck.TLSSNI)
						a.True(*elasticIP4.Healthcheck.TLSSkipVerify)
						a.Equal(testAccResourceElasticIPHealthcheckTimeoutUpdated, int64(elasticIP4.Healthcheck.Timeout.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckURIUpdated, *elasticIP4.Healthcheck.URI)

						return nil
					},
					checkResourceState(r4, checkResourceStateValidateAttributes(testAttrs{
						resElasticIPAttrDescription:                                           validateString(testAccResourceElasticIPDescriptionUpdated),
						resElasticIPAttrAddressFamily:                                         validateString(testAccResourceElasticIPAddressFamily4),
						resElasticIPAttrCIDR:                                                  validation.ToDiagFunc(validation.IsCIDR),
						resElasticIPAttrIPAddress:                                             validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrReverseDNS:                                            validateString(testAccResourceElasticIPReverseDNS),
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
						return fmt.Sprintf("%s@%s", *elasticIP4.ID, testZoneName), nil
					}
				}(&elasticIP4),
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resElasticIPAttrDescription:                                           validateString(testAccResourceElasticIPDescriptionUpdated),
							resElasticIPAttrAddressFamily:                                         validateString(testAccResourceElasticIPAddressFamily4),
							resElasticIPAttrCIDR:                                                  validation.ToDiagFunc(validation.IsCIDR),
							resElasticIPAttrIPAddress:                                             validation.ToDiagFunc(validation.IsIPAddress),
							resElasticIPAttrReverseDNS:                                            validateString(testAccResourceElasticIPReverseDNS),
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
			{
				// Update Address Family (new resource)
				Config: testAccResourceElasticIP4ConfigUpdate2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceElasticIPExists(r4, &elasticIP6),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceElasticIPDescriptionUpdated, *elasticIP6.Description)
						a.NotEqual(ElasticIPID, *elasticIP6.ID)
						a.NotNil(elasticIP6.Healthcheck)
						a.Equal(testAccResourceElasticIPHealthcheckIntervalUpdated, int64(elasticIP6.Healthcheck.Interval.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckModeUpdated, *elasticIP6.Healthcheck.Mode)
						a.Equal(testAccResourceElasticIPHealthcheckPortUpdated, *elasticIP6.Healthcheck.Port)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesFailUpdated, *elasticIP6.Healthcheck.StrikesFail)
						a.Equal(testAccResourceElasticIPHealthcheckStrikesOKUpdated, *elasticIP6.Healthcheck.StrikesOK)
						a.Equal(testAccResourceElasticIPHealthcheckTLSSNI, *elasticIP6.Healthcheck.TLSSNI)
						a.True(*elasticIP6.Healthcheck.TLSSkipVerify)
						a.Equal(testAccResourceElasticIPHealthcheckTimeoutUpdated, int64(elasticIP6.Healthcheck.Timeout.Seconds()))
						a.Equal(testAccResourceElasticIPHealthcheckURIUpdated, *elasticIP6.Healthcheck.URI)

						return nil
					},
					checkResourceState(r4, checkResourceStateValidateAttributes(testAttrs{
						resElasticIPAttrDescription:                                           validateString(testAccResourceElasticIPDescriptionUpdated),
						resElasticIPAttrAddressFamily:                                         validateString(testAccResourceElasticIPAddressFamily6),
						resElasticIPAttrCIDR:                                                  validation.ToDiagFunc(validation.IsCIDR),
						resElasticIPAttrIPAddress:                                             validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrReverseDNS:                                            validateString(testAccResourceElasticIPReverseDNS),
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
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceElasticIPDestroy(&elasticIP6),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceElasticIP6ConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceElasticIPExists(r6, &elasticIP6),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceElasticIPAddressFamily6, *elasticIP6.AddressFamily)
						a.Equal(testAccResourceElasticIPDescription, *elasticIP6.Description)

						return nil
					},
					checkResourceState(r6, checkResourceStateValidateAttributes(testAttrs{
						resElasticIPAttrDescription:      validateString(testAccResourceElasticIPDescription),
						resElasticIPAttrAddressFamily:    validateString(testAccResourceElasticIPAddressFamily6),
						resElasticIPAttrCIDR:             validation.ToDiagFunc(validation.IsCIDR),
						resElasticIPAttrIPAddress:        validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrReverseDNS:       validateString(testAccResourceElasticIPReverseDNS),
						resElasticIPAttrLabels + ".test": validateString(testAccResourceElasticIPLabelValue),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceElasticIP6ConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceElasticIPExists(r6, &elasticIP6),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceElasticIPDescriptionUpdated, *elasticIP6.Description)

						return nil
					},
					checkResourceState(r6, checkResourceStateValidateAttributes(testAttrs{
						resElasticIPAttrDescription:   validateString(testAccResourceElasticIPDescriptionUpdated),
						resElasticIPAttrAddressFamily: validateString(testAccResourceElasticIPAddressFamily6),
						//resElasticIPAttrReverseDNS:                                            validation.ToDiagFunc(validation.StringIsEmpty),
						resElasticIPAttrCIDR:             validation.ToDiagFunc(validation.IsCIDR),
						resElasticIPAttrIPAddress:        validation.ToDiagFunc(validation.IsIPAddress),
						resElasticIPAttrLabels + ".test": validateString(testAccResourceElasticIPLabelValueUpdated),
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
						return fmt.Sprintf("%s@%s", *elasticIP6.ID, testZoneName), nil
					}
				}(&elasticIP6),
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resElasticIPAttrDescription:      validateString(testAccResourceElasticIPDescriptionUpdated),
							resElasticIPAttrAddressFamily:    validateString(testAccResourceElasticIPAddressFamily6),
							resElasticIPAttrCIDR:             validation.ToDiagFunc(validation.IsCIDR),
							resElasticIPAttrIPAddress:        validation.ToDiagFunc(validation.IsIPAddress),
							resElasticIPAttrReverseDNS:       validation.ToDiagFunc(validation.StringIsEmpty),
							resElasticIPAttrLabels + ".test": validateString(testAccResourceElasticIPLabelValueUpdated),
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

		client, err := egoscale.NewClient(
			os.Getenv("EXOSCALE_API_KEY"),
			os.Getenv("EXOSCALE_API_SECRET"),
		)
		if err != nil {
			return err
		}

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
		client, err := egoscale.NewClient(
			os.Getenv("EXOSCALE_API_KEY"),
			os.Getenv("EXOSCALE_API_SECRET"),
		)
		if err != nil {
			return err
		}
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testAccResourceSKSClusterLocalZone),
		)

		_, err = client.GetElasticIP(ctx, testZoneName, *elasticIP.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Elastic IP still exists")
	}
}
