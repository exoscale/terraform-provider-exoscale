package kms

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	testKeyStoreID    = "a1b2c3d4-e5f6-4a5b-8c9d-0123456789ab"
	testEndpoint      = "https://xks.example.com/kms/xks/v1"
	testOtherEndpoint = "https://xks-new.example.com/kms/xks/v1"
)

const (
	routeCreateKeyStore     = "POST /key-store"
	routeGetKeyStore        = "GET /key-store/" + testKeyStoreID
	routeUpdateKeyStore     = "POST /key-store/" + testKeyStoreID + "/update"
	routeConnectKeyStore    = "POST /key-store/" + testKeyStoreID + "/connect"
	routeDisconnectKeyStore = "POST /key-store/" + testKeyStoreID + "/disconnect"
	routeDeleteKeyStore     = "DELETE /key-store/" + testKeyStoreID
)

// keyStoreJSON is a key store as returned by the API.
func keyStoreJSON(status, description, endpoint string) string {
	return fmt.Sprintf(`{
		"id": %q, "name": "my-key-store", "type": "external-key-store",
		"description": %q, "status": %q,
		"status-since": "2026-10-01T12:00:00Z", "created-at": "2026-10-01T12:00:00Z",
		"proxy": {"endpoint": %q, "auth": {"key": "access-key"}}
	}`, testKeyStoreID, description, status, endpoint)
}

func kmsKeyStoreSchema(t *testing.T) schema.Schema {
	t.Helper()

	var resp resource.SchemaResponse
	NewResourceKMSKeyStore().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	return resp.Schema
}

// kmsKeyStoreValue builds a resource object where every attribute is null except the given ones.
func kmsKeyStoreValue(t *testing.T, values map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	objType := kmsKeyStoreSchema(t).Type().TerraformType(context.Background()).(tftypes.Object)
	attrs := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		attrs[name] = tftypes.NewValue(typ, nil)
	}
	for name, v := range values {
		attrs[name] = v
	}

	return tftypes.NewValue(objType, attrs)
}

var proxyType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"endpoint":              tftypes.String,
	"access_key":            tftypes.String,
	"secret_key_wo":         tftypes.String,
	"secret_key_wo_version": tftypes.Number,
}}

// proxyValue builds a proxy object. The secret is only set in the config: pass nil for a state or a plan.
func proxyValue(endpoint, accessKey string, secret any, version int64) tftypes.Value {
	return tftypes.NewValue(proxyType, map[string]tftypes.Value{
		"endpoint":              str(endpoint),
		"access_key":            str(accessKey),
		"secret_key_wo":         str(secret),
		"secret_key_wo_version": tftypes.NewValue(tftypes.Number, version),
	})
}

// keyStore describes a key store, to build its Terraform values.
type keyStore struct {
	description string
	endpoint    string
	version     int64
	connected   bool
	status      string
}

func (k keyStore) values() map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"id":          str(testKeyStoreID),
		"name":        str("my-key-store"),
		"zone":        str(testZone),
		"description": str(k.description),
		"connected":   tftypes.NewValue(tftypes.Bool, k.connected),
	}
}

func (k keyStore) state(t *testing.T) tftypes.Value {
	v := k.values()
	v["proxy"] = proxyValue(k.endpoint, "access-key", nil, k.version)
	v["type"] = str("external-key-store")
	v["created_at"] = str("2026-10-01T12:00:00Z")
	v["status"] = str(k.status)
	v["status_since"] = str("2026-10-01T12:00:00Z")
	return kmsKeyStoreValue(t, v)
}

// plan is the plan of an update: the computed values that may change are unknown.
func (k keyStore) plan(t *testing.T) tftypes.Value {
	v := k.values()
	v["proxy"] = proxyValue(k.endpoint, "access-key", nil, k.version)
	v["type"] = str("external-key-store")
	v["created_at"] = str("2026-10-01T12:00:00Z")
	v["status"] = str(tftypes.UnknownValue)
	v["status_since"] = str(tftypes.UnknownValue)
	healthType := kmsKeyStoreValue(t, nil).Type().(tftypes.Object).AttributeTypes["health"]
	v["health"] = tftypes.NewValue(healthType, tftypes.UnknownValue)
	return kmsKeyStoreValue(t, v)
}

func (k keyStore) config(t *testing.T) tftypes.Value {
	v := k.values()
	v["proxy"] = proxyValue(k.endpoint, "access-key", "secret-key", k.version)
	return kmsKeyStoreValue(t, v)
}

