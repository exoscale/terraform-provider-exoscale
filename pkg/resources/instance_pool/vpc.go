package instance_pool

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// VPCModel maps the `vpc` block of exoscale_instance_pool.
type VPCModel struct {
	ID        types.String `tfsdk:"id"`
	SubnetIDs types.List   `tfsdk:"subnet_ids"`
}

// Types returns nested data model types to be used for conversion.
func (m VPCModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"id":         types.StringType,
		"subnet_ids": types.ListType{ElemType: types.StringType},
	}
}

var vpcType = types.ObjectType{AttrTypes: VPCModel{}.Types()}

// noVPC is the value of the `vpc` block when there is none.
func noVPC() types.List {
	return types.ListValueMust(vpcType, []attr.Value{})
}

// vpcSubnets decodes the `vpc` block: nil when there is none (or it is not
// known yet). The subnet IDs are in the configured order, unknown ones being
// left out.
func vpcSubnets(ctx context.Context, list types.List) (*VPCModel, []string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.Length(basetypes.CollectionLengthOptions{
		UnhandledNullAsZero:    true,
		UnhandledUnknownAsZero: true,
	}) == 0 {
		return nil, nil, diags
	}

	var vpcs []VPCModel
	diags.Append(list.ElementsAs(ctx, &vpcs, false)...)
	if diags.HasError() {
		return nil, nil, diags
	}
	vpc := vpcs[0]

	var ids []string
	if !vpc.SubnetIDs.IsNull() && !vpc.SubnetIDs.IsUnknown() {
		var values []types.String
		diags.Append(vpc.SubnetIDs.ElementsAs(ctx, &values, false)...)
		for _, v := range values {
			if !v.IsUnknown() {
				ids = append(ids, v.ValueString())
			}
		}
	}

	return &vpc, ids, diags
}

// vpcList encodes the `vpc` block, an empty list (no block) when vpcID is
// empty.
func vpcList(ctx context.Context, vpcID string, subnetIDs []string) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if vpcID == "" {
		return noVPC(), diags
	}

	if subnetIDs == nil {
		subnetIDs = []string{}
	}

	subnets, dg := types.ListValueFrom(ctx, types.StringType, subnetIDs)
	diags.Append(dg...)
	if diags.HasError() {
		return noVPC(), diags
	}

	list, dg := types.ListValueFrom(ctx, vpcType, []VPCModel{{ID: types.StringValue(vpcID), SubnetIDs: subnets}})
	diags.Append(dg...)

	return list, diags
}

// subnetLookup returns a VPC subnet from the API by its ID.
type subnetLookup func(id string) (*exoscale.Subnet, error)

// isPublicSubnet tells whether a subnet has public IPv4 addresses.
func isPublicSubnet(subnet *exoscale.Subnet) bool {
	if subnet.AddressSpace != "" && subnet.AddressSpace != exoscale.SubnetAddressSpacePrivate {
		return true
	}

	// A subnet without a (valid) block is an API bug and therefore should
	// not be skipped and appear to the user.
	ip, _, err := net.ParseCIDR(subnet.Ipv4Block)

	return err == nil && !ip.IsPrivate()
}

// splitVPCSubnets splits the subnets of the pool VPC as known by the API into
// the ones Terraform manages, and the ones it leaves alone: the subnets with
// public IPv4 addresses, unless managed returns true for them (they are
// listed in the configuration or in the state). Only the subnets not managed
// cost an API call.
func splitVPCSubnets(
	vpc *exoscale.InstancePoolVpc,
	managed func(id string) bool,
	lookup subnetLookup,
) (visible, hidden []string, err error) {
	if vpc == nil {
		return nil, nil, nil
	}

	for _, subnet := range vpc.Subnets {
		id := subnet.ID.String()
		if managed(id) {
			visible = append(visible, id)
			continue
		}

		details, err := lookup(id)
		switch {
		case errors.Is(err, exoscale.ErrNotFound):
			visible = append(visible, id)
		case err != nil:
			return nil, nil, fmt.Errorf("unable to retrieve VPC subnet %s: %w", id, err)
		case isPublicSubnet(details):
			hidden = append(hidden, id)
		default:
			visible = append(visible, id)
		}
	}

	return visible, hidden, nil
}

