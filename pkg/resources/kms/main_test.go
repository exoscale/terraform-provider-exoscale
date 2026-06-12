package kms_test

import "testing"

func TestKMS(t *testing.T) {
	t.Parallel()

	t.Run("ResourceKMSKey", testResourceKMSKey)
}
