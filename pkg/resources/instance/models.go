package instance

import (
	"context"
	"errors"
	"fmt"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NetworkInterfaceModel maps the `network_interface` block of
// exoscale_compute_instance.
type NetworkInterfaceModel struct {
	IPAddress  types.String `tfsdk:"ip_address"`
	MACAddress types.String `tfsdk:"mac_address"`
	NetworkID  types.String `tfsdk:"network_id"`
}

// Types returns nested data model types to be used for conversion.
func (m NetworkInterfaceModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"ip_address":  types.StringType,
		"mac_address": types.StringType,
		"network_id":  types.StringType,
	}
}

var networkInterfaceType = types.ObjectType{AttrTypes: NetworkInterfaceModel{}.Types()}

// networkInterface is a Private Network interface of an instance, as known by
// the API.
type networkInterface struct {
	NetworkID  string
	MACAddress string
	// IPAddress is the static DHCP lease of the instance in a managed
	// Private Network, "" if it has none: the state has always held an empty
	// string there rather than null.
	IPAddress string
}

// remoteNetworkInterfaces returns the Private Network interfaces of the
// instance. The static leases are held by the Private Networks, they cost an
// API call per interface: they are only fetched when withLeases is true.
func remoteNetworkInterfaces(
	ctx context.Context,
	client *exoscale.Client,
	instance *exoscale.Instance,
	withLeases bool,
) ([]networkInterface, error) {
	nifs := make([]networkInterface, 0, len(instance.PrivateNetworks))

	for _, attached := range instance.PrivateNetworks {
		nif := networkInterface{
			NetworkID:  attached.ID.String(),
			MACAddress: attached.MACAddress,
		}

		if withLeases {
			privateNetwork, err := client.GetPrivateNetwork(ctx, attached.ID)
			if err != nil {
				return nil, err
			}

			for _, lease := range privateNetwork.Leases {
				if lease.InstanceID == instance.ID {
					nif.IPAddress = lease.IP.String()
					break
				}
			}
		}

		nifs = append(nifs, nif)
	}

	return nifs, nil
}

// networkInterfaces decodes the `network_interface` blocks.
func networkInterfaces(ctx context.Context, set types.Set) ([]NetworkInterfaceModel, diag.Diagnostics) {
	var nifs []NetworkInterfaceModel

	if set.IsNull() || set.IsUnknown() {
		return nifs, nil
	}

	diags := set.ElementsAs(ctx, &nifs, false)

	return nifs, diags
}

// networkInterfaceSet encodes the `network_interface` blocks. A block is never
// null: no interface is an empty set.
func networkInterfaceSet(ctx context.Context, nifs []NetworkInterfaceModel) (types.Set, diag.Diagnostics) {
	if nifs == nil {
		nifs = []NetworkInterfaceModel{}
	}

	return types.SetValueFrom(ctx, networkInterfaceType, nifs)
}

// stringSetValues decodes a set of strings, null being an empty set.
func stringSetValues(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var values []string

	if set.IsNull() || set.IsUnknown() {
		return values, nil
	}

	diags := set.ElementsAs(ctx, &values, false)

	return values, diags
}

// stringSetDiff returns the values to add to and to remove from old to get cur.
func stringSetDiff(old, cur []string) (added, removed []string) {
	for _, v := range cur {
		if !utils.In(old, v) {
			added = append(added, v)
		}
	}

	for _, v := range old {
		if !utils.In(cur, v) {
			removed = append(removed, v)
		}
	}

	return added, removed
}

// refreshString updates a state value from the API, null and "" being the
// same value: whichever the state holds is kept when the API has none.
func refreshString(state *types.String, remote string) {
	if remote == "" {
		if state.IsUnknown() || state.ValueString() != "" {
			*state = types.StringNull()
		}
		return
	}

	*state = types.StringValue(remote)
}

// refreshStringSet is refreshString for sets of strings.
func refreshStringSet(ctx context.Context, state *types.Set, remote []string) diag.Diagnostics {
	if len(remote) == 0 {
		if state.IsUnknown() || len(state.Elements()) > 0 {
			*state = types.SetNull(types.StringType)
		}
		return nil
	}

	set, diags := types.SetValueFrom(ctx, types.StringType, remote)
	if diags.HasError() {
		return diags
	}
	*state = set

	return diags
}

// refreshLabels is refreshString for labels.
func refreshLabels(ctx context.Context, state *types.Map, remote exoscale.Labels) diag.Diagnostics {
	if len(remote) == 0 {
		if state.IsUnknown() || len(state.Elements()) > 0 {
			*state = types.MapNull(types.StringType)
		}
		return nil
	}

	labels, diags := types.MapValueFrom(ctx, types.StringType, remote)
	if diags.HasError() {
		return diags
	}
	*state = labels

	return diags
}

