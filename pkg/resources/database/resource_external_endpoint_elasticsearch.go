package database

import (
	"context"
	"fmt"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ExternalEndpointElasticsearchResource{}
var _ resource.ResourceWithImportState = &ExternalEndpointElasticsearchResource{}

func NewExternalEndpointElasticsearchResource() resource.Resource {
	return &ExternalEndpointElasticsearchResource{}
}

type ExternalEndpointElasticsearchResource struct {
	client *v3.Client
}

type ExternalEndpointElasticsearchResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Name         types.String   `tfsdk:"name"`
	Zone         types.String   `tfsdk:"zone"`
	URL          types.String   `tfsdk:"url"`
	IndexPrefix  types.String   `tfsdk:"index_prefix"`
	IndexDaysMax types.Int64    `tfsdk:"index_days_max"`
	Timeout      types.Int64    `tfsdk:"timeout"`
	CA           types.String   `tfsdk:"ca"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func (r *ExternalEndpointElasticsearchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ExternalEndpointElasticsearchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_external_endpoint_elasticsearch"
}

func (r *ExternalEndpointElasticsearchResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Exoscale DBaaS [External Endpoint](https://community.exoscale.com/documentation/dbaas/) for Elasticsearch logs integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "❗ The endpoint name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(40),
				},
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Elasticsearch connection URL.",
				Required:            true,
			},
			"index_prefix": schema.StringAttribute{
				MarkdownDescription: "Elasticsearch index prefix.",
				Required:            true,
			},
			"index_days_max": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of days of logs to keep (1-10000).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 10000),
				},
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Elasticsearch request timeout in seconds (10-120).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(10, 120),
				},
			},
			"ca": schema.StringAttribute{
				MarkdownDescription: "PEM encoded CA certificate.",
				Optional:            true,
				Sensitive:           true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ExternalEndpointElasticsearchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExternalEndpointElasticsearchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, diags := data.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	settings := &v3.DBAASEndpointElasticsearchInputCreateSettings{
		URL:         data.URL.ValueString(),
		IndexPrefix: data.IndexPrefix.ValueString(),
	}
	if !data.IndexDaysMax.IsNull() && !data.IndexDaysMax.IsUnknown() {
		settings.IndexDaysMax = data.IndexDaysMax.ValueInt64()
	}
	if !data.Timeout.IsNull() && !data.Timeout.IsUnknown() {
		settings.Timeout = data.Timeout.ValueInt64()
	}
	if !data.CA.IsNull() {
		settings.CA = data.CA.ValueString()
	}

	op, err := client.CreateDBAASExternalEndpointElasticsearch(ctx, data.Name.ValueString(), v3.DBAASEndpointElasticsearchInputCreate{Settings: settings})
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating elasticsearch external endpoint: %s", err))
		return
	}

	op, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating elasticsearch external endpoint: %s", err))
		return
	}

	data.ID = types.StringValue(op.Reference.ID.String())

	if found := readElasticsearchEndpointIntoModel(ctx, client, &data, &resp.Diagnostics); !found {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError("create", "endpoint not found after create")
		}
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, "resource created", map[string]any{"id": data.ID.ValueString()})
}

func (r *ExternalEndpointElasticsearchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExternalEndpointElasticsearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	found := readElasticsearchEndpointIntoModel(ctx, client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalEndpointElasticsearchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ExternalEndpointElasticsearchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, diags := stateData.Timeouts.Update(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, v3.ZoneName(planData.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	endpointID, err := v3.ParseUUID(stateData.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("parse ID", fmt.Sprintf("error parsing endpoint ID: %s", err))
		return
	}

	settings := &v3.DBAASEndpointElasticsearchInputUpdateSettings{}
	if !planData.URL.IsNull() {
		settings.URL = planData.URL.ValueString()
	}
	if !planData.IndexPrefix.IsNull() {
		settings.IndexPrefix = planData.IndexPrefix.ValueString()
	}
	if !planData.IndexDaysMax.IsNull() && !planData.IndexDaysMax.IsUnknown() {
		settings.IndexDaysMax = planData.IndexDaysMax.ValueInt64()
	}
	if !planData.Timeout.IsNull() && !planData.Timeout.IsUnknown() {
		settings.Timeout = planData.Timeout.ValueInt64()
	}
	if !planData.CA.IsNull() {
		settings.CA = planData.CA.ValueString()
	}

	op, err := client.UpdateDBAASExternalEndpointElasticsearch(ctx, endpointID, v3.DBAASEndpointElasticsearchInputUpdate{Settings: settings})
	if err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating elasticsearch external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating elasticsearch external endpoint: %s", err))
		return
	}

	planData.ID = stateData.ID
	planData.Timeouts = stateData.Timeouts

	if found := readElasticsearchEndpointIntoModel(ctx, client, &planData, &resp.Diagnostics); !found {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError("update", "endpoint not found after update")
		}
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
	tflog.Trace(ctx, "resource updated", map[string]any{"id": planData.ID.ValueString()})
}

func (r *ExternalEndpointElasticsearchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExternalEndpointElasticsearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, diags := data.Timeouts.Delete(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	endpointID, err := v3.ParseUUID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("parse ID", fmt.Sprintf("error parsing endpoint ID: %s", err))
		return
	}

	op, err := client.DeleteDBAASExternalEndpointElasticsearch(ctx, endpointID)
	if err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting elasticsearch external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting elasticsearch external endpoint: %s", err))
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": data.ID.ValueString()})
}

func (r *ExternalEndpointElasticsearchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	endpointID, zone, err := parseZonedImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("import ID", fmt.Sprintf("error parsing import ID: %s", err))
		return
	}

	var data ExternalEndpointElasticsearchResourceModel
	var t timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &t)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Timeouts = t
	data.ID = types.StringValue(endpointID)
	data.Zone = types.StringValue(zone)

	timeout, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	found := readElasticsearchEndpointIntoModel(ctx, client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("import", "endpoint not found")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func readElasticsearchEndpointIntoModel(ctx context.Context, client *v3.Client, data *ExternalEndpointElasticsearchResourceModel, diagnostics *diag.Diagnostics) bool {
	endpointID, err := v3.ParseUUID(data.ID.ValueString())
	if err != nil {
		diagnostics.AddError("parse ID", fmt.Sprintf("error parsing endpoint ID: %s", err))
		return false
	}

	endpoint, found := pollEndpoint(ctx, func() (*v3.DBAASEndpointElasticsearchOutput, error) {
		return client.GetDBAASExternalEndpointElasticsearch(ctx, endpointID)
	}, diagnostics, "error reading elasticsearch external endpoint")
	if !found {
		return false
	}

	data.Name = types.StringValue(endpoint.Name)
	if endpoint.Settings != nil {
		data.URL = types.StringValue(endpoint.Settings.URL)
		data.IndexPrefix = types.StringValue(endpoint.Settings.IndexPrefix)
		data.IndexDaysMax = types.Int64Value(endpoint.Settings.IndexDaysMax)
		data.Timeout = types.Int64Value(endpoint.Settings.Timeout)
	}
	// CA is write-only (in secrets), preserve existing value
	return true
}
