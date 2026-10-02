package instance

import (
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/go-cty/cty"
)

type VPCInterface struct {
	VPCID       string
	SubnetID    string
	IPv4Address *string
}

// ipv4 returns the configured IPv4 address, or nil when none was requested
func (v VPCInterface) ipv4() net.IP {
	if v.IPv4Address == nil || *v.IPv4Address == "" {
		return nil
	}

	return net.ParseIP(*v.IPv4Address)
}

// toMap returns the interface as a vpc.interface block.
func (v VPCInterface) toMap() map[string]any {
	ipv4 := ""
	if v.IPv4Address != nil {
		ipv4 = *v.IPv4Address
	}

	return map[string]any{
		vpcInterfaceSubnetIDAttr: v.SubnetID,
		vpcInterfaceIPv4Attr:     ipv4,
	}
}

// vpcInterfacesFromState decodes the vpc block, in the order the interfaces are stored in.
func vpcInterfacesFromState(raw []any) ([]VPCInterface, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	vpc, ok := raw[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected %s value %T", AttrVPC, raw[0])
	}

	vpcID, _ := vpc[vpcIDAttr].(string)
	interfaces, _ := vpc[vpcInterfacesAttr].([]any)

	vifs := make([]VPCInterface, 0, len(interfaces))
	for _, item := range interfaces {
		vif, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unexpected %s.%s value %T", AttrVPC, vpcInterfacesAttr, item)
		}

		subnetID, _ := vif[vpcInterfaceSubnetIDAttr].(string)
		if subnetID == "" {
			return nil, fmt.Errorf("an %s.%s element has no %s",
				AttrVPC, vpcInterfacesAttr, vpcInterfaceSubnetIDAttr)
		}

		ipv4, _ := vif[vpcInterfaceIPv4Attr].(string)
		vifs = append(vifs, VPCInterface{
			VPCID:       vpcID,
			SubnetID:    subnetID,
			IPv4Address: vpcInterfaceIPv4Request(ipv4),
		})
	}

	return vifs, nil
}

const (
	vpcIDAttr                = "id"
	vpcInterfacesAttr        = "interfaces"
	vpcInterfaceChangesAttr  = "interface_changes"
	vpcInterfaceSubnetIDAttr = "subnet_id"
	vpcInterfaceIPv4Attr     = "ipv4_address"
)

const vpcInterfaceIPv4Auto = "auto"

func vpcInterfaceIPv4Request(declared string) *string {
	if declared == "" || declared == vpcInterfaceIPv4Auto {
		return nil
	}

	return &declared
}

func validateVPCInterfaceIPv4(v any, path cty.Path) diag.Diagnostics {
	declared, ok := v.(string)
	if ok && (declared == vpcInterfaceIPv4Auto || net.ParseIP(declared).To4() != nil) {
		return nil
	}

	return diag.Diagnostics{{
		Severity:      diag.Error,
		Summary:       "Invalid IPv4 address",
		Detail:        fmt.Sprintf("Expected an IPv4 address or %q, got %q.", vpcInterfaceIPv4Auto, v),
		AttributePath: path,
	}}
}

// vpcBlock renders the attachments as the vpc block, which is empty when the
// instance is attached to no VPC at all.
func vpcBlock(vifs []VPCInterface) []any {
	if len(vifs) == 0 {
		return []any{}
	}

	interfaces := make([]any, 0, len(vifs))
	for _, vif := range vifs {
		interfaces = append(interfaces, vif.toMap())
	}

	return []any{map[string]any{
		// Every attachment of an instance belongs to the same VPC.
		vpcIDAttr:         vifs[0].VPCID,
		vpcInterfacesAttr: interfaces,
		// The description of a planned change is not state, so every read clears
		// it: a plan then shows what the next apply does to the attachments,
		// instead of diffing the previous description against the new one.
		vpcInterfaceChangesAttr: []string{},
	}}
}

