package kms_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func TestKMSKey(t *testing.T) {
	t.Parallel()

	fullResourceName := "exoscale_kms_key.test"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/001.kms_key_create.tf.tmpl",
					&testdataSpec,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "id"),
					resource.TestCheckResourceAttr(fullResourceName, "name", testutils.ResourceName(testdataSpec.ID)),
					resource.TestCheckResourceAttr(fullResourceName, "description", "acceptance test key"),
					resource.TestCheckResourceAttr(fullResourceName, "zone", testutils.TestZoneName),
					resource.TestCheckResourceAttr(fullResourceName, "usage", "encrypt-decrypt"),
					resource.TestCheckResourceAttr(fullResourceName, "status", "enabled"),
					resource.TestCheckResourceAttr(fullResourceName, "key_spec", "AES_256"),
					resource.TestCheckResourceAttr(fullResourceName, "source", "exoscale-kms"),
					resource.TestCheckNoResourceAttr(fullResourceName, "xks"),
				),
			},
			// Import
			{
				ResourceName: fullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return fmt.Sprintf(
							"%s@%s",
							s.RootModule().Resources[fullResourceName].Primary.ID,
							testdataSpec.Zone,
						), nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// keyStoreTestdataSpec adds the XKS proxy to the key store testdata template.
type keyStoreTestdataSpec struct {
	testutils.TestdataSpec
	ProxyEndpoint    string
	ProxyAccessKey   string
	ProxySecretKey   string
	Description      string
	SecretKeyVersion int
	Connected        bool
}

// The test XKS proxy that KMS deploys in each zone (sof1 is the location of testutils.TestZoneName),
// with its dummy credentials: see .kargo/xks-proxy.edn in the KMS repository.
const (
	testXKSProxyEndpoint  = "https://sof1.prod.kms-test.exoscale.ch/example/uri/path/prefix/kms/xks/v1"
	testXKSProxyAccessKey = "BETWEEN2TENAND3TENCHARACTERS"
	testXKSProxySecretKey = "PleaseReplaceThisWithSomeSecretOfLength43To64"
)

func TestKMSKeyStore(t *testing.T) {
	t.Parallel()

	proxyEndpoint := testXKSProxyEndpoint
	proxyAccessKey := testXKSProxyAccessKey
	proxySecretKey := testXKSProxySecretKey
	zone := testutils.TestZoneName

	fullResourceName := "exoscale_kms_key_store.test"
	byName := "data.exoscale_kms_key_store.by_name"
	byID := "data.exoscale_kms_key_store.by_id"

	spec := keyStoreTestdataSpec{
		TestdataSpec:     testutils.TestdataSpec{ID: time.Now().UnixNano(), Zone: zone},
		ProxyEndpoint:    proxyEndpoint,
		ProxyAccessKey:   proxyAccessKey,
		ProxySecretKey:   proxySecretKey,
		Description:      "acceptance test key store",
		SecretKeyVersion: 1,
	}
	config := func(change func(*keyStoreTestdataSpec)) string {
		change(&spec)
		return testutils.ParseTestdataConfig("./testdata/002.kms_key_store.tf.tmpl", &spec)
	}

	checkKeyStore := func(status, description string, version int) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrSet(fullResourceName, "id"),
			resource.TestCheckResourceAttr(fullResourceName, "name", testutils.ResourceName(spec.ID)),
			resource.TestCheckResourceAttr(fullResourceName, "zone", zone),
			resource.TestCheckResourceAttr(fullResourceName, "description", description),
			resource.TestCheckResourceAttr(fullResourceName, "type", "external-key-store"),
			resource.TestCheckResourceAttr(fullResourceName, "status", status),
			resource.TestCheckResourceAttr(fullResourceName, "connected", fmt.Sprint(status == "connected")),
			resource.TestCheckResourceAttrSet(fullResourceName, "created_at"),
			resource.TestCheckResourceAttr(fullResourceName, "proxy.endpoint", proxyEndpoint),
			resource.TestCheckResourceAttr(fullResourceName, "proxy.access_key", proxyAccessKey),
			resource.TestCheckResourceAttr(fullResourceName, "proxy.secret_key_wo_version", fmt.Sprint(version)),
			resource.TestCheckNoResourceAttr(fullResourceName, "proxy.secret_key_wo"),

			resource.TestCheckResourceAttrPair(byName, "id", fullResourceName, "id"),
			resource.TestCheckResourceAttrPair(byID, "name", fullResourceName, "name"),
			resource.TestCheckResourceAttr(byID, "proxy.endpoint", proxyEndpoint),
			resource.TestCheckResourceAttr(byID, "proxy.access_key", proxyAccessKey),
			resource.TestCheckResourceAttr(byID, "type", "external-key-store"),
		)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             checkKeyStoreDestroyed(zone),
		Steps: []resource.TestStep{
			// A key store is created disconnected unless connected = true.
			{
				Config: config(func(s *keyStoreTestdataSpec) {}),
				Check:  checkKeyStore("disconnected", "acceptance test key store", 1),
			},
			{
				Config: config(func(s *keyStoreTestdataSpec) {
					s.Connected = true
					s.Description = "acceptance test key store updated"
				}),
				Check: checkKeyStore("connected", "acceptance test key store updated", 1),
			},
			// A new secret version sends the credentials again, which runs a health check.
			{
				Config: config(func(s *keyStoreTestdataSpec) { s.SecretKeyVersion = 2 }),
				Check:  checkKeyStore("connected", "acceptance test key store updated", 2),
			},
			{
				Config: config(func(s *keyStoreTestdataSpec) { s.Connected = false }),
				Check:  checkKeyStore("disconnected", "acceptance test key store updated", 2),
			},
			{
				ResourceName: fullResourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s@%s", s.RootModule().Resources[fullResourceName].Primary.ID, zone), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				// The secret is never stored, its version only exists in Terraform, and the health
				// check runs on its own schedule.
				ImportStateVerifyIgnore: []string{"proxy.secret_key_wo", "proxy.secret_key_wo_version", "health", "timeouts"},
			},
		},
	})
}

// checkKeyStoreDestroyed checks that the key store is gone, as the provider detects it:
// the API answers with a 400 of type #/key-store/not-found.
func checkKeyStoreDestroyed(zone string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := testutils.APIClientV3()
		if err != nil {
			return err
		}
		ctx := context.Background()
		client, err = utils.SwitchClientZone(ctx, client, exoscale.ZoneName(zone))
		if err != nil {
			return err
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_kms_key_store" {
				continue
			}

			_, err := client.GetKeyStore(ctx, exoscale.UUID(rs.Primary.ID))
			if err == nil {
				return fmt.Errorf("key store %s still exists", rs.Primary.ID)
			}
			if problemType(err) != "#/key-store/not-found" {
				return fmt.Errorf("unexpected error reading the deleted key store %s: %w", rs.Primary.ID, err)
			}
		}

		return nil
	}
}

// problemType returns the problem type of a KMS API error, or "" for any other error.
func problemType(err error) string {
	var apiErr *exoscale.APIError
	if !errors.As(err, &apiErr) {
		return ""
	}
	if resp, ok := apiErr.Response.(*exoscale.ErrorResponse); ok {
		return resp.Type
	}
	return ""
}
