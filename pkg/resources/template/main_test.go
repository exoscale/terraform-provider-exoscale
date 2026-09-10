package template_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func TestTemplate(t *testing.T) {
	t.Parallel()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test skipped without TF_ACC")
	}
	testutils.AccPreCheck(t)

	var (
		fullResourceName = "exoscale_template.test"
		zone             = testutils.TestZoneName
		id               = time.Now().UnixNano()
		name             = testutils.ResourceName(id)
		nameUpdated      = name + "-updated"
	)

	url, checksum := exportedSnapshotFixture(t, zone, name)

	config := func(tplName, description string) string {
		return fmt.Sprintf(`
resource "exoscale_template" "test" {
  zone             = %q
  name             = %q
  description      = %q
  url              = %q
  checksum         = %q
  password_enabled = false
  ssh_key_enabled  = true
}
`, zone, tplName, description, url, checksum)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             checkTemplateDestroy(fullResourceName, zone),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: config(name, "acceptance test template"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "id"),
					resource.TestCheckResourceAttr(fullResourceName, "name", name),
					resource.TestCheckResourceAttr(fullResourceName, "description", "acceptance test template"),
					resource.TestCheckResourceAttr(fullResourceName, "zone", zone),
					resource.TestCheckResourceAttr(fullResourceName, "url", url),
					resource.TestCheckResourceAttr(fullResourceName, "checksum", checksum),
					resource.TestCheckResourceAttr(fullResourceName, "password_enabled", "false"),
					resource.TestCheckResourceAttr(fullResourceName, "ssh_key_enabled", "true"),
					resource.TestCheckResourceAttr(fullResourceName, "visibility", "private"),
					resource.TestCheckResourceAttrSet(fullResourceName, "created_at"),
				),
			},
			// Update (name + description are the only mutable attributes).
			{
				Config: config(nameUpdated, "acceptance test template (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "name", nameUpdated),
					resource.TestCheckResourceAttr(fullResourceName, "description", "acceptance test template (updated)"),
				),
			},
			// Import.
			{
				ResourceName: fullResourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf(
						"%s@%s",
						s.RootModule().Resources[fullResourceName].Primary.ID,
						zone,
					), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// exportedSnapshotFixture creates an instance, snapshots it, exports the snapshot
// and returns the exported disk image pre-signed URL and its MD5 checksum. The
// instance and snapshot are removed via t.Cleanup (last registered runs first,
// so the snapshot is deleted before the instance).
func exportedSnapshotFixture(t *testing.T, zone, name string) (url, checksum string) {
	t.Helper()

	ctx := context.Background()

	client, err := testutils.APIClientV3()
	if err != nil {
		t.Fatalf("init v3 client: %s", err)
	}
	client, err = utils.SwitchClientZone(ctx, client, exoscale.ZoneName(zone))
	if err != nil {
		t.Fatalf("switch client zone: %s", err)
	}

	// Resolve a public template to boot the instance from.
	templates, err := client.ListTemplates(ctx, exoscale.ListTemplatesWithVisibility(exoscale.ListTemplatesVisibilityPublic))
	if err != nil {
		t.Fatalf("list templates: %s", err)
	}
	bootTemplate, err := templates.FindTemplate(testutils.TestInstanceTemplateName)
	if err != nil {
		t.Fatalf("find boot template %q: %s", testutils.TestInstanceTemplateName, err)
	}

	instanceTypeID, err := exoscale.ParseUUID(testutils.TestInstanceTypeIDTiny)
	if err != nil {
		t.Fatalf("parse instance type ID: %s", err)
	}

	// Create the instance.
	op, err := client.CreateInstance(ctx, exoscale.CreateInstanceRequest{
		Name:         name,
		DiskSize:     10,
		InstanceType: &exoscale.InstanceType{ID: instanceTypeID},
		Template:     &exoscale.Template{ID: bootTemplate.ID},
	})
	if err != nil {
		t.Fatalf("create instance: %s", err)
	}
	op, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		t.Fatalf("wait instance creation: %s", err)
	}
	instanceID := refID(t, op, "instance creation")

	t.Cleanup(func() {
		op, err := client.DeleteInstance(context.Background(), instanceID)
		if err != nil {
			t.Errorf("cleanup: delete instance %s: %s", instanceID, err)
			return
		}
		if _, err := client.Wait(context.Background(), op, exoscale.OperationStateSuccess); err != nil {
			t.Errorf("cleanup: wait instance %s deletion: %s", instanceID, err)
		}
	})

	// Wait for the instance to be running before snapshotting.
	if err := poll(ctx, 5*time.Minute, func() (bool, error) {
		instance, err := client.GetInstance(ctx, instanceID)
		if err != nil {
			return false, err
		}
		return instance.State == exoscale.InstanceStateRunning, nil
	}); err != nil {
		t.Fatalf("wait instance %s running: %s", instanceID, err)
	}

	// Snapshot the instance.
	op, err = client.CreateSnapshot(ctx, instanceID)
	if err != nil {
		t.Fatalf("create snapshot: %s", err)
	}
	op, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		t.Fatalf("wait snapshot creation: %s", err)
	}
	snapshotID := refID(t, op, "snapshot creation")

	t.Cleanup(func() {
		op, err := client.DeleteSnapshot(context.Background(), snapshotID)
		if err != nil {
			t.Errorf("cleanup: delete snapshot %s: %s", snapshotID, err)
			return
		}
		if _, err := client.Wait(context.Background(), op, exoscale.OperationStateSuccess); err != nil {
			t.Errorf("cleanup: wait snapshot %s deletion: %s", snapshotID, err)
		}
	})

	// Export the snapshot.
	op, err = client.ExportSnapshot(ctx, snapshotID)
	if err != nil {
		t.Fatalf("export snapshot: %s", err)
	}
	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		t.Fatalf("wait snapshot export: %s", err)
	}

	var snapshot *exoscale.Snapshot
	if err := poll(ctx, 10*time.Minute, func() (bool, error) {
		snapshot, err = client.GetSnapshot(ctx, snapshotID)
		if err != nil {
			return false, err
		}
		return snapshot.Export != nil &&
			snapshot.Export.PresignedURL != "" &&
			snapshot.Export.Md5sum != "", nil
	}); err != nil {
		t.Fatalf("wait snapshot %s export details: %s", snapshotID, err)
	}

	return snapshot.Export.PresignedURL, snapshot.Export.Md5sum
}

// poll invokes fn every 5s until it returns true or ctx/timeout expires.
func poll(ctx context.Context, timeout time.Duration, fn func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		done, err := fn()
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func refID(t *testing.T, op *exoscale.Operation, what string) exoscale.UUID {
	t.Helper()

	if op == nil || op.Reference == nil {
		t.Fatalf("%s: operation completed without a resource reference", what)
	}

	return op.Reference.ID
}

func checkTemplateDestroy(resourceName, zone string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}

		client, err := testutils.APIClientV3()
		if err != nil {
			return err
		}

		ctx := context.Background()

		client, err = utils.SwitchClientZone(ctx, client, exoscale.ZoneName(zone))
		if err != nil {
			return err
		}

		templateID, err := exoscale.ParseUUID(res.Primary.ID)
		if err != nil {
			return err
		}

		_, err = client.GetTemplate(ctx, templateID)
		if err != nil {
			if errors.Is(err, exoscale.ErrNotFound) {
				return nil
			}

			return err
		}

		return fmt.Errorf("template %s still exists", res.Primary.ID)
	}
}
