package kms

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func kmsKeySchema(t *testing.T) schema.Schema {
	t.Helper()

	var resp resource.SchemaResponse
	NewResourceKMSKey().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	return resp.Schema
}

// kmsKeyValue builds a resource object where every attribute is null except the given ones.
func kmsKeyValue(t *testing.T, values map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	objType := kmsKeySchema(t).Type().TerraformType(context.Background()).(tftypes.Object)
	attrs := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		attrs[name] = tftypes.NewValue(typ, nil)
	}
	for name, v := range values {
		attrs[name] = v
	}

	return tftypes.NewValue(objType, attrs)
}

func str(v any) tftypes.Value { return tftypes.NewValue(tftypes.String, v) }

var xksType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"key_store_id":    tftypes.String,
	"external_key_id": tftypes.String,
}}

func xksValue() tftypes.Value {
	return tftypes.NewValue(xksType, map[string]tftypes.Value{
		"key_store_id":    str("a1b2c3d4-e5f6-4a5b-8c9d-0123456789ab"),
		"external_key_id": str("my-external-key"),
	})
}

// assertErrorPaths checks that diags holds exactly one error per expected path.
func assertErrorPaths(t *testing.T, diags diag.Diagnostics, want []path.Path) {
	t.Helper()

	errs := diags.Errors()
	if len(errs) != len(want) {
		t.Fatalf("got %d errors, want %d: %v", len(errs), len(want), errs)
	}
	for _, p := range want {
		found := false
		for _, d := range errs {
			if wp, ok := d.(diag.DiagnosticWithPath); ok && wp.Path().Equal(p) {
				found = true
			}
		}
		if !found {
			t.Errorf("missing error on %s: %v", p, errs)
		}
	}
}

func TestResourceKMSKeyValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		values    map[string]tftypes.Value
		wantError []path.Path
	}{
		{
			name: "no xks",
			values: map[string]tftypes.Value{
				"key_spec":   str("RSA_4096"),
				"multi_zone": tftypes.NewValue(tftypes.Bool, true),
			},
		},
		{
			name: "sign-verify with asymmetric key_spec",
			values: map[string]tftypes.Value{
				"usage":    str("sign-verify"),
				"key_spec": str("ECC_NIST_P256"),
			},
		},
		{
			name: "sign-verify with unknown key_spec",
			values: map[string]tftypes.Value{
				"usage":    str("sign-verify"),
				"key_spec": str(tftypes.UnknownValue),
			},
		},
		{
			name:      "sign-verify with default key_spec",
			values:    map[string]tftypes.Value{"usage": str("sign-verify")},
			wantError: []path.Path{path.Root("key_spec")},
		},
		{
			name: "sign-verify with AES_256",
			values: map[string]tftypes.Value{
				"usage":    str("sign-verify"),
				"key_spec": str("AES_256"),
			},
			wantError: []path.Path{path.Root("key_spec")},
		},
		{
			name:   "xks with defaults",
			values: map[string]tftypes.Value{"xks": xksValue()},
		},
		{
			name: "xks with supported values",
			values: map[string]tftypes.Value{
				"xks":        xksValue(),
				"usage":      str("encrypt-decrypt"),
				"key_spec":   str("AES_256"),
				"multi_zone": tftypes.NewValue(tftypes.Bool, false),
			},
		},
		{
			name: "xks with unknown values",
			values: map[string]tftypes.Value{
				"xks":        xksValue(),
				"key_spec":   str(tftypes.UnknownValue),
				"multi_zone": tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
			},
		},
		{
			name: "xks with unsupported values",
			values: map[string]tftypes.Value{
				"xks":        xksValue(),
				"usage":      str("sign-verify"),
				"key_spec":   str("RSA_4096"),
				"multi_zone": tftypes.NewValue(tftypes.Bool, true),
			},
			wantError: []path.Path{path.Root("usage"), path.Root("key_spec"), path.Root("multi_zone")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resource.ValidateConfigRequest{Config: tfsdk.Config{
				Schema: kmsKeySchema(t),
				Raw:    kmsKeyValue(t, tt.values),
			}}
			resp := &resource.ValidateConfigResponse{}

			(&ResourceKMSKey{}).ValidateConfig(context.Background(), req, resp)

			assertErrorPaths(t, resp.Diagnostics, tt.wantError)
		})
	}
}

func TestResourceKMSKeyModifyPlan(t *testing.T) {
	existing := map[string]tftypes.Value{
		"id":          str("019ef461-c0ae-7c9c-94bd-081e70fe2b08"),
		"name":        str("my-key"),
		"zone":        str("ch-gva-2"),
		"description": str("my description"),
		"multi_zone":  tftypes.NewValue(tftypes.Bool, false),
		"usage":       str("encrypt-decrypt"),
		"key_spec":    str("AES_256"),
		"source":      str("exoscale-kms"),
		"status":      str("enabled"),
	}
	with := func(changes map[string]tftypes.Value) map[string]tftypes.Value {
		values := make(map[string]tftypes.Value, len(existing))
		for k, v := range existing {
			values[k] = v
		}
		for k, v := range changes {
			values[k] = v
		}
		return values
	}

	tests := []struct {
		name      string
		state     tftypes.Value
		plan      map[string]tftypes.Value
		config    map[string]tftypes.Value
		wantError []path.Path
	}{
		{
			name:   "create",
			state:  tftypes.NewValue(kmsKeyValue(t, nil).Type(), nil),
			plan:   existing,
			config: existing,
		},
		{
			name:   "no change",
			state:  kmsKeyValue(t, existing),
			plan:   existing,
			config: existing,
		},
		{
			name:      "replace with same name",
			state:     kmsKeyValue(t, existing),
			plan:      with(map[string]tftypes.Value{"description": str("new description")}),
			config:    with(map[string]tftypes.Value{"description": str("new description")}),
			wantError: []path.Path{path.Root("name")},
		},
		{
			name:      "add xks with same name",
			state:     kmsKeyValue(t, existing),
			plan:      with(map[string]tftypes.Value{"xks": xksValue()}),
			config:    with(map[string]tftypes.Value{"xks": xksValue()}),
			wantError: []path.Path{path.Root("name")},
		},
		{
			name:  "replace with new name",
			state: kmsKeyValue(t, existing),
			plan: with(map[string]tftypes.Value{
				"name":        str("my-new-key"),
				"description": str("new description"),
			}),
			config: with(map[string]tftypes.Value{
				"name":        str("my-new-key"),
				"description": str("new description"),
			}),
		},
		{
			name:   "unconfigured computed value",
			state:  kmsKeyValue(t, existing),
			plan:   with(map[string]tftypes.Value{"description": str(tftypes.UnknownValue)}),
			config: with(map[string]tftypes.Value{"description": str(nil)}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := kmsKeySchema(t)
			req := resource.ModifyPlanRequest{
				State:  tfsdk.State{Schema: s, Raw: tt.state},
				Plan:   tfsdk.Plan{Schema: s, Raw: kmsKeyValue(t, tt.plan)},
				Config: tfsdk.Config{Schema: s, Raw: kmsKeyValue(t, tt.config)},
			}
			resp := &resource.ModifyPlanResponse{Plan: req.Plan}

			(&ResourceKMSKey{}).ModifyPlan(context.Background(), req, resp)

			assertErrorPaths(t, resp.Diagnostics, tt.wantError)
		})
	}
}
