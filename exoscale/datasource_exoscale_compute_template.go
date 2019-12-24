package exoscale

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func dataSourceComputeTemplate() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"zone": {
				Type:        schema.TypeString,
				Description: "Name of the zone",
				Required:    true,
			},
			"name": {
				Type:          schema.TypeString,
				Description:   "Name of the template",
				Optional:      true,
				ConflictsWith: []string{"id"},
			},
			"id": {
				Type:          schema.TypeString,
				Description:   "ID of the template",
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			"filter": {
				Type:        schema.TypeString,
				Description: "Template filter to apply",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("(?:featured|community|mine)"),
					`must be either "featured", "community" or "mine"`),
				Optional: true,
				Default:  "featured",
			},

			"username": {
				Type:        schema.TypeString,
				Description: "Username for logging into a compute instance based on this template",
				Computed:    true,
			},
		},

		Read: dataSourceComputeTemplateRead,
	}
}

func dataSourceComputeTemplateRead(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	req := egoscale.ListTemplates{
		ZoneID:         zone.ID,
		TemplateFilter: d.Get("filter").(string),
	}

	// Template filter "mine" is a friendlier alias for "self"
	if req.TemplateFilter == "mine" {
		req.TemplateFilter = "self"
	}

	templateName, byName := d.GetOk("name")
	templateID, byID := d.GetOk("id")
	if !byName && !byID {
		return errors.New("either name or id must be specified")
	}

	req.Name = templateName.(string)

	if templateID != "" {
		if req.ID, err = egoscale.ParseUUID(templateID.(string)); err != nil {
			return fmt.Errorf("invalid value for id: %s", err)
		}
	}

	resp, err := client.RequestWithContext(ctx, &req)
	if err != nil {
		return fmt.Errorf("templates list query failed: %s", err)
	}

	var template egoscale.Template
	nt := resp.(*egoscale.ListTemplatesResponse).Count
	switch {
	case nt == 0:
		return errors.New("template not found")

	case nt > 1:
		return errors.New("multiple results returned, expected only one")

	default:
		template = resp.(*egoscale.ListTemplatesResponse).Template[0]
	}

	d.SetId(template.ID.String())

	if err := d.Set("id", d.Id()); err != nil {
		return err
	}
	if err := d.Set("name", template.Name); err != nil {
		return err
	}

	if username, ok := template.Details["username"]; ok {
		if err := d.Set("username", username); err != nil {
			return err
		}
	} else {
		// If no username information provided in the template details,
		// attempt an educated guess based on the template name
		if err := d.Set("username", getSSHUsername(template.Name)); err != nil {
			return err
		}
	}

	return nil
}
