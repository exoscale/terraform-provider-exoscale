package instance

import (
	"context"
	"errors"
	"fmt"
	"net"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stringSetChange returns the values added to and removed from a set of
// strings between the state and the plan.
func stringSetChange(ctx context.Context, state, plan types.Set) (added, removed []string, diags diag.Diagnostics) {
	old, dg := stringSetValues(ctx, state)
	diags.Append(dg...)
	cur, dg := stringSetValues(ctx, plan)
	diags.Append(dg...)

	added, removed = stringSetDiff(old, cur)

	return added, removed, diags
}

// attachment attaches resources of one kind to an instance, and detaches them,
// by their ID.
type attachment struct {
	name   string
	attach func(ctx context.Context, client *exoscale.Client, instanceID exoscale.UUID, id string) error
	detach func(ctx context.Context, client *exoscale.Client, instanceID exoscale.UUID, id string) error
}

var (
	blockStorageVolumes = attachment{"block storage volume", attachBlockStorageVolume, detachBlockStorageVolume}
	elasticIPs          = attachment{"Elastic IP", attachElasticIP, detachElasticIP}
	securityGroups      = attachment{"Security Group", attachSecurityGroup, detachSecurityGroup}
)

// update attaches the IDs added to the set from state to plan, then detaches
// the removed ones; a resource already deleted counts as detached.
func (a attachment) update(ctx context.Context, client *exoscale.Client, instanceID exoscale.UUID, state, plan types.Set) diag.Diagnostics {
	if plan.Equal(state) {
		return nil
	}

	added, removed, diags := stringSetChange(ctx, state, plan)
	if diags.HasError() {
		return diags
	}

	for _, id := range added {
		if err := a.attach(ctx, client, instanceID, id); err != nil {
			diags.AddError("unable to attach "+a.name+" "+id, err.Error())
			return diags
		}
	}

	for _, id := range removed {
		if err := a.detach(ctx, client, instanceID, id); err != nil && !errors.Is(err, exoscale.ErrNotFound) {
			diags.AddError("unable to detach "+a.name+" "+id, err.Error())
			return diags
		}
	}

	return diags
}

func attachElasticIP(ctx context.Context, client *exoscale.Client, id exoscale.UUID, elasticIPID string) error {
	return utils.Wait(ctx, client)(client.AttachInstanceToElasticIP(
		ctx,
		exoscale.UUID(elasticIPID),
		exoscale.AttachInstanceToElasticIPRequest{Instance: &exoscale.InstanceTarget{ID: id}},
	))
}

func detachElasticIP(ctx context.Context, client *exoscale.Client, id exoscale.UUID, elasticIPID string) error {
	return utils.Wait(ctx, client)(client.DetachInstanceFromElasticIP(
		ctx,
		exoscale.UUID(elasticIPID),
		exoscale.DetachInstanceFromElasticIPRequest{Instance: &exoscale.InstanceTarget{ID: id}},
	))
}

func attachSecurityGroup(ctx context.Context, client *exoscale.Client, id exoscale.UUID, securityGroupID string) error {
	return utils.Wait(ctx, client)(client.AttachInstanceToSecurityGroup(
		ctx,
		exoscale.UUID(securityGroupID),
		exoscale.AttachInstanceToSecurityGroupRequest{Instance: &exoscale.Instance{ID: id}},
	))
}

func detachSecurityGroup(ctx context.Context, client *exoscale.Client, id exoscale.UUID, securityGroupID string) error {
	return utils.Wait(ctx, client)(client.DetachInstanceFromSecurityGroup(
		ctx,
		exoscale.UUID(securityGroupID),
		exoscale.DetachInstanceFromSecurityGroupRequest{Instance: &exoscale.Instance{ID: id}},
	))
}

func attachPrivateNetwork(ctx context.Context, client *exoscale.Client, id exoscale.UUID, nif NetworkInterfaceModel) error {
	return utils.Wait(ctx, client)(client.AttachInstanceToPrivateNetwork(
		ctx,
		exoscale.UUID(nif.NetworkID.ValueString()),
		exoscale.AttachInstanceToPrivateNetworkRequest{
			Instance: &exoscale.AttachInstanceToPrivateNetworkRequestInstance{ID: id},
			// The IP address is unknown or null unless the configuration sets one.
			IP: net.ParseIP(nif.IPAddress.ValueString()),
		},
	))
}

func detachPrivateNetwork(ctx context.Context, client *exoscale.Client, id exoscale.UUID, networkID string) error {
	return utils.Wait(ctx, client)(client.DetachInstanceFromPrivateNetwork(
		ctx,
		exoscale.UUID(networkID),
		exoscale.DetachInstanceFromPrivateNetworkRequest{Instance: &exoscale.Instance{ID: id}},
	))
}

func attachBlockStorageVolume(ctx context.Context, client *exoscale.Client, id exoscale.UUID, volumeID string) error {
	volume, err := exoscale.ParseUUID(volumeID)
	if err != nil {
		return fmt.Errorf("unable to parse block storage volume ID: %w", err)
	}

	return utils.Wait(ctx, client)(client.AttachBlockStorageVolumeToInstance(
		ctx,
		volume,
		exoscale.AttachBlockStorageVolumeToInstanceRequest{Instance: &exoscale.InstanceTarget{ID: id}},
	))
}

func detachBlockStorageVolume(ctx context.Context, client *exoscale.Client, _ exoscale.UUID, volumeID string) error {
	volume, err := exoscale.ParseUUID(volumeID)
	if err != nil {
		return fmt.Errorf("unable to parse block storage volume ID: %w", err)
	}

	return utils.DetachBlockStorageVolume(ctx, client, volume)
}