func proxyOf(t *testing.T, m ResourceKMSKeyStoreModel) ResourceKMSKeyStoreProxyModel {
	t.Helper()

	var p ResourceKMSKeyStoreProxyModel
	if diags := m.Proxy.As(context.Background(), &p, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatal(diags)
	}
	return p
}

// withoutRetryDelays makes connectKeyStoreWithRetry retry immediately.
func withoutRetryDelays(t *testing.T) {
	delays := connectKeyStoreRetryDelays
	connectKeyStoreRetryDelays = make([]time.Duration, len(delays))
	t.Cleanup(func() { connectKeyStoreRetryDelays = delays })
}

// expectConnectFailures queues a proxy failure for every connect attempt.
func expectConnectFailures(api *mockAPI) {
	for range len(connectKeyStoreRetryDelays) + 1 {
		api.expect(routeConnectKeyStore, 400, mockProblem(kmsErrKeyStoreProxyUnhealthy, "Key store proxy is unhealthy"))
	}
}

func TestResourceKMSKeyStoreSchema(t *testing.T) {
	if diags := kmsKeyStoreSchema(t).ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid schema: %v", diags)
	}
}

func TestResourceKMSKeyStoreModifyPlan(t *testing.T) {
	existing := map[string]tftypes.Value{
		"id":    str(testKeyStoreID),
		"name":  str("my-key-store"),
		"zone":  str(testZone),
		"proxy": proxyValue(testEndpoint, "access-key", nil, 1),
	}
	with := func(proxy tftypes.Value) map[string]tftypes.Value {
		values := make(map[string]tftypes.Value, len(existing))
		for k, v := range existing {
			values[k] = v
		}
		values["proxy"] = proxy
		return values
	}

	tests := []struct {
		name      string
		state     tftypes.Value
		plan      map[string]tftypes.Value
		wantError []path.Path
	}{
		{
			name:  "create",
			state: tftypes.NewValue(kmsKeyStoreValue(t, nil).Type(), nil),
			plan:  existing,
		},
		{
			name:  "no change",
			state: kmsKeyStoreValue(t, existing),
			plan:  existing,
		},
		{
			name:  "new secret version only",
			state: kmsKeyStoreValue(t, existing),
			plan:  with(proxyValue(testEndpoint, "access-key", nil, 2)),
		},
		{
			name:  "new access key with new secret version",
			state: kmsKeyStoreValue(t, existing),
			plan:  with(proxyValue(testEndpoint, "new-access-key", nil, 2)),
		},
		{
			name:      "new access key without new secret version",
			state:     kmsKeyStoreValue(t, existing),
			plan:      with(proxyValue(testEndpoint, "new-access-key", nil, 1)),
			wantError: []path.Path{path.Root("proxy").AtName("secret_key_wo_version")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := kmsKeyStoreSchema(t)
			plan := kmsKeyStoreValue(t, tt.plan)
			req := resource.ModifyPlanRequest{
				State:  tfsdk.State{Schema: s, Raw: tt.state},
				Plan:   tfsdk.Plan{Schema: s, Raw: plan},
				Config: tfsdk.Config{Schema: s, Raw: plan},
			}
			resp := &resource.ModifyPlanResponse{Plan: req.Plan}

			(&ResourceKMSKeyStore{}).ModifyPlan(context.Background(), req, resp)

			assertErrorPaths(t, resp.Diagnostics, tt.wantError)
		})
	}
}

