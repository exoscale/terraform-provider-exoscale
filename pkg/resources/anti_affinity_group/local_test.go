//go:build local_integration

package anti_affinity_group_test

import (
	"flag"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var flagAccount = flag.String("account", testutils.DefaultLocalAccount, "account name substring in exoscale.toml")

func TestAntiAffinityGroupLocal(t *testing.T) {
	testutils.LoadLocalCreds(t, *flagAccount)
	TestAntiAffinityGroup(t)
}
