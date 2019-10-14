package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
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
						"template":        ValidateString(defaultExoscaleTemplate),
						"zone":            ValidateString(defaultExoscaleZone),
						"name":            ValidateString("instance-pool-test"),
						"serviceoffering": ValidateString("Medium"),
						"size":            ValidateString("3"),
						"key_pair":        ValidateString("terraform-test-keypair"),
					}),
				),
			},
			{
				Config: testAccResourceInstancePoolConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceInstancePoolExists("exoscale_instance_pool.pool", pool),
					testAccCheckResourceInstancePool(pool),
					testAccCheckResourceInstancePoolAttributes(testAttrs{
						"description": ValidateString("test description"),
						"user_data":   ValidateString("#cloud-config\npackage_upgrade: true\n"),
						"size":        ValidateString("1"),
					}),
				),
			},
			{
				ResourceName:      "exoscale_instance_pool.pool",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"template":        ValidateString(defaultExoscaleTemplate),
							"zone":            ValidateString(defaultExoscaleZone),
							"name":            ValidateString("instance-pool-test"),
							"description":     ValidateString("test description"),
							"serviceoffering": ValidateString("Medium"),
							"size":            ValidateString("1"),
							"key_pair":        ValidateString("terraform-test-keypair"),
							"user_data":       ValidateString("#cloud-config\npackage_upgrade: true\n"),
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

		resp, err := client.Get(&egoscale.Zone{
			Name: defaultExoscaleZone,
		})
		if err != nil {
			return err
		}
		zone := resp.(*egoscale.Zone)

		req := &egoscale.GetInstancePool{ID: id, ZoneID: zone.ID}
		r, err := client.Request(req)
		if err != nil {
			return err
		}
		instancePool := r.(*egoscale.GetInstancePoolsResponse).ListInstancePoolsResponse[0]

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

		resp, err := client.Get(&egoscale.Zone{
			Name: defaultExoscaleZone,
		})
		if err != nil {
			return err
		}
		zone := resp.(*egoscale.Zone)

		pool := &egoscale.GetInstancePool{ID: id, ZoneID: zone.ID}
		_, err = client.Request(pool)
		if err != nil {
			return nil
		}
	}
	return errors.New("instance pool still exists")
}

var testAccResourceInstancePoolConfigCreate = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-test-keypair"
}

variable "zone" {
	default = %q
}

resource "exoscale_instance_pool" "pool" {
  name = "instance-pool-test"
  template = %q
  serviceoffering = "medium
  size = 3
  key_pair = "${exoscale_ssh_keypair.key.name}"
  zone = "${var.zone}"

  timeouts {
    create = "10m"
  }
}
`,
	defaultExoscaleZone,
	defaultExoscaleTemplate,
)

var testAccResourceInstancePoolConfigUpdate = fmt.Sprintf(`

variable "zone" {
	  default = %q
}
  
resource "exoscale_instance_pool" "pool" {
  description = "test description"
  user_data = "#cloud-config\npackage_upgrade: true\n"
  size = 1
  zone = "${var.zone}"

  timeouts {
	update = "10m"
  }
}
`,
	defaultExoscaleZone,
)
