package exoscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

var (
	testAccResourceSKSClusterLocalZone              = "ch-gva-2" // TODO: replace with testZoneName when blockstorage becomes available in all zones
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
	testAccResourceSKSClusterFeatureGate            = "GracefulNodeShutdown"

	// For OIDC update testing
	testAccResourceSKSClusterOIDCClientIDUpdated           = testAccResourceSKSClusterOIDCClientID + "-updated"
	testAccResourceSKSClusterOIDCGroupsClaimUpdated        = testAccResourceSKSClusterOIDCGroupsClaim + "-updated"
	testAccResourceSKSClusterOIDCGroupsPrefixUpdated       = testAccResourceSKSClusterOIDCGroupsPrefix + "-updated"
	testAccResourceSKSClusterOIDCIssuerURLUpdated          = "https://id-updated.example.net"
	testAccResourceSKSClusterOIDCRequiredClaimValueUpdated = testAccResourceSKSClusterOIDCRequiredClaimValue + "-updated"
	testAccResourceSKSClusterOIDCUsernameClaimUpdated      = testAccResourceSKSClusterOIDCUsernameClaim + "-updated"
	testAccResourceSKSClusterOIDCUsernamePrefixUpdated     = testAccResourceSKSClusterOIDCUsernamePrefix + "-updated"

	// Only in audit testing scenario
	testAccResourceSKSClusterAuditInitBackoff = "30s"
	testAccResourceSKSClusterAuditRemoteURL   = "https://audit.example.exoscale.net"
	testAccResourceSKSClusterAuditBearerToken = "supersecretbearertoken"
	// For re-enable audit test with new URL
	testAccResourceSKSClusterAuditRemoteURLUpdated   = "https://audit-updated.example.exoscale.net"
	testAccResourceSKSClusterAuditBearerTokenUpdated = "newsupersecretbearertoken"

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
  feature_gates = ["%s"]

  enable_kube_proxy = true

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
		testAccResourceSKSClusterLocalZone,
		testAccResourceSKSClusterName,
		testAccResourceSKSClusterDescription,
		testAccResourceSKSClusterLabelValue,
		testAccResourceSKSClusterFeatureGate,
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
  exoscale_csi = true
  metrics_server = false
  auto_upgrade = true
  labels = {
    test = "%s"
  }
  feature_gates = []

  enable_kube_proxy = true

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
		testAccResourceSKSClusterLocalZone,
		testAccResourceSKSClusterNameUpdated,
		testAccResourceSKSClusterDescriptionUpdated,
		testAccResourceSKSClusterLabelValueUpdated,
	)

	testAccResourceSKSClusterConfig2Format = `
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test" {
  zone = local.zone
  name = "%s"
  auto_upgrade = false
  enable_kube_proxy = true
  version = "%s"

  timeouts {
    create = "10m"
  }
}`
	testAccRessourceSKSClusterCreateWithAudit = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test-with-audit" {
  zone = local.zone
  name = "%s"
  description = "%s"
  exoscale_ccm = true
  metrics_server = false
  auto_upgrade = true
  labels = {
    test = "%s"
  }

  audit {
    enabled = true
    endpoint = "%s"
	initial_backoff = "%s"
	bearer_token = "%s"
  }

  enable_kube_proxy = true

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
`,
		testAccResourceSKSClusterLocalZone,
		testAccResourceSKSClusterName,
		testAccResourceSKSClusterDescription,
		testAccResourceSKSClusterLabelValue,
		testAccResourceSKSClusterAuditRemoteURL,
		testAccResourceSKSClusterAuditInitBackoff,
		testAccResourceSKSClusterAuditBearerToken,
		testAccResourceSKSClusterOIDCClientIDUpdated,
		testAccResourceSKSClusterOIDCGroupsClaimUpdated,
		testAccResourceSKSClusterOIDCGroupsPrefixUpdated,
		testAccResourceSKSClusterOIDCIssuerURLUpdated,
		testAccResourceSKSClusterOIDCRequiredClaimValueUpdated,
		testAccResourceSKSClusterOIDCUsernameClaimUpdated,
		testAccResourceSKSClusterOIDCUsernamePrefixUpdated,
	)

	testAccRessourceSKSClusterUpdateDisableAudit = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test-with-audit" {
  zone = local.zone
  name = "%s"
  description = "%s"
  exoscale_ccm = true
  metrics_server = false
  auto_upgrade = true
  labels = {
    test = "%s"
  }

  audit {
    enabled = false
  }

  timeouts {
    create = "10m"
  }
}
`,
		testAccResourceSKSClusterLocalZone,
		testAccResourceSKSClusterName,
		testAccResourceSKSClusterDescription,
		testAccResourceSKSClusterLabelValue,
	)

	testAccRessourceSKSClusterReEnableAuditWithNewURL = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test-with-audit" {
  zone = local.zone
  name = "%s"
  description = "%s"
  exoscale_ccm = true
  metrics_server = false
  auto_upgrade = true
  labels = {
    test = "%s"
  }

  audit {
    enabled = true
    endpoint = "%s"
    bearer_token = "%s"
  }

  timeouts {
    create = "10m"
  }
}
`,
		testAccResourceSKSClusterLocalZone,
		testAccResourceSKSClusterName,
		testAccResourceSKSClusterDescription,
		testAccResourceSKSClusterLabelValue,
		testAccResourceSKSClusterAuditRemoteURLUpdated,
		testAccResourceSKSClusterAuditBearerTokenUpdated,
	)

	testAccResourceSKSClusterCreateWithKarpenter = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test-with-karpenter" {
  zone = local.zone
  name = "%s"
  description = "%s"
  exoscale_ccm = true
  metrics_server = false
  auto_upgrade = true
  labels = {
    test = "%s"
  }

  enable_karpenter = true

  timeouts {
    create = "10m"
  }
}
`,
		testAccResourceSKSClusterLocalZone,
		testAccResourceSKSClusterName,
		testAccResourceSKSClusterDescription,
		testAccResourceSKSClusterLabelValue,
	)

	testAccResourceSKSClusterUpdateDisableKarpenter = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test-with-karpenter" {
  zone = local.zone
  name = "%s"
  description = "%s"
  exoscale_ccm = true
  metrics_server = false
  auto_upgrade = true
  labels = {
    test = "%s"
  }

  enable_karpenter = false

  timeouts {
    create = "10m"
  }
}
`,
		testAccResourceSKSClusterLocalZone,
		testAccResourceSKSClusterName,
		testAccResourceSKSClusterDescription,
		testAccResourceSKSClusterLabelValue,
	)

	testAccResourceSKSClusterReEnableKarpenter = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test-with-karpenter" {
  zone = local.zone
  name = "%s"
  description = "%s"
  exoscale_ccm = true
  metrics_server = false
  auto_upgrade = true
  labels = {
    test = "%s"
  }

  enable_karpenter = true

  timeouts {
    create = "10m"
  }
}
`,
		testAccResourceSKSClusterLocalZone,
		testAccResourceSKSClusterName,
		testAccResourceSKSClusterDescription,
		testAccResourceSKSClusterLabelValue,
	)
)

