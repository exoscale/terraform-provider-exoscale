package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	exov2 "github.com/exoscale/egoscale/v2"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const testV2APIEndpoint = "https://gateway.internal:8443/v2"

type recordingTransport struct {
	request *http.Request
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.request = req

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader("{}")),
	}, nil
}

func recordV2Request(t *testing.T, client *exov2.Client) *http.Request {
	t.Helper()

	transport := new(recordingTransport)
	client.SetHTTPClient(&http.Client{Transport: transport})

	if _, err := client.ListSSHKeys(context.Background(), "ch-gva-2"); err != nil {
		t.Fatal(err)
	}
	if transport.request == nil {
		t.Fatal("v2 client sent no request")
	}

	return transport.request
}

func TestExoscaleProviderConfigureHonoursV2APIEndpoint(t *testing.T) {
	t.Setenv("EXOSCALE_API_ENDPOINT", testV2APIEndpoint)

	ctx := context.Background()
	p := new(ExoscaleProvider)
	schemaResp := new(frameworkprovider.SchemaResponse)
	p.Schema(ctx, frameworkprovider.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("provider schema diagnostics: %s", schemaResp.Diagnostics)
	}

	config := tftypes.NewValue(
		schemaResp.Schema.Type().TerraformType(ctx),
		map[string]tftypes.Value{
			KeyAttrName:         tftypes.NewValue(tftypes.String, "EXOxxxxxxxxxxxxxxxxxxxxxxxx"),
			SecretAttrName:      tftypes.NewValue(tftypes.String, "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"),
			EnvironmentAttrName: tftypes.NewValue(tftypes.String, nil),
			SOSEndpointAttrName: tftypes.NewValue(tftypes.String, nil),
			TimeoutAttrName:     tftypes.NewValue(tftypes.Number, 1),
		},
	)

	configureResp := new(frameworkprovider.ConfigureResponse)
	p.Configure(ctx, frameworkprovider.ConfigureRequest{
		Config: tfsdk.Config{Raw: config, Schema: schemaResp.Schema},
	}, configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("provider configure diagnostics: %s", configureResp.Diagnostics)
	}

	providerData, ok := configureResp.ResourceData.(*providerConfig.ExoscaleProviderConfig)
	if !ok {
		t.Fatalf("provider data type = %T, want *config.ExoscaleProviderConfig", configureResp.ResourceData)
	}

	req := recordV2Request(t, providerData.ClientV2)
	if got, want := req.URL.String(), testV2APIEndpoint+"/ssh-key"; got != want {
		t.Errorf("request URL = %q, want %q", got, want)
	}
	if got, want := req.Host, "gateway.internal:8443"; got != want {
		t.Errorf("request host = %q, want %q", got, want)
	}
}
