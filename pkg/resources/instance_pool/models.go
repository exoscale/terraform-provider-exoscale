package instance_pool

import (
	"context"
	"fmt"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// InstanceModel maps the elements of the `instances` attribute of
// exoscale_instance_pool.
type InstanceModel struct {
	ID              types.String `tfsdk:"id"`
	IPv6Address     types.String `tfsdk:"ipv6_address"`
	Name            types.String `tfsdk:"name"`
	PublicIPAddress types.String `tfsdk:"public_ip_address"`
}

// Types returns nested data model types to be used for conversion.
func (m InstanceModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                types.StringType,
		"ipv6_address":      types.StringType,
		"name":              types.StringType,
		"public_ip_address": types.StringType,
	}
}

var instanceObjectType = types.ObjectType{AttrTypes: InstanceModel{}.Types()}

// idValues decodes a set of IDs into the []any the utils.*IDsTo* helpers
// take, null being an empty set.
func idValues(ctx context.Context, set types.Set) ([]any, diag.Diagnostics) {
	var ids []string

	if set.IsNull() || set.IsUnknown() {
		return []any{}, nil
	}

	diags := set.ElementsAs(ctx, &ids, false)

	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id
	}

	return values, diags
}

// antiAffinityGroupIDs returns the anti-affinity groups of the pool from
// whichever of `anti_affinity_group_ids` or the deprecated
// `affinity_group_ids` is in use.
func antiAffinityGroupIDs(ctx context.Context, model *ResourceModel) ([]any, diag.Diagnostics) {
	if len(model.AntiAffinityGroupIDs.Elements()) > 0 {
		return idValues(ctx, model.AntiAffinityGroupIDs)
	}

	return idValues(ctx, model.AffinityGroupIDs)
}

// poolInstanceType returns the `<family>.<size>` name of the instance type of
// the pool.
func poolInstanceType(ctx context.Context, client *exoscale.Client, pool *exoscale.InstancePool) (string, error) {
	if pool.InstanceType == nil {
		return "", fmt.Errorf("instance pool %s has no instance type", pool.ID)
	}

	instanceType, err := client.GetInstanceType(ctx, pool.InstanceType.ID)
	if err != nil {
		return "", err
	}

	return strings.ToLower(string(instanceType.Family)) + "." + strings.ToLower(string(instanceType.Size)), nil
}

// poolInstances returns the `instances` and `virtual_machines` values of the
// pool. The instance details cost an API call per instance.
func poolInstances(ctx context.Context, client *exoscale.Client, pool *exoscale.InstancePool) (types.Set, types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	ids := make([]string, 0, len(pool.Instances))
	instances := make([]InstanceModel, 0, len(pool.Instances))

	for _, i := range pool.Instances {
		instance, err := client.GetInstance(ctx, i.ID)
		if err != nil {
			diags.AddError("unable to retrieve instance pool member "+i.ID.String(), err.Error())
			return types.SetNull(instanceObjectType), types.SetNull(types.StringType), diags
		}

		// Nested values have always been "" rather than null in the state.
		publicIP := ""
		if len(instance.PublicIP) > 0 {
			publicIP = instance.PublicIP.String()
		}

		ids = append(ids, instance.ID.String())
		instances = append(instances, InstanceModel{
			ID:              types.StringValue(instance.ID.String()),
			IPv6Address:     types.StringValue(instance.Ipv6Address),
			Name:            types.StringValue(instance.Name),
			PublicIPAddress: types.StringValue(publicIP),
		})
	}

	instanceSet, dg := types.SetValueFrom(ctx, instanceObjectType, instances)
	diags.Append(dg...)
	idSet, dg := types.SetValueFrom(ctx, types.StringType, ids)
	diags.Append(dg...)

	return instanceSet, idSet, diags
}

