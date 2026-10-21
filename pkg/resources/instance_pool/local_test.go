//go:build local_integration

package instance_pool_test

import (
	"flag"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var flagAccount = flag.String("account", testutils.DefaultLocalAccount, "account name substring in exoscale.toml")

func TestInstancePoolLocal(t *testing.T) {
	testutils.LoadLocalCreds(t, *flagAccount)
	TestInstancePool(t)
}
