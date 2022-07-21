package exoscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

var (
	testAccResourceSKSClusterLabelValue             = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSKSClusterLabelValueUpdated      = testAccResourceSKSClusterLabelValue + "-updated"
	testAccResourceSKSClusterName                   = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSKSClusterNameUpdated            = testAccResourceSKSClusterName + "-updated"
	testAccResourceSKSClusterOIDCClientID           = acctest.RandString(10)
	testAccResourceSKSClusterOIDCGroupsClaim        = acctest.RandString(10)
	testAccResourceSKSClusterOIDCGroupsPrefix       = acctest.RandString(10)
	testAccResourceSKSClusterOIDCIssuerURL          = "https://id.example.net"
	testAccResourceSKSClusterOIDCRequiredClaimValue = acctest.RandString(10)
	testAccResourceSKSClusterOIDCUsernameClaim      = acctest.RandString(10)
	testAccResourceSKSClusterOIDCUsernamePrefix     = acctest.RandString(10)
	testAccResourceSKSClusterDescription            = acctest.RandString(10)
	testAccResourceSKSClusterDescriptionUpdated     = testAccResourceSKSClusterDescription + "-updated"

	testAccResourceSKSClusterConfigCreate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test" {
  zone = local.zone
  name = "%s"
  description = "%s"
  exoscale_ccm = true
  metrics_server = false
  auto_upgrade = true
  labels = {
    test = "%s"
  }

  oidc {
    client_id  = "%s"
    groups_claim = "%s"
    groups_prefix = "%s"
    issuer_url = "%s"
    required_claim = { test = "%s" }
    username_claim = "%s"
    username_prefix = "%s"
  }

  timeouts {
    create = "10m"
  }
}

resource "exoscale_sks_nodepool" "test" {
  zone = local.zone
  cluster_id = exoscale_sks_cluster.test.id
  name = "test"
  instance_type = "standard.small"
  disk_size = 20
  size = 1

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceSKSClusterName,
		testAccResourceSKSClusterDescription,
		testAccResourceSKSClusterLabelValue,
		testAccResourceSKSClusterOIDCClientID,
		testAccResourceSKSClusterOIDCGroupsClaim,
		testAccResourceSKSClusterOIDCGroupsPrefix,
		testAccResourceSKSClusterOIDCIssuerURL,
		testAccResourceSKSClusterOIDCRequiredClaimValue,
		testAccResourceSKSClusterOIDCUsernameClaim,
		testAccResourceSKSClusterOIDCUsernamePrefix,
	)

	testAccResourceSKSClusterConfigUpdate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test" {
  zone = local.zone
  name = "%s"
  description = "%s"
  exoscale_ccm = true
  metrics_server = false
  auto_upgrade = false
  labels = {
    test = "%s"
  }

  timeouts {
    create = "10m"
  }
}

