package kms

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
			name:      "sign-only key_spec with default usage",
			values:    map[string]tftypes.Value{"key_spec": str("ECC_NIST_P256")},
			wantError: []path.Path{path.Root("usage")},
		},
		{
			name: "sign-only key_spec with encrypt-decrypt",
			values: map[string]tftypes.Value{
				"usage":    str("encrypt-decrypt"),
				"key_spec": str("ML_DSA_65"),
			},
			wantError: []path.Path{path.Root("usage")},
		},
		{
			name: "RSA with encrypt-decrypt",
			values: map[string]tftypes.Value{
				"usage":    str("encrypt-decrypt"),
				"key_spec": str("RSA_3072"),
			},
		},
		{
			name: "sign-only key_spec with unknown usage",
			values: map[string]tftypes.Value{
				"usage":    str(tftypes.UnknownValue),
				"key_spec": str("ECC_NIST_P384"),
			},
		},
		{
			name: "unknown xks",
			values: map[string]tftypes.Value{
				"xks":      tftypes.NewValue(xksType, tftypes.UnknownValue),
				"usage":    str("sign-verify"),
				"key_spec": str("RSA_4096"),
			},
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
			name:  "new zone with same name",
			state: kmsKeyValue(t, existing),
			plan: with(map[string]tftypes.Value{
				"zone":        str("de-fra-1"),
				"description": str("new description"),
			}),
			config: with(map[string]tftypes.Value{
				"zone":        str("de-fra-1"),
				"description": str("new description"),
			}),
		},
		{
			name: "new zone with same name for a multi-zone key",
			state: kmsKeyValue(t, with(map[string]tftypes.Value{
				"multi_zone": tftypes.NewValue(tftypes.Bool, true),
			})),
			plan: with(map[string]tftypes.Value{
				"zone":       str("de-fra-1"),
				"multi_zone": tftypes.NewValue(tftypes.Bool, true),
			}),
			config: with(map[string]tftypes.Value{
				"zone":       str("de-fra-1"),
				"multi_zone": tftypes.NewValue(tftypes.Bool, true),
			}),
			wantError: []path.Path{path.Root("name")},
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

const testKeyID = "019ef461-c0ae-7c9c-94bd-081e70fe2b08"

// kmsKeyJSON is an External Key Store key as returned by the API.
func kmsKeyJSON(status string) string {
	return fmt.Sprintf(`{
		"id": %q, "name": "my-key", "description": "", "status": %q,
		"usage": "encrypt-decrypt", "key-spec": "AES_256", "multi-zone": false,
		"source": "external-key-store", "xks": {"xks-id": %q, "external-key-id": "my-external-key"}
	}`, testKeyID, status, testKeyStoreID)
}

func TestResourceKMSKeyRead(t *testing.T) {
	// The API returns the key store ID in lower case.
	configuredKeyStoreID := strings.ToUpper(testKeyStoreID)

	tests := []struct {
		name        string
		status      int
		body        string
		wantRemoved bool
	}{
		{name: "xks key", status: 200, body: kmsKeyJSON("enabled")},
		{name: "pending deletion", status: 200, body: kmsKeyJSON("pending-deletion"), wantRemoved: true},
		{name: "not found", status: 400, body: mockProblem(kmsErrKeyNotFound, "Key not found"), wantRemoved: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newMockAPI(t).expect("GET /kms-key/"+testKeyID, tt.status, tt.body)
			s := kmsKeySchema(t)
			state := kmsKeyValue(t, map[string]tftypes.Value{
				"id":   str(testKeyID),
				"zone": str(testZone),
				"xks": tftypes.NewValue(xksType, map[string]tftypes.Value{
					"key_store_id":    str(configuredKeyStoreID),
					"external_key_id": str("my-external-key"),
				}),
			})

			resp := &resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: state}}
			(&ResourceKMSKey{client: api.client()}).Read(context.Background(), resource.ReadRequest{
				State: tfsdk.State{Schema: s, Raw: state},
			}, resp)

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if tt.wantRemoved {
				if !resp.State.Raw.IsNull() {
					t.Error("key must be removed from the state")
				}
				return
			}

			var got ResourceKMSKeyModel
			var gotXks ResourceKMSKeyXksModel
			resp.Diagnostics.Append(resp.State.Get(context.Background(), &got)...)
			resp.Diagnostics.Append(got.Xks.As(context.Background(), &gotXks, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if gotXks.KeyStoreID.ValueString() != configuredKeyStoreID || gotXks.ExternalKeyID.ValueString() != "my-external-key" {
				t.Errorf("unexpected xks %+v", gotXks)
			}
			if got.Source.ValueString() != "external-key-store" || got.KeySpec.ValueString() != "AES_256" {
				t.Errorf("unexpected key %+v", got)
			}
		})
	}
}

