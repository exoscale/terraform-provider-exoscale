package kms

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func TestAPIErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "not an API error",
			err:  errors.New("boom"),
			want: "boom",
		},
		{
			name: "title only",
			err:  fmt.Errorf("DeleteKeyStore: %w", &exoscale.APIError{Title: "Key store is referenced", Detail: kmsNoDetail}),
			want: "Key store is referenced",
		},
		{
			name: "title, detail and proxy error",
			err: fmt.Errorf("CreateKeyStore: %w", &exoscale.APIError{
				Title:  "Key store proxy request failed",
				Detail: "proxy health",
				Response: &exoscale.ErrorResponse{XksProxyError: &exoscale.XksProxyErrorDetail{
					ErrorName:    "AuthenticationFailedException",
					ErrorMessage: "bad signature",
				}},
			}),
			want: "Key store proxy request failed: proxy health (XKS proxy error: AuthenticationFailedException: bad signature)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiErrorMessage(tt.err); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsAPIError(t *testing.T) {
	notFound := fmt.Errorf("GetKeyStore: %w", &exoscale.APIError{
		Response: &exoscale.ErrorResponse{Type: kmsErrKeyStoreNotFound},
	})

	if !isAPIError(notFound, kmsErrKeyStoreNotFound) {
		t.Error("the problem type must be matched")
	}
	if isAPIError(notFound, kmsErrKeyStoreIsReferenced) {
		t.Error("another problem type must not be matched")
	}
	if isAPIError(errors.New("boom"), kmsErrKeyStoreNotFound) {
		t.Error("a non-API error must not be matched")
	}
	if !isNotFound(fmt.Errorf("GetKeyStore: %w", exoscale.ErrNotFound), kmsErrKeyStoreNotFound) {
		t.Error("an HTTP 404 must count as not found")
	}
}

// TestAPIErrorTypes checks against the real API that KMS still answers with the problem types the provider
// relies on, as they are copied from the KMS service. It runs with the acceptance tests (TF_ACC).
func TestAPIErrorTypes(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	// Same as testutils.TestZoneName: testutils cannot be imported here (it imports this package).
	zone := "bg-sof-1"

	ctx := context.Background()
	client, err := exoscale.NewClient(credentials.NewStaticCredentials(
		os.Getenv("EXOSCALE_API_KEY"),
		os.Getenv("EXOSCALE_API_SECRET"),
	))
	if err != nil {
		t.Fatal(err)
	}
	client, err = utils.SwitchClientZone(ctx, client, exoscale.ZoneName(zone))
	if err != nil {
		t.Fatal(err)
	}

	unknownID := exoscale.UUID("00000000-0000-4000-8000-000000000000")
	name := fmt.Sprintf("terraform-provider-test-%d", time.Now().UnixNano())

	assertType := func(t *testing.T, err error, problemTypes ...string) {
		t.Helper()
		for _, problemType := range problemTypes {
			if isAPIError(err, problemType) {
				return
			}
		}
		t.Errorf("got error %v, want problem type %v", err, problemTypes)
	}

	t.Run("key store not found", func(t *testing.T) {
		_, err := client.GetKeyStore(ctx, unknownID)
		assertType(t, err, kmsErrKeyStoreNotFound)
	})

	t.Run("key not found", func(t *testing.T) {
		_, err := client.GetKmsKey(ctx, unknownID)
		assertType(t, err, kmsErrKeyNotFound)
	})

	t.Run("proxy unreachable", func(t *testing.T) {
		keyStore, err := client.CreateKeyStore(ctx, exoscale.CreateKeyStoreRequest{
			Name: name,
			Proxy: &exoscale.KeyStoreProxy{
				Endpoint: "https://xks-proxy.invalid/kms/xks/v1",
				Auth: &exoscale.KeyStoreProxyAuth{
					Key:    "TERRAFORMPROVIDERTESTKEY",
					Secret: "terraform-provider-test-secret-of-at-least-43-chars",
				},
			},
		})
		if err == nil {
			_, _ = client.DeleteKeyStore(ctx, keyStore.ID)
			t.Fatal("a key store with an unreachable proxy must not be created")
		}
		assertType(t, err, kmsErrKeyStoreProxyRequestFailed, kmsErrKeyStoreProxyUnhealthy)
	})

	t.Run("key already pending deletion", func(t *testing.T) {
		key, err := client.CreateKmsKey(ctx, exoscale.CreateKmsKeyRequest{
			Name:        name,
			Description: "terraform provider error types test",
		})
		if err != nil {
			t.Fatal(apiErrorMessage(err))
		}

		req := exoscale.ScheduleKmsKeyDeletionRequest{DelayDays: kmsKeyDeletionDelayDays}
		if _, err := client.ScheduleKmsKeyDeletion(ctx, key.ID, req); err != nil {
			t.Fatal(apiErrorMessage(err))
		}
		_, err = client.ScheduleKmsKeyDeletion(ctx, key.ID, req)
		assertType(t, err, kmsErrKeyPendingDeletion)
	})
}
