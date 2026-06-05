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

var _ resource.Resource = &ExternalEndpointOpensearchResource{}
var _ resource.ResourceWithImportState = &ExternalEndpointOpensearchResource{}

func NewExternalEndpointOpensearchResource() resource.Resource {
	return &ExternalEndpointOpensearchResource{}
}

type ExternalEndpointOpensearchResource struct {
	client *v3.Client
}

type ExternalEndpointOpensearchResourceModel struct {
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

func (r *ExternalEndpointOpensearchResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ExternalEndpointOpensearchResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_external_endpoint_opensearch"
}

func (r *ExternalEndpointOpensearchResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Exoscale DBaaS [External Endpoint](https://community.exoscale.com/documentation/dbaas/) for OpenSearch logs integration.",
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
				MarkdownDescription: "OpenSearch connection URL.",
				Required:            true,
			},
			"index_prefix": schema.StringAttribute{
				MarkdownDescription: "OpenSearch index prefix.",
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
				MarkdownDescription: "OpenSearch request timeout in seconds (10-120).",
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

func (r *ExternalEndpointOpensearchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExternalEndpointOpensearchResourceModel
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

	settings := &v3.DBAASEndpointOpensearchInputCreateSettings{
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

	op, err := client.CreateDBAASExternalEndpointOpensearch(ctx, data.Name.ValueString(), v3.DBAASEndpointOpensearchInputCreate{Settings: settings})
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating opensearch external endpoint: %s", err))
		return
	}

	op, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating opensearch external endpoint: %s", err))
		return
	}

	data.ID = types.StringValue(op.Reference.ID.String())

	if found := readOpensearchEndpointIntoModel(ctx, client, &data, &resp.Diagnostics); !found {
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

func (r *ExternalEndpointOpensearchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExternalEndpointOpensearchResourceModel
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

	found := readOpensearchEndpointIntoModel(ctx, client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalEndpointOpensearchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ExternalEndpointOpensearchResourceModel
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

	settings := &v3.DBAASEndpointOpensearchInputUpdateSettings{}
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

	op, err := client.UpdateDBAASExternalEndpointOpensearch(ctx, endpointID, v3.DBAASEndpointOpensearchInputUpdate{Settings: settings})
	if err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating opensearch external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating opensearch external endpoint: %s", err))
		return
	}

	planData.ID = stateData.ID
	planData.Timeouts = stateData.Timeouts

	if found := readOpensearchEndpointIntoModel(ctx, client, &planData, &resp.Diagnostics); !found {
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

func (r *ExternalEndpointOpensearchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExternalEndpointOpensearchResourceModel
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

	op, err := client.DeleteDBAASExternalEndpointOpensearch(ctx, endpointID)
	if err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting opensearch external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting opensearch external endpoint: %s", err))
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": data.ID.ValueString()})
}

func (r *ExternalEndpointOpensearchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	endpointID, zone, err := parseZonedImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("import ID", fmt.Sprintf("error parsing import ID: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), endpointID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone"), zone)...)
}

func readOpensearchEndpointIntoModel(ctx context.Context, client *v3.Client, data *ExternalEndpointOpensearchResourceModel, diagnostics *diag.Diagnostics) bool {
	endpointID, err := v3.ParseUUID(data.ID.ValueString())
	if err != nil {
		diagnostics.AddError("parse ID", fmt.Sprintf("error parsing endpoint ID: %s", err))
		return false
	}

	endpoint, found := pollEndpoint(ctx, func() (*v3.DBAASEndpointOpensearchOutput, error) {
		return client.GetDBAASExternalEndpointOpensearch(ctx, endpointID)
	}, diagnostics, "error reading opensearch external endpoint")
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