// applyPoolComputed fills in the attributes of the model left unknown by the
// plan from the API.
//
// Create and Update use this rather than a full refresh: overwriting the
// configured attributes there would make the applied state diverge from the
// plan on any API-side normalisation, which the framework rejects outright
// ("Provider produced inconsistent result after apply"). Leaving them as
// planned lets the next Read report the difference as ordinary drift.
func applyPoolComputed(
	ctx context.Context,
	client *exoscale.Client,
	model *ResourceModel,
	pool *exoscale.InstancePool,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if model.ID.IsUnknown() {
		model.ID = types.StringValue(pool.ID.String())
	}
	if model.DiskSize.IsUnknown() {
		model.DiskSize = types.Int64Value(pool.DiskSize)
	}
	if model.MinAvailable.IsUnknown() {
		model.MinAvailable = types.Int64Value(pool.MinAvailable)
	}
	if model.State.IsUnknown() {
		model.State = types.StringValue(string(pool.State))
	}

	if model.Instances.IsUnknown() || model.VirtualMachines.IsUnknown() {
		instances, ids, dg := poolInstances(ctx, client, pool)
		diags.Append(dg...)
		if diags.HasError() {
			return diags
		}
		if model.Instances.IsUnknown() {
			model.Instances = instances
		}
		if model.VirtualMachines.IsUnknown() {
			model.VirtualMachines = ids
		}
	}

	return diags
}

// applyPool refreshes the whole model from the API. Only Read should use this;
// see applyPoolComputed. Attributes the API does not return (zone, timeouts)
// are left untouched.
func applyPool( //nolint:gocyclo
	ctx context.Context,
	client *exoscale.Client,
	model *ResourceModel,
	pool *exoscale.InstancePool,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(pool.ID.String())
	model.Name = types.StringValue(pool.Name)
	model.DiskSize = types.Int64Value(pool.DiskSize)
	model.Size = types.Int64Value(pool.Size)
	model.MinAvailable = types.Int64Value(pool.MinAvailable)
	model.State = types.StringValue(string(pool.State))
	model.InstancePrefix = types.StringValue(pool.InstancePrefix)
	model.IPv6 = types.BoolValue(utils.DefaultBool(pool.Ipv6Enabled, false))

	if pool.Template != nil {
		model.TemplateID = types.StringValue(pool.Template.ID.String())
	}

	utils.RefreshString(&model.Description, pool.Description)

	deployTargetID := ""
	if pool.DeployTarget != nil {
		deployTargetID = pool.DeployTarget.ID.String()
	}
	utils.RefreshString(&model.DeployTargetID, deployTargetID)

	keyPair := ""
	if pool.SSHKey != nil {
		keyPair = pool.SSHKey.Name
	}
	utils.RefreshString(&model.KeyPair, keyPair)

	diags.Append(utils.RefreshLabels(ctx, &model.Labels, pool.Labels)...)

	// Only refresh the anti-affinity group attribute in use: on import (both
	// null), the non-deprecated one gets the groups.
	antiAffinityGroups := utils.AntiAffiniGroupsToAntiAffinityGroupIDs(pool.AntiAffinityGroups)
	if model.AntiAffinityGroupIDs.IsNull() && !model.AffinityGroupIDs.IsNull() {
		utils.RefreshStringSet(ctx, &diags, &model.AffinityGroupIDs, antiAffinityGroups)
	} else {
		utils.RefreshStringSet(ctx, &diags, &model.AntiAffinityGroupIDs, antiAffinityGroups)
	}

	utils.RefreshStringSet(ctx, &diags, &model.ElasticIPIDs, utils.ElasticIPsToElasticIPIDs(pool.ElasticIPS))
	utils.RefreshStringSet(ctx, &diags, &model.NetworkIDs, utils.PrivateNetworksToPrivateNetworkIDs(pool.PrivateNetworks))
	utils.RefreshStringSet(ctx, &diags, &model.SecurityGroupIDs, utils.SecurityGroupsToSecurityGroupIDs(pool.SecurityGroups))
	if diags.HasError() {
		return diags
	}

	name, err := poolInstanceType(ctx, client, pool)
	if err != nil {
		diags.AddError("unable to retrieve instance type", err.Error())
		return diags
	}
	// Case differences with the configuration are not drift.
	if !strings.EqualFold(name, model.InstanceType.ValueString()) {
		model.InstanceType = types.StringValue(name)
	}

	if pool.UserData == "" {
		utils.RefreshString(&model.UserData, "")
	} else {
		userData, err := utils.DecodeUserData(pool.UserData)
		if err != nil {
			diags.AddError("unable to decode user data", err.Error())
			return diags
		}
		// Encoding differences with the configuration are not drift.
		if model.UserData.IsNull() || !sameUserData(userData, model.UserData.ValueString()) {
			model.UserData = types.StringValue(userData)
		}
	}

	instances, ids, dg := poolInstances(ctx, client, pool)
	diags.Append(dg...)
	if diags.HasError() {
		return diags
	}
	model.Instances = instances
	model.VirtualMachines = ids

	return diags
}
