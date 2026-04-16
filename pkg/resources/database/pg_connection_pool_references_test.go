package database

import (
	"reflect"
	"testing"

	v3 "github.com/exoscale/egoscale/v3"
)

func TestFindPGConnectionPools(t *testing.T) {
	t.Parallel()

	pools := []v3.DBAASServicePGConnectionPools{
		{
			Name:     "beta-pool",
			Database: "foo_db",
			Username: "foo",
		},
		{
			Name:     "alpha-pool",
			Database: "foo_db",
			Username: "bar",
		},
		{
			Name:     "gamma-pool",
			Database: "bar_db",
			Username: "foo",
		},
	}

	userPools := findPGConnectionPools(pools, func(pool v3.DBAASServicePGConnectionPools) bool {
		return string(pool.Username) == "foo"
	})

	expectedUserPools := []string{"beta-pool", "gamma-pool"}
	if !reflect.DeepEqual(userPools, expectedUserPools) {
		t.Fatalf("unexpected connection pools for user: got %v want %v", userPools, expectedUserPools)
	}

	databasePools := findPGConnectionPools(pools, func(pool v3.DBAASServicePGConnectionPools) bool {
		return string(pool.Database) == "foo_db"
	})

	expectedDatabasePools := []string{"alpha-pool", "beta-pool"}
	if !reflect.DeepEqual(databasePools, expectedDatabasePools) {
		t.Fatalf("unexpected connection pools for database: got %v want %v", databasePools, expectedDatabasePools)
	}
}

func TestFormatPGConnectionPoolReferences(t *testing.T) {
	t.Parallel()

	got := formatPGConnectionPoolReferences([]string{"alpha-pool", "beta-pool"})
	want := `"alpha-pool", "beta-pool"`

	if got != want {
		t.Fatalf("unexpected formatted connection pools: got %q want %q", got, want)
	}
}