func TestAccResourceSKSCluster(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
		return
	}

	var (
		r          = "exoscale_sks_cluster.test"
		sksCluster egoscale.SKSCluster
	)

	versions := testGetSKSClusterVersions(t)

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

						latestVersion := versions[0]

						a.Equal([]string{sksClusterAddonExoscaleCCM}, sksCluster.Addons)
						a.True(defaultBool(sksCluster.AutoUpgrade, false))
						a.Equal(defaultSKSClusterCNI, string(sksCluster.Cni))
						a.Equal(testAccResourceSKSClusterDescription, sksCluster.Description)
						a.Equal(testAccResourceSKSClusterLabelValue, sksCluster.Labels["test"])
						a.Equal(testAccResourceSKSClusterName, sksCluster.Name)
						a.Equal(defaultSKSClusterServiceLevel, string(sksCluster.Level))
						a.Equal(latestVersion, sksCluster.Version)
						a.Len(sksCluster.FeatureGates, 1)
						a.Equal(testAccResourceSKSClusterFeatureGate, sksCluster.FeatureGates[0])
						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrAggregationLayerCA:  validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Aggregation CA must be a PEM certificate")),
						resSKSClusterAttrAutoUpgrade:         validateString("true"),
						resSKSClusterAttrCNI:                 validateString(defaultSKSClusterCNI),
						resSKSClusterAttrControlPlaneCA:      validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Control-plane CA must be a PEM certificate")),
						resSKSClusterAttrCreatedAt:           validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrDescription:         validateString(testAccResourceSKSClusterDescription),
						resSKSClusterAttrEndpoint:            validation.ToDiagFunc(isDNSName),
						resSKSClusterAttrEnableKubeProxy:     validateString("true"),
						resSKSClusterAttrExoscaleCCM:         validateString("true"),
						resSKSClusterAttrFeatureGates + ".#": validateString("1"),
						resSKSClusterAttrKubeletCA:           validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Kubelet CA must be a PEM certificate")),
						resSKSClusterAttrMetricsServer:       validateString("false"),
						resSKSClusterAttrExoscaleCSI:         validateString("false"),
						resSKSClusterAttrLabels + ".test":    validateString(testAccResourceSKSClusterLabelValue),
						resSKSClusterAttrName:                validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrServiceLevel:        validateString(defaultSKSClusterServiceLevel),
						resSKSClusterAttrState:               validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrVersion:             validation.ToDiagFunc(validation.NoZeroValues),
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

						a.Len(sksCluster.FeatureGates, 0)
						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrAggregationLayerCA:  validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Aggregation CA must be a PEM certificate")),
						resSKSClusterAttrAutoUpgrade:         validateString("true"),
						resSKSClusterAttrCNI:                 validateString(defaultSKSClusterCNI),
						resSKSClusterAttrControlPlaneCA:      validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Control-plane CA must be a PEM certificate")),
						resSKSClusterAttrCreatedAt:           validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrDescription:         validateString(testAccResourceSKSClusterDescriptionUpdated),
						resSKSClusterAttrEndpoint:            validation.ToDiagFunc(isDNSName),
						resSKSClusterAttrExoscaleCCM:         validateString("true"),
						resSKSClusterAttrFeatureGates + ".#": validateString("0"),
						resSKSClusterAttrKubeletCA:           validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Kubelet CA must be a PEM certificate")),
						resSKSClusterAttrMetricsServer:       validateString("false"),
						resSKSClusterAttrExoscaleCSI:         validateString("true"),
						resSKSClusterAttrLabels + ".test":    validateString(testAccResourceSKSClusterLabelValueUpdated),
						resSKSClusterAttrName:                validateString(testAccResourceSKSClusterNameUpdated),
						resSKSClusterAttrServiceLevel:        validateString(defaultSKSClusterServiceLevel),
						resSKSClusterAttrState:               validation.ToDiagFunc(validation.NoZeroValues),

						resSKSClusterAttrOIDC(resSKSClusterAttrOIDCClientID):       validateString(testAccResourceSKSClusterOIDCClientIDUpdated),
						resSKSClusterAttrOIDC(resSKSClusterAttrOIDCGroupsClaim):    validateString(testAccResourceSKSClusterOIDCGroupsClaimUpdated),
						resSKSClusterAttrOIDC(resSKSClusterAttrOIDCGroupsPrefix):   validateString(testAccResourceSKSClusterOIDCGroupsPrefixUpdated),
						resSKSClusterAttrOIDC(resSKSClusterAttrOIDCIssuerURL):      validateString(testAccResourceSKSClusterOIDCIssuerURLUpdated),
						resSKSClusterAttrOIDC(resSKSClusterAttrOIDCUsernameClaim):  validateString(testAccResourceSKSClusterOIDCUsernameClaimUpdated),
						resSKSClusterAttrOIDC(resSKSClusterAttrOIDCUsernamePrefix): validateString(testAccResourceSKSClusterOIDCUsernamePrefixUpdated),
					})),
				),
			},
			{
				// Update again (remove OIDC and add CSI)
				Config: testAccResourceSKSClusterConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrExoscaleCSI:      validateString("true"),
						resSKSClusterAttrLabels + ".test": validateString(testAccResourceSKSClusterLabelValueUpdated),
						resSKSClusterAttrName:             validateString(testAccResourceSKSClusterNameUpdated),
						resSKSClusterAttrServiceLevel:     validateString(defaultSKSClusterServiceLevel),
						resSKSClusterAttrState:            validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(sksCluster *egoscale.SKSCluster) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", sksCluster.ID, testAccResourceSKSClusterLocalZone), nil
					}
				}(&sksCluster),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"oidc.#",
					"oidc.0.%",
					"addons",
					resSKSClusterAttrEnableKubeProxy,
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
							resSKSClusterAttrAutoUpgrade:        validateString("true"),
							resSKSClusterAttrCNI:                validateString(defaultSKSClusterCNI),
							resSKSClusterAttrControlPlaneCA:     validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Control-plane CA must be a PEM certificate")),
							resSKSClusterAttrCreatedAt:          validation.ToDiagFunc(validation.NoZeroValues),
							resSKSClusterAttrDescription:        validateString(testAccResourceSKSClusterDescriptionUpdated),
							resSKSClusterAttrEndpoint:           validation.ToDiagFunc(isDNSName),
							resSKSClusterAttrExoscaleCCM:        validateString("true"),
							resSKSClusterAttrKubeletCA:          validation.ToDiagFunc(validation.StringMatch(testPemCertificateFormatRegex, "Kubelet CA must be a PEM certificate")),
							resSKSClusterAttrMetricsServer:      validateString("false"),
							resSKSClusterAttrExoscaleCSI:        validateString("true"),
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

	// Test cluster Upgrade
	sksCluster = egoscale.SKSCluster{}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				// Create old version cluster
				Config: fmt.Sprintf(testAccResourceSKSClusterConfig2Format, testAccResourceSKSClusterLocalZone, testAccResourceSKSClusterName, versions[1]),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceSKSClusterName, sksCluster.Name)
						a.Equal(versions[1], sksCluster.Version)
						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrAutoUpgrade: validateString("false"),
						resSKSClusterAttrName:        validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrState:       validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrVersion:     validateString(versions[1]),
					})),
				),
			},
			{
				// Upgrade cluster
				Config: fmt.Sprintf(testAccResourceSKSClusterConfig2Format, testAccResourceSKSClusterLocalZone, testAccResourceSKSClusterName, versions[0]),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceSKSClusterName, sksCluster.Name)
						a.Equal(versions[0], sksCluster.Version)
						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrAutoUpgrade: validateString("false"),
						resSKSClusterAttrName:        validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrState:       validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrVersion:     validateString(versions[0]),
					})),
				),
			},
		},
	})
}

