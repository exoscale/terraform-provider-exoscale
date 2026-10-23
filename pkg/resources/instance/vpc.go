package instance

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// vpcIPv4Auto lets the platform pick the IPv4 address of a VPC interface.
const vpcIPv4Auto = "auto"

// VPCModel maps the `vpc` block of exoscale_compute_instance.
type VPCModel struct {
	ID         types.String `tfsdk:"id"`
	Interfaces types.List   `tfsdk:"interfaces"`
}

// Types returns nested data model types to be used for conversion.
func (m VPCModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"id":         types.StringType,
		"interfaces": types.ListType{ElemType: vpcInterfaceType},
	}
}

// VPCInterfaceModel maps an element of `vpc.interfaces`.
type VPCInterfaceModel struct {
	SubnetID             types.String `tfsdk:"subnet_id"`
	IPv4Address          types.String `tfsdk:"ipv4_address"`
	IPv4AddressAllocated types.String `tfsdk:"ipv4_address_allocated"`
}

// Types returns nested data model types to be used for conversion.
func (m VPCInterfaceModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"subnet_id":              types.StringType,
		"ipv4_address":           types.StringType,
		"ipv4_address_allocated": types.StringType,
	}
}

var (
	vpcType          = types.ObjectType{AttrTypes: VPCModel{}.Types()}
	vpcInterfaceType = types.ObjectType{AttrTypes: VPCInterfaceModel{}.Types()}
)

// requestedIPv4 returns the IPv4 address to request when attaching the
// interface, nil letting the platform pick one.
func (m VPCInterfaceModel) requestedIPv4() net.IP {
	if m.IPv4Address.ValueString() == vpcIPv4Auto {
		return nil
	}

	return net.ParseIP(m.IPv4Address.ValueString())
}

// vpcInterfaces decodes the `vpc` block: nil when there is none (or it is not
// known yet).
func vpcInterfaces(ctx context.Context, list types.List) (*VPCModel, []VPCInterfaceModel, diag.Diagnostics) {
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

	var ifaces []VPCInterfaceModel
	if !vpc.Interfaces.IsNull() && !vpc.Interfaces.IsUnknown() {
		diags.Append(vpc.Interfaces.ElementsAs(ctx, &ifaces, false)...)
	}

	return &vpc, ifaces, diags
}

// vpcList encodes the `vpc` block, an empty list (no block) when vpcID is
// null.
func vpcList(ctx context.Context, vpcID types.String, ifaces []VPCInterfaceModel) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if vpcID.IsNull() {
		return types.ListValueMust(vpcType, []attr.Value{}), diags
	}

	if ifaces == nil {
		ifaces = []VPCInterfaceModel{}
	}

	interfaces, dg := types.ListValueFrom(ctx, vpcInterfaceType, ifaces)
	diags.Append(dg...)
	if diags.HasError() {
		return types.ListValueMust(vpcType, []attr.Value{}), diags
	}

	list, dg := types.ListValueFrom(ctx, vpcType, []VPCModel{{ID: vpcID, Interfaces: interfaces}})
	diags.Append(dg...)

	return list, diags
}

// vpcInterfaceUnchanged tells whether the planned interface leaves the current
// one as is: same subnet, and same IPv4 address unless the platform picks it.
func vpcInterfaceUnchanged(current, planned VPCInterfaceModel) bool {
	if planned.SubnetID.IsUnknown() || planned.IPv4Address.IsUnknown() ||
		!planned.SubnetID.Equal(current.SubnetID) {
		return false
	}

	ip := planned.IPv4Address.ValueString()

	return ip == vpcIPv4Auto || ip == current.IPv4AddressAllocated.ValueString()
}

// keptVPCInterfaces returns how many of the current interfaces are kept as
// they are. The interfaces are strictly ordered: from the first position at
// which the planned interface differs from the current one, every current
// interface is detached and every planned interface is (re)attached, in order.
func keptVPCInterfaces(current, planned []VPCInterfaceModel) int {
	kept := 0
	for kept < len(current) && kept < len(planned) && vpcInterfaceUnchanged(current[kept], planned[kept]) {
		kept++
	}

	return kept
}

