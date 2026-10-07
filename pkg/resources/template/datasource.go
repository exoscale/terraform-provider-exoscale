package template

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const markdownDescriptionDataSourceTemplate = `Fetch Exoscale [Compute Instance Templates](https://community.exoscale.com/product/compute/instances/how-to/custom-templates/) data.

Exoscale instance templates are regularly updated to include the latest updates. Whenever this happens, the template ID also changes which can lead terraform to plan the recreation of an instance. To work around this you may find [this issue](https://github.com/exoscale/terraform-provider-exoscale/issues/366) helpful.`

var _ datasource.DataSourceWithConfigure = (*DataSourceTemplate)(nil)

type DataSourceTemplate struct {
	client *exoscale.Client
}

func NewDataSourceTemplate() datasource.DataSource {
	return &DataSourceTemplate{}
}

type DataSourceTemplateModel struct {
	ID          types.String `tfsdk:"id"`
	Zone        types.String `tfsdk:"zone"`
	Name        types.String `tfsdk:"name"`
	Visibility  types.String `tfsdk:"visibility"`
	DefaultUser types.String `tfsdk:"default_user"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceTemplate) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (d *DataSourceTemplate) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Fetch Exoscale Compute Instance Templates data.",
		MarkdownDescription: markdownDescriptionDataSourceTemplate,
		Attributes: map[string]schema.Attribute{
			"zone": schema.StringAttribute{
				Description:         "The Exoscale Zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The template name to match (conflicts with 'id') (when multiple templates have the same name, the newest one will be returned).",
				MarkdownDescription: "The template name to match (conflicts with `id`) (when multiple templates have the same name, the newest one will be returned).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("id")),
				},
			},
			"id": schema.StringAttribute{
				Description:         "The compute instance template ID to match (conflicts with 'name').",
				MarkdownDescription: "The compute instance template ID to match (conflicts with `name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("name")),
				},
			},
			"visibility": schema.StringAttribute{
				Description:         "A template category filter (default: 'public'); among: - 'public' - official Exoscale templates - 'private' - custom templates private to my organization",
				MarkdownDescription: "A template category filter (default: `public`); among: - `public` - official Exoscale templates - `private` - custom templates private to my organization",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(exoscale.ListTemplatesVisibilityPublic),
						string(exoscale.ListTemplatesVisibilityPrivate),
					),
				},
			},
			"default_user": schema.StringAttribute{
				Description:         "Username to use to log into a compute instance based on this template",
				MarkdownDescription: "Username to use to log into a compute instance based on this template",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (d *DataSourceTemplate) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceTemplate) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceTemplateModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	if data.Visibility.IsNull() {
		data.Visibility = types.StringValue(string(exoscale.ListTemplatesVisibilityPublic))
	}

	var template *exoscale.Template
	if !data.ID.IsNull() {
		id, err := exoscale.ParseUUID(data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse template ID", err.Error())
			return
		}

		template, err = client.GetTemplate(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("unable to get template", err.Error())
			return
		}
	} else {
		visibility := exoscale.ListTemplatesVisibility(data.Visibility.ValueString())
		template, err = findTemplateByName(ctx, client, data.Name.ValueString(), visibility)
		// A name not found among the public templates is looked up in the private ones.
		if errors.Is(err, exoscale.ErrNotFound) && visibility == exoscale.ListTemplatesVisibilityPublic {
			template, err = findTemplateByName(ctx, client, data.Name.ValueString(), exoscale.ListTemplatesVisibilityPrivate)
		}
		if err != nil {
			resp.Diagnostics.AddError("unable to find template", err.Error())
			return
		}
	}

	data.ID = types.StringValue(template.ID.String())
	data.Name = types.StringValue(template.Name)
	data.DefaultUser = types.StringValue(template.DefaultUser)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "data source read", map[string]any{
		"id": data.ID.ValueString(),
	})
}

// findTemplateByName returns the newest template with the given name: several
// private templates can share a name.
func findTemplateByName(
	ctx context.Context,
	client *exoscale.Client,
	name string,
	visibility exoscale.ListTemplatesVisibility,
) (*exoscale.Template, error) {
	list, err := client.ListTemplates(ctx, exoscale.ListTemplatesWithVisibility(visibility))
	if err != nil {
		return nil, err
	}

	var newest *exoscale.Template
	for i, template := range list.Templates {
		if template.Name == name && (newest == nil || template.CreatedAT.After(newest.CreatedAT)) {
			newest = &list.Templates[i]
		}
	}
	if newest == nil {
		return nil, fmt.Errorf("template %q not found in %s templates: %w", name, visibility, exoscale.ErrNotFound)
	}

	return newest, nil
}