func TestAccResourceSKSClusterSKSClusterWithAudit(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
		return
	}

	var (
		r          = "exoscale_sks_cluster.test-with-audit"
		sksCluster egoscale.SKSCluster
	)

	versions := testGetSKSClusterVersions(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				// Create cluster with audit enabled
				Config: testAccRessourceSKSClusterCreateWithAudit,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(testAccResourceSKSClusterName, sksCluster.Name)
						a.Equal(testAccResourceSKSClusterDescription, sksCluster.Description)

						// Verify audit is enabled in the API response
						assert.NotNil(t, sksCluster.Audit)
						if sksCluster.Audit != nil {
							if a.NotNil(sksCluster.Audit.Enabled) {
								a.True(*sksCluster.Audit.Enabled)
							}
							a.Equal(testAccResourceSKSClusterAuditRemoteURL, string(sksCluster.Audit.Endpoint))
							a.Equal(testAccResourceSKSClusterAuditInitBackoff, string(sksCluster.Audit.InitialBackoff))
						} else {
							t.Error("Audit should not be nil when audit is enabled")
						}

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrName:                                     validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrDescription:                              validateString(testAccResourceSKSClusterDescription),
						resSKSClusterAttrAutoUpgrade:                              validateString("true"),
						resSKSClusterAttrExoscaleCCM:                              validateString("true"),
						resSKSClusterAttrMetricsServer:                            validateString("false"),
						resSKSClusterAttrLabels + ".test":                         validateString(testAccResourceSKSClusterLabelValue),
						resSKSClusterAttrAudit(resSKSClusterAttrAuditEnabled):     validateString("true"),
						resSKSClusterAttrAudit(resSKSClusterAttrAuditEndpoint):    validateString(testAccResourceSKSClusterAuditRemoteURL),
						resSKSClusterAttrAudit(resSKSClusterAttrAuditInitBackoff): validateString(testAccResourceSKSClusterAuditInitBackoff),
						resSKSClusterAttrAudit(resSKSClusterAttrAuditBearerToken): validateString(testAccResourceSKSClusterAuditBearerToken),
					})),
				),
			},
			{
				// Update cluster to disable audit
				Config: testAccRessourceSKSClusterUpdateDisableAudit,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(testAccResourceSKSClusterName, sksCluster.Name)

						// Verify audit is disabled in the API response
						if assert.NotNil(t, sksCluster.Audit) {
							a.False(*sksCluster.Audit.Enabled)
						}

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrName:                                 validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrDescription:                          validateString(testAccResourceSKSClusterDescription),
						resSKSClusterAttrAutoUpgrade:                          validateString("true"),
						resSKSClusterAttrExoscaleCCM:                          validateString("true"),
						resSKSClusterAttrMetricsServer:                        validateString("false"),
						resSKSClusterAttrLabels + ".test":                     validateString(testAccResourceSKSClusterLabelValue),
						resSKSClusterAttrAudit(resSKSClusterAttrAuditEnabled): validateString("false"),
					})),
				),
			},
			{
				// Re-enable audit with new URL and default backoff
				Config: testAccRessourceSKSClusterReEnableAuditWithNewURL,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(testAccResourceSKSClusterName, sksCluster.Name)

						// Verify audit is enabled again with new URL
						assert.NotNil(t, sksCluster.Audit)
						if sksCluster.Audit != nil {
							a.True(*sksCluster.Audit.Enabled)
							a.Equal(testAccResourceSKSClusterAuditRemoteURLUpdated, string(sksCluster.Audit.Endpoint))
							// Backoff should use default value when not specified in config
							a.NotEmpty(string(sksCluster.Audit.InitialBackoff))
						} else {
							t.Error("Audit should not be nil when audit is re-enabled")
						}

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrName:                                     validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrDescription:                              validateString(testAccResourceSKSClusterDescription),
						resSKSClusterAttrAutoUpgrade:                              validateString("true"),
						resSKSClusterAttrExoscaleCCM:                              validateString("true"),
						resSKSClusterAttrMetricsServer:                            validateString("false"),
						resSKSClusterAttrLabels + ".test":                         validateString(testAccResourceSKSClusterLabelValue),
						resSKSClusterAttrAudit(resSKSClusterAttrAuditEnabled):     validateString("true"),
						resSKSClusterAttrAudit(resSKSClusterAttrAuditEndpoint):    validateString(testAccResourceSKSClusterAuditRemoteURLUpdated),
						resSKSClusterAttrAudit(resSKSClusterAttrAuditInitBackoff): validateString(defaultSKSClusterAuditInitBackoff),
						resSKSClusterAttrAudit(resSKSClusterAttrAuditBearerToken): validateString(testAccResourceSKSClusterAuditBearerTokenUpdated),
					})),
				),
			},
		},
	})
}

