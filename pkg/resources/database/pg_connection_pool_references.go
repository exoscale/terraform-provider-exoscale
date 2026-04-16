package database

import (
	"context"
	"fmt"
	"sort"
	"strings"

	v3 "github.com/exoscale/egoscale/v3"
)

func getPGConnectionPoolsUsingDatabase(ctx context.Context, client *v3.Client, serviceName, databaseName string) ([]string, error) {
	return getPGConnectionPools(ctx, client, serviceName, func(pool v3.DBAASServicePGConnectionPools) bool {
		return string(pool.Database) == databaseName
	})
}

func getPGConnectionPoolsUsingUsername(ctx context.Context, client *v3.Client, serviceName, username string) ([]string, error) {
	return getPGConnectionPools(ctx, client, serviceName, func(pool v3.DBAASServicePGConnectionPools) bool {
		return string(pool.Username) == username
	})
}

func getPGConnectionPools(
	ctx context.Context,
	client *v3.Client,
	serviceName string,
	matches func(v3.DBAASServicePGConnectionPools) bool,
) ([]string, error) {
	svc, err := waitForDBAASServiceReadyForFn(ctx, client.GetDBAASServicePG, serviceName, func(t *v3.DBAASServicePG) bool {
		return t.State == v3.EnumServiceStateRunning
	})
	if err != nil {
		return nil, err
	}

	return findPGConnectionPools(svc.ConnectionPools, matches), nil
}

func findPGConnectionPools(
	pools []v3.DBAASServicePGConnectionPools,
	matches func(v3.DBAASServicePGConnectionPools) bool,
) []string {
	blockingPools := make([]string, 0)

	for _, pool := range pools {
		if matches(pool) {
			blockingPools = append(blockingPools, string(pool.Name))
		}
	}

	sort.Strings(blockingPools)

	return blockingPools
}

func formatPGConnectionPoolReferences(poolNames []string) string {
	quoted := make([]string, len(poolNames))

	for i, poolName := range poolNames {
		quoted[i] = fmt.Sprintf("%q", poolName)
	}

	return strings.Join(quoted, ", ")
}
