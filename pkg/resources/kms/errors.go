package kms

import (
	"errors"
	"fmt"

	exoscale "github.com/exoscale/egoscale/v3"
)

const kmsNoDetail = "no details."

const (
	kmsErrKeyNotFound          = "#/kms-key/not-found"
	kmsErrKeyStoreNotFound     = "#/key-store/not-found"
	kmsErrKeyStoreIsReferenced = "#/key-store/is-referenced-by-key"

	kmsErrKeyStoreProxyUnhealthy     = "#/key-store/proxy-unhealthy"
	kmsErrKeyStoreProxyRequestFailed = "#/key-store/proxy-request-failed"
)

// isAPIError reports whether err is a KMS API error of the given problem type.
func isAPIError(err error, problemType string) bool {
	var apiErr *exoscale.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	resp, ok := apiErr.Response.(*exoscale.ErrorResponse)
	return ok && resp.Type == problemType
}

func isNotFound(err error, problemType string) bool {
	return isAPIError(err, problemType) || errors.Is(err, exoscale.ErrNotFound)
}

// apiErrorMessage formats a KMS API error from its problem details (title, detail and XKS proxy error).
// egoscale parses them into APIError, but APIError.Error() only prints the status and the "message" field,
// which KMS error responses do not have.
func apiErrorMessage(err error) string {
	var apiErr *exoscale.APIError
	if !errors.As(err, &apiErr) || apiErr.Title == "" {
		return err.Error()
	}

	msg := apiErr.Title
	if apiErr.Detail != "" && apiErr.Detail != kmsNoDetail {
		msg += ": " + apiErr.Detail
	}

	if resp, ok := apiErr.Response.(*exoscale.ErrorResponse); ok && resp.XksProxyError != nil {
		msg += fmt.Sprintf(" (XKS proxy error: %s", resp.XksProxyError.ErrorName)
		if resp.XksProxyError.ErrorMessage != "" {
			msg += ": " + resp.XksProxyError.ErrorMessage
		}
		msg += ")"
	}

	return msg
}