func TestResourceKMSKeyDelete(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantError bool
	}{
		{name: "scheduled", status: 200, body: `{"delete-at":"2026-10-30T12:00:00Z"}`},
		{name: "already gone", status: 400, body: mockProblem(kmsErrKeyNotFound, "Key not found")},
		{
			name:      "other error",
			status:    400,
			body:      mockProblem("#/kms-key/operation-not-allowed-on-default-key", "Operation not allowed on default key"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newMockAPI(t).expect("POST /kms-key/"+testKeyID+"/schedule-deletion", tt.status, tt.body)
			s := kmsKeySchema(t)
			state := kmsKeyValue(t, map[string]tftypes.Value{"id": str(testKeyID), "zone": str(testZone)})

			resp := &resource.DeleteResponse{State: tfsdk.State{Schema: s, Raw: state}}
			(&ResourceKMSKey{client: api.client()}).Delete(context.Background(), resource.DeleteRequest{
				State: tfsdk.State{Schema: s, Raw: state},
			}, resp)

			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("got errors %v, want error: %t", resp.Diagnostics.Errors(), tt.wantError)
			}
		})
	}
}

func TestResourceKMSKeyImportState(t *testing.T) {
	assertImportState(t, &ResourceKMSKey{}, testKeyID)
}

func TestResourceKMSKeyCreate(t *testing.T) {
	ctx := context.Background()
	api := newMockAPI(t).expect("POST /kms-key", 200, kmsKeyJSON("enabled"))
	s := kmsKeySchema(t)

	// An XKS key with every optional attribute left to the API.
	values := map[string]tftypes.Value{
		"name": str("my-key"),
		"zone": str(testZone),
		"xks": tftypes.NewValue(xksType, map[string]tftypes.Value{
			"key_store_id":    str(testKeyStoreID),
			"external_key_id": str("my-external-key"),
		}),
	}
	plan := map[string]tftypes.Value{}
	for k, v := range values {
		plan[k] = v
	}
	for _, name := range []string{"id", "description", "multi_zone", "usage", "key_spec", "source", "status"} {
		plan[name] = tftypes.NewValue(kmsKeyValue(t, nil).Type().(tftypes.Object).AttributeTypes[name], tftypes.UnknownValue)
	}

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(kmsKeyValue(t, nil).Type(), nil)}}
	(&ResourceKMSKey{client: api.client()}).Create(ctx, resource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: s, Raw: kmsKeyValue(t, plan)},
		Config: tfsdk.Config{Schema: s, Raw: kmsKeyValue(t, values)},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	body := api.requestBody("POST /kms-key", 0)
	xks, _ := body["xks"].(map[string]any)
	if body["source"] != "external-key-store" || xks["xks-id"] != testKeyStoreID || xks["external-key-id"] != "my-external-key" {
		t.Errorf("unexpected create body %v", body)
	}
	if body["usage"] != nil || body["key-spec"] != nil || body["multi-zone"] != nil {
		t.Errorf("unset attributes must be left to the API: %v", body)
	}
	if !resp.State.Raw.IsFullyKnown() {
		t.Errorf("state must be fully known after create: %s", resp.State.Raw)
	}

	var got ResourceKMSKeyModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.ID.ValueString() != testKeyID || got.Source.ValueString() != "external-key-store" ||
		got.KeySpec.ValueString() != "AES_256" || got.Usage.ValueString() != "encrypt-decrypt" {
		t.Errorf("unexpected state %+v", got)
	}
}

func TestResourceKMSKeyUpdate(t *testing.T) {
	ctx := context.Background()
	s := kmsKeySchema(t)
	objType := kmsKeyValue(t, nil).Type().(tftypes.Object)

	state := map[string]tftypes.Value{
		"id":          str(testKeyID),
		"name":        str("my-key"),
		"zone":        str(testZone),
		"description": str(""),
		"multi_zone":  tftypes.NewValue(tftypes.Bool, false),
		"usage":       str("encrypt-decrypt"),
		"key_spec":    str("AES_256"),
		"source":      str("exoscale-kms"),
		"status":      str("enabled"),
	}
	timeoutsType := objType.AttributeTypes["timeouts"].(tftypes.Object)
	timeouts := map[string]tftypes.Value{}
	for name, typ := range timeoutsType.AttributeTypes {
		timeouts[name] = tftypes.NewValue(typ, nil)
	}
	timeouts["update"] = str("10m")

	// Only timeouts changed: Terraform plans the computed attributes without plan modifier as unknown.
	plan := map[string]tftypes.Value{"timeouts": tftypes.NewValue(timeoutsType, timeouts)}
	for k, v := range state {
		plan[k] = v
	}
	for _, name := range []string{"description", "multi_zone", "usage", "status"} {
		plan[name] = tftypes.NewValue(objType.AttributeTypes[name], tftypes.UnknownValue)
	}

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: kmsKeyValue(t, state)}}
	(&ResourceKMSKey{}).Update(ctx, resource.UpdateRequest{
		State: tfsdk.State{Schema: s, Raw: kmsKeyValue(t, state)},
		Plan:  tfsdk.Plan{Schema: s, Raw: kmsKeyValue(t, plan)},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !resp.State.Raw.IsFullyKnown() {
		t.Fatalf("state must be fully known after update: %s", resp.State.Raw)
	}
	var got ResourceKMSKeyModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.Status.ValueString() != "enabled" || got.Timeouts.IsNull() {
		t.Errorf("update must keep the state and save the new timeouts: %+v", got)
	}
}
