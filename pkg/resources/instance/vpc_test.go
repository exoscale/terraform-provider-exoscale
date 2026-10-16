package instance

import (
	"context"
	"net"
	"testing"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func iface(subnet, ip, allocated string) VPCInterfaceModel {
	m := VPCInterfaceModel{
		SubnetID:             types.StringValue(subnet),
		IPv4Address:          types.StringValue(ip),
		IPv4AddressAllocated: types.StringUnknown(),
	}
	if allocated != "" {
		m.IPv4AddressAllocated = types.StringValue(allocated)
	}
	return m
}

func subnetIDs(ifaces []VPCInterfaceModel) []string {
	ids := []string{}
	for _, i := range ifaces {
		ids = append(ids, i.SubnetID.ValueString())
	}
	return ids
}

func TestVPCChanges(t *testing.T) {
	vpc := &VPCModel{ID: types.StringValue("vpc")}
	otherVPC := &VPCModel{ID: types.StringValue("other")}

	a := iface("a", "10.0.0.22", "10.0.0.22")
	b := iface("b", "auto", "10.0.1.131")
	c := iface("c", "auto", "10.0.2.5")
	current := []VPCInterfaceModel{a, b, c}

	tests := []struct {
		name           string
		currentVPC     *VPCModel
		current        []VPCInterfaceModel
		plannedVPC     *VPCModel
		planned        []VPCInterfaceModel
		kept           int
		detach, attach []string
	}{
		{"unchanged", vpc, current, vpc, []VPCInterfaceModel{iface("a", "10.0.0.22", ""), iface("b", "auto", ""), iface("c", "auto", "")}, 3, []string{}, []string{}},
		{"auto to allocated IP", vpc, current, vpc, []VPCInterfaceModel{a, iface("b", "10.0.1.131", ""), c}, 3, []string{}, []string{}},
		{"swap", vpc, current[:2], vpc, []VPCInterfaceModel{b, a}, 0, []string{"a", "b"}, []string{"b", "a"}},
		{"insert in the middle", vpc, current[:2], vpc, []VPCInterfaceModel{a, c, b}, 1, []string{"b"}, []string{"c", "b"}},
		{"remove the first", vpc, current, vpc, []VPCInterfaceModel{b, c}, 0, []string{"a", "b", "c"}, []string{"b", "c"}},
		{"remove the last", vpc, current, vpc, []VPCInterfaceModel{a, b}, 2, []string{"c"}, []string{}},
		{"append", vpc, current[:2], vpc, current, 2, []string{}, []string{"c"}},
		{"IP change", vpc, current, vpc, []VPCInterfaceModel{a, iface("b", "10.0.1.7", ""), c}, 1, []string{"b", "c"}, []string{"b", "c"}},
		{"VPC change", vpc, current[:1], otherVPC, []VPCInterfaceModel{a}, 0, []string{"a"}, []string{"a"}},
		{"block added", nil, nil, vpc, current[:1], 0, []string{}, []string{"a"}},
		{"block removed", vpc, current, nil, nil, 0, []string{"a", "b", "c"}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kept, detach, attach := vpcChanges(tt.currentVPC, tt.current, tt.plannedVPC, tt.planned)
			require.Equal(t, tt.kept, kept)
			require.Equal(t, tt.detach, subnetIDs(detach))
			require.Equal(t, tt.attach, subnetIDs(attach))
		})
	}
}

func TestDescribeVPCChanges(t *testing.T) {
	a := iface("a", "10.0.0.22", "10.0.0.22")
	b := iface("b", "auto", "10.0.1.131")
	c := iface("c", "auto", "10.0.2.5")

	require.Empty(t, describeVPCChanges(2, nil, nil))
	require.Equal(t,
		"The following VPC subnet attachments will be changed:\n"+
			"  - detach subnet c (10.0.2.5)\n"+
			"  - reattach subnet b as interfaces[0] (10.0.1.131)\n"+
			"  - reattach subnet a as interfaces[1] (10.0.0.22)",
		describeVPCChanges(0, []VPCInterfaceModel{a, b, c}, []VPCInterfaceModel{b, a}),
	)
	require.Equal(t,
		"The following VPC subnet attachments will be changed:\n"+
			"  - attach subnet c as interfaces[1] (auto)",
		describeVPCChanges(1, nil, []VPCInterfaceModel{iface("c", "auto", "")}),
	)
}

func TestRemoteVPCInterfacesSkipsPublicIPs(t *testing.T) {
	instance := &exoscale.Instance{Vpc: &exoscale.InstanceVpc{
		ID: exoscale.UUID("vpc"),
		Subnets: []exoscale.InstanceVpcSubnets{
			{ID: exoscale.UUID("a"), Ipv4: net.ParseIP("10.0.0.22")},
			{ID: exoscale.UUID("public"), Ipv4: net.ParseIP("194.182.160.1")},
			{ID: exoscale.UUID("b"), Ipv4: net.ParseIP("192.168.1.1")},
		},
	}}

	vpcID, subnets := vpcInterfacesFromAPI(instance, nil)
	require.Equal(t, "vpc", vpcID)
	require.Len(t, subnets, 2)
	require.Equal(t, exoscale.UUID("a"), subnets[0].ID)
	require.Equal(t, exoscale.UUID("b"), subnets[1].ID)

	vpcID, subnets = vpcInterfacesFromAPI(&exoscale.Instance{}, nil)
	require.Empty(t, vpcID)
	require.Empty(t, subnets)
}

