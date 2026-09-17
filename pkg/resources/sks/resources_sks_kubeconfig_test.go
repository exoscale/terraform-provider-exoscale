package sks_test

import (
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/sks"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

const (
	testAccResourceSKSKubeconfigAttrEarlyRenewalSeconds = int64(600)
	testAccResourceSKSKubeconfigAttrGroup               = "kube-group"
	testAccResourceSKSKubeconfigAttrTTLSeconds          = int64(3600)
	testAccResourceSKSKubeconfigAttrUser                = "kube-user"
)

func TestAccResourceSKSKubeconfig(t *testing.T) {
	t.Parallel()

	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
		return
	}

	var (
		r          = "exoscale_sks_kubeconfig.test_admin"
		sksCluster egoscale.SKSCluster
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				Config: parseSKSConfig(t, "./testdata/016.sks_kubeconfig_create.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists("exoscale_sks_cluster.test", &sksCluster),
					testAccCheckResourceSKSKubeconfigExists(r),
					func(s *terraform.State) error {
						a := require.New(t)

						rs := s.RootModule().Resources[r]
						kubeconfig := rs.Primary.Attributes["kubeconfig"]

						_, certificates, err := sks.KubeconfigExtractCertificates(kubeconfig)
						a.NoError(err)
						a.Len(certificates, 1)

						clientCertificate := *(certificates[0])
						certificateTTL := int64(clientCertificate.NotAfter.Sub(clientCertificate.NotBefore).Seconds())

						a.InDelta(testAccResourceSKSKubeconfigAttrTTLSeconds, certificateTTL, 10)
						a.Equal(testAccResourceSKSKubeconfigAttrUser, clientCertificate.Subject.CommonName)
						a.Equal(testAccResourceSKSKubeconfigAttrGroup, clientCertificate.Subject.Organization[0])

						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						"groups.#":              testutils.ValidateString("1"),
						"groups.0":              testutils.ValidateString(testAccResourceSKSKubeconfigAttrGroup),
						"ttl_seconds":           testutils.ValidateString(strconv.FormatInt(testAccResourceSKSKubeconfigAttrTTLSeconds, 10)),
						"user":                  testutils.ValidateString(testAccResourceSKSKubeconfigAttrUser),
						"early_renewal_seconds": testutils.ValidateString(strconv.FormatInt(testAccResourceSKSKubeconfigAttrEarlyRenewalSeconds, 10)),
						"ready_for_renewal":     testutils.ValidateString("false"),
					})),
				),
			},
		},
	})
}

func testAccCheckResourceSKSKubeconfigExists(r string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		if _, ok := rs.Primary.Attributes["kubeconfig"]; !ok {
			return errors.New("kubeconfig attribute not found in the resource")
		}

		return nil
	}
}
