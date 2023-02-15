package datasource

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func AddAttributes(dataSourceResource *schema.Resource, resourceSchema map[string]*schema.Schema) {
	for attributeIdentifier, attributeValue := range resourceSchema {
		_, attributeAlreadySet := dataSourceResource.Schema[attributeIdentifier]
		if !attributeAlreadySet {
			newSchema := &schema.Schema{}
			*newSchema = *attributeValue
			newSchema.Required = false
			newSchema.Optional = true
			newSchema.Default = nil

			dataSourceResource.Schema[attributeIdentifier] = newSchema
		}
	}
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
