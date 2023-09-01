package iam

import "github.com/exoscale/egoscale/v3/oapi"

type IAMIface interface {
	Roles() RolesIface
	OrgPolicy() OrgPolicyIface
	AccessKey() AccessKeyIface
}

// IAM provides access to [Exoscale IAM] API resources.
//
// [Exoscale IAM]: https://community.exoscale.com/documentation/iam/
type IAM struct {
	oapiClient *oapi.ClientWithResponses
}

// NewIAM initializes IAM with provided oapi Client.
func NewIAM(c *oapi.ClientWithResponses) IAMIface {
	return &IAM{c}
}

func (a *IAM) Roles() RolesIface {
	return NewRoles(a.oapiClient)
}

func (a *IAM) OrgPolicy() OrgPolicyIface {
	return NewOrgPolicy(a.oapiClient)
}

func (a *IAM) AccessKey() AccessKeyIface {
	return NewAccessKey(a.oapiClient)
}

type MockIAM struct {
}

// NewMockIAM initializes MockIAM with provided oapi Client.
func NewMockIAM() *MockIAM {
	return &MockIAM{}
}

func (a *MockIAM) Roles() RolesIface {
	return NewMockRoles()
}

func (a *MockIAM) OrgPolicy() OrgPolicyIface {
	return NewMockOrgPolicy()
}

func (a *MockIAM) AccessKey() AccessKeyIface {
	return NewMockAccessKey()
}
