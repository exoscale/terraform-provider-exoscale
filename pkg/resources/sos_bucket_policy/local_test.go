//go:build local_integration

package sos_bucket_policy_test

import (
	"flag"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var flagAccount = flag.String("account", testutils.DefaultLocalAccount, "account name substring in exoscale.toml")

func TestSOSBucketPolicyLocal(t *testing.T) {
	testutils.LoadLocalCreds(t, *flagAccount)
	TestSOSBucketPolicy(t)
}
