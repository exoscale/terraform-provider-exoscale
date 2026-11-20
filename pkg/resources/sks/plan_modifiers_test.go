package sks

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

var nodepoolTemplateTestSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"template_id":        schema.StringAttribute{Optional: true, Computed: true},
		"template_selection": schema.StringAttribute{Computed: true},
	},
}

func nodepoolTemplateTestValue(templateID, selection tftypes.Value) tftypes.Value {
	return tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"template_id":        tftypes.String,
			"template_selection": tftypes.String,
		}},
		map[string]tftypes.Value{
			"template_id":        templateID,
			"template_selection": selection,
		},
	)
}

var (
	tfNull    = tftypes.NewValue(tftypes.String, nil)
	tfUnknown = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
)

func tfString(s string) tftypes.Value {
	return tftypes.NewValue(tftypes.String, s)
}

func TestNodepoolTemplateSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		templateID tftypes.Value
		expected   string
	}{
		{name: "template_id unset", templateID: tfNull, expected: "auto"},
		{name: "template_id set", templateID: tfString("tpl"), expected: "pinned"},
		{name: "template_id unknown", templateID: tfUnknown, expected: "pinned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				Config: tfsdk.Config{Schema: nodepoolTemplateTestSchema, Raw: nodepoolTemplateTestValue(tt.templateID, tfNull)},
			}
			resp := &planmodifier.StringResponse{PlanValue: types.StringUnknown()}

			nodepoolTemplateSelection{}.PlanModifyString(context.Background(), req, resp)

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			require.Equal(t, types.StringValue(tt.expected), resp.PlanValue)
		})
	}
}

func TestNodepoolTemplateIDUnpinned(t *testing.T) {
	t.Parallel()

	state := func(selection string) tftypes.Value {
		return nodepoolTemplateTestValue(tfString("tpl"), tfString(selection))
	}

	tests := []struct {
		name     string
		config   types.String
		state    tftypes.Value
		expected types.String
	}{
		{
			name:     "creation",
			config:   types.StringNull(),
			state:    tftypes.NewValue(nodepoolTemplateTestValue(tfNull, tfNull).Type(), nil),
			expected: types.StringValue("tpl"),
		},
		{
			name:     "pinned and configured",
			config:   types.StringValue("tpl"),
			state:    state("pinned"),
			expected: types.StringValue("tpl"),
		},
		{
			name:     "pinned and removed from configuration",
			config:   types.StringNull(),
			state:    state("pinned"),
			expected: types.StringUnknown(),
		},
		{
			name:     "auto",
			config:   types.StringNull(),
			state:    state("auto"),
			expected: types.StringValue("tpl"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				ConfigValue: tt.config,
				State:       tfsdk.State{Schema: nodepoolTemplateTestSchema, Raw: tt.state},
			}
			resp := &planmodifier.StringResponse{PlanValue: types.StringValue("tpl")}

			nodepoolTemplateIDUnpinned{}.PlanModifyString(context.Background(), req, resp)

			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			require.Equal(t, tt.expected, resp.PlanValue)
		})
	}
}
