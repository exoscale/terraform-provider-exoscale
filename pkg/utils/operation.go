package utils

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v3"
)

// Wait returns a function waiting for the success of an operation, to be
// wrapped around the API call starting it:
//
//	err := utils.Wait(ctx, client)(client.DeleteSSHKey(ctx, name))
func Wait(ctx context.Context, client *exoscale.Client) func(*exoscale.Operation, error) error {
	return func(op *exoscale.Operation, err error) error {
		_, err = WaitOperation(ctx, client)(op, err)
		return err
	}
}

// WaitOperation is Wait for callers needing the finished operation, e.g. for
// the ID of the resource it created (op.Reference.ID).
func WaitOperation(ctx context.Context, client *exoscale.Client) func(*exoscale.Operation, error) (*exoscale.Operation, error) {
	return func(op *exoscale.Operation, err error) (*exoscale.Operation, error) {
		if err != nil {
			return nil, err
		}

		return client.Wait(ctx, op, exoscale.OperationStateSuccess)
	}
}