func TestAccResourceSKSClusterWithKarpenter(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
		return
	}

	var (
		r          = "exoscale_sks_cluster.test-with-karpenter"
		sksCluster egoscale.SKSCluster
	)

	versions := testGetSKSClusterVersions(t)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				// Create cluster with Karpenter enabled
				Config: testAccResourceSKSClusterCreateWithKarpenter,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(testAccResourceSKSClusterName, sksCluster.Name)
						a.Equal(testAccResourceSKSClusterDescription, sksCluster.Description)

						// Verify Karpenter addon is present in the API response
						assert.NotNil(t, sksCluster.Addons)
						if sksCluster.Addons != nil {
							found := false
							for _, addon := range sksCluster.Addons {
								if addon == sksClusterAddonKarpenter {
									found = true
									break
								}
							}
							a.True(found, "Karpenter addon should be present when enabled")
						} else {
							t.Error("Addons should not be nil when Karpenter is enabled")
						}

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrName:             validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrDescription:      validateString(testAccResourceSKSClusterDescription),
						resSKSClusterAttrAutoUpgrade:      validateString("true"),
						resSKSClusterAttrExoscaleCCM:      validateString("true"),
						resSKSClusterAttrMetricsServer:    validateString("false"),
						resSKSClusterAttrLabels + ".test": validateString(testAccResourceSKSClusterLabelValue),
						resSKSClusterAttrEnableKarpenter:  validateString("true"),
					})),
				),
			},
			{
				// Update cluster to disable Karpenter
				Config: testAccResourceSKSClusterUpdateDisableKarpenter,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(testAccResourceSKSClusterName, sksCluster.Name)

						// Verify Karpenter addon is not present in the API response
						if assert.NotNil(t, sksCluster.Addons) {
							found := false
							for _, addon := range sksCluster.Addons {
								if addon == sksClusterAddonKarpenter {
									found = true
									break
								}
							}
							a.False(found, "Karpenter addon should not be present when disabled")
						}

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrName:             validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrDescription:      validateString(testAccResourceSKSClusterDescription),
						resSKSClusterAttrAutoUpgrade:      validateString("true"),
						resSKSClusterAttrExoscaleCCM:      validateString("true"),
						resSKSClusterAttrMetricsServer:    validateString("false"),
						resSKSClusterAttrLabels + ".test": validateString(testAccResourceSKSClusterLabelValue),
						resSKSClusterAttrEnableKarpenter:  validateString("false"),
					})),
				),
			},
			{
				// Re-enable Karpenter
				Config: testAccResourceSKSClusterReEnableKarpenter,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(testAccResourceSKSClusterName, sksCluster.Name)

						// Verify Karpenter addon is present again in the API response
						assert.NotNil(t, sksCluster.Addons)
						if sksCluster.Addons != nil {
							found := false
							for _, addon := range sksCluster.Addons {
								if addon == sksClusterAddonKarpenter {
									found = true
									break
								}
							}
							a.True(found, "Karpenter addon should be present when re-enabled")
						} else {
							t.Error("Addons should not be nil when Karpenter is re-enabled")
						}

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrName:             validateString(testAccResourceSKSClusterName),
						resSKSClusterAttrDescription:      validateString(testAccResourceSKSClusterDescription),
						resSKSClusterAttrAutoUpgrade:      validateString("true"),
						resSKSClusterAttrExoscaleCCM:      validateString("true"),
						resSKSClusterAttrMetricsServer:    validateString("false"),
						resSKSClusterAttrLabels + ".test": validateString(testAccResourceSKSClusterLabelValue),
						resSKSClusterAttrEnableKarpenter:  validateString("true"),
					})),
				),
			},
		},
	})
}

