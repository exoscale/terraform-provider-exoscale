package exoscale

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"exoscale": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {
	key := os.Getenv("CLOUDSTACK_KEY")
	secret := os.Getenv("CLOUDSTACK_SECRET")
	if key == "" || secret == "" {
		t.Fatal("CLOUDSTACK_KEY and CLOUDSTACK_SECRET must be set for acceptance tests")
	}
}

var EXOSCALE_ZONE = "ch-dk-2"
var EXOSCALE_TEMPLATE = "Linux Ubuntu 17.10 64-bit"
var EXOSCALE_NETWORK_OFFERING = "PrivNet"
