package kms

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const routeListKeyStores = "GET /key-store"

func kmsKeyStoreDataSourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	var resp datasource.SchemaResponse
	NewDataSourceKMSKeyStore().Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	return resp.Schema
}

// readKMSKeyStoreDataSource reads the data source with a config where every attribute is null except the given ones.
func readKMSKeyStoreDataSource(t *testing.T, api *mockAPI, values map[string]tftypes.Value) (*datasource.ReadResponse, DataSourceKMSKeyStoreModel) {
	t.Helper()

	s := kmsKeyStoreDataSourceSchema(t)
	objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
	attrs := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		attrs[name] = tftypes.NewValue(typ, nil)
	}
	for name, v := range values {
		attrs[name] = v
	}

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}}
	(&DataSourceKMSKeyStore{client: api.client()}).Read(context.Background(), datasource.ReadRequest{
		Config: tfsdk.Config{Schema: s, Raw: tftypes.NewValue(objType, attrs)},
	}, resp)

	var got DataSourceKMSKeyStoreModel
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Get(context.Background(), &got)...)
	}
	return resp, got
}

func TestDataSourceKMSKeyStoreSchema(t *testing.T) {
	if diags := kmsKeyStoreDataSourceSchema(t).ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid schema: %v", diags)
	}
}

func TestDataSourceKMSKeyStoreRead(t *testing.T) {
	keyStores := fmt.Sprintf(`{"key-stores": [%s]}`, keyStoreJSON("connected", "desc", testEndpoint))

	tests := []struct {
		name      string
		config    map[string]tftypes.Value
		expect    func(api *mockAPI)
		wantCalls []string
		wantError string
	}{
		{
			name:   "by id",
			config: map[string]tftypes.Value{"zone": str(testZone), "id": str(testKeyStoreID)},
			expect: func(api *mockAPI) {
				api.expect(routeGetKeyStore, 200, keyStoreJSON("connected", "desc", testEndpoint))
			},
			wantCalls: []string{routeGetKeyStore},
		},
		{
			name:   "by name",
			config: map[string]tftypes.Value{"zone": str(testZone), "name": str("my-key-store")},
			expect: func(api *mockAPI) {
				api.expect(routeListKeyStores, 200, keyStores).
					expect(routeGetKeyStore, 200, keyStoreJSON("connected", "desc", testEndpoint))
			},
			wantCalls: []string{routeListKeyStores, routeGetKeyStore},
		},
		{
			name:   "unknown name",
			config: map[string]tftypes.Value{"zone": str(testZone), "name": str("other-key-store")},
			expect: func(api *mockAPI) {
				api.expect(routeListKeyStores, 200, keyStores)
			},
			wantCalls: []string{routeListKeyStores},
			wantError: `KMS key store "other-key-store" not found`,
		},
		{
			name:   "name matching another store's id",
			config: map[string]tftypes.Value{"zone": str(testZone), "name": str(testKeyStoreID)},
			expect: func(api *mockAPI) {
				api.expect(routeListKeyStores, 200, keyStores)
			},
			wantCalls: []string{routeListKeyStores},
			wantError: fmt.Sprintf(`KMS key store %q not found`, testKeyStoreID),
		},
		{
			name:   "unknown id",
			config: map[string]tftypes.Value{"zone": str(testZone), "id": str(testKeyStoreID)},
			expect: func(api *mockAPI) {
				api.expect(routeGetKeyStore, 400, mockProblem(kmsErrKeyStoreNotFound, "Key store not found"))
			},
			wantCalls: []string{routeGetKeyStore},
			wantError: "KMS key store " + testKeyStoreID + " not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newMockAPI(t)
			tt.expect(api)

			resp, got := readKMSKeyStoreDataSource(t, api, tt.config)

			api.assertCalls(tt.wantCalls...)
			errs := resp.Diagnostics.Errors()
			if tt.wantError != "" {
				if len(errs) != 1 || !strings.Contains(errs[0].Summary(), tt.wantError) {
					t.Errorf("got %v, want an error about %q", errs, tt.wantError)
				}
				return
			}
			if len(errs) > 0 {
				t.Fatal(errs)
			}

			var proxy DataSourceKMSKeyStoreProxyModel
			if diags := got.Proxy.As(context.Background(), &proxy, basetypes.ObjectAsOptions{}); diags.HasError() {
				t.Fatal(diags)
			}
			if got.ID.ValueString() != testKeyStoreID || got.Name.ValueString() != "my-key-store" ||
				got.Status.ValueString() != "connected" || got.Description.ValueString() != "desc" {
				t.Errorf("unexpected key store %+v", got)
			}
			if proxy.Endpoint.ValueString() != testEndpoint || proxy.AccessKey.ValueString() != "access-key" {
				t.Errorf("unexpected proxy %+v", proxy)
			}
		})
	}
}
