package testutils

// DataSourceTemplateModel maps to datasource_template.tmpl
type DataSourceTemplateModel struct {
	ResourceName string

	Zone       string
	ID         string
	Name       string
	Visibility string
}

// ResourceInstancePoolModel maps to resource_instance_pool.tmpl
type ResourceInstancePoolModel struct {
	ResourceName string

	Zone       string
	Name       string
	Size       int64
	TemplateID string
	Type       string
	DiskSize   int64
}

// ResourceIAMOrgPolicyModel maps to resource_iam_org_policy.tmpl
type ResourceIAMOrgPolicyModel struct {
	ResourceName string

	DefaultServiceStrategy string
	Services               map[string]ResourceIAMPolicyServicesModel
}

// ResourceIAMPolicyServicesModel defines nested structure within IAM Policy.
type ResourceIAMPolicyServicesModel struct {
	Type  string
	Rules []ResourceIAMPolicyServiceRules
}

// ResourceIAMPolicyServiceRules defines nested structure within IAM Policy Service.
type ResourceIAMPolicyServiceRules struct {
	Action     string
	Expression string
	Resources  string
}

// ResourceIAMRole maps to resource_iam_role.tmpl
type ResourceIAMRole struct {
	ResourceName string

	Name        string
	Description string
	Editable    bool
	Labels      map[string]string
	Permissions string

	Policy *ResourceIAMOrgPolicyModel
}

// ResourceAPIKeyModel maps to resource_iam_api_key.tmpl
type ResourceAPIKeyModel struct {
	ResourceName string

	Name   string
	RoleID string
}
