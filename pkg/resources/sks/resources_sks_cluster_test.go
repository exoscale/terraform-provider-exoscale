package sks_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"
	"text/template"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

var testZoneName = testutils.TestZoneName

const (
	defaultSKSClusterCNI              = "calico"
	defaultSKSClusterServiceLevel     = "pro"
	defaultSKSClusterAuditInitBackoff = "10s"

	sksClusterAddonExoscaleCCM = "exoscale-cloud-controller"
	sksClusterAddonKarpenter   = "karpenter"

	testAccSKSClusterFeatureGate       = "GracefulNodeShutdown"
	testAccSKSClusterAuditRemoteURL    = "https://audit.example.exoscale.net"
	testAccSKSClusterAuditInitBackoff  = "30s"
	testAccSKSClusterAuditBearerToken  = "supersecretbearertoken"
	testAccSKSClusterAuditRemoteURLUpd = "https://audit-updated.example.exoscale.net"
	testAccSKSClusterAuditBearerTokUpd = "newsupersecretbearertoken"
)

// testPemCertificateFormatRegex validates a PEM encoded certificate.
var testPemCertificateFormatRegex = regexp.MustCompile(`^-----BEGIN CERTIFICATE-----\n(.|\s)+\n-----END CERTIFICATE-----\n$`)

// validDNSNameRegex represents a valid DNS name pattern (RFC 1123/952).
var validDNSNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

// testUUIDRegex validates a UUID.
var testUUIDRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isDNSName(v string) error {
	if v == "" || len(v) > 253 || !validDNSNameRegex.MatchString(v) {
		return fmt.Errorf("expected a valid DNS name, got %q", v)
	}

	return nil
}

func defaultBool(v *bool, def bool) bool {
	if v != nil {
		return *v
	}

	return def
}

// sksTestdata is the data model injected into the testdata templates.
type sksTestdata struct {
	Zone    string
	ID      int64
	Version string
}

