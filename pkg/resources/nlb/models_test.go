package nlb

import (
	"context"
	"encoding/json"
	"maps"
	"testing"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

func TestLabelSelectorToAPI(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// A null attribute leaves the selector alone, so the key must not be sent.
	selector, diags := labelSelectorToAPI(ctx, types.MapNull(types.StringType))
	require.False(t, diags.HasError())
	require.Nil(t, selector)

	// An empty selector asks for the removal, which only the request body can
	// express. Neither an omitted key nor a null value removes anything.
	selector, diags = labelSelectorToAPI(ctx, types.MapValueMust(types.StringType, map[string]attr.Value{}))
	require.False(t, diags.HasError())
	require.NotNil(t, selector)

	body, err := json.Marshal(exoscale.UpdateLoadBalancerServiceRequest{LabelSelector: selector})
	require.NoError(t, err)
	require.Equal(t, `{"label-selector":{"labels":{}}}`, string(body))

	// Label keys are sent exactly as they are.
	selector, diags = labelSelectorToAPI(ctx, types.MapValueMust(types.StringType, map[string]attr.Value{
		"role":                    types.StringValue("web"),
		"exoscale.com/managed-by": types.StringValue("terraform"),
	}))
	require.False(t, diags.HasError())
	require.Equal(t, exoscale.Labels{
		"role":                    "web",
		"exoscale.com/managed-by": "terraform",
	}, selector.Labels)
}

func TestLabelSelectorFromAPI(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// A service with no selector has no `label-selector` key in its response.
	// The API never returns an object with no labels, but read one as a null
	// attribute too rather than turning it into an empty map.
	for _, in := range []*exoscale.LoadBalancerServiceLabelSelector{
		nil,
		{},
		{Labels: exoscale.Labels{}},
	} {
		selector, diags := labelSelectorFromAPI(ctx, in)
		require.False(t, diags.HasError())
		require.Equal(t, types.MapNull(types.StringType), selector)
	}

	selector, diags := labelSelectorFromAPI(ctx, &exoscale.LoadBalancerServiceLabelSelector{
		Labels: exoscale.Labels{
			"role":                    "web",
			"exoscale.com/managed-by": "terraform",
		},
	})
	require.False(t, diags.HasError())
	require.Equal(t, types.MapValueMust(types.StringType, map[string]attr.Value{
		"role":                    types.StringValue("web"),
		"exoscale.com/managed-by": types.StringValue("terraform"),
	}), selector)
}

// TestResourceServiceBackendValidators checks the schema validators on the two
// backend attributes. A service requires an instance pool, a label selector, or
// both, and a label selector requires at least one label.
func TestResourceServiceBackendValidators(t *testing.T) {
	t.Parallel()

	const poolID = "5bea9f0a-4b0b-4b0b-8b0b-0b0b0b0b0b0b"

	tests := []struct {
		name      string
		values    map[string]tftypes.Value
		wantError []path.Path
	}{
		{
			name:   "instance pool only",
			values: map[string]tftypes.Value{"instance_pool_id": strValue(poolID)},
		},
		{
			name:   "label selector only",
			values: map[string]tftypes.Value{"label_selector": labelSelectorValue(map[string]string{"role": "web"})},
		},
		{
			// A pool created in the same apply is still unknown at this point.
			name:   "unknown instance pool",
			values: map[string]tftypes.Value{"instance_pool_id": strValue(tftypes.UnknownValue)},
		},
		{
			name: "unknown label selector",
			values: map[string]tftypes.Value{
				"instance_pool_id": strValue(poolID),
				"label_selector":   tftypes.NewValue(labelSelectorType, tftypes.UnknownValue),
			},
		},
		{
			// Both backend values are unknown or null, so AtLeastOneOf defers the
			// check rather than rejecting the configuration.
			name: "null pool, unknown selector",
			values: map[string]tftypes.Value{
				"label_selector": tftypes.NewValue(labelSelectorType, tftypes.UnknownValue),
			},
		},
		{
			name:      "no backend source",
			wantError: []path.Path{path.Root("instance_pool_id")},
		},
		{
			// The API cannot hold a selector with no labels, so an empty map
			// could never be read back.
			name: "empty label selector",
			values: map[string]tftypes.Value{
				"instance_pool_id": strValue(poolID),
				"label_selector":   labelSelectorValue(nil),
			},
			wantError: []path.Path{path.Root("label_selector")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := serviceSchema(t)
			config := serviceConfig(t, tt.values)

			var (
				poolIDValue    types.String
				labelValue     types.Map
				poolPath       = path.Root("instance_pool_id")
				labelPath      = path.Root("label_selector")
				poolAttribute  = s.Attributes["instance_pool_id"].(schema.StringAttribute)
				labelAttribute = s.Attributes["label_selector"].(schema.MapAttribute)
				diags          diag.Diagnostics
			)

			diags.Append(config.GetAttribute(ctx, poolPath, &poolIDValue)...)
			diags.Append(config.GetAttribute(ctx, labelPath, &labelValue)...)
			require.False(t, diags.HasError(), "%v", diags.Errors())

			// The framework runs these validators on its own during a plan, but
			// calling them directly keeps the test independent of a real run.
			for _, v := range poolAttribute.Validators {
				resp := &validator.StringResponse{}
				v.ValidateString(ctx, validator.StringRequest{
					Path:           poolPath,
					PathExpression: poolPath.Expression(),
					Config:         config,
					ConfigValue:    poolIDValue,
				}, resp)
				diags.Append(resp.Diagnostics...)
			}

			for _, v := range labelAttribute.Validators {
				resp := &validator.MapResponse{}
				v.ValidateMap(ctx, validator.MapRequest{
					Path:           labelPath,
					PathExpression: labelPath.Expression(),
					Config:         config,
					ConfigValue:    labelValue,
				}, resp)
				diags.Append(resp.Diagnostics...)
			}

			errs := diags.Errors()
			require.Len(t, errs, len(tt.wantError), "%v", errs)

			for _, p := range tt.wantError {
				found := false
				for _, d := range errs {
					if dp, ok := d.(diag.DiagnosticWithPath); ok && dp.Path().Equal(p) {
						found = true
					}
				}
				require.True(t, found, "missing error on %s: %v", p, errs)
			}
		})
	}
}

func serviceSchema(t *testing.T) schema.Schema {
	t.Helper()

	var resp resource.SchemaResponse
	NewResourceService().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	return resp.Schema
}

// serviceConfig builds a resource configuration in which every attribute is
// null except the ones given.
func serviceConfig(t *testing.T, values map[string]tftypes.Value) tfsdk.Config {
	t.Helper()

	s := serviceSchema(t)

	objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
	attrs := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		attrs[name] = tftypes.NewValue(typ, nil)
	}
	maps.Copy(attrs, values)

	return tfsdk.Config{Schema: s, Raw: tftypes.NewValue(objType, attrs)}
}

var labelSelectorType = tftypes.Map{ElementType: tftypes.String}

// labelSelectorValue builds a `label_selector` value. A nil map gives an empty
// one, which the API reads as a removal.
func labelSelectorValue(labels map[string]string) tftypes.Value {
	elements := make(map[string]tftypes.Value, len(labels))
	for k, v := range labels {
		elements[k] = tftypes.NewValue(tftypes.String, v)
	}

	return tftypes.NewValue(labelSelectorType, elements)
}

func strValue(v any) tftypes.Value {
	return tftypes.NewValue(tftypes.String, v)
}
