package utils

import (
	"context"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
)

// DetachBlockStorageVolume detaches a volume and waits for the operation.
// A volume not attached is not an error; exoscale.ErrNotFound is returned as is.
func DetachBlockStorageVolume(ctx context.Context, client *exoscale.Client, id exoscale.UUID) error {
	op, err := client.DetachBlockStorageVolume(ctx, id)
	if err == nil {
		_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	}
	// The API has no error code for it, only this message.
	if err != nil && strings.HasSuffix(err.Error(), "Volume not attached") {
		return nil
	}

	return err
}
