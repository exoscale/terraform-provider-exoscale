package instance_test

import "testing"

func TestInstance(t *testing.T) {
	t.Run("DataSource", testDataSource)
	t.Run("DataSourceList", testListDataSource)
	t.Run("Resource", testResource)
}
