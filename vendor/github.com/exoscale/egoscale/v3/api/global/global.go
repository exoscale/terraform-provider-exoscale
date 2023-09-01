package global

import "github.com/exoscale/egoscale/v3/oapi"

type GlobalIface interface {
	OrgQuotas() OrgQuotasIface
	Operations() OperationIface
	Zones() ZonesIface
}

// Global provides access to global API resources.
type Global struct {
	oapiClient *oapi.ClientWithResponses
}

// NewGlobal initializes Global with provided oapi Client.
func NewGlobal(c *oapi.ClientWithResponses) GlobalIface {
	return &Global{c}
}

func (a *Global) OrgQuotas() OrgQuotasIface {
	return NewOrgQuotas(a.oapiClient)
}

func (a *Global) Operations() OperationIface {
	return NewOperation(a.oapiClient)
}

func (a *Global) Zones() ZonesIface {
	return NewZones(a.oapiClient)
}

// MockGlobal provides access to global API resources.
type MockGlobal struct {
}

// NewMockGlobal initializes MockGlobal with provided oapi Client.
func NewMockGlobal() *MockGlobal {
	return &MockGlobal{}
}

func (a *MockGlobal) OrgQuotas() OrgQuotasIface {
	return NewMockOrgQuotas()
}

func (a *MockGlobal) Operations() OperationIface {
	return NewMockOperation()
}

func (a *MockGlobal) Zones() ZonesIface {
	return NewMockZones()
}