func testGetSKSClusterVersions(t *testing.T) []string {
	defaultClient, err := APIClientV3()
	if err != nil {
		t.Fatalf("unable to initialize Exoscale client: %s", err)
	}
	ctx := context.Background()
	client, err := utils.SwitchClientZone(
		ctx,
		defaultClient,
		egoscale.ZoneName(testAccResourceSKSClusterLocalZone),
	)
	if err != nil {
		t.Fatalf("unable to initialize Exoscale client: %s", err)
	}

	versionsResponse, err := client.ListSKSClusterVersions(ctx)
	if err != nil {
		t.Fatalf("unable to retrieve SKS versions: %s", err)
	}
	if versionsResponse == nil || len(versionsResponse.SKSClusterVersions) == 0 {
		t.Fatal("no version returned by the API")
	}
	versions := versionsResponse.SKSClusterVersions

	return versions
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

		defaultClient, err := APIClientV3()
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}
		ctx := context.Background()
		client, err := utils.SwitchClientZone(
			ctx,
			defaultClient,
			egoscale.ZoneName(testAccResourceSKSClusterLocalZone),
		)
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}

		res, err := client.GetSKSCluster(ctx, egoscale.UUID(rs.Primary.ID))
		if err != nil {
			return err
		}

		*sksCluster = *res
		return nil
	}
}