func TestResourceKMSKeyStoreCreate(t *testing.T) {
	withoutRetryDelays(t)

	create := func(t *testing.T, api *mockAPI, connected bool) (*resource.CreateResponse, ResourceKMSKeyStoreModel) {
		t.Helper()

		s := kmsKeyStoreSchema(t)
		k := keyStore{endpoint: testEndpoint, version: 1, connected: connected}

		// On create, every computed value is unknown and the description is not set.
		plan := k.values()
		plan["proxy"] = proxyValue(k.endpoint, "access-key", nil, k.version)
		for _, name := range []string{"id", "description", "type", "status", "status_since", "created_at"} {
			plan[name] = str(tftypes.UnknownValue)
		}
		plan["health"] = tftypes.NewValue(kmsKeyStoreValue(t, nil).Type().(tftypes.Object).AttributeTypes["health"], tftypes.UnknownValue)
		config := k.values()
		config["proxy"] = proxyValue(k.endpoint, "access-key", "secret-key", k.version)
		config["id"] = str(nil)
		config["description"] = str(nil)

		resp := &resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(kmsKeyStoreValue(t, nil).Type(), nil)}}
		(&ResourceKMSKeyStore{client: api.client()}).Create(context.Background(), resource.CreateRequest{
			Plan:   tfsdk.Plan{Schema: s, Raw: kmsKeyStoreValue(t, plan)},
			Config: tfsdk.Config{Schema: s, Raw: kmsKeyStoreValue(t, config)},
		}, resp)

		var got ResourceKMSKeyStoreModel
		resp.Diagnostics.Append(resp.State.Get(context.Background(), &got)...)
		return resp, got
	}

	t.Run("created disconnected", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeCreateKeyStore, 200, keyStoreJSON("disconnected", "", testEndpoint)).
			expect(routeGetKeyStore, 200, keyStoreJSON("disconnected", "", testEndpoint))

		resp, got := create(t, api, false)

		if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 0 {
			t.Fatal(resp.Diagnostics)
		}
		api.assertCalls(routeCreateKeyStore, routeGetKeyStore)
		if got.Connected.ValueBool() || got.Status.ValueString() != "disconnected" {
			t.Errorf("unexpected state %+v", got)
		}
	})

	t.Run("created and connected", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeCreateKeyStore, 200, keyStoreJSON("disconnected", "", testEndpoint)).
			expect(routeConnectKeyStore, 200, mockSuccess).
			expect(routeGetKeyStore, 200, keyStoreJSON("connected", "", testEndpoint))

		resp, got := create(t, api, true)

		if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 0 {
			t.Fatal(resp.Diagnostics)
		}
		api.assertCalls(routeCreateKeyStore, routeConnectKeyStore, routeGetKeyStore)
		body := api.requestBody(routeCreateKeyStore, 0)
		proxy, _ := body["proxy"].(map[string]any)
		auth, _ := proxy["auth"].(map[string]any)
		if body["description"] != nil || auth["key"] != "access-key" || auth["secret"] != "secret-key" {
			t.Errorf("unexpected create body %v", body)
		}
		if got.Status.ValueString() != "connected" || !proxyOf(t, got).SecretKeyWO.IsNull() {
			t.Errorf("unexpected state %+v", got)
		}
	})

	t.Run("connect fails", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeCreateKeyStore, 200, keyStoreJSON("disconnected", "", testEndpoint)).
			expect(routeGetKeyStore, 200, keyStoreJSON("disconnected", "", testEndpoint))
		expectConnectFailures(api)

		resp, got := create(t, api, true)

		if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
			t.Fatalf("expected one warning and no error: %v", resp.Diagnostics)
		}
		if !got.Connected.ValueBool() || got.Status.ValueString() != "disconnected" {
			t.Errorf("connected must keep its planned value and status the actual one: %+v", got)
		}
	})

	t.Run("read back fails", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeCreateKeyStore, 200, keyStoreJSON("disconnected", "", testEndpoint)).
			expect(routeConnectKeyStore, 200, mockSuccess).
			expect(routeGetKeyStore, 400, mockProblem("#/http-problem-types/request-invalid-parameters", "Invalid parameters"))

		resp, got := create(t, api, true)

		if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
			t.Fatalf("expected one warning and no error: %v", resp.Diagnostics)
		}
		if got.ID.ValueString() != testKeyStoreID || got.Status.ValueString() != "connected" || !got.Health.IsNull() {
			t.Errorf("state must come from the creation response: %+v", got)
		}
	})
}

