package utils

import (
	"testing"

	exoscale "github.com/exoscale/egoscale/v3"
)

func TestInstanceTypeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		family exoscale.InstanceTypeFamily
		size   exoscale.InstanceTypeSize
		want   string
	}{
		{exoscale.InstanceTypeFamilyStandard, exoscale.InstanceTypeSizeMedium, "standard.medium"},
		{exoscale.InstanceTypeFamilyGpua30, exoscale.InstanceTypeSizeSmall, "gpua30.small"},
		{exoscale.InstanceTypeFamilyStandard, exoscale.InstanceTypeSizeExtraLarge, "standard.extra-large"},
		{"GPU2", "Medium", "gpu2.medium"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			got := InstanceTypeName(&exoscale.InstanceType{Family: tt.family, Size: tt.size})
			if got != tt.want {
				t.Errorf("InstanceTypeName() = %q, want %q", got, tt.want)
			}
		})
	}
}
