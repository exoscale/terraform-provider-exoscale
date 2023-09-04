package testutils

// DataSourceNlbServiceListModel maps to datasource_nlb_service_list.tmpl
type DataSourceNlbServiceListModel struct {
	ResourceName string

	Zone string
	ID   string
	Name string

	RawConfig string
}

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

// ResourceNLBModel maps to resource_nlb.tmpl
type ResourceNLBModel struct {
	ResourceName string

	Zone        string
	Name        string
	Description string
	Labels      string
}

// ResourceNLBServiceModel maps to resource_nlb_service.tmpl
type ResourceNLBServiceModel struct {
	ResourceName string

	Zone                string
	Name                string
	NLBID               string
	InstancePoolID      string
	Port                int64
	TargetPort          int64
	Description         string
	Protocol            string
	Strategy            string
	HealthcheckPort     int64
	HealthcheckInterval int64
	HealthcheckMode     string
	HealthcheckRetries  int64
	HealthcheckTimeout  int64
	HealthcheckTLSSNI   string
	HealthcheckURI      string
}
