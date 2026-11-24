package instance_pool

import (
	"context"
	"testing"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func vpcModel(t *testing.T, id string, subnets ...string) *VPCModel {
	t.Helper()

	list, diags := types.ListValueFrom(context.Background(), types.StringType, subnets)
	require.False(t, diags.HasError())

	return &VPCModel{ID: types.StringValue(id), SubnetIDs: list}
}

func TestVPCChanges(t *testing.T) {
	vpc, other := vpcModel(t, "vpc"), vpcModel(t, "other")

	tests := []struct {
		name           string
		currentVPC     *VPCModel
		current        []string
		plannedVPC     *VPCModel
		planned        []string
		kept           int
		detach, attach []string
	}{
		{"unchanged", vpc, []string{"a", "b"}, vpc, []string{"a", "b"}, 2, []string{}, []string{}},
		{"swap", vpc, []string{"a", "b"}, vpc, []string{"b", "a"}, 0, []string{"a", "b"}, []string{"b", "a"}},
		{"insert in the middle", vpc, []string{"a", "b"}, vpc, []string{"a", "c", "b"}, 1, []string{"b"}, []string{"c", "b"}},
		{"remove the first", vpc, []string{"a", "b"}, vpc, []string{"b"}, 0, []string{"a", "b"}, []string{"b"}},
		{"remove the last", vpc, []string{"a", "b"}, vpc, []string{"a"}, 1, []string{"b"}, []string{}},
		{"append", vpc, []string{"a"}, vpc, []string{"a", "b"}, 1, []string{}, []string{"b"}},
		{"VPC change", vpc, []string{"a"}, other, []string{"a"}, 0, []string{"a"}, []string{"a"}},
		{"VPC removed", vpc, []string{"a", "b"}, nil, nil, 0, []string{"a", "b"}, []string{}},
		{"VPC added", nil, nil, vpc, []string{"a"}, 0, []string{}, []string{"a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kept, detach, attach := vpcChanges(tt.currentVPC, tt.current, tt.plannedVPC, tt.planned)
			require.Equal(t, tt.kept, kept)
			require.ElementsMatch(t, tt.detach, detach)
			require.Equal(t, len(tt.attach), len(attach))
			for i := range attach {
				require.Equal(t, tt.attach[i], attach[i])
			}
		})
	}
}

func TestDescribeVPCChanges(t *testing.T) {
	require.Equal(t, "", describeVPCChanges(nil, nil))

	got := describeVPCChanges([]string{"b"}, []string{"c", "b"})
	require.Contains(t, got, "attach subnet c")
	require.Contains(t, got, "reattach subnet b")
	require.NotContains(t, got, "detach subnet b")
}

func TestPlanVPC(t *testing.T) {
	ctx := context.Background()

	var state ResourceModel

	list, dg := vpcList(ctx, "vpc", []string{"a", "b"})
	require.False(t, dg.HasError())
	state.VPC = list

	list, dg = vpcList(ctx, "vpc", []string{"b"})
	require.False(t, dg.HasError())
	plan := ResourceModel{VPC: list}

	got, dg := planVPC(ctx, &plan, &state, nil)
	require.False(t, dg.HasError())
	require.Contains(t, got, "reattach subnet b")
	require.Contains(t, got, "detach subnet a")

	// Subnets yet to create are described at plan time, at their position.
	unknownSubnets, dg := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("a"), types.StringUnknown(), types.StringValue("b"),
	})
	require.False(t, dg.HasError())
	plan.VPC = types.ListValueMust(vpcType, []attr.Value{types.ObjectValueMust(VPCModel{}.Types(), map[string]attr.Value{
		"id":         types.StringValue("vpc"),
		"subnet_ids": unknownSubnets,
	})})
	got, dg = planVPC(ctx, &plan, &state, nil)
	require.False(t, dg.HasError())
	require.Equal(t, "The following VPC subnet attachments of the managed instances will be changed:\n"+
		"  - attach subnet (known after apply)\n"+
		"  - reattach subnet b", got)

	// An unknown VPC ID detaches everything.
	plan.VPC = types.ListValueMust(vpcType, []attr.Value{types.ObjectValueMust(VPCModel{}.Types(), map[string]attr.Value{
		"id":         types.StringUnknown(),
		"subnet_ids": unknownSubnets,
	})})
	got, dg = planVPC(ctx, &plan, &state, nil)
	require.False(t, dg.HasError())
	require.Contains(t, got, "reattach subnet a")
	require.Contains(t, got, "reattach subnet b")

	// Unknown plans produce no description.
	plan.VPC = types.ListUnknown(vpcType)
	got, dg = planVPC(ctx, &plan, &state, nil)
	require.False(t, dg.HasError())
	require.Empty(t, got)
}