// vpcChanges returns the interfaces to detach (from the current VPC) and to
// attach (to the planned one), in that order. A nil VPC is no VPC block.
func vpcChanges(
	currentVPC *VPCModel, current []VPCInterfaceModel,
	plannedVPC *VPCModel, planned []VPCInterfaceModel,
) (kept int, detach, attach []VPCInterfaceModel) {
	if currentVPC != nil && plannedVPC != nil && !plannedVPC.ID.IsUnknown() && plannedVPC.ID.Equal(currentVPC.ID) {
		kept = keptVPCInterfaces(current, planned)
	}

	if currentVPC != nil {
		detach = current[min(kept, len(current)):]
	}
	if plannedVPC != nil {
		attach = planned[min(kept, len(planned)):]
	}

	return kept, detach, attach
}

// describeVPCChanges describes for the user the attachments an update
// performs, "" if none.
func describeVPCChanges(kept int, detach, attach []VPCInterfaceModel) string {
	if len(detach) == 0 && len(attach) == 0 {
		return ""
	}

	ipv4 := func(m VPCInterfaceModel) string {
		if v := m.IPv4AddressAllocated; !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
			return v.ValueString()
		}
		if v := m.IPv4Address; !v.IsUnknown() && v.ValueString() != vpcIPv4Auto {
			return v.ValueString()
		}
		return "auto"
	}

	reattached := map[string]bool{}
	for _, a := range attach {
		for _, d := range detach {
			if a.SubnetID.Equal(d.SubnetID) {
				reattached[d.SubnetID.ValueString()] = true
			}
		}
	}

	var lines []string
	for _, d := range detach {
		if !reattached[d.SubnetID.ValueString()] {
			lines = append(lines, fmt.Sprintf("detach subnet %s (%s)", d.SubnetID.ValueString(), ipv4(d)))
		}
	}
	for i, a := range attach {
		verb := "attach"
		if reattached[a.SubnetID.ValueString()] {
			verb = "reattach"
		}
		lines = append(lines, fmt.Sprintf(
			"%s subnet %s as interfaces[%d] (%s)",
			verb, a.SubnetID.ValueString(), kept+i, ipv4(a),
		))
	}

	return "The following VPC subnet attachments will be changed:\n  - " +
		strings.Join(lines, "\n  - ")
}

// isPublicAttachment tells whether an attachment has a public IPv4 address.
func isPublicAttachment(subnet exoscale.InstanceVpcSubnets) bool {
	// The not-nil check is needed because net.IP(nil).IsPrivate()
	// would just return false. An attachment without an IP is
	// an API bug and therefore should not be skipped and appear
	// to the user.
	return subnet.Ipv4 != nil && !subnet.Ipv4.IsPrivate()
}

// vpcInterfacesFromAPI returns the VPC ID and the subnet attachments of the
// instance as known by the API. Attachments unmanaged by terraform
// default are left out.
func vpcInterfacesFromAPI(
	instance *exoscale.Instance,
	isManagedByTF func(subnetID string) bool,
) (string, []exoscale.InstanceVpcSubnets) {
	if instance.Vpc == nil {
		return "", nil
	}

	var subnets []exoscale.InstanceVpcSubnets
	for _, subnet := range instance.Vpc.Subnets {
		if isPublicAttachment(subnet) && (isManagedByTF == nil || !isManagedByTF(subnet.ID.String())) {
			continue
		}
		subnets = append(subnets, subnet)
	}

	return instance.Vpc.ID.String(), subnets
}

func hasSubnet(ifaces ...[]VPCInterfaceModel) func(string) bool {
	return func(subnetID string) bool {
		for _, list := range ifaces {
			for _, iface := range list {
				if iface.SubnetID.ValueString() == subnetID {
					return true
				}
			}
		}
		return false
	}
}