func parseSKSConfig(t *testing.T, path string, data sksTestdata) string {
	t.Helper()

	tpl, err := template.ParseFiles(path)
	if err != nil {
		t.Fatalf("parse %s: %s", path, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		t.Fatalf("render %s: %s", path, err)
	}

	return buf.String()
}

func TestAccResourceSKSCluster(t *testing.T) {
	t.Parallel()

	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
		return
	}

	var (
		r          = "exoscale_sks_cluster.test"
		sksCluster egoscale.SKSCluster
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}
	name := testutils.ResourceName(td.ID)

	versions := testGetSKSClusterVersions(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				// Create
				Config: parseSKSConfig(t, "./testdata/001.sks_create.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal([]string{sksClusterAddonExoscaleCCM}, sksCluster.Addons)
						a.True(defaultBool(sksCluster.AutoUpgrade, false))
						a.Equal(defaultSKSClusterCNI, string(sksCluster.Cni))
						a.Equal("description-test", sksCluster.Description)
						a.Equal("acctest", sksCluster.Labels["test"])
						a.Equal(name, sksCluster.Name)
						a.Equal(defaultSKSClusterServiceLevel, string(sksCluster.Level))
						a.Equal(versions[0], sksCluster.Version)
						a.Len(sksCluster.FeatureGates, 1)
						a.Equal(testAccSKSClusterFeatureGate, sksCluster.FeatureGates[0])
						return nil
					},
					resource.TestMatchResourceAttr(r, "aggregation_ca", testPemCertificateFormatRegex),
					resource.TestCheckResourceAttr(r, "auto_upgrade", "true"),
					resource.TestCheckResourceAttr(r, "cni", defaultSKSClusterCNI),
					resource.TestMatchResourceAttr(r, "control_plane_ca", testPemCertificateFormatRegex),
					resource.TestCheckResourceAttrSet(r, "created_at"),
					resource.TestCheckResourceAttr(r, "create_default_security_group", "true"),
					resource.TestCheckResourceAttr(r, "description", "description-test"),
					resource.TestCheckResourceAttrWith(r, "endpoint", isDNSName),
					resource.TestCheckResourceAttr(r, "enable_kube_proxy", "true"),
					resource.TestCheckResourceAttr(r, "exoscale_ccm", "true"),
					resource.TestCheckResourceAttr(r, "feature_gates.#", "1"),
					resource.TestMatchResourceAttr(r, "kubelet_ca", testPemCertificateFormatRegex),
					resource.TestCheckResourceAttr(r, "metrics_server", "false"),
					resource.TestCheckResourceAttr(r, "exoscale_csi", "false"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest"),
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "service_level", defaultSKSClusterServiceLevel),
					resource.TestCheckResourceAttrSet(r, "state"),
					resource.TestCheckResourceAttrSet(r, "version"),

					// OIDC checks
					resource.TestCheckResourceAttr(r, "oidc.0.client_id", "test-client"),
					resource.TestCheckResourceAttr(r, "oidc.0.groups_claim", "test-groups-claim"),
					resource.TestCheckResourceAttr(r, "oidc.0.groups_prefix", "test-groups-prefix"),
					resource.TestCheckResourceAttr(r, "oidc.0.issuer_url", "https://id.example.net"),
					resource.TestCheckResourceAttr(r, "oidc.0.username_claim", "test-username-claim"),
					resource.TestCheckResourceAttr(r, "oidc.0.username_prefix", "test-username-prefix"),
				),
			},
			{
				// Update
				Config: parseSKSConfig(t, "./testdata/002.sks_update.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Len(sksCluster.FeatureGates, 0)
						return nil
					},
					resource.TestMatchResourceAttr(r, "aggregation_ca", testPemCertificateFormatRegex),
					resource.TestCheckResourceAttr(r, "auto_upgrade", "true"),
					resource.TestCheckResourceAttr(r, "cni", defaultSKSClusterCNI),
					resource.TestMatchResourceAttr(r, "control_plane_ca", testPemCertificateFormatRegex),
					resource.TestCheckResourceAttrSet(r, "created_at"),
					resource.TestCheckResourceAttr(r, "description", "description-test-updated"),
					resource.TestCheckResourceAttrWith(r, "endpoint", isDNSName),
					resource.TestCheckResourceAttr(r, "exoscale_ccm", "true"),
					resource.TestCheckResourceAttr(r, "feature_gates.#", "0"),
					resource.TestMatchResourceAttr(r, "kubelet_ca", testPemCertificateFormatRegex),
					resource.TestCheckResourceAttr(r, "metrics_server", "false"),
					resource.TestCheckResourceAttr(r, "exoscale_csi", "true"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest-updated"),
					resource.TestCheckResourceAttr(r, "name", name+"-updated"),
					resource.TestCheckResourceAttr(r, "service_level", defaultSKSClusterServiceLevel),
					resource.TestCheckResourceAttrSet(r, "state"),
					resource.TestCheckResourceAttrSet(r, "version"),

					resource.TestCheckResourceAttr(r, "oidc.0.client_id", "test-client-updated"),
					resource.TestCheckResourceAttr(r, "oidc.0.groups_claim", "test-groups-claim-updated"),
					resource.TestCheckResourceAttr(r, "oidc.0.groups_prefix", "test-groups-prefix-updated"),
					resource.TestCheckResourceAttr(r, "oidc.0.issuer_url", "https://id-updated.example.net"),
					resource.TestCheckResourceAttr(r, "oidc.0.username_claim", "test-username-claim-updated"),
					resource.TestCheckResourceAttr(r, "oidc.0.username_prefix", "test-username-prefix-updated"),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(sksCluster *egoscale.SKSCluster) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", sksCluster.ID, testZoneName), nil
					}
				}(&sksCluster),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"oidc.#",
					"oidc.0.%",
					"create_default_security_group",
					"enable_kube_proxy",
					"oidc.0.client_id",
					"oidc.0.groups_claim",
					"oidc.0.groups_prefix",
					"oidc.0.issuer_url",
					"oidc.0.required_claim",
					"oidc.0.username_claim",
					"oidc.0.username_prefix",
				},
			},
		},
	})

	// Test cluster upgrade.
	sksCluster = egoscale.SKSCluster{}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				// Create old version cluster
				Config: parseSKSConfig(t, "./testdata/003.sks_version.tf.tmpl", sksTestdata{Zone: td.Zone, ID: td.ID, Version: versions[1]}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(name, sksCluster.Name)
						a.Equal(versions[1], sksCluster.Version)
						return nil
					},
					resource.TestCheckResourceAttr(r, "auto_upgrade", "false"),
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttrSet(r, "state"),
					resource.TestCheckResourceAttr(r, "version", versions[1]),
				),
			},
			{
				// Upgrade cluster
				Config: parseSKSConfig(t, "./testdata/003.sks_version.tf.tmpl", sksTestdata{Zone: td.Zone, ID: td.ID, Version: versions[0]}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(name, sksCluster.Name)
						a.Equal(versions[0], sksCluster.Version)
						return nil
					},
					resource.TestCheckResourceAttr(r, "auto_upgrade", "false"),
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttrSet(r, "state"),
					resource.TestCheckResourceAttr(r, "version", versions[0]),
				),
			},
		},
	})
}

