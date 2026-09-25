package instance

import (
	"net"
	"testing"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/hashicorp/go-cty/cty"
)

func vif(vpcID, subnetID, ipv4 string) VPCInterface {
	return VPCInterface{VPCID: vpcID, SubnetID: subnetID, IPv4Address: &ipv4}
}

func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestVPCInterfaceIPv4(t *testing.T) {
	if ip := vif("v", "s", "").ipv4(); ip != nil {
		t.Fatalf("expected nil for an empty address, got %s", ip)
	}
	if ip := (VPCInterface{VPCID: "v", SubnetID: "s"}).ipv4(); ip != nil {
		t.Fatalf("expected nil for a nil address, got %s", ip)
	}
	if ip := vif("v", "s", "10.0.0.1").ipv4(); ip == nil || ip.String() != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1, got %v", ip)
	}
}

func TestInDeclaredOrder(t *testing.T) {
	cases := []struct {
		name  string
		order []string
		vifs  []VPCInterface
		want  []string
	}{
		{"empty", nil, nil, []string{}},
		{
			"declared order is followed",
			[]string{"subnet-2", "subnet-1"},
			[]VPCInterface{vif("vpc-a", "subnet-1", ""), vif("vpc-a", "subnet-2", "")},
			[]string{"subnet-2", "subnet-1"},
		},
		{
			"only the configured subset is ordered",
			[]string{"subnet-3", "subnet-2", "subnet-1"},
			[]VPCInterface{vif("vpc-a", "subnet-1", ""), vif("vpc-a", "subnet-3", "")},
			[]string{"subnet-3", "subnet-1"},
		},
		{
			"unknown subnets are appended, sorted",
			[]string{"subnet-3"},
			[]VPCInterface{vif("vpc-a", "subnet-2", ""), vif("vpc-a", "subnet-1", ""), vif("vpc-a", "subnet-3", "")},
			[]string{"subnet-3", "subnet-1", "subnet-2"},
		},
		{
			// Terraform sends no configuration during a plain refresh.
			"without an order the subnets are sorted",
			nil,
			[]VPCInterface{vif("vpc-a", "subnet-2", ""), vif("vpc-a", "subnet-1", "")},
			[]string{"subnet-1", "subnet-2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inDeclaredOrder(tc.order, tc.vifs)
			if len(got) != len(tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, subnetIDs(got))
			}
			for i := range got {
				if got[i].SubnetID != tc.want[i] {
					t.Fatalf("expected %v, got %v", tc.want, subnetIDs(got))
				}
			}
		})
	}
}

func TestVPCInterfacesFromConfig(t *testing.T) {
	elem := func(vpcID, subnetID string, ipv4 cty.Value) cty.Value {
		return cty.ObjectVal(map[string]cty.Value{
			"vpc_id":       cty.StringVal(vpcID),
			"subnet_id":    cty.StringVal(subnetID),
			"ipv4_address": ipv4,
		})
	}
	resource := func(vpcInterface cty.Value) cty.Value {
		return cty.ObjectVal(map[string]cty.Value{
			"name":           cty.StringVal("test"),
			AttrVPCInterface: vpcInterface,
		})
	}

	config := resource(cty.ListVal([]cty.Value{
		elem("vpc-a", "subnet-2", cty.NullVal(cty.String)),
		elem("vpc-a", "subnet-1", cty.StringVal("10.0.0.1")),
	}))
	vifs, err := vpcInterfacesFromConfig(config)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if got := subnetIDs(vifs); !equalIDs(got, []string{"subnet-2", "subnet-1"}) {
		t.Fatalf("expected the blocks in the order they are declared, got %v", got)
	}
	// An unset address is left to the platform, rather than read as a request.
	if vifs[0].ipv4() != nil {
		t.Fatalf("expected no address for subnet-2, got %s", vifs[0].ipv4())
	}
	if vifs[1].ipv4() == nil || vifs[1].ipv4().String() != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1 for subnet-1, got %v", vifs[1].ipv4())
	}

	// A Subnet that belongs to a resource which doesn't exist yet: we can't
	// attach to it, and Terraform never asks us to.
	unknown := resource(cty.TupleVal([]cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			"vpc_id":       cty.StringVal("vpc-a"),
			"subnet_id":    cty.UnknownVal(cty.String),
			"ipv4_address": cty.NullVal(cty.String),
		}),
	}))
	if _, err := vpcInterfacesFromConfig(unknown); err == nil {
		t.Fatal("expected an error for an unknown subnet_id, got none")
	}

	// No configuration at all: a plain refresh, or a destroy.
	for _, empty := range []cty.Value{
		cty.NullVal(cty.EmptyObject),
		cty.DynamicVal,
		resource(cty.NullVal(cty.List(cty.EmptyObject))),
	} {
		vifs, err := vpcInterfacesFromConfig(empty)
		if err != nil {
			t.Fatalf("expected no error, got %s", err)
		}
		if len(vifs) != 0 {
			t.Fatalf("expected no interfaces, got %v", subnetIDs(vifs))
		}
	}
}

