package exoscale

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

var (
	testAccResourceSKSKubeconfigAttrEarlyRenewalSeconds = int64(600)
	testAccResourceSKSKubeconfigAttrGroup               = "kube-group"
	testAccResourceSKSKubeconfigAttrTTLSeconds          = int64(3600)
	testAccResourceSKSKubeconfigAttrUser                = "kube-user"

	testAccResourceSKSKubeconfigConfigCreate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test" {
  zone = local.zone
  name = "%s"

  timeouts {
    create = "10m"
  }
}

resource "exoscale_sks_kubeconfig" "test_admin" {
	zone = local.zone

	ttl_seconds = %d
	early_renewal_seconds = %d
	cluster_id = exoscale_sks_cluster.test.id
	user = "%s"
	groups = ["%s"]
}
`,
		testZoneName,
		testAccResourceSKSClusterName,
		testAccResourceSKSKubeconfigAttrTTLSeconds,
		testAccResourceSKSKubeconfigAttrEarlyRenewalSeconds,
		testAccResourceSKSKubeconfigAttrUser,
		testAccResourceSKSKubeconfigAttrGroup,
	)
)

func TestAccResourceSKSKubeconfig(t *testing.T) {
	var (
		r             = "exoscale_sks_kubeconfig.test_admin"
		sksCluster    egoscale.SKSCluster
		sksKubeconfig string
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSKSKubeconfigConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists("exoscale_sks_cluster.test", &sksCluster),
					testAccCheckResourceSKSKubeconfigExists(r, &sksKubeconfig),
					func(s *terraform.State) error {
						a := require.New(t)

						_, certificates, _ := KubeconfigExtractCertificates(sksKubeconfig)

						a.Len(certificates, 1)
						clientCertificate := *(certificates[0])

						certificateTTL := int64(clientCertificate.NotAfter.Sub(clientCertificate.NotBefore).Seconds())

						a.InDelta(testAccResourceSKSKubeconfigAttrTTLSeconds, certificateTTL, 10)
						a.Equal(testAccResourceSKSKubeconfigAttrUser, clientCertificate.Subject.CommonName)
						a.Equal(testAccResourceSKSKubeconfigAttrGroup, clientCertificate.Subject.Organization[0])

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSKubeconfigAttrGroups + ".#":       validateString("1"),
						resSKSKubeconfigAttrGroups + ".0":       validateString(testAccResourceSKSKubeconfigAttrGroup),
						resSKSKubeconfigAttrTTLSeconds:          validateString(strconv.FormatInt(testAccResourceSKSKubeconfigAttrTTLSeconds, 10)),
						resSKSKubeconfigAttrUser:                validateString(testAccResourceSKSKubeconfigAttrUser),
						resSKSKubeconfigAttrEarlyRenewalSeconds: validateString(strconv.FormatInt(testAccResourceSKSKubeconfigAttrEarlyRenewalSeconds, 10)),
					})),
				),
			},
		},
	})
}

func testAccCheckResourceSKSKubeconfigExists(r string, sksKubeconfig *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		kubeconfig, ok := rs.Primary.Attributes[resSKSKubeconfigAttrKubeconfig]
		if !ok {
			return errors.New("attribute not found in the resource")
		}

		*sksKubeconfig = kubeconfig
		return nil
	}
}
