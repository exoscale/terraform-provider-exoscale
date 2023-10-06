package iam_test

import "testing"

func TestIAM(t *testing.T) {
	t.Run("DataSourceOrgPolicy", testDataSourceOrgPolicy)
	t.Run("ResourceOrgPolicy", testResourceOrgPolicy)
	t.Run("ResourceRole", testResourceRole)
	t.Run("DataSourceRole", testDataSourceRole)
	t.Run("DataSourceAPIKey", testDataSourceAPIKey)
	t.Run("ResourceAPIKey", testResourceAPIKey)
}