// vpcInterfacesFromInstance returns the Subnet attachments the instance has accodring to the API.
func vpcInterfacesFromInstance(instance *v3.Instance) ([]VPCInterface, error) {
	if instance == nil || instance.Vpc == nil {
		return nil, nil
	}

	vifs := make([]VPCInterface, 0, len(instance.Vpc.Subnets))
	for _, subnet := range instance.Vpc.Subnets {
		if subnet.Ipv4 == nil {
			return nil, fmt.Errorf(
				"invalid attachment: subnet %s does not have an IP address", subnet.Name)
		}

		// TODO once IPv6 support is added, we also need to filter out attachments
		// of public IPv6 addresses here until the public interfaces are migrated to VPC
		if !subnet.Ipv4.IsPrivate() {
			continue
		}

		ipv4 := subnet.Ipv4.String()
		vifs = append(vifs, VPCInterface{
			VPCID:       instance.Vpc.ID.String(),
			SubnetID:    subnet.ID.String(),
			IPv4Address: &ipv4,
		})
	}

	return vifs, nil
}

// vpcInterfacesFromConfig decodes the interfaces of the vpc block of a raw
// resource configuration, in the order they are written.
func vpcInterfacesFromConfig(config cty.Value) ([]VPCInterface, error) {
	vpc := vpcAttr(config)
	if vpc.IsNull() || !vpc.IsKnown() || vpc.LengthInt() == 0 {
		return nil, nil
	}

	// The block is declared at most once, so the first element is the VPC.
	it := vpc.ElementIterator()
	it.Next()
	_, block := it.Element()
	if block.IsNull() || !block.IsKnown() {
		return nil, nil
	}

	vpcID := ctyString(block, vpcIDAttr)
	if vpcID == "" {
		return nil, fmt.Errorf("the id of the %s block is not known yet", AttrVPC)
	}

	interfaces := ctyAttr(block, vpcInterfacesAttr)
	if interfaces.IsNull() || !interfaces.IsKnown() {
		return nil, nil
	}

	vifs := make([]VPCInterface, 0, interfaces.LengthInt())
	for it := interfaces.ElementIterator(); it.Next(); {
		_, elem := it.Element()
		if elem.IsNull() || !elem.IsKnown() {
			continue
		}

		vif := VPCInterface{VPCID: vpcID, SubnetID: ctyString(elem, vpcInterfaceSubnetIDAttr)}
		if vif.SubnetID == "" {
			return nil, fmt.Errorf("the %s of an %s.%s element is not known yet",
				vpcInterfaceSubnetIDAttr, AttrVPC, vpcInterfacesAttr)
		}

		// Because ipv4_address is Optional + Computed, the value the SDK hands out
		// for a block that doesn't set one is whatever used to sit at the same position,
		// which belongs to another Subnet if the blocks are reordered.
		vif.IPv4Address = vpcInterfaceIPv4Request(ctyString(elem, vpcInterfaceIPv4Attr))

		vifs = append(vifs, vif)
	}

	return vifs, nil
}

func vpcAttr(resource cty.Value) cty.Value {
	if resource.IsNull() || !resource.IsKnown() {
		return cty.NullVal(cty.EmptyObject)
	}

	return ctyAttr(resource, AttrVPC)
}

// ctyAttr reads an attribute, and returns a null value for anything we cannot
// read one from.
func ctyAttr(obj cty.Value, attr string) cty.Value {
	t := obj.Type()
	if !t.IsObjectType() || !t.HasAttribute(attr) {
		return cty.NullVal(cty.EmptyObject)
	}

	return obj.GetAttr(attr)
}

// ctyString reads a string attribute, and returns an empty string for anything
// we cannot read a value from e.g. an address that is only known after apply, or a
// Subnet that belongs to a resource that doesn't exist yet.
func ctyString(obj cty.Value, attr string) string {
	t := obj.Type()
	if !t.IsObjectType() || !t.HasAttribute(attr) {
		return ""
	}

	v := obj.GetAttr(attr)
	if v.IsNull() || !v.IsKnown() {
		return ""
	}

	return v.AsString()
}