// instanceTypeName returns the `<family>.<size>` name of an instance type.
func instanceTypeName(instanceType *exoscale.InstanceType) string {
	return fmt.Sprintf(
		"%s.%s",
		strings.ToLower(string(instanceType.Family)),
		strings.ToLower(string(instanceType.Size)),
	)
}

// reverseDNS returns the reverse DNS record of the instance, "" if it has none.
func reverseDNS(ctx context.Context, client *exoscale.Client, id exoscale.UUID) (string, error) {
	record, err := client.GetReverseDNSInstance(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	return string(record.DomainName), nil
}

func privateNetworkIDs(instance *exoscale.Instance) []string {
	ids := make([]string, len(instance.PrivateNetworks))
	for i, privateNetwork := range instance.PrivateNetworks {
		ids[i] = privateNetwork.ID.String()
	}

	return ids
}

func sshKeyNames(instance *exoscale.Instance) []string {
	names := make([]string, len(instance.SSHKeys))
	for i, key := range instance.SSHKeys {
		names[i] = key.Name
	}

	return names
}

// applyInstanceComputed fills in the attributes of the model left unknown by
// the plan from the API.
//
// Create and Update use this rather than a full refresh: overwriting the
// configured attributes there would make the applied state diverge from the
// plan on any API-side normalisation, which the framework rejects outright
// ("Provider produced inconsistent result after apply"). Leaving them as
// planned lets the next Read report the difference as ordinary drift.
func applyInstanceComputed(
	ctx context.Context,
	client *exoscale.Client,
	model *ResourceModel,
	instance *exoscale.Instance,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if model.ID.IsUnknown() {
		model.ID = types.StringValue(instance.ID.String())
	}
	if model.CreatedAt.IsUnknown() {
		model.CreatedAt = types.StringValue(instance.CreatedAT.String())
	}
	if model.IPv6Address.IsUnknown() {
		model.IPv6Address = utils.OptionalString(instance.Ipv6Address)
	}
	if model.MACAddress.IsUnknown() {
		model.MACAddress = utils.OptionalString(instance.MACAddress)
	}
	if model.PublicIPAddress.IsUnknown() {
		model.PublicIPAddress = types.StringNull()
		if instance.PublicIP != nil {
			model.PublicIPAddress = types.StringValue(instance.PublicIP.String())
		}
	}
	if model.State.IsUnknown() {
		model.State = types.StringValue(string(instance.State))
	}

	if model.PrivateNetworkIDs.IsUnknown() {
		model.PrivateNetworkIDs = types.SetNull(types.StringType)
		if ids := privateNetworkIDs(instance); len(ids) > 0 {
			set, dg := types.SetValueFrom(ctx, types.StringType, ids)
			diags.Append(dg...)
			if diags.HasError() {
				return diags
			}
			model.PrivateNetworkIDs = set
		}
	}

	// Network interfaces: the MAC address is always computed, the IP address
	// is when the configuration has none.
	planned, dg := networkInterfaces(ctx, model.NetworkInterface)
	diags.Append(dg...)
	if diags.HasError() {
		return diags
	}

	withLeases := false
	for _, nif := range planned {
		withLeases = withLeases || nif.IPAddress.IsUnknown()
	}

	remote, err := remoteNetworkInterfaces(ctx, client, instance, withLeases)
	if err != nil {
		diags.AddError("unable to retrieve instance network interfaces", err.Error())
		return diags
	}

	for i := range planned {
		var found networkInterface
		for _, nif := range remote {
			if nif.NetworkID == planned[i].NetworkID.ValueString() {
				found = nif
				break
			}
		}

		if planned[i].MACAddress.IsUnknown() {
			planned[i].MACAddress = utils.OptionalString(found.MACAddress)
		}
		if planned[i].IPAddress.IsUnknown() {
			planned[i].IPAddress = types.StringValue(found.IPAddress)
		}
	}

	model.NetworkInterface, dg = networkInterfaceSet(ctx, planned)
	diags.Append(dg...)

	return diags
}

// applyInstance refreshes the whole model from the API. Only Read should use
// this; see applyInstanceComputed. Attributes the API does not return (zone,
// destroy_protected, block_storage_volume_ids, timeouts) are left untouched.
func applyInstance( //nolint:gocyclo
	ctx context.Context,
	client *exoscale.Client,
	model *ResourceModel,
	instance *exoscale.Instance,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(instance.ID.String())
	model.CreatedAt = types.StringValue(instance.CreatedAT.String())
	model.Name = types.StringValue(instance.Name)
	model.DiskSize = types.Int64Value(instance.DiskSize)
	model.State = types.StringValue(string(instance.State))
	model.IPv6 = types.BoolValue(instance.Ipv6Address != "")
	model.IPv6Address = utils.OptionalString(instance.Ipv6Address)
	model.MACAddress = utils.OptionalString(instance.MACAddress)

	model.PublicIPAddress = types.StringNull()
	if instance.PublicIP != nil {
		model.PublicIPAddress = types.StringValue(instance.PublicIP.String())
	}

	// The configuration is the only source of truth for `private` once the
	// instance exists, as it was with the SDKv2: only an import has to
	// deduce it.
	if model.Private.IsNull() || model.Private.IsUnknown() {
		model.Private = types.BoolValue(instance.PublicIPAssignment == exoscale.PublicIPAssignmentNone)
	}

	if instance.Template != nil {
		model.TemplateID = types.StringValue(instance.Template.ID.String())
	}

	deployTargetID := ""
	if instance.DeployTarget != nil {
		deployTargetID = instance.DeployTarget.ID.String()
	}
	refreshString(&model.DeployTargetID, deployTargetID)

	// The API omits these on instances predating the features.
	if instance.SecurebootEnabled != nil {
		model.EnableSecureBoot = types.BoolValue(*instance.SecurebootEnabled)
	} else if model.EnableSecureBoot.IsNull() || model.EnableSecureBoot.IsUnknown() {
		model.EnableSecureBoot = types.BoolValue(false)
	}
	if instance.TpmEnabled != nil {
		model.EnableTPM = types.BoolValue(*instance.TpmEnabled)
	} else if model.EnableTPM.IsNull() || model.EnableTPM.IsUnknown() {
		model.EnableTPM = types.BoolValue(false)
	}

	diags.Append(refreshLabels(ctx, &model.Labels, instance.Labels)...)
	diags.Append(refreshStringSet(
		ctx,
		&model.AntiAffinityGroupIDs,
		utils.AntiAffiniGroupsToAntiAffinityGroupIDs(instance.AntiAffinityGroups),
	)...)
	diags.Append(refreshStringSet(
		ctx,
		&model.ElasticIPIDs,
		utils.ElasticIPsToElasticIPIDs(instance.ElasticIPS),
	)...)
	diags.Append(refreshStringSet(
		ctx,
		&model.SecurityGroupIDs,
		utils.SecurityGroupsToSecurityGroupIDs(instance.SecurityGroups),
	)...)
	diags.Append(refreshStringSet(ctx, &model.PrivateNetworkIDs, privateNetworkIDs(instance))...)
	if diags.HasError() {
		return diags
	}

	// The API populates both fields regardless of which of `ssh_keys` or the
	// deprecated `ssh_key` was used: only refresh the one in use.
	if instance.SSHKey != nil && model.SSHKey.ValueString() != "" {
		model.SSHKey = types.StringValue(instance.SSHKey.Name)
	} else if instance.SSHKeys != nil {
		diags.Append(refreshStringSet(ctx, &model.SSHKeys, sshKeyNames(instance))...)
		if diags.HasError() {
			return diags
		}
	}
	if model.SSHKeys.IsUnknown() {
		model.SSHKeys = types.SetNull(types.StringType)
	}

	remote, err := remoteNetworkInterfaces(ctx, client, instance, true)
	if err != nil {
		diags.AddError("unable to retrieve instance network interfaces", err.Error())
		return diags
	}
	nifs := make([]NetworkInterfaceModel, 0, len(remote))
	for _, nif := range remote {
		nifs = append(nifs, NetworkInterfaceModel{
			IPAddress:  types.StringValue(nif.IPAddress),
			MACAddress: utils.OptionalString(nif.MACAddress),
			NetworkID:  types.StringValue(nif.NetworkID),
		})
	}
	nifSet, dg := networkInterfaceSet(ctx, nifs)
	diags.Append(dg...)
	if diags.HasError() {
		return diags
	}
	model.NetworkInterface = nifSet

	rdns, err := reverseDNS(ctx, client, instance.ID)
	if err != nil {
		diags.AddError("unable to retrieve instance reverse-dns", err.Error())
		return diags
	}
	refreshString(&model.ReverseDNS, strings.TrimSuffix(rdns, "."))

	if instance.InstanceType != nil {
		instanceTypes, err := client.ListInstanceTypes(ctx)
		if err != nil {
			diags.AddError("unable to find instance type", err.Error())
			return diags
		}
		instanceType, err := instanceTypes.FindInstanceType(instance.InstanceType.ID.String())
		if err != nil {
			diags.AddError("unable to find instance type", err.Error())
			return diags
		}
		// Case differences with the configuration are not drift.
		if name := instanceTypeName(&instanceType); !strings.EqualFold(name, model.Type.ValueString()) {
			model.Type = types.StringValue(name)
		}
	}

	// TODO(egoscale): an instance without user data is indistinguishable from
	// one whose user data could not be cleared (see Update), so the state is
	// only refreshed when the API returns some.
	if instance.UserData != "" {
		userData, err := utils.DecodeUserData(instance.UserData)
		if err != nil {
			diags.AddError("unable to decode user data", err.Error())
			return diags
		}
		// Encoding differences with the configuration are not drift.
		if model.UserData.IsNull() || !sameUserData(userData, model.UserData.ValueString()) {
			model.UserData = types.StringValue(userData)
		}
	}

	return diags
}
