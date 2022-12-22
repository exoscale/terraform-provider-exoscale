package exoscale

import (
	"context"
	"regexp"

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	dsTemplateAttrDefaultUser = "default_user"
	dsTemplateAttrID          = "id"
	dsTemplateAttrName        = "name"
	dsTemplateAttrVisibility  = "visibility"
	dsTemplateAttrZone        = "zone"
)

func dataSourceTemplate() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			dsTemplateAttrZone: {
				Type:        schema.TypeString,
				Description: "Name of the zone",
				Required:    true,
			},
			dsTemplateAttrName: {
				Type:          schema.TypeString,
				Description:   "Name of the template",
				Optional:      true,
				ConflictsWith: []string{"id"},
			},
			dsTemplateAttrID: {
				Type:          schema.TypeString,
				Description:   "ID of the template",
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			dsTemplateAttrVisibility: {
				Type:        schema.TypeString,
				Description: "template visibility (public|private)",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("(?:public|private)"),
					`must be either "public" or "private"`),
				Optional: true,
				Default:  "public",
			},
			dsTemplateAttrDefaultUser: {
				Type:        schema.TypeString,
				Description: "Template default user",
				Computed:    true,
			},
		},

		ReadContext: dataSourceTemplateRead,
	}
}

func dataSourceTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceIDString(d, "exoscale_template"),
	})

	zone := d.Get(dsTemplateAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	templateID, byTemplateID := d.GetOk(dsTemplateAttrID)
	templateName, byTemplateName := d.GetOk(dsTemplateAttrName)
	if !byTemplateID && !byTemplateName {
		return diag.Errorf(
			"either %s or %s must be specified",
			dsTemplateAttrID,
			dsTemplateAttrName,
		)
	}
	visibility := d.Get(dsTemplateAttrVisibility).(string)

	var template *v2.Template
	var err error
	if byTemplateID {
		template, err = client.GetTemplate(ctx, zone, templateID.(string))

	} else {

		template, err = client.GetTemplateByName(ctx, zone, templateName.(string), visibility)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*template.ID)

	if err := d.Set(dsTemplateAttrName, defaultString(template.Name, "")); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(dsTemplateAttrDefaultUser, defaultString(template.DefaultUser, "")); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceIDString(d, "exoscale_template"),
	})

	return nil
}
