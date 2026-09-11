package exoscale

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

// testAccCheckResourceSKSClusterExists is shared by the SKS nodepool and
// kubeconfig acceptance tests. The `exoscale_sks_cluster` resource itself now
// lives in pkg/resources/sks_cluster.
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