func testAccCheckResourceSKSClusterDestroy(sksCluster *egoscale.SKSCluster) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		defaultClient, err := APIClientV3()
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}
		ctx := context.Background()
		client, err := utils.SwitchClientZone(
			ctx,
			defaultClient,
			egoscale.ZoneName(testAccResourceSKSClusterLocalZone),
		)
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}

		_, err = client.GetSKSCluster(ctx, sksCluster.ID)
		if err != nil {
			if errors.Is(err, egoscale.ErrNotFound) {
				return nil
			}
			return err
		}

		return errors.New("SKS cluster still exists")
	}
}

// Unit tests for addon helper functions

func TestAddonsSetToSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []string
	}{
		{
			name:     "empty set",
			input:    []interface{}{},
			expected: []string{},
		},
		{
			name:     "single addon",
			input:    []interface{}{"addon1"},
			expected: []string{"addon1"},
		},
		{
			name:     "multiple addons",
			input:    []interface{}{"addon1", "addon2", "addon3"},
			expected: []string{"addon1", "addon2", "addon3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a schema.Set from the input
			set := &schema.Set{F: schema.HashString}
			for _, v := range tt.input {
				set.Add(v)
			}

			result := addonsSetToSlice(set)

			assert.Len(t, result, len(tt.expected))
			// Convert result to map for easier comparison (order doesn't matter in a set)
			resultMap := make(map[string]bool)
			for _, v := range result {
				resultMap[v] = true
			}
			for _, expected := range tt.expected {
				assert.True(t, resultMap[expected], "expected addon %s not found in result", expected)
			}
		})
	}
}

