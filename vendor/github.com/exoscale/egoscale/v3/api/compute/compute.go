package compute

import "github.com/exoscale/egoscale/v3/oapi"

type ComputeIface interface {
	InstanceTypes() InstanceTypesIface
	PrivateNetworks() PrivateNetworksIface
	SSHKeys() SSHKeysIface
}

// Compute provides access to [Exoscale Compute] API resources.
//
// [Exoscale Compute]: https://community.exoscale.com/documentation/compute/
type Compute struct {
	oapiClient *oapi.ClientWithResponses
}

// NewCompute initializes Compute with provided oapi Client.
func NewCompute(c *oapi.ClientWithResponses) *Compute {
	return &Compute{c}
}

//func (a *Compute) Instances() *Instaces {
//return NewInstances(a.oapiClient)
//}

func (a *Compute) InstanceTypes() InstanceTypesIface {
	return NewInstanceTypes(a.oapiClient)
}

func (c *Compute) PrivateNetworks() PrivateNetworksIface {
	return NewPrivateNetworks(c.oapiClient)
}

func (c *Compute) SSHKeys() SSHKeysIface {
	return NewSSHKeys(c.oapiClient)
}

type MockCompute struct {
}

// NewMockCompute initializes MockCompute with provided oapi Client.
func NewMockCompute() *MockCompute {
	return &MockCompute{}
}

//func (a *MockCompute) Instances() *Instaces {
//return NewInstances(a.oapiClient)
//}

func (a *MockCompute) InstanceTypes() InstanceTypesIface {
	return NewMockInstanceTypes()
}

func (c *MockCompute) PrivateNetworks() PrivateNetworksIface {
	return NewMockPrivateNetworks()
}

func (c *MockCompute) SSHKeys() SSHKeysIface {
	return NewMockSSHKeys()
}
