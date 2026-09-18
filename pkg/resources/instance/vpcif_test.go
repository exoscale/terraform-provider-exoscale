package instance

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// unknownConfigValue is the sentinel the SDK shims an unknown value to in the
// legacy ResourceConfig, copied from
// vendor/github.com/hashicorp/terraform-plugin-sdk/v2/internal/configs/hcl2shim,
// which is internal and cannot be imported.
const unknownConfigValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

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

func subnetIDs(vifs []VPCInterface) []string {
	ids := make([]string, len(vifs))
	for i, vif := range vifs {
		ids[i] = vif.SubnetID
	}

	return ids
}

// vpcConfig builds the cty value of a vpc block: one VPC and its interfaces, in
// the order they are declared.
func vpcConfig(vpcID string, interfaces ...cty.Value) cty.Value {
	block := map[string]cty.Value{vpcIDAttr: cty.StringVal(vpcID)}
	if len(interfaces) == 0 {
		block[vpcInterfacesAttr] = cty.ListValEmpty(cty.Object(map[string]cty.Type{
			vpcInterfaceSubnetIDAttr: cty.String, vpcInterfaceIPv4Attr: cty.String,
		}))
	} else {
		block[vpcInterfacesAttr] = cty.ListVal(interfaces)
	}

	return cty.ListVal([]cty.Value{cty.ObjectVal(block)})
}

// vpcInterfaceConfig builds the cty value of one vpc.interface block.
func vpcInterfaceConfig(subnetID string, ipv4 cty.Value) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		vpcInterfaceSubnetIDAttr: cty.StringVal(subnetID),
		vpcInterfaceIPv4Attr:     ipv4,
	})
}

// vpcStateKey returns the flatmap key of an attribute of the vpc block.
func vpcStateKey(attr string) string {
	return AttrVPC + ".0." + attr
}

// vpcState builds the attributes Terraform stores for a vpc block.
func vpcState(vpcID string, interfaces ...VPCInterface) map[string]string {
	attrs := map[string]string{
		"vpc.#":                               "1",
		"vpc.0." + vpcIDAttr:                  vpcID,
		vpcStateKey(vpcInterfacesAttr) + ".#": strconv.Itoa(len(interfaces)),
		"vpc.0.interface_changes.#":           "0",
	}
	for i, vif := range interfaces {
		attrs[fmt.Sprintf("%s.%d.%s", vpcStateKey(vpcInterfacesAttr), i, vpcInterfaceSubnetIDAttr)] = vif.SubnetID
		if vif.IPv4Address != nil {
			attrs[fmt.Sprintf("%s.%d.%s", vpcStateKey(vpcInterfacesAttr), i, vpcInterfaceIPv4Attr)] = *vif.IPv4Address
		}
	}

	return attrs
}

