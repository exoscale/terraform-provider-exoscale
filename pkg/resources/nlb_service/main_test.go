package nlb_service_test

import "testing"

func TestNlbService(t *testing.T) {
	t.Parallel()

	t.Run("DataSourceList", testListDataSource)
}