func TestAccResourceSKSClusterWithAudit(t *testing.T) {
	t.Parallel()

	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
		return
	}

	var (
		r          = "exoscale_sks_cluster.test-with-audit"
		sksCluster egoscale.SKSCluster
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}
	name := testutils.ResourceName(td.ID)

	versions := testGetSKSClusterVersions(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				// Create cluster with audit enabled
				Config: parseSKSConfig(t, "./testdata/004.sks_audit_create.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(name, sksCluster.Name)
						a.Equal("description-test", sksCluster.Description)

						assert.NotNil(t, sksCluster.Audit)
						if sksCluster.Audit != nil {
							if a.NotNil(sksCluster.Audit.Enabled) {
								a.True(*sksCluster.Audit.Enabled)
							}
							a.Equal(testAccSKSClusterAuditRemoteURL, string(sksCluster.Audit.Endpoint))
							a.Equal(testAccSKSClusterAuditInitBackoff, string(sksCluster.Audit.InitialBackoff))
						} else {
							t.Error("Audit should not be nil when audit is enabled")
						}

						return nil
					},
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "description", "description-test"),
					resource.TestCheckResourceAttr(r, "auto_upgrade", "true"),
					resource.TestCheckResourceAttr(r, "exoscale_ccm", "true"),
					resource.TestCheckResourceAttr(r, "metrics_server", "false"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest"),
					resource.TestCheckResourceAttr(r, "audit.0.enabled", "true"),
					resource.TestCheckResourceAttr(r, "audit.0.endpoint", testAccSKSClusterAuditRemoteURL),
					resource.TestCheckResourceAttr(r, "audit.0.initial_backoff", testAccSKSClusterAuditInitBackoff),
					resource.TestCheckResourceAttr(r, "audit.0.bearer_token", testAccSKSClusterAuditBearerToken),
				),
			},
			{
				// Update cluster to disable audit
				Config: parseSKSConfig(t, "./testdata/005.sks_audit_disable.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(name, sksCluster.Name)

						if assert.NotNil(t, sksCluster.Audit) {
							a.False(*sksCluster.Audit.Enabled)
						}

						return nil
					},
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "description", "description-test"),
					resource.TestCheckResourceAttr(r, "auto_upgrade", "true"),
					resource.TestCheckResourceAttr(r, "exoscale_ccm", "true"),
					resource.TestCheckResourceAttr(r, "metrics_server", "false"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest"),
					resource.TestCheckResourceAttr(r, "audit.0.enabled", "false"),
				),
			},
			{
				// Re-enable audit with new URL and default backoff
				Config:             parseSKSConfig(t, "./testdata/006.sks_audit_reenable.tf.tmpl", td),
				PlanOnly:           true, // TODO: remove once sks-orch is fixed
				ExpectNonEmptyPlan: true, // TODO: remove once sks-orch is fixed
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(name, sksCluster.Name)

						assert.NotNil(t, sksCluster.Audit)
						if sksCluster.Audit != nil {
							a.True(*sksCluster.Audit.Enabled)
							a.Equal(testAccSKSClusterAuditRemoteURLUpd, string(sksCluster.Audit.Endpoint))
							a.NotEmpty(string(sksCluster.Audit.InitialBackoff))
						} else {
							t.Error("Audit should not be nil when audit is re-enabled")
						}

						return nil
					},
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "description", "description-test"),
					resource.TestCheckResourceAttr(r, "auto_upgrade", "true"),
					resource.TestCheckResourceAttr(r, "exoscale_ccm", "true"),
					resource.TestCheckResourceAttr(r, "metrics_server", "false"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest"),
					resource.TestCheckResourceAttr(r, "audit.0.enabled", "true"),
					resource.TestCheckResourceAttr(r, "audit.0.endpoint", testAccSKSClusterAuditRemoteURLUpd),
					resource.TestCheckResourceAttr(r, "audit.0.initial_backoff", defaultSKSClusterAuditInitBackoff),
					resource.TestCheckResourceAttr(r, "audit.0.bearer_token", testAccSKSClusterAuditBearerTokUpd),
				),
			},
		},
	})
}

