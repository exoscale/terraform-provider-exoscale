package sks

import (
	"context"
	"testing"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestSKSNodepoolTaintsValue(t *testing.T) {
	t.Parallel()

	empty := types.MapValueMust(types.StringType, map[string]attr.Value{})
	nonempty := types.MapValueMust(types.StringType, map[string]attr.Value{
		"dedicated": types.StringValue("system:NoSchedule"),
	})
	apiTaints := v3.SKSNodepoolTaints{
		"dedicated": {Value: "system", Effect: "NoSchedule"},
	}

	tests := []struct {
		name    string
		current types.Map
		api     v3.SKSNodepoolTaints
		want    types.Map
	}{
		{"omitted attribute and nil API map", types.MapNull(types.StringType), nil, types.MapNull(types.StringType)},
		{"omitted attribute and empty API map", types.MapNull(types.StringType), v3.SKSNodepoolTaints{}, types.MapNull(types.StringType)},
		{"explicit empty map and nil API map", empty, nil, empty},
		{"explicit empty map and empty API map", empty, v3.SKSNodepoolTaints{}, empty},
		{"unknown attribute and no API taints", types.MapUnknown(types.StringType), nil, types.MapNull(types.StringType)},
		{"nonempty taints", nonempty, apiTaints, nonempty},
		{"import discovers API taints", types.MapNull(types.StringType), apiTaints, nonempty},
		{"unknown attribute discovers API taints", types.MapUnknown(types.StringType), apiTaints, nonempty},
		{"API taints cleared out of band", nonempty, nil, empty},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diagnostics := sksNodepoolTaintsValue(context.Background(), tt.api, tt.current)
			require.False(t, diagnostics.HasError(), "unexpected diagnostics: %v", diagnostics)
			require.True(t, got.Equal(tt.want), "got %s, want %s", got, tt.want)
		})
	}
}