// subnetLookupFor returns the lookup of the subnets of a VPC.
func subnetLookupFor(ctx context.Context, client *exoscale.Client, vpc *exoscale.InstancePoolVpc) subnetLookup {
	return func(id string) (*exoscale.Subnet, error) {
		return client.GetSubnet(ctx, vpc.ID, exoscale.UUID(id))
	}
}

// containsAny returns a function telling whether an ID is in one of the lists.
func containsAny(lists ...[]string) func(string) bool {
	return func(id string) bool {
		for _, list := range lists {
			if slices.Contains(list, id) {
				return true
			}
		}
		return false
	}
}

// vpcFromAPI returns the `vpc` block value for the VPC of the pool and the
// managed subnets, no block if there are none.
func vpcFromAPI(ctx context.Context, vpc *exoscale.InstancePoolVpc, visible []string) (types.List, diag.Diagnostics) {
	if vpc == nil || len(visible) == 0 {
		return noVPC(), nil
	}

	return vpcList(ctx, vpc.ID.String(), visible)
}

// subnetRefs converts subnet IDs to the subnet references of an API request,
// all of which are a struct holding only the subnet ID.
func subnetRefs[T ~struct {
	ID exoscale.UUID `json:"id" validate:"required"`
}](ids []string) []T {
	refs := make([]T, len(ids))
	for i, id := range ids {
		refs[i] = T{ID: exoscale.UUID(id)}
	}

	return refs
}

// createRequestVPC returns the VPC to request at the pool creation, nil if
// there is none.
func createRequestVPC(ctx context.Context, model *ResourceModel) (*exoscale.CreateInstancePoolRequestVpc, diag.Diagnostics) {
	vpc, ids, diags := vpcSubnets(ctx, model.VPC)
	if diags.HasError() || vpc == nil {
		return nil, diags
	}

	return &exoscale.CreateInstancePoolRequestVpc{
		ID:      exoscale.UUID(vpc.ID.ValueString()),
		Subnets: subnetRefs[exoscale.CreateInstancePoolRequestVpcSubnets](ids),
	}, diags
}

// updateRequestVPC returns the VPC to request at the pool update, nil if
// there is none: the API then detaches the pool from its VPC, so it must be
// sent along with every update. The API replaces all the subnets, so the
// hidden ones (see splitVPCSubnets) are requested after the planned ones, as
// long as the VPC remains the same, to be left alone.
func updateRequestVPC(
	ctx context.Context,
	model *ResourceModel,
	current *exoscale.InstancePoolVpc,
	hidden []string,
) (*exoscale.UpdateInstancePoolRequestVpc, diag.Diagnostics) {
	vpc, ids, diags := vpcSubnets(ctx, model.VPC)
	if diags.HasError() {
		return nil, diags
	}

	var vpcID exoscale.UUID
	switch {
	case vpc != nil:
		vpcID = exoscale.UUID(vpc.ID.ValueString())
	case len(hidden) > 0 && current != nil:
		vpcID = current.ID
	default:
		return nil, diags
	}

	if current != nil && current.ID == vpcID {
		ids = append(ids, hidden...)
	}
	if len(ids) == 0 {
		return nil, diags
	}

	return &exoscale.UpdateInstancePoolRequestVpc{
		ID:      vpcID,
		Subnets: subnetRefs[exoscale.UpdateInstancePoolRequestVpcSubnets](ids),
	}, diags
}