func TestRequestsAndAPIRoundTrip(t *testing.T) {
	ctx := context.Background()

	list, dg := vpcList(ctx, "11111111-1111-1111-1111-111111111111", []string{
		"22222222-2222-2222-2222-222222222222",
		"33333333-3333-3333-3333-333333333333",
	})
	require.False(t, dg.HasError())
	model := &ResourceModel{VPC: list}

	create, dg := createRequestVPC(ctx, model)
	require.False(t, dg.HasError())
	require.Len(t, create.Subnets, 2)
	require.Equal(t, exoscale.UUID("33333333-3333-3333-3333-333333333333"), create.Subnets[1].ID)

	update, dg := updateRequestVPC(ctx, model, nil, nil)
	require.False(t, dg.HasError())
	require.Equal(t, exoscale.UUID("22222222-2222-2222-2222-222222222222"), update.Subnets[0].ID)

	back, dg := vpcFromAPI(ctx, &exoscale.InstancePoolVpc{
		ID: exoscale.UUID("11111111-1111-1111-1111-111111111111"),
		Subnets: []exoscale.InstancePoolVpcSubnets{
			{ID: update.Subnets[0].ID}, {ID: update.Subnets[1].ID},
		},
	}, []string{string(update.Subnets[0].ID), string(update.Subnets[1].ID)})
	require.False(t, dg.HasError())
	require.True(t, back.Equal(list))

	none, dg := createRequestVPC(ctx, &ResourceModel{VPC: noVPC()})
	require.False(t, dg.HasError())
	require.Nil(t, none)
	require.True(t, func() bool { l, _ := vpcFromAPI(ctx, nil, nil); return l.Equal(noVPC()) }())
}

func TestSplitVPCSubnets(t *testing.T) {
	const (
		priv = "00000000-0000-0000-0000-00000000000a"
		pub  = "00000000-0000-0000-0000-00000000000b"
		gone = "00000000-0000-0000-0000-00000000000c"
	)
	vpc := &exoscale.InstancePoolVpc{
		ID: "00000000-0000-0000-0000-000000000001",
		Subnets: []exoscale.InstancePoolVpcSubnets{
			{ID: priv}, {ID: pub}, {ID: gone},
		},
	}
	lookups := 0
	lookup := func(id string) (*exoscale.Subnet, error) {
		lookups++
		switch id {
		case priv:
			return &exoscale.Subnet{Ipv4Block: "10.0.0.0/24"}, nil
		case pub:
			return &exoscale.Subnet{Ipv4Block: "185.19.28.0/24"}, nil
		}
		return nil, exoscale.ErrNotFound
	}

	visible, hidden, err := splitVPCSubnets(vpc, containsAny(nil), lookup)
	require.NoError(t, err)
	require.Equal(t, []string{priv, gone}, visible)
	require.Equal(t, []string{pub}, hidden)
	require.Equal(t, 3, lookups)

	// Listed (managed) public subnets are kept, with no lookup.
	lookups = 0
	visible, hidden, err = splitVPCSubnets(vpc, containsAny([]string{pub, priv, gone}), lookup)
	require.NoError(t, err)
	require.Equal(t, []string{priv, pub, gone}, visible)
	require.Empty(t, hidden)
	require.Zero(t, lookups)

	// Hidden subnets are sent along with the update, after the planned ones.
	ctx := context.Background()
	list, _ := vpcList(ctx, string(vpc.ID), []string{priv})
	req, dg := updateRequestVPC(ctx, &ResourceModel{VPC: list}, vpc, []string{pub})
	require.False(t, dg.HasError())
	require.Len(t, req.Subnets, 2)
	require.Equal(t, exoscale.UUID(pub), req.Subnets[1].ID)

	// Without block, only the hidden ones remain; none means no VPC.
	req, _ = updateRequestVPC(ctx, &ResourceModel{VPC: noVPC()}, vpc, []string{pub})
	require.Len(t, req.Subnets, 1)
	req, _ = updateRequestVPC(ctx, &ResourceModel{VPC: noVPC()}, vpc, nil)
	require.Nil(t, req)

	// Moving to another VPC drops them.
	other, _ := vpcList(ctx, "00000000-0000-0000-0000-000000000002", []string{priv})
	req, _ = updateRequestVPC(ctx, &ResourceModel{VPC: other}, vpc, []string{pub})
	require.Len(t, req.Subnets, 1)
}
