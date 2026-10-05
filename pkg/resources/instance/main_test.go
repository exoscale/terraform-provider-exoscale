package instance_test

import "testing"

func TestInstance(t *testing.T) {
	t.Parallel()

	t.Run("DataSource", testDataSource)
	t.Run("DataSourceList", testListDataSource)
	t.Run("Resource", testResource)
	t.Run("Resource/ManagedNetworkInterface", testResourceManagedNetworkInterface)
	t.Run("Resource/SSHKeys", testResourceSSHKeys)
	t.Run("DestroyProtection/ExplicitValue", testExplicitDestroyProtection)
	t.Run("DestroyProtection/DefaultValue", testDefaultDestroyProtection)
}
