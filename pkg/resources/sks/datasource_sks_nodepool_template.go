package sks

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const markdownDescriptionNodepoolTemplateDataSource = `Fetch an Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) Node Pool template, to pin an [exoscale_sks_nodepool](../resources/sks_nodepool.md) on it with ` + "`template_id`" + `.

Without ` + "`name`" + `, the latest template of the Kubernetes version and variant is returned.`

var sksNodepoolTemplateVariants = []string{
	string(exoscale.ListSKSNodepoolTemplatesVariantStandard),
	string(exoscale.ListSKSNodepoolTemplatesVariantNvidia),
}

var _ datasource.DataSourceWithConfigure = (*DataSourceNodepoolTemplate)(nil)

type DataSourceNodepoolTemplate struct {
	client *exoscale.Client
}

func NewDataSourceNodepoolTemplate() datasource.DataSource {
	return &DataSourceNodepoolTemplate{}
}

type DataSourceNodepoolTemplateModel struct {
	Zone        types.String `tfsdk:"zone"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	CreatedAt   types.String `tfsdk:"created_at"`
	KubeVersion types.String `tfsdk:"kube_version"`
	Variant     types.String `tfsdk:"variant"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceNodepoolTemplate) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sks_nodepool_template"
}

func (d *DataSourceNodepoolTemplate) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescriptionNodepoolTemplateDataSource,

		Attributes: map[string]schema.Attribute{
			"zone": schema.StringAttribute{
				Description:         "The Exoscale zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"id": schema.StringAttribute{
				Description:         "The SKS node pool template ID.",
				MarkdownDescription: "The SKS node pool template ID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The SKS node pool template name to match (default: the latest template).",
				MarkdownDescription: "The SKS node pool template name to match (default: the latest template).",
				Optional:            true,
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The template registration date.",
				MarkdownDescription: "The template registration date.",
				Computed:            true,
			},
			"kube_version": schema.StringAttribute{
				Description:         "The Kubernetes version of the node pool ('<major>.<minor>' or '<major>.<minor>.<patch>').",
				MarkdownDescription: "The Kubernetes version of the node pool (`<major>.<minor>` or `<major>.<minor>.<patch>`).",
				Required:            true,
			},
			"variant": schema.StringAttribute{
				Description:         "The template variant: 'standard', or 'nvidia' for GPU instance types (default: 'standard').",
				MarkdownDescription: "The template variant: `standard`, or `nvidia` for GPU instance types (default: `standard`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(sksNodepoolTemplateVariants...),
				},
			},
		},

		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (d *DataSourceNodepoolTemplate) Configure(ctx context.Context, r datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceNodepoolTemplate) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataSourceNodepoolTemplateModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(
		ctx,
		d.client,
		exoscale.ZoneName(state.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	if state.Variant.IsNull() {
		state.Variant = types.StringValue(string(exoscale.ListSKSNodepoolTemplatesVariantStandard))
	}
	kubeVersion, variant := state.KubeVersion.ValueString(), state.Variant.ValueString()

	listResp, err := client.ListSKSNodepoolTemplates(
		ctx,
		kubeVersion,
		exoscale.ListSKSNodepoolTemplatesWithVariant(exoscale.ListSKSNodepoolTemplatesVariant(variant)),
	)
	if err != nil {
		resp.Diagnostics.AddError("error listing SKS nodepool templates", err.Error())
		return
	}

	nameOrID := state.Name.ValueString()
	if state.Name.IsNull() {
		active, err := client.GetActiveNodepoolTemplate(ctx, kubeVersion, exoscale.GetActiveNodepoolTemplateVariant(variant))
		if err != nil {
			resp.Diagnostics.AddError("error getting the active SKS nodepool template", err.Error())
			return
		}
		nameOrID = active.ActiveTemplate.String()
	}

	template, err := listResp.FindSKSNodepoolTemplate(nameOrID)
	if err != nil {
		resp.Diagnostics.AddError("error finding SKS nodepool template", err.Error())
		return
	}

	state.ID = types.StringValue(template.ID.String())
	state.Name = types.StringValue(template.Name)
	state.CreatedAt = types.StringValue(template.CreatedAT.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