resource "exoscale_sks_nodepool" "test" {
  zone = local.zone
  cluster_id = exoscale_sks_cluster.test.id
  name = "test"
  instance_type = "standard.small"
  disk_size = 20
  size = 1

  timeouts {
    create = "10m"
  }
}
`,
		testZoneName,
		testAccResourceSKSClusterNameUpdated,
		testAccResourceSKSClusterDescriptionUpdated,
		testAccResourceSKSClusterLabelValueUpdated,
	)
)

func TestAccResourceSKSCluster(t *testing.T) {
	var (
		r          = "exoscale_sks_cluster.test"
		sksCluster egoscale.SKSCluster
	)

	client, err := egoscale.NewClient(
		os.Getenv("EXOSCALE_API_KEY"),
		os.Getenv("EXOSCALE_API_SECRET"),
		egoscale.ClientOptCond(func() bool {
			if v := os.Getenv("EXOSCALE_TRACE"); v != "" {
				return true
			}
			return false
		}, egoscale.ClientOptWithTrace()))
	if err != nil {
		t.Fatalf("unable to initialize Exoscale client: %s", err)
	}
	clientctx := exoapi.WithEndpoint(
		context.Background(),
		exoapi.NewReqEndpoint(os.Getenv("EXOSCALE_API_ENVIRONMENT"), testZoneName),
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceSKSClusterConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						// Retrieve the latest SKS version available to test the
						// exoscale_sks_cluster.version attribute default value.
						versions, err := client.ListSKSClusterVersions(clientctx)
						if err != nil || len(versions) == 0 {
							if len(versions) == 0 {
								err = errors.New("no version returned by the API")
							}
							return fmt.Errorf("unable to retrieve SKS versions: %s", err)
						}
						latestVersion := versions[0]

						a.Equal([]string{sksClusterAddonExoscaleCCM}, *sksCluster.AddOns)
						a.True(defaultBool(sksCluster.AutoUpgrade, false))
						a.Equal(defaultSKSClusterCNI, *sksCluster.CNI)
						a.Equal(testAccResourceSKSClusterDescription, *sksCluster.Description)
						a.Equal(testAccResourceSKSClusterLabelValue, (*sksCluster.Labels)["test"])
						a.Equal(testAccResourceSKSClusterName, *sksCluster.Name)
						a.Equal(defaultSKSClusterServiceLevel, *sksCluster.ServiceLevel)
						a.Equal(latestVersion, *sksCluster.Version)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrAggregationLayerCA: validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Aggregation CA must be a PEM certificate")),
						resSKSClusterAttrAutoUpgrade:        validateString("true"),
						resSKSClusterAttrCNI:                validateString(defaultSKSClusterCNI),
						resSKSClusterAttrControlPlaneCA:     validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Control-plane CA must be a PEM certificate")),
						resSKSClusterAttrCreatedAt:          validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrDescription:        validateString(testAccResourceSKSClusterDescription),
						resSKSClusterAttrEndpoint:           validation.ToDiagFunc(validation.IsURLWithHTTPS),
						resSKSClusterAttrExoscaleCCM:        validateString("true"),
						resSKSClusterAttrKubeletCA:          validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Kubelet CA must be a PEM certificate")),
						resSKSClusterAttrMetricsServer:      validateString("false"),
						resSKSClusterAttrLabels + ".test":   validateString(testAccResourceSKSClusterLabelValue),
						resSKSClusterAttrName:               validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrServiceLevel:       validateString(defaultSKSClusterServiceLevel),
						resSKSClusterAttrState:              validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrVersion:            validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceSKSClusterConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.False(defaultBool(sksCluster.AutoUpgrade, false))
						a.Equal(testAccResourceSKSClusterDescriptionUpdated, *sksCluster.Description)
						a.Equal(testAccResourceSKSClusterLabelValueUpdated, (*sksCluster.Labels)["test"])
						a.Equal(testAccResourceSKSClusterNameUpdated, *sksCluster.Name)

						// Wait for the cluster to be in the Running state
						for i := 0; i < 60; i++ {
							c, err := client.GetSKSCluster(clientctx, testZoneName, *sksCluster.ID)
							if err != nil {
								return fmt.Errorf("failed to fetch sks cluster: %s", err)
							}
							if *c.State == "running" {
								return nil
							}
							t.Logf("waiting for cluster to be Running, current state: %s", *c.State)
							time.Sleep(10 * time.Second)
						}

						return fmt.Errorf("timeout waiting for the cluster to be Running: current state")
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrAggregationLayerCA: validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Aggregation CA must be a PEM certificate")),
						resSKSClusterAttrAutoUpgrade:        validateString("false"),
						resSKSClusterAttrCNI:                validateString(defaultSKSClusterCNI),
						resSKSClusterAttrControlPlaneCA:     validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Control-plane CA must be a PEM certificate")),
						resSKSClusterAttrCreatedAt:          validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrDescription:        validateString(testAccResourceSKSClusterDescriptionUpdated),
						resSKSClusterAttrEndpoint:           validation.ToDiagFunc(validation.IsURLWithHTTPS),
						resSKSClusterAttrExoscaleCCM:        validateString("true"),
						resSKSClusterAttrKubeletCA:          validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Kubelet CA must be a PEM certificate")),
						resSKSClusterAttrMetricsServer:      validateString("false"),
						resSKSClusterAttrLabels + ".test":   validateString(testAccResourceSKSClusterLabelValueUpdated),
						resSKSClusterAttrName:               validateString(testAccResourceSKSClusterNameUpdated),
						resSKSClusterAttrServiceLevel:       validateString(defaultSKSClusterServiceLevel),
						resSKSClusterAttrState:              validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrVersion:            validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(sksCluster *egoscale.SKSCluster) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *sksCluster.ID, testZoneName), nil
					}
				}(&sksCluster),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"oidc.#",
					"oidc.0.%",
					resSKSClusterAttrOIDC(resSKSClusterAttrOIDCClientID),
					resSKSClusterAttrOIDC(resSKSClusterAttrOIDCGroupsClaim),
					resSKSClusterAttrOIDC(resSKSClusterAttrOIDCGroupsPrefix),
					resSKSClusterAttrOIDC(resSKSClusterAttrOIDCIssuerURL),
					resSKSClusterAttrOIDC(resSKSClusterAttrOIDCRequiredClaim),
					resSKSClusterAttrOIDC(resSKSClusterAttrOIDCUsernameClaim),
					resSKSClusterAttrOIDC(resSKSClusterAttrOIDCUsernamePrefix),
				},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resSKSClusterAttrAggregationLayerCA: validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Aggregation CA must be a PEM certificate")),
							resSKSClusterAttrAutoUpgrade:        validateString("false"),
							resSKSClusterAttrCNI:                validateString(defaultSKSClusterCNI),
							resSKSClusterAttrControlPlaneCA:     validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Control-plane CA must be a PEM certificate")),
							resSKSClusterAttrCreatedAt:          validation.ToDiagFunc(validation.NoZeroValues),
							resSKSClusterAttrDescription:        validateString(testAccResourceSKSClusterDescriptionUpdated),
							resSKSClusterAttrEndpoint:           validation.ToDiagFunc(validation.IsURLWithHTTPS),
							resSKSClusterAttrExoscaleCCM:        validateString("true"),
							resSKSClusterAttrKubeletCA:          validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Kubelet CA must be a PEM certificate")),
							resSKSClusterAttrMetricsServer:      validateString("false"),
							resSKSClusterAttrLabels + ".test":   validateString(testAccResourceSKSClusterLabelValueUpdated),
							resSKSClusterAttrName:               validateString(testAccResourceSKSClusterNameUpdated),
							resSKSClusterAttrServiceLevel:       validateString(defaultSKSClusterServiceLevel),
							resSKSClusterAttrState:              validation.ToDiagFunc(validation.NoZeroValues),
							resSKSClusterAttrVersion:            validation.ToDiagFunc(validation.NoZeroValues),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSKSClusterExists(r string, sksCluster *egoscale.SKSCluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)
		res, err := client.GetSKSCluster(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*sksCluster = *res
		return nil
	}
}

func testAccCheckResourceSKSClusterDestroy(sksCluster *egoscale.SKSCluster) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		_, err := client.GetSKSCluster(ctx, testZoneName, *sksCluster.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("SKS cluster still exists")
	}
}