func TestDiffVPCInterfaces(t *testing.T) {
	cases := []struct {
		name       string
		prior      []VPCInterface
		desired    []VPCInterface
		wantDetach []string
		wantAttach []string
	}{
		{"both empty", nil, nil, nil, nil},
		{
			"no change",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			nil, nil,
		},
		{
			"reordered blocks attach nothing",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-2", "10.1.0.1"), vif("vpc-a", "subnet-1", "10.0.0.1")},
			nil, nil,
		},
		{
			// prior comes from the instance itself, so an attachment made out of
			// band is already there and declaring it is not a change.
			"a Subnet attached out of band and then declared attaches nothing",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{
				vif("vpc-a", "subnet-1", "10.0.0.1"),
				{VPCID: "vpc-a", SubnetID: "subnet-2"},
			},
			nil, nil,
		},
		{
			"a Subnet attached out of band and declared with its address attaches nothing",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-2", "10.1.0.1"), vif("vpc-a", "subnet-1", "10.0.0.1")},
			nil, nil,
		},
		{
			"a block inserted first only attaches that Subnet",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{
				vif("vpc-a", "subnet-3", ""),
				vif("vpc-a", "subnet-1", "10.0.0.1"),
				vif("vpc-a", "subnet-2", "10.1.0.1"),
			},
			nil, []string{"subnet-3"},
		},
		{
			"an appended block only attaches that Subnet",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "")},
			nil, []string{"subnet-2"},
		},
		{
			"a removed block only detaches that Subnet",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]string{"subnet-1"}, nil,
		},
		{
			"a changed address reattaches its own Subnet",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.9"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]string{"subnet-1"}, []string{"subnet-1"},
		},
		{
			"an unset address is satisfied by the allocated one",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.231")},
			[]VPCInterface{{VPCID: "vpc-a", SubnetID: "subnet-1"}},
			nil, nil,
		},
		{
			// The attachments come out in the order the blocks are declared in.
			"everything at once",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{
				vif("vpc-a", "subnet-4", ""),
				vif("vpc-a", "subnet-2", "10.1.0.9"),
				vif("vpc-a", "subnet-3", ""),
			},
			[]string{"subnet-2", "subnet-1"},
			[]string{"subnet-4", "subnet-2", "subnet-3"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			detach, attach := diffVPCInterfaces(tc.prior, tc.desired)
			if got := subnetIDs(detach); !equalIDs(got, tc.wantDetach) {
				t.Fatalf("expected to detach %v, got %v", tc.wantDetach, got)
			}
			if got := subnetIDs(attach); !equalIDs(got, tc.wantAttach) {
				t.Fatalf("expected to attach %v, got %v", tc.wantAttach, got)
			}
		})
	}
}

// The VPC of an existing attachment is the one its Subnet belongs to, so a
// reattachment keeps it rather than trusting the configuration.
func TestDiffVPCInterfacesKeepsThePriorVPC(t *testing.T) {
	_, attach := diffVPCInterfaces(
		[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1")},
		[]VPCInterface{vif("vpc-b", "subnet-1", "10.0.0.9")},
	)
	if len(attach) != 1 || attach[0].VPCID != "vpc-a" {
		t.Fatalf("expected the prior VPC to be kept, got %v", attach)
	}
}

func TestVPCInterfacesFromInstance(t *testing.T) {
	if vifs, err := vpcInterfacesFromInstance(nil); err != nil || vifs != nil {
		t.Fatalf("got %v, %v for an instance we have nothing for", vifs, err)
	}

	instance := &v3.Instance{Vpc: &v3.InstanceVpc{
		ID: v3.UUID("vpc-a"),
		Subnets: []v3.InstanceVpcSubnets{
			{ID: v3.UUID("subnet-1"), Ipv4: net.ParseIP("10.0.0.1")},
			// a public address is not a Subnet attachment we manage
			{ID: v3.UUID("subnet-2"), Ipv4: net.ParseIP("194.182.160.1")},
			{ID: v3.UUID("subnet-3"), Ipv4: net.ParseIP("10.1.0.1")},
		},
	}}

	vifs, err := vpcInterfacesFromInstance(instance)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"subnet-1", "subnet-3"}; !equalIDs(subnetIDs(vifs), want) {
		t.Errorf("got %v, want %v", subnetIDs(vifs), want)
	}
	for i, want := range []string{"10.0.0.1", "10.1.0.1"} {
		if vifs[i].IPv4Address == nil || *vifs[i].IPv4Address != want {
			t.Errorf("address %d: got %v, want %s", i, vifs[i].IPv4Address, want)
		}
	}
	if vifs[0].VPCID != "vpc-a" {
		t.Errorf("got VPC %q, want vpc-a", vifs[0].VPCID)
	}

	instance.Vpc.Subnets = []v3.InstanceVpcSubnets{{ID: v3.UUID("subnet-1"), Name: "no address"}}
	if _, err := vpcInterfacesFromInstance(instance); err == nil {
		t.Error("an attachment without an address should be an error")
	}
}
