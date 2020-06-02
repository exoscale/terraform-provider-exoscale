package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceInstancePoolSSHKeyName      = testPrefix + "-" + testRandomString()
	testAccResourceInstancePoolZoneName        = testZoneName
	testAccResourceInstancePoolName            = testPrefix + "-" + testRandomString()
	testAccResourceInstancePoolNameUpdated     = testAccResourceInstancePoolName + "-updated"
	testAccResourceInstancePoolDescription     = testDescription
	testAccResourceInstancePoolTemplateID      = testInstanceTemplateID
	testAccResourceInstancePoolServiceOffering = "medium"
	testAccResourceInstancePoolSize            = 2
	testAccResourceInstancePoolDiskSize        = 10
	testAccResourceInstancePoolSizeUpdated     = 1
	testAccResourceInstancePoolUserData        = `#cloud-config
package_upgrade: true
`

	testAccResourceInstancePoolConfigCreate = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}

resource "exoscale_instance_pool" "pool" {
  zone = "%s"
  name = "%s"
  template_id = "%s"
  service_offering = "%s"
  size = %d
  disk_size = %d
  key_pair = exoscale_ssh_keypair.key.name
  user_data = <<EOF
%s
EOF

  timeouts {
    delete = "10m"
  }
}
`,
		testAccResourceInstancePoolSSHKeyName,
		testAccResourceInstancePoolZoneName,
		testAccResourceInstancePoolName,
		testAccResourceInstancePoolTemplateID,
		testAccResourceInstancePoolServiceOffering,
		testAccResourceInstancePoolSize,
		testAccResourceInstancePoolDiskSize,
		testAccResourceInstancePoolUserData,
	)

	testAccResourceInstancePoolConfigUpdate = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}

resource "exoscale_instance_pool" "pool" {
  zone = "%s"
  name = "%s"
  description = "%s"
  template_id = "%s"
  service_offering = "%s"
  size = %d
  disk_size = %d
  key_pair = exoscale_ssh_keypair.key.name
  user_data = <<EOF
%s
EOF

  timeouts {
    delete = "10m"
  }
}
`,
		testAccResourceInstancePoolSSHKeyName,
		testAccResourceInstancePoolZoneName,
		testAccResourceInstancePoolNameUpdated,
		testAccResourceInstancePoolDescription,
		testAccResourceInstancePoolTemplateID,
		testAccResourceInstancePoolServiceOffering,
		testAccResourceInstancePoolSizeUpdated,
		testAccResourceInstancePoolDiskSize,
		testAccResourceInstancePoolUserData,
	)
)

func TestAccResourceInstancePool(t *testing.T) {
	pool := new(egoscale.InstancePool)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceInstancePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceInstancePoolConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceInstancePoolExists("exoscale_instance_pool.pool", pool),
					testAccCheckResourceInstancePool(pool),
					testAccCheckResourceInstancePoolAttributes(testAttrs{
						"zone":               ValidateString(testAccResourceInstancePoolZoneName),
						"name":               ValidateString(testAccResourceInstancePoolName),
						"template_id":        ValidateString(testAccResourceInstancePoolTemplateID),
						"service_offering":   ValidateString(testAccResourceInstancePoolServiceOffering),
						"size":               ValidateString(fmt.Sprint(testAccResourceInstancePoolSize)),
						"disk_size":          ValidateString(fmt.Sprint(testAccResourceInstancePoolDiskSize)),
						"key_pair":           ValidateString(testAccResourceInstancePoolSSHKeyName),
						"virtual_machines.#": ValidateStringNot("0"),
					}),
				),
			},
			{
				Config: testAccResourceInstancePoolConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceInstancePoolExists("exoscale_instance_pool.pool", pool),
					testAccCheckResourceInstancePool(pool),
					testAccCheckResourceInstancePoolAttributes(testAttrs{
						"zone":               ValidateString(testAccResourceInstancePoolZoneName),
						"name":               ValidateString(testAccResourceInstancePoolNameUpdated),
						"description":        ValidateString(testAccResourceInstancePoolDescription),
						"template_id":        ValidateString(testAccResourceInstancePoolTemplateID),
						"service_offering":   ValidateString(testAccResourceInstancePoolServiceOffering),
						"size":               ValidateString(fmt.Sprint(testAccResourceInstancePoolSizeUpdated)),
						"disk_size":          ValidateString(fmt.Sprint(testAccResourceInstancePoolDiskSize)),
						"key_pair":           ValidateString(testAccResourceInstancePoolSSHKeyName),
						"virtual_machines.#": ValidateStringNot("0"),
					}),
				),
			},
			{
				ResourceName:            "exoscale_instance_pool.pool",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"state"},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"zone":             ValidateString(testAccResourceInstancePoolZoneName),
							"name":             ValidateString(testAccResourceInstancePoolNameUpdated),
							"description":      ValidateString(testAccResourceInstancePoolDescription),
							"template_id":      ValidateString(testAccResourceInstancePoolTemplateID),
							"service_offering": ValidateString(testAccResourceInstancePoolServiceOffering),
							"size":             ValidateString(fmt.Sprint(testAccResourceInstancePoolSizeUpdated)),
							"disk_size":        ValidateString(fmt.Sprint(testAccResourceInstancePoolDiskSize)),
							"key_pair":         ValidateString(testAccResourceInstancePoolSSHKeyName),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceInstancePoolExists(n string, pool *egoscale.InstancePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := GetComputeClient(testAccProvider.Meta())

		zone, err := getZoneByName(context.TODO(), client, testZoneName)
		if err != nil {
			return err
		}

		req := &egoscale.GetInstancePool{ID: id, ZoneID: zone.ID}
		r, err := client.Request(req)
		if err != nil {
			return err
		}
		instancePool := r.(*egoscale.GetInstancePoolResponse).InstancePools[0]

		return Copy(pool, &instancePool)
	}
}

func testAccCheckResourceInstancePool(pool *egoscale.InstancePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if pool.ID == nil {
			return errors.New("instance pool ID is nil")
		}

		return nil
	}
}

func testAccCheckResourceInstancePoolAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_instance_pool" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceInstancePoolDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_instance_pool" {
			continue
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		zone, err := getZoneByName(context.TODO(), client, testZoneName)
		if err != nil {
			return err
		}

		// this time.Sleep() is here to prevent race condition when
		// an instance pool is destroyed, to wait till instance pool state changes
		// from "running" to "destroying"
		time.Sleep(time.Second * 10)

		pool := &egoscale.GetInstancePool{ID: id, ZoneID: zone.ID}
		r, err := client.Request(pool)
		if err != nil {
			return nil
		}
		instancePool := r.(*egoscale.GetInstancePoolResponse).InstancePools[0]

		if instancePool.State == egoscale.InstancePoolDestroying {
			return nil
		}

	}
	return errors.New("instance pool still exists")
}
