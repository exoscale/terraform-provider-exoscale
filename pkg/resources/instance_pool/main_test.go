package instance_pool_test

import "testing"

func TestInstancePool(t *testing.T) {
	t.Parallel()

	t.Run("DataSource", testDataSource)
	t.Run("DataSourceList", testListDataSource)
	t.Run("Resource", testResource)
	t.Run("ResourcePrivate", testResourcePrivate)
}