func TestVPCInterfacesNoBlock(t *testing.T) {
	ctx := context.Background()

	for _, list := range []types.List{
		types.ListNull(vpcType),
		types.ListUnknown(vpcType),
		types.ListValueMust(vpcType, []attr.Value{}),
	} {
		vpc, ifaces, diags := vpcInterfaces(ctx, list)
		require.False(t, diags.HasError())
		require.Nil(t, vpc)
		require.Nil(t, ifaces)
	}

	list, diags := vpcList(ctx, types.StringValue("vpc"), []VPCInterfaceModel{iface("a", "auto", "10.0.0.1")})
	require.False(t, diags.HasError())
	vpc, ifaces, diags := vpcInterfaces(ctx, list)
	require.False(t, diags.HasError())
	require.Equal(t, "vpc", vpc.ID.ValueString())
	require.Equal(t, []string{"a"}, subnetIDs(ifaces))

	list, diags = vpcList(ctx, types.StringNull(), nil)
	require.False(t, diags.HasError())
	require.Empty(t, list.Elements())
}

// adoptionInstance is attached to a private subnet a, then to a subnet p
// unmanaged by default.
func adoptionInstance() *exoscale.Instance {
	return &exoscale.Instance{
		ID: exoscale.UUID("instance"),
		Vpc: &exoscale.InstanceVpc{
			ID: exoscale.UUID("vpc"),
			Subnets: []exoscale.InstanceVpcSubnets{
				{ID: exoscale.UUID("a"), Ipv4: net.ParseIP("10.0.0.22")},
				{ID: exoscale.UUID("p"), Ipv4: net.ParseIP("194.182.160.5")},
			},
		},
	}
}

func TestRemoteVPCInterfacesKeep(t *testing.T) {
	_, subnets := vpcInterfacesFromAPI(adoptionInstance(), func(string) bool { return false })
	require.Len(t, subnets, 1)

	_, subnets = vpcInterfacesFromAPI(adoptionInstance(), func(id string) bool { return id == "p" })
	require.Len(t, subnets, 2)
}

func TestVPCAdoption(t *testing.T) {
	ctx := context.Background()
	instance := adoptionInstance()
	vpc := &VPCModel{ID: types.StringValue("vpc")}
	a := iface("a", "10.0.0.22", "10.0.0.22")

	stateModel := func(ifaces ...VPCInterfaceModel) *ResourceModel {
		list, diags := vpcList(ctx, types.StringValue("vpc"), ifaces)
		require.False(t, diags.HasError())
		return &ResourceModel{VPC: list}
	}

	t.Run("not configured stays hidden", func(t *testing.T) {
		model := stateModel(a)
		require.False(t, refreshVPC(ctx, model, instance).HasError())
		_, ifaces, _ := vpcInterfaces(ctx, model.VPC)
		require.Equal(t, []string{"a"}, subnetIDs(ifaces))

		currentVPC, current := currentVPCInterfaces(instance, vpc, []VPCInterfaceModel{a}, []VPCInterfaceModel{a})
		kept, detach, attach := vpcChanges(currentVPC, current, vpc, []VPCInterfaceModel{a})
		require.Equal(t, 1, kept)
		require.Empty(t, detach)
		require.Empty(t, attach)
	})

	t.Run("adopt in place", func(t *testing.T) {
		state := stateModel(a)
		plan := stateModel(a, iface("p", "auto", ""))

		warning, diags := planVPC(ctx, plan, state, instance)
		require.False(t, diags.HasError())
		require.Empty(t, warning)

		_, planned, _ := vpcInterfaces(ctx, plan.VPC)
		require.Equal(t, "194.182.160.5", planned[1].IPv4AddressAllocated.ValueString())
	})

	t.Run("adopt with reorder", func(t *testing.T) {
		state := stateModel(a)
		plan := stateModel(iface("p", "auto", ""), iface("a", "10.0.0.22", ""))

		warning, diags := planVPC(ctx, plan, state, instance)
		require.False(t, diags.HasError())
		require.Contains(t, warning, "reattach subnet p as interfaces[0]")
		require.Contains(t, warning, "reattach subnet a as interfaces[1]")
	})

	t.Run("adopted stays managed", func(t *testing.T) {
		p := iface("p", "auto", "194.182.160.5")
		model := stateModel(a, p)
		require.False(t, refreshVPC(ctx, model, instance).HasError())
		_, ifaces, _ := vpcInterfaces(ctx, model.VPC)
		require.Equal(t, []string{"a", "p"}, subnetIDs(ifaces))
		require.Equal(t, "auto", ifaces[1].IPv4Address.ValueString())

		currentVPC, current := currentVPCInterfaces(instance, vpc, []VPCInterfaceModel{a, p}, []VPCInterfaceModel{a})
		_, detach, attach := vpcChanges(currentVPC, current, vpc, []VPCInterfaceModel{a})
		require.Equal(t, []string{"p"}, subnetIDs(detach))
		require.Empty(t, attach)
	})
}
