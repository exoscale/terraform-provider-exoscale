//go:build local_integration

package nlb_service_test

import (
	"flag"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var flagAccount = flag.String("account", testutils.DefaultLocalAccount, "account name substring in exoscale.toml")

func TestNlbServiceLocal(t *testing.T) {
	testutils.LoadLocalCreds(t, *flagAccount)
	TestNlbService(t)
}
