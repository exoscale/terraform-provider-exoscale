package exoscale

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
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
		Description: `Fetch Exoscale [Compute Instance Templates](https://community.exoscale.com/documentation/compute/custom-templates/) data.

Exoscale instance templates are regularly updated to include the latest updates. Whenever this happens, the template ID also changes which can lead terraform to plan the recreation of an instance. To work around this you may find [this issue](https://github.com/exoscale/terraform-provider-exoscale/issues/366) helpful.`,
		Schema: map[string]*schema.Schema{
			dsTemplateAttrZone: {
				Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			dsTemplateAttrName: {
				Description:   "The template name to match (conflicts with `id`) (when multiple templates have the same name, the newest one will be returned).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
			},
			dsTemplateAttrID: {
				Description:   "The compute instance template ID to match (conflicts with `name`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			dsTemplateAttrVisibility: {
				Description: "A template category filter (default: `public`); among: - `public` - official Exoscale templates - `private` - custom templates private to my organization",
				Type:        schema.TypeString,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("(?:public|private)"),
					`must be either "public" or "private"`),
				Optional: true,
				Default:  "public",
			},
			dsTemplateAttrDefaultUser: {
				Description: "Username to use to log into a compute instance based on this template",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},

		ReadContext: dataSourceTemplateRead,
	}
}

func dataSourceTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": general.ResourceIDString(d, "exoscale_template"),
	})

	zone := d.Get(dsTemplateAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

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
		"id": general.ResourceIDString(d, "exoscale_template"),
	})

	return nil
}
