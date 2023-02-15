package datasource

import "time"

type TerraformObject = map[string]interface{}

func AssignTime(data TerraformObject, attributeIdentifier string, value *time.Time) {
	if value == nil {
		return
	}

	data[attributeIdentifier] = value.Format(time.RFC3339)
}

func Assign[T any](data TerraformObject, attributeIdentifier string, value *T) {
	if value == nil {
		return
	}

	data[attributeIdentifier] = *value
}