func vpcInterfaceRawConfig(vpcID string, interfaces ...map[string]any) map[string]any {
	blocks := make([]any, 0, len(interfaces))
	for _, vif := range interfaces {
		blocks = append(blocks, vif)
	}

	return map[string]any{AttrVPC: []any{map[string]any{
		vpcIDAttr: vpcID, vpcInterfacesAttr: blocks,
	}}}
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

func TestVPCInterfacesFromConfig(t *testing.T) {
	resource := func(vpc cty.Value) cty.Value {
		return cty.ObjectVal(map[string]cty.Value{
			"name":  cty.StringVal("test"),
			AttrVPC: vpc,
		})
	}

	config := resource(vpcConfig("vpc-a",
		vpcInterfaceConfig("subnet-2", cty.NullVal(cty.String)),
		vpcInterfaceConfig("subnet-1", cty.StringVal("10.0.0.1")),
		vpcInterfaceConfig("subnet-3", cty.StringVal(vpcInterfaceIPv4Auto)),
	))
	vifs, err := vpcInterfacesFromConfig(config)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if got := subnetIDs(vifs); !equalIDs(got, []string{"subnet-2", "subnet-1", "subnet-3"}) {
		t.Fatalf("expected the blocks in the order they are declared, got %v", got)
	}
	// The VPC is declared once, and belongs to every interface.
	for _, vif := range vifs {
		if vif.VPCID != "vpc-a" {
			t.Fatalf("expected vpc-a, got %q", vif.VPCID)
		}
	}
	// An unset address is left to the platform to allocate
	if vifs[0].ipv4() != nil {
		t.Fatalf("expected no address for subnet-2, got %s", vifs[0].ipv4())
	}
	if vifs[1].ipv4() == nil || vifs[1].ipv4().String() != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1 for subnet-1, got %v", vifs[1].ipv4())
	}
	// An "auto" address is left to the platform to allocate
	if vifs[2].IPv4Address != nil {
		t.Fatalf("expected no address for subnet-3, got %q", *vifs[2].IPv4Address)
	}

	// A Subnet that belongs to a resource which doesn't exist yet: we can't
	// attach to it, and Terraform never asks us to.
	unknown := resource(cty.TupleVal([]cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			vpcIDAttr: cty.StringVal("vpc-a"),
			vpcInterfacesAttr: cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					vpcInterfaceSubnetIDAttr: cty.UnknownVal(cty.String),
					vpcInterfaceIPv4Attr:     cty.NullVal(cty.String),
				}),
			}),
		}),
	}))
	if _, err := vpcInterfacesFromConfig(unknown); err == nil {
		t.Fatal("expected an error for an unknown subnet_id, got none")
	}

	// A VPC that doesn't exist yet either.
	unknownVPC := resource(cty.TupleVal([]cty.Value{
		cty.ObjectVal(map[string]cty.Value{
			vpcIDAttr:         cty.UnknownVal(cty.String),
			vpcInterfacesAttr: cty.ListValEmpty(cty.EmptyObject),
		}),
	}))
	if _, err := vpcInterfacesFromConfig(unknownVPC); err == nil {
		t.Fatal("expected an error for an unknown VPC id, got none")
	}

	// No configuration at all: a plain refresh, a destroy, or an instance that is
	// attached to no VPC.
	for _, empty := range []cty.Value{
		cty.NullVal(cty.EmptyObject),
		cty.DynamicVal,
		resource(cty.NullVal(cty.List(cty.EmptyObject))),
		resource(cty.ListValEmpty(cty.EmptyObject)),
		resource(vpcConfig("vpc-a")),
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

// The state we write, the configuration we read and the schema have to agree on every key.
func TestVPCBlockMatchesTheSchema(t *testing.T) {
	vpc := Resource().Schema[AttrVPC].Elem.(*schema.Resource).Schema
	block, ok := vpcBlock([]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1")})[0].(map[string]any)
	if !ok {
		t.Fatal("vpcBlock does not write a block")
	}

	compare := func(what string, written map[string]any, sch map[string]*schema.Schema) {
		for key := range written {
			if _, ok := sch[key]; !ok {
				t.Errorf("%s writes %q, which its schema does not have", what, key)
			}
		}
		for key := range sch {
			if _, ok := written[key]; !ok {
				t.Errorf("the %s schema has %q, which is never written", what, key)
			}
		}
	}

	compare(AttrVPC, block, vpc)

	interfaces, ok := block[vpcInterfacesAttr].([]any)
	if !ok || len(interfaces) == 0 {
		t.Fatalf("expected interfaces under %q, got %v", vpcInterfacesAttr, block[vpcInterfacesAttr])
	}
	compare(AttrVPC+"."+vpcInterfacesAttr, interfaces[0].(map[string]any),
		vpc[vpcInterfacesAttr].Elem.(*schema.Resource).Schema)
}

// The vpc block Terraform stores decodes back into the attachments it was built
// from: the VPC is held once, and the interfaces keep their order.
func TestVPCBlockRoundTrip(t *testing.T) {
	vifs := []VPCInterface{
		vif("vpc-a", "subnet-2", "10.1.0.1"),
		{VPCID: "vpc-a", SubnetID: "subnet-1"},
	}

	block := vpcBlock(vifs)
	if len(block) != 1 {
		t.Fatalf("expected one vpc block, got %v", block)
	}

	got, err := vpcInterfacesFromState(block)
	if err != nil {
		t.Fatal(err)
	}
	if ids := subnetIDs(got); !equalIDs(ids, []string{"subnet-2", "subnet-1"}) {
		t.Fatalf("expected the stored order, got %v", ids)
	}
	if got[0].VPCID != "vpc-a" || got[1].VPCID != "vpc-a" {
		t.Fatalf("expected both interfaces in vpc-a, got %+v", got)
	}
	if got[0].IPv4Address == nil || *got[0].IPv4Address != "10.1.0.1" {
		t.Fatalf("expected 10.1.0.1 for subnet-2, got %v", got[0].IPv4Address)
	}
	if got[1].IPv4Address != nil {
		t.Fatalf("expected no address for subnet-1, got %q", *got[1].IPv4Address)
	}

	// An instance attached to no VPC holds no block at all.
	if block := vpcBlock(nil); len(block) != 0 {
		t.Fatalf("expected no vpc block, got %v", block)
	}
	if vifs, err := vpcInterfacesFromState(nil); err != nil || vifs != nil {
		t.Fatalf("got %v, %v for an instance with no vpc block", vifs, err)
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
			"an appended block only attaches that Subnet",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "")},
			nil, []string{"subnet-2"},
		},
		{
			"removing the last block only detaches that Subnet",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1")},
			[]string{"subnet-2"}, nil,
		},
		{
			"removing the first block reattaches the ones after it",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]string{"subnet-1", "subnet-2"}, []string{"subnet-2"},
		},
		{
			"swapped blocks are reattached in the declared order",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{vif("vpc-a", "subnet-2", "10.1.0.1"), vif("vpc-a", "subnet-1", "10.0.0.1")},
			[]string{"subnet-1", "subnet-2"}, []string{"subnet-2", "subnet-1"},
		},
		{
			"a block inserted first reattaches everything",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{
				vif("vpc-a", "subnet-3", ""),
				vif("vpc-a", "subnet-1", "10.0.0.1"),
				vif("vpc-a", "subnet-2", "10.1.0.1"),
			},
			[]string{"subnet-1", "subnet-2"},
			[]string{"subnet-3", "subnet-1", "subnet-2"},
		},
		{
			"a changed address reattaches its block and the ones after it",
			[]VPCInterface{
				vif("vpc-a", "subnet-1", "10.0.0.1"),
				vif("vpc-a", "subnet-2", "10.1.0.1"),
				vif("vpc-a", "subnet-3", "10.2.0.1"),
			},
			[]VPCInterface{
				vif("vpc-a", "subnet-1", "10.0.0.1"),
				vif("vpc-a", "subnet-2", "10.1.0.9"),
				vif("vpc-a", "subnet-3", "10.2.0.1"),
			},
			[]string{"subnet-2", "subnet-3"}, []string{"subnet-2", "subnet-3"},
		},
		{
			"an unset address is satisfied by the allocated one",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.231")},
			[]VPCInterface{{VPCID: "vpc-a", SubnetID: "subnet-1"}},
			nil, nil,
		},
		{
			"a Subnet attached out of band and declared last attaches nothing",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{
				vif("vpc-a", "subnet-1", "10.0.0.1"),
				{VPCID: "vpc-a", SubnetID: "subnet-2"},
			},
			nil, nil,
		},
		{
			// The price of the strict ordering guarantee: the same out of band
			// attachment declared first has to be made again, in that order.
			"a Subnet attached out of band and declared first reattaches everything",
			[]VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1"), vif("vpc-a", "subnet-2", "10.1.0.1")},
			[]VPCInterface{
				{VPCID: "vpc-a", SubnetID: "subnet-2"},
				vif("vpc-a", "subnet-1", "10.0.0.1"),
			},
			[]string{"subnet-1", "subnet-2"}, []string{"subnet-2", "subnet-1"},
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

// Because ipv4_address is Optional + Computed, the SDK fills it from the prior
// value at the same index: the new value of a block that declares no address
// carries the address of the block that used to sit there, which belongs to
// another Subnet. Attaching with it would ask the platform to pin an address
// from the wrong Subnet, so rUpdate has to read the desired interfaces from the
// raw configuration rather than from the diff.
func TestVPCInterfaceDiffCarriesTheComputedAddressForward(t *testing.T) {
	sch := map[string]*schema.Schema{AttrVPC: Resource().Schema[AttrVPC]}
	state := &terraform.InstanceState{
		ID:         "i",
		Attributes: vpcState("vpc-a", vif("vpc-a", "subnet-1", "10.0.0.1")),
	}

	// subnet-2 is declared before the Subnet that is already attached, and
	// leaves its address to the platform.
	config := terraform.NewResourceConfigRaw(vpcInterfaceRawConfig("vpc-a",
		map[string]any{"subnet_id": "subnet-2"},
		map[string]any{"subnet_id": "subnet-1", "ipv4_address": "10.0.0.1"},
	))

	diff, err := schema.InternalMap(sch).Diff(context.Background(), state, config, nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil {
		t.Fatal("expected a diff for the inserted block, got none")
	}

	merged := state.MergeDiff(diff).Attributes
	if got := merged[vpcStateKey(vpcInterfacesAttr)+".0."+vpcInterfaceSubnetIDAttr]; got != "subnet-2" {
		t.Fatalf("expected subnet-2 at index 0, got %q", got)
	}
	if got := merged[vpcStateKey(vpcInterfacesAttr)+".0."+vpcInterfaceIPv4Attr]; got != "10.0.0.1" {
		t.Fatalf("expected the prior address to be carried forward, got %q", got)
	}

	// The raw configuration says what was written: no address for subnet-2.
	vifs, err := vpcInterfacesFromConfig(cty.ObjectVal(map[string]cty.Value{
		AttrVPC: vpcConfig("vpc-a",
			vpcInterfaceConfig("subnet-2", cty.NullVal(cty.String)),
			vpcInterfaceConfig("subnet-1", cty.StringVal("10.0.0.1")),
		),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if vifs[0].ipv4() != nil {
		t.Fatalf("expected no address for subnet-2, got %s", vifs[0].ipv4())
	}
}

// rawResourceConfig builds the value Terraform sends for a resource: the given
// attributes, and a null for every other one.
func rawResourceConfig(resource *schema.Resource, attrs map[string]cty.Value) cty.Value {
	ty := schema.InternalMap(resource.Schema).CoreConfigSchema().ImpliedType()

	vals := make(map[string]cty.Value, len(ty.AttributeTypes()))
	for name, attrType := range ty.AttributeTypes() {
		if val, ok := attrs[name]; ok {
			vals[name] = val
			continue
		}

		vals[name] = cty.NullVal(attrType)
	}

	return cty.ObjectVal(vals)
}

func TestCustomizeDiffVPCInterfaces(t *testing.T) {
	resource := Resource()
	state := &terraform.InstanceState{
		ID: "i",
		Attributes: vpcState("vpc-a",
			vif("vpc-a", "subnet-1", "10.0.0.1"),
			vif("vpc-a", "subnet-2", "10.1.0.1"),
		),
	}

	// The two blocks are swapped, and subnet-2 does not pin its address.
	// The raw configuration reaches CustomizeDiff through the state, which is
	// where Terraform puts it while planning.
	state.RawConfig = rawResourceConfig(resource, map[string]cty.Value{
		AttrVPC: vpcConfig("vpc-a",
			vpcInterfaceConfig("subnet-2", cty.NullVal(cty.String)),
			vpcInterfaceConfig("subnet-1", cty.StringVal("10.0.0.1")),
		),
	})
	config := terraform.NewResourceConfigRaw(vpcInterfaceRawConfig("vpc-a",
		map[string]any{"subnet_id": "subnet-2"},
		map[string]any{"subnet_id": "subnet-1", "ipv4_address": "10.0.0.1"},
	))

	diff, err := schema.InternalMap(resource.Schema).Diff(
		context.Background(), state, config, resource.CustomizeDiff, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil {
		t.Fatal("expected a diff for the swapped blocks, got none")
	}

	got := []string{}
	for i := 0; ; i++ {
		attr, ok := diff.Attributes[fmt.Sprintf("%s.0.%s.%d", AttrVPC, vpcInterfaceChangesAttr, i)]
		if !ok {
			break
		}
		// Against the empty value a read leaves behind, each sentence is an
		// addition: the plan shows what this change does, and not the previous
		// description alongside it.
		if attr.Old != "" {
			t.Errorf("expected no previous description at index %d, got %q", i, attr.Old)
		}

		got = append(got, attr.New)
	}
	want := []string{"reattaching subnet-2, subnet-1"}
	if !equalIDs(got, want) {
		t.Fatalf("expected the plan to say %q, got %q", want, got)
	}
}

// The vpc block is Computed, because only the provider can fill
// interface_changes, and the SDK keeps a Computed block that the configuration
// drops. Planning that removal is therefore up to CustomizeDiff - without it,
// deleting the block would detach nothing.
func TestCustomizeDiffVPCInterfacesPlansTheRemoval(t *testing.T) {
	resource := Resource()
	state := &terraform.InstanceState{
		ID: "i",
		Attributes: vpcState("vpc-a",
			vif("vpc-a", "subnet-1", "10.0.0.1"),
			vif("vpc-a", "subnet-2", "10.1.0.1"),
		),
	}
	// A configuration with no vpc block at all.
	state.RawConfig = rawResourceConfig(resource, nil)

	diff, err := schema.InternalMap(resource.Schema).Diff(
		context.Background(), state, terraform.NewResourceConfigRaw(map[string]any{}),
		resource.CustomizeDiff, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil {
		t.Fatal("expected the removal to be planned, got no diff")
	}

	for key, want := range map[string]string{
		"vpc.#":                               "0",
		vpcStateKey(vpcInterfacesAttr) + ".#": "0",
	} {
		attr, ok := diff.Attributes[key]
		if !ok {
			t.Errorf("expected %s in the plan, got nothing", key)
			continue
		}
		if attr.New != want {
			t.Errorf("expected %s to become %q, got %q", key, want, attr.New)
		}
	}
}

// If the user deletes the whole VPC block, this is invisible in the terraform diff.
// The first half of the test verifies this assumption(perhaps unnecessarily).
// The second half verifies that our custom diff picks this change up and
// returns all attachments for detachment.
func TestVPCRemovalIsInvisibleToTheApplyDiff(t *testing.T) {
	resource := Resource()
	ty := resource.CoreConfigSchema().ImpliedType()

	resourceValue := func(vpc cty.Value) cty.Value {
		vals := make(map[string]cty.Value, len(ty.AttributeTypes()))
		for name, attrType := range ty.AttributeTypes() {
			vals[name] = cty.NullVal(attrType)
		}
		vals["id"] = cty.StringVal("i")
		vals[AttrVPC] = vpc

		return cty.ObjectVal(vals)
	}

	prior := []VPCInterface{
		vif("vpc-a", "subnet-1", "10.0.0.1"),
		vif("vpc-a", "subnet-2", "10.1.0.1"),
	}
	attached := resourceValue(cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
		vpcIDAttr:               cty.StringVal("vpc-a"),
		vpcInterfaceChangesAttr: cty.ListValEmpty(cty.String),
		vpcInterfacesAttr: cty.ListVal([]cty.Value{
			vpcInterfaceConfig("subnet-1", cty.StringVal("10.0.0.1")),
			vpcInterfaceConfig("subnet-2", cty.StringVal("10.1.0.1")),
		}),
	})}))
	// A configuration, and so a planned value, with no vpc block at all.
	detached := resourceValue(cty.ListValEmpty(ty.AttributeType(AttrVPC).ElementType()))

	diff, err := schema.DiffFromValues(context.Background(), attached, detached, detached, resource)
	if err != nil {
		t.Fatal(err)
	}

	state, err := resource.ShimInstanceStateFromValue(attached)
	if err != nil {
		t.Fatal(err)
	}
	data, err := schema.InternalMap(resource.Schema).Data(state, diff)
	if err != nil {
		t.Fatal(err)
	}

	if data.HasChange(AttrVPC) {
		t.Errorf("the apply diff now reports the removal of %s:"+
			" rUpdate may go back to gating the detach on d.HasChange", AttrVPC)
	}

	// What rUpdate reads instead says the attachments are all going away.
	desired, err := vpcInterfacesFromConfig(detached)
	if err != nil {
		t.Fatal(err)
	}
	detach, attach := diffVPCInterfaces(prior, desired)
	if !equalIDs(subnetIDs(detach), []string{"subnet-1", "subnet-2"}) {
		t.Errorf("expected both Subnets to be detached, got %v", subnetIDs(detach))
	}
	if len(attach) != 0 {
		t.Errorf("expected nothing to be attached, got %v", subnetIDs(attach))
	}
}

// Every read clears `interface_changes`. This ensures that we don't display
// outdated plans to the user.
func TestCustomizeDiffVPCInterfacesClearsALeftoverDescription(t *testing.T) {
	resource := Resource()
	state := &terraform.InstanceState{
		ID:         "i",
		Attributes: vpcState("vpc-a", vif("vpc-a", "subnet-1", "10.0.0.1")),
	}
	state.Attributes["vpc.0.interface_changes.#"] = "1"
	state.Attributes["vpc.0.interface_changes.0"] = "attaching subnet-1"
	state.RawConfig = rawResourceConfig(resource, map[string]cty.Value{
		AttrVPC: vpcConfig("vpc-a", vpcInterfaceConfig("subnet-1", cty.StringVal("10.0.0.1"))),
	})

	diff, err := schema.InternalMap(resource.Schema).Diff(
		context.Background(), state,
		terraform.NewResourceConfigRaw(vpcInterfaceRawConfig("vpc-a",
			map[string]any{"subnet_id": "subnet-1", "ipv4_address": "10.0.0.1"})),
		resource.CustomizeDiff, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	key := AttrVPC + ".0." + vpcInterfaceChangesAttr + ".#"
	attr, ok := diff.Attributes[key]
	if !ok || attr.New != "0" {
		t.Fatalf("expected %s to be cleared, got %v", key, attr)
	}
}

// A Computed attribute that no apply ever writes is planned as "known after
// apply" again on every plan, a difference that never settles. The plan must
// therefore carry a concrete value, and leave it alone when nothing about the
// attachments changes.
func TestCustomizeDiffVPCInterfacesSettles(t *testing.T) {
	resource := Resource()
	attached := vif("vpc-a", "subnet-1", "10.0.0.1")
	config := vpcInterfaceRawConfig("vpc-a",
		map[string]any{"subnet_id": "subnet-1", "ipv4_address": "10.0.0.1"})

	cases := []struct {
		name  string
		state map[string]string
	}{
		// An instance created before the attribute existed holds no value for it.
		{"no value in state", map[string]string{"vpc.0.interface_changes.#": ""}},
		{"the empty value a read leaves behind", map[string]string{
			"vpc.0.interface_changes.#": "0",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := &terraform.InstanceState{ID: "i", Attributes: vpcState("vpc-a", attached)}
			for k, v := range tc.state {
				state.Attributes[k] = v
			}
			state.RawConfig = rawResourceConfig(resource, map[string]cty.Value{
				AttrVPC: vpcConfig("vpc-a",
					vpcInterfaceConfig("subnet-1", cty.StringVal("10.0.0.1"))),
			})

			diff, err := schema.InternalMap(resource.Schema).Diff(
				context.Background(), state, terraform.NewResourceConfigRaw(config),
				resource.CustomizeDiff, nil, true)
			if err != nil {
				t.Fatal(err)
			}

			// Nothing about the attachments changes, so the attribute must not
			// appear in the plan at all - and above all not as a value that is
			// only known after apply.
			prefix := AttrVPC + ".0." + vpcInterfaceChangesAttr
			for key, attr := range diff.Attributes {
				if strings.HasPrefix(key, prefix) {
					t.Errorf("expected no plan for %s, got %q -> %q (computed: %v)",
						key, attr.Old, attr.New, attr.NewComputed)
				}
			}
		})
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

// The auto keyword asks for any address, so the one the platform already
// allocated satisfies it: it must not diff against the stored address, and it
// must not be read as a request that reattaches the interface.
func TestVPCInterfaceAutoKeepsTheAllocatedAddress(t *testing.T) {
	resource := Resource()
	state := &terraform.InstanceState{
		ID:         "i",
		Attributes: vpcState("vpc-a", vif("vpc-a", "subnet-1", "10.0.0.1")),
	}
	state.RawConfig = rawResourceConfig(resource, map[string]cty.Value{
		AttrVPC: vpcConfig("vpc-a",
			vpcInterfaceConfig("subnet-1", cty.StringVal(vpcInterfaceIPv4Auto))),
	})

	diff, err := schema.InternalMap(resource.Schema).Diff(
		context.Background(), state,
		terraform.NewResourceConfigRaw(vpcInterfaceRawConfig("vpc-a", map[string]any{
			vpcInterfaceSubnetIDAttr: "subnet-1",
			vpcInterfaceIPv4Attr:     vpcInterfaceIPv4Auto,
		})),
		resource.CustomizeDiff, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	address := fmt.Sprintf("%s.0.%s", vpcStateKey(vpcInterfacesAttr), vpcInterfaceIPv4Attr)
	if diff != nil {
		if attr, ok := diff.Attributes[address]; ok && attr.Old != attr.New {
			t.Errorf("expected %s to be left alone, got %q -> %q", address, attr.Old, attr.New)
		}
		for key, attr := range diff.Attributes {
			if strings.HasPrefix(key, vpcStateKey(vpcInterfaceChangesAttr)) && attr.New != "0" {
				t.Errorf("expected no described change, got %s = %q", key, attr.New)
			}
		}
	}

	// And the keyword is not a pinned address, so nothing is reattached over it.
	prior := []VPCInterface{vif("vpc-a", "subnet-1", "10.0.0.1")}
	desired, err := vpcInterfacesFromConfig(state.RawConfig)
	if err != nil {
		t.Fatal(err)
	}
	if desired[0].IPv4Address != nil {
		t.Errorf("expected %q to decode to no address, got %q",
			vpcInterfaceIPv4Auto, *desired[0].IPv4Address)
	}
	if detach, attach := diffVPCInterfaces(prior, desired); len(detach) != 0 || len(attach) != 0 {
		t.Errorf("expected nothing to move, got detach %v attach %v",
			subnetIDs(detach), subnetIDs(attach))
	}
}

func TestValidateVPCInterfaceIPv4(t *testing.T) {
	for _, valid := range []any{vpcInterfaceIPv4Auto, "10.0.0.1", "192.168.1.254"} {
		if diags := validateVPCInterfaceIPv4(valid, nil); diags.HasError() {
			t.Errorf("expected %v to be valid, got %s", valid, diags[0].Detail)
		}
	}

	for _, invalid := range []any{"", "dhcp", "10.0.0.256", "fd00::1", "10.0.0.1/24", 42} {
		diags := validateVPCInterfaceIPv4(invalid, nil)
		if !diags.HasError() {
			t.Errorf("expected %v to be rejected, got no error", invalid)
			continue
		}
		if !strings.Contains(diags[0].Detail, vpcInterfaceIPv4Auto) {
			t.Errorf("expected the error for %v to name %q, got %q",
				invalid, vpcInterfaceIPv4Auto, diags[0].Detail)
		}
	}
}

// A VPC, its Subnets and the instance created by the same apply are unknown
// while the plan is made, and known by the time Terraform expands that plan
// during the apply. Both runs of CustomizeDiff have to agree on the address of
// an interface that asks for auto.
func TestCustomizeDiffVPCInterfacesPlansUnknownSubnetsAsUnknown(t *testing.T) {
	resource := Resource()
	// Nothing in state: the instance is created by this apply.
	state := &terraform.InstanceState{}
	state.RawConfig = rawResourceConfig(resource, map[string]cty.Value{
		AttrVPC: cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			vpcIDAttr: cty.UnknownVal(cty.String),
			vpcInterfacesAttr: cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					vpcInterfaceSubnetIDAttr: cty.UnknownVal(cty.String),
					vpcInterfaceIPv4Attr:     cty.StringVal("10.0.0.22"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					vpcInterfaceSubnetIDAttr: cty.UnknownVal(cty.String),
					vpcInterfaceIPv4Attr:     cty.StringVal(vpcInterfaceIPv4Auto),
				}),
			}),
		})}),
	})
	config := terraform.NewResourceConfigRaw(vpcInterfaceRawConfig(
		unknownConfigValue,
		map[string]any{
			vpcInterfaceSubnetIDAttr: unknownConfigValue,
			vpcInterfaceIPv4Attr:     "10.0.0.22",
		},
		map[string]any{
			vpcInterfaceSubnetIDAttr: unknownConfigValue,
			vpcInterfaceIPv4Attr:     vpcInterfaceIPv4Auto,
		},
	))

	diff, err := schema.InternalMap(resource.Schema).Diff(
		context.Background(), state, config, resource.CustomizeDiff, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil {
		t.Fatal("expected a diff for the created attachments, got none")
	}

	attr, ok := diff.Attributes["vpc.#"]
	if !ok || !attr.NewComputed {
		t.Fatalf("expected the vpc block to be planned unknown, got %#v", attr)
	}

	// The keyword must not survive as the planned address: the apply resolves it
	// to whatever the platform allocates, which no plan can name.
	key := fmt.Sprintf("%s.1.%s", vpcStateKey(vpcInterfacesAttr), vpcInterfaceIPv4Attr)
	if planned, ok := diff.Attributes[key]; ok && !planned.NewComputed {
		t.Fatalf("expected %s to be unknown, got %q", key, planned.New)
	}
}

// An unknown vpc block is a block the configuration declares, not one it drops.
// Planning it as a removal would detach every Subnet the apply then attaches
// again, so this pins the block down as unknown - which is what the SDK does
// with it on its own, and what customizeDiffVPCInterfaces must not undo.
func TestCustomizeDiffVPCInterfacesKeepsAnUnknownBlock(t *testing.T) {
	resource := Resource()
	state := &terraform.InstanceState{
		ID: "i",
		Attributes: vpcState("vpc-a",
			vif("vpc-a", "subnet-1", "10.0.0.1"),
			vif("vpc-a", "subnet-2", "10.1.0.1"),
		),
	}
	blockType := schema.InternalMap(resource.Schema).
		CoreConfigSchema().ImpliedType().AttributeType(AttrVPC)
	state.RawConfig = rawResourceConfig(resource, map[string]cty.Value{
		AttrVPC: cty.UnknownVal(blockType),
	})
	config := terraform.NewResourceConfigRaw(
		map[string]any{AttrVPC: unknownConfigValue})

	diff, err := schema.InternalMap(resource.Schema).Diff(
		context.Background(), state, config, resource.CustomizeDiff, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil {
		t.Fatal("expected the unknown block to be planned, got no diff")
	}

	attr, ok := diff.Attributes["vpc.#"]
	if !ok {
		t.Fatal("expected the vpc block in the plan, got nothing")
	}
	if !attr.NewComputed {
		t.Fatalf("expected the vpc block to be planned unknown, got %q", attr.New)
	}
}
