package general

import (
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type SchemaMap map[string]*schema.Schema

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

func Apply(data TerraformObject, d *schema.ResourceData, schema map[string]*schema.Schema) error {
	for attrIdentifier, attrVal := range data {
		_, hasAttribute := schema[attrIdentifier]
		if hasAttribute {
			if err := d.Set(attrIdentifier, attrVal); err != nil {
				return err
			}
		}
	}

	return nil
}

func AddAttributes(res *schema.Resource, resourceSchema map[string]*schema.Schema) {
	for attributeIdentifier, attributeValue := range resourceSchema {
		_, attributeAlreadySet := res.Schema[attributeIdentifier]
		if !attributeAlreadySet {
			newSchema := &schema.Schema{}
			*newSchema = *attributeValue
			newSchema.Required = false
			newSchema.Optional = true
			newSchema.Default = nil

			res.Schema[attributeIdentifier] = newSchema
		}
	}
}
