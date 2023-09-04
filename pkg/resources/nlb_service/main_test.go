package nlb_service_test

import "testing"

func TestNlbService(t *testing.T) {
	t.Run("DataSourceList", testListDataSource)
}
