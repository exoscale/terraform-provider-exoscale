package utils

import (
	"context"
	"errors"
	"testing"

	exoscale "github.com/exoscale/egoscale/v3"
)

func TestWaitReturnsAPIError(t *testing.T) {
	apiErr := errors.New("API error")

	// The client is never reached when the API call failed.
	if err := Wait(context.Background(), nil)(nil, apiErr); !errors.Is(err, apiErr) {
		t.Errorf("Wait() = %v, want %v", err, apiErr)
	}

	op, err := WaitOperation(context.Background(), nil)(&exoscale.Operation{}, apiErr)
	if !errors.Is(err, apiErr) || op != nil {
		t.Errorf("WaitOperation() = %v, %v, want nil, %v", op, err, apiErr)
	}
}
