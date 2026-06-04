//go:build local_integration

package block_storage_test

import (
	"flag"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var flagAccount = flag.String("account", testutils.DefaultLocalAccount, "account name substring in exoscale.toml")

func TestBlockStorageLocal(t *testing.T) {
	testutils.LoadLocalCreds(t, *flagAccount)
	TestBlockStorage(t)
}