func TestResourceKMSKeyStoreUpdate(t *testing.T) {
	withoutRetryDelays(t)

	update := func(t *testing.T, api *mockAPI, prior, next keyStore) (*resource.UpdateResponse, ResourceKMSKeyStoreModel) {
		t.Helper()

		s := kmsKeyStoreSchema(t)
		resp := &resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: prior.state(t)}}
		(&ResourceKMSKeyStore{client: api.client()}).Update(context.Background(), resource.UpdateRequest{
			State:  tfsdk.State{Schema: s, Raw: prior.state(t)},
			Plan:   tfsdk.Plan{Schema: s, Raw: next.plan(t)},
			Config: tfsdk.Config{Schema: s, Raw: next.config(t)},
		}, resp)

		var got ResourceKMSKeyStoreModel
		resp.Diagnostics.Append(resp.State.Get(context.Background(), &got)...)
		return resp, got
	}

	connected := keyStore{description: "desc", endpoint: testEndpoint, version: 1, connected: true, status: "connected"}

	t.Run("description only", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeUpdateKeyStore, 200, keyStoreJSON("connected", "new", testEndpoint)).
			expect(routeGetKeyStore, 200, keyStoreJSON("connected", "new", testEndpoint))

		next := connected
		next.description = "new"
		resp, got := update(t, api, connected, next)

		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		api.assertCalls(routeUpdateKeyStore, routeGetKeyStore)
		if body := api.requestBody(routeUpdateKeyStore, 0); body["description"] != "new" || body["proxy"] != nil {
			t.Errorf("unexpected update body %v", body)
		}
		if got.Description.ValueString() != "new" {
			t.Errorf("description not saved: %s", got.Description)
		}
	})

	t.Run("endpoint of a connected key store", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeDisconnectKeyStore, 200, mockSuccess).
			expect(routeUpdateKeyStore, 200, keyStoreJSON("disconnected", "desc", testOtherEndpoint)).
			expect(routeConnectKeyStore, 200, mockSuccess).
			expect(routeGetKeyStore, 200, keyStoreJSON("connected", "desc", testOtherEndpoint))

		next := connected
		next.endpoint = testOtherEndpoint
		resp, got := update(t, api, connected, next)

		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		api.assertCalls(routeDisconnectKeyStore, routeUpdateKeyStore, routeConnectKeyStore, routeGetKeyStore)
		proxy, _ := api.requestBody(routeUpdateKeyStore, 0)["proxy"].(map[string]any)
		if proxy["endpoint"] != testOtherEndpoint || proxy["auth"] != nil {
			t.Errorf("unexpected proxy update %v", proxy)
		}
		if !got.Connected.ValueBool() || proxyOf(t, got).Endpoint.ValueString() != testOtherEndpoint {
			t.Errorf("unexpected state %+v", got)
		}
	})

	t.Run("new secret version sends the credentials", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeUpdateKeyStore, 200, keyStoreJSON("connected", "desc", testEndpoint)).
			expect(routeGetKeyStore, 200, keyStoreJSON("connected", "desc", testEndpoint))

		next := connected
		next.version = 2
		resp, got := update(t, api, connected, next)

		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		api.assertCalls(routeUpdateKeyStore, routeGetKeyStore)
		proxy, _ := api.requestBody(routeUpdateKeyStore, 0)["proxy"].(map[string]any)
		auth, _ := proxy["auth"].(map[string]any)
		if auth["key"] != "access-key" || auth["secret"] != "secret-key" || proxy["endpoint"] != nil {
			t.Errorf("unexpected proxy update %v", proxy)
		}
		if p := proxyOf(t, got); !p.SecretKeyWO.IsNull() || p.SecretKeyWOVersion.ValueInt64() != 2 {
			t.Errorf("unexpected proxy state %+v", p)
		}
	})

	t.Run("disconnect with an update", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeDisconnectKeyStore, 200, mockSuccess).
			expect(routeUpdateKeyStore, 200, keyStoreJSON("disconnected", "new", testEndpoint)).
			expect(routeGetKeyStore, 200, keyStoreJSON("disconnected", "new", testEndpoint))

		next := connected
		next.description = "new"
		next.connected = false
		resp, _ := update(t, api, connected, next)

		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		api.assertCalls(routeDisconnectKeyStore, routeUpdateKeyStore, routeGetKeyStore)
	})

	t.Run("connects a key store left disconnected at creation", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeUpdateKeyStore, 200, keyStoreJSON("disconnected", "new", testEndpoint)).
			expect(routeConnectKeyStore, 200, mockSuccess).
			expect(routeGetKeyStore, 200, keyStoreJSON("connected", "new", testEndpoint))

		// State after a create that ended with a warning: connected as planned, status as it is.
		prior := connected
		prior.status = "disconnected"
		next := connected
		next.description = "new"
		resp, _ := update(t, api, prior, next)

		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		api.assertCalls(routeUpdateKeyStore, routeConnectKeyStore, routeGetKeyStore)
	})

	t.Run("failed endpoint update reconnects", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeDisconnectKeyStore, 200, mockSuccess).
			expect(routeUpdateKeyStore, 400, mockProblem(kmsErrKeyStoreProxyUnhealthy, "Key store proxy is unhealthy")).
			expect(routeConnectKeyStore, 200, mockSuccess).
			expect(routeGetKeyStore, 200, keyStoreJSON("connected", "desc", testEndpoint))

		next := connected
		next.endpoint = testOtherEndpoint
		resp, got := update(t, api, connected, next)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error")
		}
		api.assertCalls(routeDisconnectKeyStore, routeUpdateKeyStore, routeConnectKeyStore, routeGetKeyStore)
		if proxyOf(t, got).Endpoint.ValueString() != testEndpoint || !got.Connected.ValueBool() {
			t.Errorf("state must keep the previous settings: %+v", got)
		}
	})

	t.Run("failed connect keeps the update", func(t *testing.T) {
		api := newMockAPI(t).
			expect(routeDisconnectKeyStore, 200, mockSuccess).
			expect(routeUpdateKeyStore, 200, keyStoreJSON("disconnected", "desc", testOtherEndpoint)).
			expect(routeGetKeyStore, 200, keyStoreJSON("disconnected", "desc", testOtherEndpoint))
		expectConnectFailures(api)

		next := connected
		next.endpoint = testOtherEndpoint
		next.version = 2
		resp, got := update(t, api, connected, next)

		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error")
		}
		p := proxyOf(t, got)
		if p.Endpoint.ValueString() != testOtherEndpoint || p.SecretKeyWOVersion.ValueInt64() != 2 || got.Connected.ValueBool() {
			t.Errorf("state must hold the update and the actual status: %+v %+v", got, p)
		}
	})
}

