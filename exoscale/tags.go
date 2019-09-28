package exoscale

import (
	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// addTags adds the tags structure to the schema at the given key
func addTags(s map[string]*schema.Schema, key string) {
	s[key] = &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Computed: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Description: "Map of tags (key: value)",
	}
}

// createTags create the tags for the given resource (provide a resource type)
func createTags(d *schema.ResourceData, key, resourceType string) (egoscale.Command, error) {
	if t, ok := d.GetOk(key); ok {
		m := t.(map[string]interface{})
		tags := make([]egoscale.ResourceTag, 0, len(m))
		for k, v := range m {
			tags = append(tags, egoscale.ResourceTag{
				Key:   k,
				Value: v.(string),
			})
		}

		id, err := egoscale.ParseUUID(d.Id())
		if err != nil {
			return nil, err
		}

		return &egoscale.CreateTags{
			ResourceIDs:  []egoscale.UUID{*id},
			ResourceType: resourceType,
			Tags:         tags,
		}, nil
	}

	return nil, nil
}

// updateTags create the commands to delete / create the tags for a resource
func updateTags(d *schema.ResourceData, key, resourceType string) ([]egoscale.Command, error) {
	requests := make([]egoscale.Command, 0)

	if d.HasChange(key) {
		d.SetPartial(key)
		o, n := d.GetChange(key)

		oldTags := o.(map[string]interface{})
		newTags := n.(map[string]interface{})

		// Remove the intersection between the two sets of tag
		for k, v := range oldTags {
			if value, ok := newTags[k]; ok && v == value {
				delete(oldTags, k)
				delete(newTags, k)
			}
		}

		id, err := egoscale.ParseUUID(d.Id())
		if err != nil {
			return requests, err
		}

		if len(oldTags) > 0 {
			deleteTags := &egoscale.DeleteTags{
				ResourceIDs:  []egoscale.UUID{*id},
				ResourceType: resourceType,
				Tags:         make([]egoscale.ResourceTag, len(oldTags)),
			}
			i := 0
			for k, v := range oldTags {
				deleteTags.Tags[i] = egoscale.ResourceTag{
					Key:   k,
					Value: v.(string),
				}
				i++
			}
			requests = append(requests, deleteTags)
		}

		if len(newTags) > 0 {
			createTags := &egoscale.CreateTags{
				ResourceIDs:  []egoscale.UUID{*id},
				ResourceType: resourceType,
				Tags:         make([]egoscale.ResourceTag, len(newTags)),
			}
			i := 0
			for k, v := range newTags {
				createTags.Tags[i] = egoscale.ResourceTag{
					Key:   k,
					Value: v.(string),
				}
				i++
			}
			requests = append(requests, createTags)
		}
	}

	return requests, nil
}
