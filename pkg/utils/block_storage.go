package utils

import (
	"context"
	"errors"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
)

// DetachBlockStorageVolume detaches a volume and waits for the operation.
// A volume already gone or not attached is not an error.
func DetachBlockStorageVolume(ctx context.Context, client *exoscale.Client, id exoscale.UUID) error {
	op, err := client.DetachBlockStorageVolume(ctx, id)
	if err == nil {
		_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	}
	if errors.Is(err, exoscale.ErrNotFound) {
		return nil
	}
	// The API has no error code for "not attached", only this message.
	if err != nil && strings.HasSuffix(err.Error(), "Volume not attached") {
		return nil
	}

	return err
}