func TestAccResourceSKSClusterWithKarpenter(t *testing.T) {
	t.Parallel()

	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
		return
	}

	var (
		r          = "exoscale_sks_cluster.test-with-karpenter"
		sksCluster egoscale.SKSCluster
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}
	name := testutils.ResourceName(td.ID)

	versions := testGetSKSClusterVersions(t)

	hasKarpenter := func(addons []string) bool {
		for _, a := range addons {
			if a == sksClusterAddonKarpenter {
				return true
			}
		}
		return false
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				// Create cluster with Karpenter enabled
				Config: parseSKSConfig(t, "./testdata/007.sks_karpenter_enable.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(name, sksCluster.Name)
						a.Equal("description-test", sksCluster.Description)
						a.True(hasKarpenter(sksCluster.Addons), "Karpenter addon should be present when enabled")
						a.ElementsMatch([]string{"NodeRepair"}, sksCluster.KarpenterFeatureGates)
						return nil
					},
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "description", "description-test"),
					resource.TestCheckResourceAttr(r, "auto_upgrade", "true"),
					resource.TestCheckResourceAttr(r, "exoscale_ccm", "true"),
					resource.TestCheckResourceAttr(r, "metrics_server", "false"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest"),
					resource.TestCheckResourceAttr(r, "enable_karpenter", "true"),
					resource.TestCheckResourceAttr(r, "karpenter_feature_gates.#", "1"),
					resource.TestCheckTypeSetElemAttr(r, "karpenter_feature_gates.*", "NodeRepair"),
				),
			},
			{
				// Update cluster to disable Karpenter
				Config: parseSKSConfig(t, "./testdata/008.sks_karpenter_disable.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(name, sksCluster.Name)
						a.False(hasKarpenter(sksCluster.Addons), "Karpenter addon should not be present when disabled")
						return nil
					},
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "description", "description-test"),
					resource.TestCheckResourceAttr(r, "auto_upgrade", "true"),
					resource.TestCheckResourceAttr(r, "exoscale_ccm", "true"),
					resource.TestCheckResourceAttr(r, "metrics_server", "false"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest"),
					resource.TestCheckResourceAttr(r, "enable_karpenter", "false"),
				),
			},
			{
				// Re-enable Karpenter
				Config: parseSKSConfig(t, "./testdata/007.sks_karpenter_enable.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(versions[0], sksCluster.Version)
						a.Equal(name, sksCluster.Name)
						a.True(hasKarpenter(sksCluster.Addons), "Karpenter addon should be present when re-enabled")
						a.ElementsMatch([]string{"NodeRepair"}, sksCluster.KarpenterFeatureGates)
						return nil
					},
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "description", "description-test"),
					resource.TestCheckResourceAttr(r, "auto_upgrade", "true"),
					resource.TestCheckResourceAttr(r, "exoscale_ccm", "true"),
					resource.TestCheckResourceAttr(r, "metrics_server", "false"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest"),
					resource.TestCheckResourceAttr(r, "enable_karpenter", "true"),
					resource.TestCheckResourceAttr(r, "karpenter_feature_gates.#", "1"),
					resource.TestCheckTypeSetElemAttr(r, "karpenter_feature_gates.*", "NodeRepair"),
				),
			},
		},
	})
}

func testGetSKSClusterVersions(t *testing.T) []string {
	defaultClient, err := testutils.APIClientV3()
	if err != nil {
		t.Fatalf("unable to initialize Exoscale client: %s", err)
	}
	ctx := context.Background()
	client, err := utils.SwitchClientZone(
		ctx,
		defaultClient,
		egoscale.ZoneName(testZoneName),
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

	return versionsResponse.SKSClusterVersions
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

		defaultClient, err := testutils.APIClientV3()
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}
		ctx := context.Background()
		client, err := utils.SwitchClientZone(
			ctx,
			defaultClient,
			egoscale.ZoneName(testZoneName),
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
		defaultClient, err := testutils.APIClientV3()
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}
		ctx := context.Background()
		client, err := utils.SwitchClientZone(
			ctx,
			defaultClient,
			egoscale.ZoneName(testZoneName),
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
