package exoscale

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	exov2 "github.com/exoscale/egoscale/v2"

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

func TestV2ClientConstructorsHonourAPIEndpoint(t *testing.T) {
	t.Setenv("EXOSCALE_API_ENDPOINT", testV2APIEndpoint)

	config := providerConfig.BaseConfig{
		Key:     "EXOxxxxxxxxxxxxxxxxxxxxxxxx",
		Secret:  "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		Timeout: time.Second,
	}

	for _, tt := range []struct {
		name      string
		newClient func() (*exov2.Client, error)
	}{
		{
			name: "CreateClient",
			newClient: func() (*exov2.Client, error) {
				return CreateClient(&config)
			},
		},
		{
			name: "getClient",
			newClient: func() (*exov2.Client, error) {
				return getClient(map[string]any{"config": config}), nil
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.newClient()
			if err != nil {
				t.Fatal(err)
			}

			req := recordV2Request(t, client)
			if got, want := req.URL.String(), testV2APIEndpoint+"/ssh-key"; got != want {
				t.Errorf("request URL = %q, want %q", got, want)
			}
			if got, want := req.Host, "gateway.internal:8443"; got != want {
				t.Errorf("request host = %q, want %q", got, want)
			}
		})
	}
}
