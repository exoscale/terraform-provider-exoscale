package kms

import (
	"errors"
	"fmt"
	"testing"

	exoscale "github.com/exoscale/egoscale/v3"
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
