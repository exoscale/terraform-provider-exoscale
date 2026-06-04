//go:build local_integration

package database_test

import (
	"flag"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var flagAccount = flag.String("account", testutils.DefaultLocalAccount, "account name substring in exoscale.toml")

// TestDatabaseLocal mirrors TestDatabase but pulls credentials from
// ~/.config/exoscale/exoscale.toml via the -account flag. Run with:
//
//	TF_ACC=1 go test -tags=local_integration -run TestDatabaseLocal \
//	  -account=<name> -timeout 60m ./pkg/resources/database/
func TestDatabaseLocal(t *testing.T) {
	testutils.LoadLocalCreds(t, *flagAccount)
	TestDatabase(t)
}
