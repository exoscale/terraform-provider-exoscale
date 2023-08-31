package oapi

type DBaaSIntegrationSettings struct {
	AdditionalProperties *bool
	Properties           *map[string]interface{}
	Title                *string
	Type                 *string
}

func FromListDbaasIntegrationSettingsResponse(r *ListDbaasIntegrationSettingsResponse) *DBaaSIntegrationSettings {
	t := r.JSON200.Settings

	return &DBaaSIntegrationSettings{
		AdditionalProperties: t.AdditionalProperties,
		Properties:           t.Properties,
		Title:                t.Title,
		Type:                 t.Type,
	}
}
