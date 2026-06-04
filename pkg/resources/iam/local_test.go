//go:build local_integration

package iam_test

import (
	"flag"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var flagAccount = flag.String("account", testutils.DefaultLocalAccount, "account name substring in exoscale.toml")

func TestIAMLocal(t *testing.T) {
	testutils.LoadLocalCreds(t, *flagAccount)
	TestIAM(t)
}