func TestResourceKMSKeyStoreDelete(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantError string
	}{
		{name: "deleted", status: 200, body: mockSuccess},
		{name: "already gone", status: 400, body: mockProblem(kmsErrKeyStoreNotFound, "Key store not found")},
		{
			name:      "referenced",
			status:    400,
			body:      mockProblem(kmsErrKeyStoreIsReferenced, "Key store is referenced"),
			wantError: "still referenced by KMS keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newMockAPI(t).expect(routeDeleteKeyStore, tt.status, tt.body)
			s := kmsKeyStoreSchema(t)
			state := keyStore{endpoint: testEndpoint, version: 1, connected: true, status: "connected"}.state(t)

			resp := &resource.DeleteResponse{State: tfsdk.State{Schema: s, Raw: state}}
			(&ResourceKMSKeyStore{client: api.client()}).Delete(context.Background(), resource.DeleteRequest{
				State: tfsdk.State{Schema: s, Raw: state},
			}, resp)

			errs := resp.Diagnostics.Errors()
			switch {
			case tt.wantError == "" && len(errs) > 0:
				t.Errorf("unexpected error: %v", errs)
			case tt.wantError != "" && (len(errs) != 1 || !strings.Contains(errs[0].Summary(), tt.wantError)):
				t.Errorf("got %v, want an error about %q", errs, tt.wantError)
			}
		})
	}
}

func TestResourceKMSKeyStoreImportState(t *testing.T) {
	assertImportState(t, &ResourceKMSKeyStore{}, testKeyStoreID)
}

// assertImportState checks that a resource is imported by id@zone, and rejects an ID without zone.
func assertImportState(t *testing.T, r resource.ResourceWithImportState, id string) {
	t.Helper()
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	empty := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}

	resp := &resource.ImportStateResponse{State: empty}
	r.ImportState(ctx, resource.ImportStateRequest{ID: id + "@" + testZone}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var gotID, gotZone types.String
	resp.State.GetAttribute(ctx, path.Root("id"), &gotID)
	resp.State.GetAttribute(ctx, path.Root("zone"), &gotZone)
	if gotID.ValueString() != id || gotZone.ValueString() != testZone {
		t.Errorf("unexpected import state: id=%s zone=%s", gotID, gotZone)
	}

	bad := &resource.ImportStateResponse{State: empty}
	r.ImportState(ctx, resource.ImportStateRequest{ID: id}, bad)
	if !bad.Diagnostics.HasError() {
		t.Error("an import ID without zone must be rejected")
	}
}