// interfacesFromAPI converts API attachments to interfaces. The IPv4
// address of the matching prior interface is kept as long as it still
// describes the attachment (e.g. "auto"), the allocated one is used otherwise.
func interfacesFromAPI(subnets []exoscale.InstanceVpcSubnets, prior []VPCInterfaceModel) []VPCInterfaceModel {
	ifaces := make([]VPCInterfaceModel, len(subnets))
	for i, subnet := range subnets {
		allocated := ""
		if subnet.Ipv4 != nil {
			allocated = subnet.Ipv4.String()
		}

		ifaces[i] = VPCInterfaceModel{
			SubnetID:             types.StringValue(subnet.ID.String()),
			IPv4Address:          types.StringValue(allocated),
			IPv4AddressAllocated: types.StringValue(allocated),
		}

		for _, p := range prior {
			if vpcInterfaceUnchanged(ifaces[i], p) {
				ifaces[i].IPv4Address = p.IPv4Address
				break
			}
		}
	}

	return ifaces
}

// currentVPCInterfaces returns the interfaces the planned ones are compared
// to: the API attachments managed by Terraform (those in the state) or about
// to be (those in the plan), in interface order. Without an instance from the
// API, the state is used as is.
func currentVPCInterfaces(
	instance *exoscale.Instance,
	stateVPC *VPCModel,
	state, planned []VPCInterfaceModel,
) (*VPCModel, []VPCInterfaceModel) {
	if instance == nil {
		return stateVPC, state
	}

	vpcID, subnets := vpcInterfacesFromAPI(instance, hasSubnet(state, planned))
	if len(subnets) == 0 {
		return nil, nil
	}

	return &VPCModel{ID: types.StringValue(vpcID)}, interfacesFromAPI(subnets, state)
}

// refreshVPC sets the `vpc` block of the model from the API. The configured
// IPv4 address is kept as long as it still describes the attachment.
func refreshVPC(ctx context.Context, model *ResourceModel, instance *exoscale.Instance) diag.Diagnostics {
	_, current, diags := vpcInterfaces(ctx, model.VPC)
	if diags.HasError() {
		return diags
	}

	// Attachments hitherto unmanaged only show up once adopted.
	// (adopted means, the user created an element in `interfaces` for it)
	vpcID, subnets := vpcInterfacesFromAPI(instance, hasSubnet(current))
	if len(subnets) == 0 {
		model.VPC = types.ListValueMust(vpcType, []attr.Value{})
		return diags
	}

	ifaces := interfacesFromAPI(subnets, current)

	obj, dg := vpcList(ctx, types.StringValue(vpcID), ifaces)
	diags.Append(dg...)
	if !diags.HasError() {
		model.VPC = obj
	}

	return diags
}

// applyVPCComputed fills in the allocated IPv4 addresses left unknown by the
// plan.
func applyVPCComputed(ctx context.Context, model *ResourceModel, instance *exoscale.Instance) diag.Diagnostics {
	vpc, planned, diags := vpcInterfaces(ctx, model.VPC)
	if diags.HasError() || vpc == nil {
		return diags
	}

	_, subnets := vpcInterfacesFromAPI(instance, func(string) bool { return true })
	for i := range planned {
		if !planned[i].IPv4AddressAllocated.IsUnknown() {
			continue
		}

		planned[i].IPv4AddressAllocated = types.StringNull()
		for _, subnet := range subnets {
			if subnet.ID.String() == planned[i].SubnetID.ValueString() && subnet.Ipv4 != nil {
				planned[i].IPv4AddressAllocated = types.StringValue(subnet.Ipv4.String())
				break
			}
		}
	}

	model.VPC, diags = vpcList(ctx, vpc.ID, planned)

	return diags
}

