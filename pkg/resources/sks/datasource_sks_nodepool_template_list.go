package sks

import (
	"context"
	"crypto/md5" //nolint:gosec // used only to derive a stable synthetic data source ID, not for security purposes
	"fmt"
	"strings"

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

const markdownDescriptionNodepoolTemplateListDataSource = `List the Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) Node Pool templates available for a Kubernetes version, latest first.

An [exoscale_sks_nodepool](../resources/sks_nodepool.md) can be pinned on one of them with ` + "`template_id`" + `.`

var _ datasource.DataSourceWithConfigure = (*DataSourceNodepoolTemplateList)(nil)

type DataSourceNodepoolTemplateList struct {
	client *exoscale.Client
}

func NewDataSourceNodepoolTemplateList() datasource.DataSource {
	return &DataSourceNodepoolTemplateList{}
}

type DataSourceNodepoolTemplateListModel struct {
	ID          types.String                              `tfsdk:"id"`
	Zone        types.String                              `tfsdk:"zone"`
	KubeVersion types.String                              `tfsdk:"kube_version"`
	Variant     types.String                              `tfsdk:"variant"`
	Templates   []DataSourceNodepoolTemplateListItemModel `tfsdk:"templates"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type DataSourceNodepoolTemplateListItemModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
	Variant   types.String `tfsdk:"variant"`
}

func (d *DataSourceNodepoolTemplateList) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sks_nodepool_template_list"
}

func (d *DataSourceNodepoolTemplateList) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescriptionNodepoolTemplateListDataSource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The ID of this resource.",
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				Description:         "The Exoscale zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"kube_version": schema.StringAttribute{
				Description:         "The Kubernetes version of the node pool ('<major>.<minor>' or '<major>.<minor>.<patch>').",
				MarkdownDescription: "The Kubernetes version of the node pool (`<major>.<minor>` or `<major>.<minor>.<patch>`).",
				Required:            true,
			},
			"variant": schema.StringAttribute{
				Description:         "The template variant to match: 'standard', or 'nvidia' for GPU instance types (default: all variants).",
				MarkdownDescription: "The template variant to match: `standard`, or `nvidia` for GPU instance types (default: all variants).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(sksNodepoolTemplateVariants...),
				},
			},
			"templates": schema.ListNestedAttribute{
				Description:         "The list of SKS node pool templates, latest first.",
				MarkdownDescription: "The list of SKS node pool templates, latest first.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description:         "The SKS node pool template ID.",
							MarkdownDescription: "The SKS node pool template ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "The SKS node pool template name.",
							MarkdownDescription: "The SKS node pool template name.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							Description:         "The template registration date.",
							MarkdownDescription: "The template registration date.",
							Computed:            true,
						},
						"variant": schema.StringAttribute{
							Description:         "The template variant ('standard' or 'nvidia').",
							MarkdownDescription: "The template variant (`standard` or `nvidia`).",
							Computed:            true,
						},
					},
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

func (d *DataSourceNodepoolTemplateList) Configure(ctx context.Context, r datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceNodepoolTemplateList) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataSourceNodepoolTemplateListModel

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

	var opts []exoscale.ListSKSNodepoolTemplatesOpt
	if !state.Variant.IsNull() {
		opts = append(opts, exoscale.ListSKSNodepoolTemplatesWithVariant(exoscale.ListSKSNodepoolTemplatesVariant(state.Variant.ValueString())))
	}

	listResp, err := client.ListSKSNodepoolTemplates(ctx, state.KubeVersion.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("error listing SKS nodepool templates", err.Error())
		return
	}

	ids := make([]string, len(listResp.SKSNodepoolTemplates))
	state.Templates = make([]DataSourceNodepoolTemplateListItemModel, len(listResp.SKSNodepoolTemplates))
	for i, template := range listResp.SKSNodepoolTemplates {
		ids[i] = template.ID.String()
		state.Templates[i] = DataSourceNodepoolTemplateListItemModel{
			ID:        types.StringValue(template.ID.String()),
			Name:      types.StringValue(template.Name),
			CreatedAt: types.StringValue(template.CreatedAT.String()),
			Variant:   types.StringValue(template.Variant),
		}
	}

	state.ID = types.StringValue(fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))) //nolint:gosec

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
