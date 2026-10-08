//go:build local_integration

package ssh_key_test

import (
	"flag"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var flagAccount = flag.String("account", testutils.DefaultLocalAccount, "account name substring in exoscale.toml")

func TestSSHKeyLocal(t *testing.T) {
	testutils.LoadLocalCreds(t, *flagAccount)
	TestSSHKey(t)
}