// diffVPCInterfaces reports which attachments reconcile prior with desired.
//
// The platform reports attachments in the order they were made, and the declared
// order is authoritative, so the interfaces are compared by position: everything
// from the first block that differs on has to be detached and attached again to
// end up in the declared order. detach comes out in the order the instance holds
// the attachments, attach in the order desired declares them.
func diffVPCInterfaces(prior, desired []VPCInterface) (detach, attach []VPCInterface) {
	keep := 0
	for keep < len(prior) && keep < len(desired) {
		have, want := prior[keep], desired[keep]
		if have.SubnetID != want.SubnetID || vpcInterfaceNeedsReattach(have, want) {
			break
		}

		keep++
	}

	return prior[keep:], desired[keep:]
}

type vpcInterfaceChangeSummary struct {
	// Attach holds the Subnets that are not attached yet.
	Attach []string
	// Detach holds the Subnets that are detached and not attached again.
	Detach []string
	// Reattach holds the Subnets that are detached and attached again, to end up
	// in the declared order.
	Reattach []string
}

// Lines renders the summary one sentence per group.
func (s vpcInterfaceChangeSummary) Lines() []string {
	lines := []string{}
	if len(s.Attach) > 0 {
		lines = append(lines, fmt.Sprintf("attaching %s", strings.Join(s.Attach, ", ")))
	}
	if len(s.Detach) > 0 {
		lines = append(lines, fmt.Sprintf("detaching %s", strings.Join(s.Detach, ", ")))
	}
	if len(s.Reattach) > 0 {
		lines = append(lines, fmt.Sprintf(
			"reattaching %s", strings.Join(s.Reattach, ", ")))
	}

	return lines
}

// summarizeVPCInterfaceChanges groups the attachments diffVPCInterfaces reports
// by what happens to them, so that a plan can name them.
func summarizeVPCInterfaceChanges(prior, desired []VPCInterface) vpcInterfaceChangeSummary {
	detach, attach := diffVPCInterfaces(prior, desired)

	detached := make(map[string]bool, len(detach))
	for _, vif := range detach {
		detached[vif.SubnetID] = true
	}

	summary := vpcInterfaceChangeSummary{}
	attached := make(map[string]bool, len(attach))
	for _, vif := range attach {
		attached[vif.SubnetID] = true

		if !detached[vif.SubnetID] {
			summary.Attach = append(summary.Attach, vif.SubnetID)
			continue
		}

		summary.Reattach = append(summary.Reattach, vif.SubnetID)
	}

	for _, vif := range detach {
		if !attached[vif.SubnetID] {
			summary.Detach = append(summary.Detach, vif.SubnetID)
		}
	}

	return summary
}

func vpcInterfaceNeedsReattach(prior, desired VPCInterface) bool {
	if desired.IPv4Address == nil || *desired.IPv4Address == "" {
		return false
	}
	if prior.IPv4Address == nil {
		return true
	}

	return *desired.IPv4Address != *prior.IPv4Address
}

// resolveVPCInterfaceAutoAddresses replaces the auto keyword with the address the
// Subnet is already attached with, so that a plan shows the address rather than
// the keyword standing in for it. Empty for a Subnet that is not attached yet:
// the platform allocates on attach, and there is nothing to show until then.
func resolveVPCInterfaceAutoAddresses(block map[string]any, prior []VPCInterface) {
	attached := make(map[string]string, len(prior))
	for _, vif := range prior {
		if vif.IPv4Address != nil {
			attached[vif.SubnetID] = *vif.IPv4Address
		}
	}

	interfaces, _ := block[vpcInterfacesAttr].([]any)
	for _, item := range interfaces {
		vif, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if declared, _ := vif[vpcInterfaceIPv4Attr].(string); declared != vpcInterfaceIPv4Auto {
			continue
		}

		subnetID, _ := vif[vpcInterfaceSubnetIDAttr].(string)
		vif[vpcInterfaceIPv4Attr] = attached[subnetID]
	}
}
