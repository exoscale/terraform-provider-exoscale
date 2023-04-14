package anti_affinity_group_test

import "testing"

func TestAntiAffinityGroup(t *testing.T) {
	t.Run("DataSource", testDataSource)
	t.Run("Resource", testResource)
}