// keptSubnets returns how many of the current subnets are kept as they are.
// The subnets are strictly ordered: from the first position at which the
// planned subnet differs from the current one, every current subnet is
// detached and every planned subnet is (re)attached, in order.
func keptSubnets(current, planned []string) int {
	kept := 0
	for kept < len(current) && kept < len(planned) && current[kept] == planned[kept] {
		kept++
	}

	return kept
}

// vpcChanges returns the subnets to detach (from the current VPC) and to
// attach (to the planned one), in that order, and the position of the first
// attached one. A nil VPC is no VPC block.
func vpcChanges(
	currentVPC *VPCModel, current []string,
	plannedVPC *VPCModel, planned []string,
) (kept int, detach, attach []string) {
	if currentVPC != nil && plannedVPC != nil && currentVPC.ID.Equal(plannedVPC.ID) {
		kept = keptSubnets(current, planned)
	}

	if currentVPC != nil {
		detach = current[min(kept, len(current)):]
	}
	if plannedVPC != nil {
		attach = planned[min(kept, len(planned)):]
	}

	return kept, detach, attach
}

// unknownSubnet stands for a planned subnet whose ID is not known yet.
const unknownSubnet = ""

// describeVPCChanges describes for the user the attachments an update
// performs, "" if none.
func describeVPCChanges(detach, attach []string) string {
	if len(detach) == 0 && len(attach) == 0 {
		return ""
	}

	reattached := map[string]bool{}
	for _, a := range attach {
		for _, d := range detach {
			if a == d {
				reattached[d] = true
			}
		}
	}

	name := func(id string) string {
		if id == unknownSubnet {
			return "(known after apply)"
		}
		return id
	}

	var lines []string
	for _, d := range detach {
		if !reattached[d] {
			lines = append(lines, "detach subnet "+d)
		}
	}
	for _, a := range attach {
		verb := "attach"
		if reattached[a] {
			verb = "reattach"
		}
		lines = append(lines, verb+" subnet "+name(a))
	}

	return "The following VPC subnet attachments of the managed instances will be changed:\n  - " +
		strings.Join(lines, "\n  - ")
}

// apiVPC is the VPC of the pool as known by the API, limited to the subnets
// managed by Terraform.
type apiVPC struct {
	ID      string
	Subnets []string
}

// planVPC returns a description of the attachment changes of the plan, "" if
// there is none or if it is not known yet. The changes are relative to the
// state, or to current when it is not nil (see vpcForAdoption).
func planVPC(ctx context.Context, plan, state *ResourceModel, current *apiVPC) (string, diag.Diagnostics) {
	if plan.VPC.IsUnknown() {
		return "", nil
	}

	plannedVPC, planned, diags := vpcSubnets(ctx, plan.VPC)
	currentVPC, currentSubnets, dg := vpcSubnets(ctx, state.VPC)
	diags.Append(dg...)
	if diags.HasError() {
		return "", diags
	}
	if current != nil {
		currentVPC, currentSubnets = nil, nil
		if len(current.Subnets) > 0 {
			currentVPC = &VPCModel{ID: types.StringValue(current.ID)}
			currentSubnets = current.Subnets
		}
	}

	// The subnets are not known yet (e.g. a for expression on resources yet
	// to create): neither are the changes.
	if plannedVPC != nil && plannedVPC.SubnetIDs.IsUnknown() {
		return "", diags
	}

	// Subnets yet to create (unknown IDs) keep their position, as
	// unknownSubnet: they differ from any current subnet. An unknown VPC ID
	// differs from the current one.
	if plannedVPC != nil {
		planned = planned[:0]
		for _, v := range plannedVPC.SubnetIDs.Elements() {
			id := unknownSubnet
			if s, ok := v.(types.String); ok && !s.IsUnknown() {
				id = s.ValueString()
			}
			planned = append(planned, id)
		}
	}

	_, detach, attach := vpcChanges(currentVPC, currentSubnets, plannedVPC, planned)

	return describeVPCChanges(detach, attach), diags
}
