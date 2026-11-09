package block_storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestResourceSnapshotReadNotFound(t *testing.T) {
	const (
		zone       = "ch-gva-2"
		snapshotID = "6f1c8b2e-3a4d-4c5e-8f9a-0b1c2d3e4f50"
	)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zone":
			_, _ = fmt.Fprintf(w, `{"zones":[{"name":%q,"api-endpoint":%q}]}`, zone, server.URL)
		case r.Method == http.MethodGet && r.URL.Path == "/block-storage-snapshot/"+snapshotID:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Block storage snapshot not found"}`))
		default:
			t.Errorf("unexpected call %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client, err := exoscale.NewClient(
		credentials.NewStaticCredentials("EXOtest", "secret"),
		exoscale.ClientOptWithEndpoint(exoscale.Endpoint(server.URL)),
	)
	if err != nil {
		t.Fatal(err)
	}

	r := &ResourceSnapshot{client: client}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatal(schemaResp.Diagnostics)
	}
	s := schemaResp.Schema

	objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
	attrs := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		attrs[name] = tftypes.NewValue(typ, nil)
	}
	attrs["id"] = tftypes.NewValue(tftypes.String, snapshotID)
	attrs["zone"] = tftypes.NewValue(tftypes.String, zone)
	state := tftypes.NewValue(objType, attrs)

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: s, Raw: state}}
	r.Read(context.Background(), resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: state}}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no error for a snapshot deleted outside of Terraform, got: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("snapshot must be removed from the state")
	}
}
