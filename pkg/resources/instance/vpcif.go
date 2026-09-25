package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"strings"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/go-cty/cty"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type VPCInterface struct {
	VPCID       string  `json:"vpc_id"`
	SubnetID    string  `json:"subnet_id"`
	IPv4Address *string `json:"ipv4_address"`
}

func NewVPCInterface(raw any) (*VPCInterface, error) {
	serialized, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	vif := VPCInterface{}
	if err := json.Unmarshal(serialized, &vif); err != nil {
		tflog.Warn(context.Background(), err.Error())
		return nil, err
	}

	return &vif, nil
}

func (v VPCInterface) ToMap() (map[string]any, error) {
	serialized, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var vif map[string]any
	if err := json.Unmarshal(serialized, &vif); err != nil {
		return nil, err
	}

	return vif, nil
}

// ipv4 returns the configured IPv4 address, or nil when none was requested
func (v VPCInterface) ipv4() net.IP {
	if v.IPv4Address == nil || *v.IPv4Address == "" {
		return nil
	}

	return net.ParseIP(*v.IPv4Address)
}

// vpcInterfacesFromList decodes the vpc_interface list, in the given order
func vpcInterfacesFromList(raw []any) ([]VPCInterface, error) {
	vifs := make([]VPCInterface, 0, len(raw))
	for _, item := range raw {
		vif, err := NewVPCInterface(item)
		if err != nil {
			return nil, err
		}

		vifs = append(vifs, *vif)
	}

	return vifs, nil
}

// vpcInterfacesFromInstance returns the Subnet attachments the instance actually
// has, which is what a change has to be reconciled against: an attachment made
// out of band is real whether Terraform knows about it or not.
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

// vpcInterfacesFromConfig decodes the vpc_interface blocks of a raw resource
// configuration, in the order they are written.
func vpcInterfacesFromConfig(config cty.Value) ([]VPCInterface, error) {
	raw := vpcInterfaceAttr(config)
	if raw.IsNull() || !raw.IsKnown() {
		return nil, nil
	}

	vifs := make([]VPCInterface, 0, raw.LengthInt())
	for it := raw.ElementIterator(); it.Next(); {
		_, elem := it.Element()
		if elem.IsNull() || !elem.IsKnown() {
			continue
		}

		vif := VPCInterface{
			VPCID:    ctyString(elem, "vpc_id"),
			SubnetID: ctyString(elem, "subnet_id"),
		}
		if vif.VPCID == "" || vif.SubnetID == "" {
			return nil, fmt.Errorf(
				"the vpc_id or subnet_id of a %s block is not known yet", AttrVPCInterface)
		}

		// Because ipv4_address is Optional + Computed, the value the SDK hands out
		// for a block that doesn't set one is whatever used to sit at the same position,
		// which belongs to another Subnet if the blocks are reordered.
		if addr := ctyString(elem, "ipv4_address"); addr != "" {
			vif.IPv4Address = &addr
		}

		vifs = append(vifs, vif)
	}

	return vifs, nil
}

// vpcInterfaceAttr returns the vpc_interface attribute of a raw resource value,
// or a null value when the resource itself is null or unknown - during a plain
// refresh Terraform sends no configuration.
func vpcInterfaceAttr(resource cty.Value) cty.Value {
	if resource.IsNull() || !resource.IsKnown() {
		return cty.NullVal(cty.EmptyObject)
	}

	t := resource.Type()
	if !t.IsObjectType() || !t.HasAttribute(AttrVPCInterface) {
		return cty.NullVal(cty.EmptyObject)
	}

	return resource.GetAttr(AttrVPCInterface)
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
// Interfaces are matched by Subnet ID rather than by position - a VM is attached
// to a given Subnet at most once - so inserting, removing or moving a block
// leaves every other attachment alone. If we relied only on the plan provided
// by terraform this wouldn't be the case.
// The returned 'attach' list comes out in the order 'desired' declares,
// so new Subnets are attached in that order.
func diffVPCInterfaces(prior, desired []VPCInterface) (detach, attach []VPCInterface) {
	priorBySubnet := make(map[string]VPCInterface, len(prior))
	for _, vif := range prior {
		priorBySubnet[vif.SubnetID] = vif
	}
	desiredBySubnet := make(map[string]VPCInterface, len(desired))
	for _, vif := range desired {
		desiredBySubnet[vif.SubnetID] = vif
	}

	for _, want := range desired {
		have, ok := priorBySubnet[want.SubnetID]
		switch {
		case !ok:
			attach = append(attach, want)
		case vpcInterfaceNeedsReattach(have, want):
			// The VPC of an existing attachment is the one the Subnet belongs
			// to, whatever the configuration claims.
			want.VPCID = have.VPCID
			detach = append(detach, have)
			attach = append(attach, want)
		}
	}

	for _, have := range prior {
		if _, ok := desiredBySubnet[have.SubnetID]; !ok {
			detach = append(detach, have)
		}
	}

	return detach, attach
}

// vpcInterfaceNeedsReattach reports whether an interface that is attached both
// before and after a change has to be detached and attached again: only an
// address that is explicitly configured and differs from the current one
// requires it. An unset address means "let the platform allocate", which the
// already attached interface satisfies.
func vpcInterfaceNeedsReattach(prior, desired VPCInterface) bool {
	if desired.IPv4Address == nil || *desired.IPv4Address == "" {
		return false
	}
	if prior.IPv4Address == nil {
		return true
	}

	return *desired.IPv4Address != *prior.IPv4Address
}

// inDeclaredOrder returns vifs ordered after the Subnet IDs in order, so that
// the vpc_interface list is stored in the order the blocks are declared in: the
// API does not guarantee any Subnet ordering, and storing the list in an order
// of our own would make the configuration plan forever. Subnets not in order -
// attached out of band - are appended, sorted by Subnet ID to keep the result
// deterministic.
func inDeclaredOrder(order []string, vifs []VPCInterface) []VPCInterface {
	bySubnet := make(map[string]VPCInterface, len(vifs))
	for _, vif := range vifs {
		bySubnet[vif.SubnetID] = vif
	}

	ordered := make([]VPCInterface, 0, len(vifs))
	for _, subnetID := range order {
		if vif, ok := bySubnet[subnetID]; ok {
			ordered = append(ordered, vif)
			// since order may contain the same ID multiple times
			// we need to remove subnets that were already added
			// to the ordered list.
			delete(bySubnet, subnetID)
		}
	}

	rest := make([]VPCInterface, 0, len(bySubnet))
	for _, vif := range bySubnet {
		rest = append(rest, vif)
	}
	slices.SortStableFunc(rest, func(a, b VPCInterface) int {
		return strings.Compare(a.SubnetID, b.SubnetID)
	})

	return append(ordered, rest...)
}

// subnetIDs returns the Subnet IDs of vifs, in the same order.
func subnetIDs(vifs []VPCInterface) []string {
	ids := make([]string, len(vifs))
	for i, vif := range vifs {
		ids[i] = vif.SubnetID
	}

	return ids
}