func TestResourceKMSKeyStoreSetFromAPI(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	proxy, diags := types.ObjectValueFrom(ctx, ResourceKMSKeyStoreProxyModel{}.Types(), ResourceKMSKeyStoreProxyModel{
		Endpoint:           types.StringValue("https://old.example.com/kms/xks/v1"),
		AccessKey:          types.StringValue("old-access-key"),
		SecretKeyWO:        types.StringValue("secret"),
		SecretKeyWOVersion: types.Int64Value(3),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	m := ResourceKMSKeyStoreModel{Proxy: proxy}

	diags = m.setFromAPI(ctx, &exoscale.GetKeyStoreResponse{
		Name:        "my-key-store",
		Type:        exoscale.GetKeyStoreResponseTypeExternalKeyStore,
		Status:      exoscale.GetKeyStoreResponseStatusDisconnected,
		StatusSince: createdAt,
		CreatedAT:   createdAt,
		Proxy: &exoscale.KeyStoreProxyResponse{
			Endpoint: testEndpoint,
			Auth:     &exoscale.KeyStoreProxyAuthResponse{Key: "access-key"},
		},
		Health: &exoscale.KeyStoreHealth{Status: exoscale.KeyStoreHealthStatusUnknown},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}

	got := proxyOf(t, m)
	if got.Endpoint.ValueString() != testEndpoint || got.AccessKey.ValueString() != "access-key" {
		t.Errorf("proxy not read from the API: %+v", got)
	}
	if !got.SecretKeyWO.IsNull() {
		t.Errorf("secret_key_wo must never be stored, got %s", got.SecretKeyWO)
	}
	if got.SecretKeyWOVersion.ValueInt64() != 3 {
		t.Errorf("secret_key_wo_version must be kept, got %s", got.SecretKeyWOVersion)
	}
	if m.Connected.ValueBool() {
		t.Error("connected must be false for a disconnected key store")
	}
	if m.CreatedAt.ValueString() != "2026-10-01T12:00:00Z" {
		t.Errorf("unexpected created_at %s", m.CreatedAt)
	}

	var health KMSKeyStoreHealthModel
	if diags := m.Health.As(ctx, &health, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatal(diags)
	}
	if !health.CheckedAt.IsNull() {
		t.Errorf("checked_at must be null before the first health check, got %s", health.CheckedAt)
	}
}

func TestConnectKeyStoreWithRetry(t *testing.T) {
	withoutRetryDelays(t)

	unhealthy := mockResponse{400, mockProblem(kmsErrKeyStoreProxyUnhealthy, "Key store proxy is unhealthy")}
	notFound := mockResponse{400, mockProblem(kmsErrKeyStoreNotFound, "Key store not found")}
	connected := mockResponse{200, mockSuccess}
	proxyError := func(name string) mockResponse {
		return mockResponse{400, fmt.Sprintf(
			`{"type":%q,"title":"Key store proxy request failed","status":400,"detail":"proxy health","xks-proxy-error":{"error-name":%q}}`,
			kmsErrKeyStoreProxyRequestFailed, name,
		)}
	}
	unreachable := mockResponse{400, mockProblem(kmsErrKeyStoreProxyRequestFailed, "Key store proxy request failed")}

	tests := []struct {
		name      string
		responses []mockResponse
		wantErr   string
	}{
		{name: "connected at once", responses: []mockResponse{connected}},
		{name: "connected after retries", responses: []mockResponse{unhealthy, unhealthy, connected}},
		{name: "not retried", responses: []mockResponse{notFound}, wantErr: kmsErrKeyStoreNotFound},
		{name: "unreachable proxy retried", responses: []mockResponse{unreachable, connected}},
		{name: "throttled proxy retried", responses: []mockResponse{proxyError("ThrottlingException"), connected}},
		{
			name:      "wrong credentials not retried",
			responses: []mockResponse{proxyError("AuthenticationFailedException")},
			wantErr:   kmsErrKeyStoreProxyRequestFailed,
		},
		{
			name:      "retries exhausted",
			responses: []mockResponse{unhealthy, unhealthy, unhealthy, unhealthy, unhealthy},
			wantErr:   kmsErrKeyStoreProxyUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newMockAPI(t)
			wantCalls := make([]string, len(tt.responses))
			for i, r := range tt.responses {
				api.expect(routeConnectKeyStore, r.status, r.body)
				wantCalls[i] = routeConnectKeyStore
			}

			err := connectKeyStoreWithRetry(context.Background(), api.client(), exoscale.UUID(testKeyStoreID))

			api.assertCalls(wantCalls...)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tt.wantErr != "" && !isAPIError(err, tt.wantErr):
				t.Errorf("got error %v, want problem type %s", err, tt.wantErr)
			}
		})
	}
}