// planVPC fills in the allocated IPv4 addresses known at plan time, and
// returns a description of the attachment changes. instance is only needed
// (non-nil) when the plan adopts attachments missing from the state.
func planVPC(ctx context.Context, plan, state *ResourceModel, instance *exoscale.Instance) (string, diag.Diagnostics) {
	if plan.VPC.IsUnknown() {
		return "", nil
	}

	plannedVPC, planned, diags := vpcInterfaces(ctx, plan.VPC)
	stateVPC, stateIfaces, dg := vpcInterfaces(ctx, state.VPC)
	diags.Append(dg...)
	if diags.HasError() {
		return "", diags
	}
	currentVPC, current := currentVPCInterfaces(instance, stateVPC, stateIfaces, planned)

	kept, detach, attach := vpcChanges(currentVPC, current, plannedVPC, planned)

	if plannedVPC != nil && !plannedVPC.Interfaces.IsUnknown() {
		for i := range planned {
			switch ip := planned[i].IPv4Address; {
			case i < kept:
				planned[i].IPv4AddressAllocated = current[i].IPv4AddressAllocated
			case !ip.IsUnknown() && !ip.IsNull() && ip.ValueString() != vpcIPv4Auto:
				planned[i].IPv4AddressAllocated = ip
			default:
				planned[i].IPv4AddressAllocated = types.StringUnknown()
			}
		}

		plan.VPC, dg = vpcList(ctx, plannedVPC.ID, planned)
		diags.Append(dg...)
		attach = planned[kept:]
	}

	return describeVPCChanges(kept, detach, attach), diags
}

// updateVPC detaches, then attaches the subnets for the instance to match
// the plan.
func updateVPC(
	ctx context.Context,
	client *exoscale.Client,
	instance *exoscale.Instance,
	plan, state *ResourceModel,
) diag.Diagnostics {
	plannedVPC, planned, diags := vpcInterfaces(ctx, plan.VPC)
	stateVPC, stateIfaces, dg := vpcInterfaces(ctx, state.VPC)
	diags.Append(dg...)
	if diags.HasError() {
		return diags
	}
	currentVPC, current := currentVPCInterfaces(instance, stateVPC, stateIfaces, planned)

	_, detach, attach := vpcChanges(currentVPC, current, plannedVPC, planned)

	// All the detachments come first, so that their IP addresses can be
	// reused by the attachments.
	for _, iface := range detach {
		err := wait(ctx, client)(client.DetachInstanceFromSubnet(
			ctx,
			exoscale.UUID(currentVPC.ID.ValueString()),
			exoscale.UUID(iface.SubnetID.ValueString()),
			exoscale.DetachInstanceFromSubnetRequest{Instance: &exoscale.InstanceRef{ID: instance.ID}},
		))
		if err != nil && !errors.Is(err, exoscale.ErrNotFound) {
			diags.AddError("unable to detach VPC subnet "+iface.SubnetID.ValueString(), err.Error())
			return diags
		}
	}

	for _, iface := range attach {
		err := wait(ctx, client)(client.AttachInstanceToSubnet(
			ctx,
			exoscale.UUID(plannedVPC.ID.ValueString()),
			exoscale.UUID(iface.SubnetID.ValueString()),
			exoscale.AttachInstanceToSubnetRequest{
				Instance: &exoscale.InstanceRef{ID: instance.ID},
				Ipv4:     iface.requestedIPv4(),
			},
		))
		if err != nil {
			diags.AddError("unable to attach VPC subnet "+iface.SubnetID.ValueString(), err.Error())
			return diags
		}
	}

	return diags
}

// createInstanceRequestVpc returns the VPC attachments to request at the instance
// creation, nil if there are none.
func createInstanceRequestVpc(ctx context.Context, model *ResourceModel) (*exoscale.CreateInstanceRequestVpc, diag.Diagnostics) {
	vpc, ifaces, diags := vpcInterfaces(ctx, model.VPC)
	if diags.HasError() || vpc == nil {
		return nil, diags
	}

	request := &exoscale.CreateInstanceRequestVpc{
		ID:      exoscale.UUID(vpc.ID.ValueString()),
		Subnets: make([]exoscale.CreateInstanceRequestVpcSubnets, len(ifaces)),
	}
	for i, iface := range ifaces {
		request.Subnets[i] = exoscale.CreateInstanceRequestVpcSubnets{
			ID:   exoscale.UUID(iface.SubnetID.ValueString()),
			Ipv4: iface.requestedIPv4(),
		}
	}

	return request, diags
}