func TestAppendAddonToSet(t *testing.T) {
	tests := []struct {
		name          string
		initialAddons []interface{}
		addonToAdd    string
		expected      []string
	}{
		{
			name:          "add to empty set",
			initialAddons: []interface{}{},
			addonToAdd:    "new-addon",
			expected:      []string{"new-addon"},
		},
		{
			name:          "add to existing set",
			initialAddons: []interface{}{"addon1", "addon2"},
			addonToAdd:    "addon3",
			expected:      []string{"addon1", "addon2", "addon3"},
		},
		{
			name:          "add duplicate addon",
			initialAddons: []interface{}{"addon1", "addon2"},
			addonToAdd:    "addon1",
			expected:      []string{"addon1", "addon2", "addon1"}, // Note: duplicates allowed in slice
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a schema.Set from the initial addons
			set := &schema.Set{F: schema.HashString}
			for _, v := range tt.initialAddons {
				set.Add(v)
			}

			result := appendAddonToSet(set, tt.addonToAdd)

			assert.Len(t, result, len(tt.expected))
			// Verify the new addon is present
			found := false
			for _, addon := range result {
				if addon == tt.addonToAdd {
					found = true
					break
				}
			}
			assert.True(t, found, "addon %s should be in the result", tt.addonToAdd)
		})
	}
}

func TestRemoveAddonFromSet(t *testing.T) {
	tests := []struct {
		name             string
		initialAddons    []interface{}
		addonToRemove    string
		expectedCount    int
		shouldContain    []string
		shouldNotContain string
	}{
		{
			name:             "remove from empty set",
			initialAddons:    []interface{}{},
			addonToRemove:    "addon1",
			expectedCount:    0,
			shouldContain:    []string{},
			shouldNotContain: "addon1",
		},
		{
			name:             "remove existing addon",
			initialAddons:    []interface{}{"addon1", "addon2", "addon3"},
			addonToRemove:    "addon2",
			expectedCount:    2,
			shouldContain:    []string{"addon1", "addon3"},
			shouldNotContain: "addon2",
		},
		{
			name:             "remove non-existent addon",
			initialAddons:    []interface{}{"addon1", "addon2"},
			addonToRemove:    "addon3",
			expectedCount:    2,
			shouldContain:    []string{"addon1", "addon2"},
			shouldNotContain: "addon3",
		},
		{
			name:             "remove last addon",
			initialAddons:    []interface{}{"addon1"},
			addonToRemove:    "addon1",
			expectedCount:    0,
			shouldContain:    []string{},
			shouldNotContain: "addon1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a schema.Set from the initial addons
			set := &schema.Set{F: schema.HashString}
			for _, v := range tt.initialAddons {
				set.Add(v)
			}

			result := removeAddonFromSet(set, tt.addonToRemove)

			assert.Len(t, result, tt.expectedCount, "expected %d addons, got %d", tt.expectedCount, len(result))

			// Verify expected addons are present
			resultMap := make(map[string]bool)
			for _, addon := range result {
				resultMap[addon] = true
			}

			for _, expected := range tt.shouldContain {
				assert.True(t, resultMap[expected], "expected addon %s not found in result", expected)
			}

			// Verify removed addon is not present
			assert.False(t, resultMap[tt.shouldNotContain], "addon %s should not be in result", tt.shouldNotContain)
		})
	}
}
