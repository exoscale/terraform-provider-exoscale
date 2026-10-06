package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRefreshString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		state  types.String
		remote string
		want   types.String
	}{
		{"unknown", types.StringUnknown(), "a", types.StringValue("a")},
		{"unknown, empty remote", types.StringUnknown(), "", types.StringValue("")},
		{"null kept", types.StringNull(), "", types.StringNull()},
		{"empty kept", types.StringValue(""), "", types.StringValue("")},
		{"changed", types.StringValue("a"), "b", types.StringValue("b")},
		{"removed remotely", types.StringValue("a"), "", types.StringValue("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.state
			RefreshString(&state, tt.remote)
			if !state.Equal(tt.want) {
				t.Errorf("got %s, want %s", state, tt.want)
			}
		})
	}
}

func TestRefreshStringPointer(t *testing.T) {
	t.Parallel()

	a := "a"
	empty := ""

	tests := []struct {
		name   string
		state  types.String
		remote *string
		want   types.String
	}{
		{"unknown", types.StringUnknown(), &a, types.StringValue("a")},
		{"unknown, nil remote", types.StringUnknown(), nil, types.StringNull()},
		{"null kept", types.StringNull(), nil, types.StringNull()},
		{"null kept on empty", types.StringNull(), &empty, types.StringNull()},
		{"empty kept", types.StringValue(""), nil, types.StringValue("")},
		{"removed remotely", types.StringValue("b"), nil, types.StringValue("")},
		{"changed", types.StringValue("b"), &a, types.StringValue("a")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.state
			RefreshStringPointer(&state, tt.remote)
			if !state.Equal(tt.want) {
				t.Errorf("got %s, want %s", state, tt.want)
			}
		})
	}
}

func TestRefreshInt64Pointer(t *testing.T) {
	t.Parallel()

	one := int64(1)

	tests := []struct {
		name   string
		state  types.Int64
		remote *int64
		want   types.Int64
	}{
		{"unknown", types.Int64Unknown(), &one, types.Int64Value(1)},
		{"unknown, nil remote", types.Int64Unknown(), nil, types.Int64Null()},
		{"null kept", types.Int64Null(), nil, types.Int64Null()},
		{"zero kept", types.Int64Value(0), nil, types.Int64Value(0)},
		{"removed remotely", types.Int64Value(2), nil, types.Int64Value(0)},
		{"changed", types.Int64Value(2), &one, types.Int64Value(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.state
			RefreshInt64Pointer(&state, tt.remote)
			if !state.Equal(tt.want) {
				t.Errorf("got %s, want %s", state, tt.want)
			}
		})
	}
}

func TestRefreshLabels(t *testing.T) {
	t.Parallel()

	labels := func(m map[string]string) types.Map {
		values := map[string]attr.Value{}
		for k, v := range m {
			values[k] = types.StringValue(v)
		}
		return types.MapValueMust(types.StringType, values)
	}

	tests := []struct {
		name   string
		state  types.Map
		remote map[string]string
		want   types.Map
	}{
		{"unknown", types.MapUnknown(types.StringType), map[string]string{"a": "b"}, labels(map[string]string{"a": "b"})},
		{"unknown, no remote labels", types.MapUnknown(types.StringType), nil, types.MapNull(types.StringType)},
		{"null kept", types.MapNull(types.StringType), nil, types.MapNull(types.StringType)},
		{"empty kept", labels(map[string]string{}), map[string]string{}, labels(map[string]string{})},
		{"changed", labels(map[string]string{"a": "b"}), map[string]string{"a": "c"}, labels(map[string]string{"a": "c"})},
		{"removed remotely", labels(map[string]string{"a": "b"}), nil, types.MapNull(types.StringType)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.state
			if dg := RefreshLabels(context.Background(), &state, tt.remote); dg.HasError() {
				t.Fatal(dg)
			}
			if !state.Equal(tt.want) {
				t.Errorf("got %s, want %s", state, tt.want)
			}
		})
	}
}

func TestRefreshStringSet(t *testing.T) {
	t.Parallel()

	set := func(values ...string) types.Set {
		elems := make([]attr.Value, len(values))
		for i, v := range values {
			elems[i] = types.StringValue(v)
		}
		return types.SetValueMust(types.StringType, elems)
	}

	tests := []struct {
		name   string
		state  types.Set
		remote []string
		want   types.Set
	}{
		{"unknown", types.SetUnknown(types.StringType), []string{"a"}, set("a")},
		{"unknown, empty remote", types.SetUnknown(types.StringType), nil, types.SetNull(types.StringType)},
		{"null kept", types.SetNull(types.StringType), nil, types.SetNull(types.StringType)},
		{"empty kept", set(), []string{}, set()},
		{"changed", set("a"), []string{"a", "b"}, set("a", "b")},
		{"removed remotely", set("a", "b"), nil, types.SetNull(types.StringType)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dg diag.Diagnostics

			state := tt.state
			RefreshStringSet(context.Background(), &dg, &state, tt.remote)
			if dg.HasError() {
				t.Fatal(dg)
			}
			if !state.Equal(tt.want) {
				t.Errorf("got %s, want %s", state, tt.want)
			}
		})
	}
}
