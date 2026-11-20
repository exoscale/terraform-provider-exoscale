package block_storage

import (
	"context"
	"errors"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

// DetachVolume detaches a volume and waits for the operation.
// A volume already gone or not attached is not an error.
func DetachVolume(ctx context.Context, client *exoscale.Client, id exoscale.UUID) error {
	err := utils.Wait(ctx, client)(client.DetachBlockStorageVolume(ctx, id))
	if errors.Is(err, exoscale.ErrNotFound) {
		return nil
	}
	// The API has no error code for "not attached", only this message.
	if err != nil && strings.HasSuffix(err.Error(), "Volume not attached") {
		return nil
	}

	return err
}
