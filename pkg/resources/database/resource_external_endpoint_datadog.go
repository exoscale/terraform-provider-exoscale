package database

import (
	"context"
	"fmt"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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

var _ resource.Resource = &ExternalEndpointDatadogResource{}
var _ resource.ResourceWithImportState = &ExternalEndpointDatadogResource{}

func NewExternalEndpointDatadogResource() resource.Resource {
	return &ExternalEndpointDatadogResource{}
}

type ExternalEndpointDatadogResource struct {
	client *v3.Client
}

type ExternalEndpointDatadogResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Name          types.String   `tfsdk:"name"`
	Zone          types.String   `tfsdk:"zone"`
	DatadogAPIKey types.String   `tfsdk:"datadog_api_key"`
	Site          types.String   `tfsdk:"site"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func (r *ExternalEndpointDatadogResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ExternalEndpointDatadogResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_external_endpoint_datadog"
}

func (r *ExternalEndpointDatadogResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Exoscale DBaaS [External Endpoint](https://community.exoscale.com/documentation/dbaas/) for Datadog integration.",
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
			"datadog_api_key": schema.StringAttribute{
				MarkdownDescription: "❗ Datadog API key.",
				Required:            true,
				Sensitive:           true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "Datadog site (`datadoghq.com`, `datadoghq.eu`, `us3.datadoghq.com`, `us5.datadoghq.com`, `ap1.datadoghq.com`, `ddog-gov.com`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(v3.EnumDatadogSiteDatadoghqCom),
						string(v3.EnumDatadogSiteDatadoghqEU),
						string(v3.EnumDatadogSiteUs3DatadoghqCom),
						string(v3.EnumDatadogSiteUs5DatadoghqCom),
						string(v3.EnumDatadogSiteAp1DatadoghqCom),
						string(v3.EnumDatadogSiteDdogGovCom),
					),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ExternalEndpointDatadogResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExternalEndpointDatadogResourceModel
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

	createReq := v3.DBAASEndpointDatadogInputCreate{
		Settings: &v3.DBAASEndpointDatadogInputCreateSettings{
			DatadogAPIKey: data.DatadogAPIKey.ValueString(),
			Site:          v3.EnumDatadogSite(data.Site.ValueString()),
		},
	}

	op, err := client.CreateDBAASExternalEndpointDatadog(ctx, data.Name.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating datadog external endpoint: %s", err))
		return
	}

	op, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating datadog external endpoint: %s", err))
		return
	}

	data.ID = types.StringValue(op.Reference.ID.String())

	if found := readDatadogEndpointIntoModel(ctx, client, &data, &resp.Diagnostics); !found {
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

func (r *ExternalEndpointDatadogResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExternalEndpointDatadogResourceModel
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

	found := readDatadogEndpointIntoModel(ctx, client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalEndpointDatadogResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ExternalEndpointDatadogResourceModel
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

	updateReq := v3.DBAASEndpointDatadogInputUpdate{
		Settings: &v3.DBAASEndpointDatadogInputUpdateSettings{
			DatadogAPIKey: planData.DatadogAPIKey.ValueString(),
			Site:          v3.EnumDatadogSite(planData.Site.ValueString()),
		},
	}

	op, err := client.UpdateDBAASExternalEndpointDatadog(ctx, endpointID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating datadog external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating datadog external endpoint: %s", err))
		return
	}

	planData.ID = stateData.ID
	planData.Timeouts = stateData.Timeouts

	if found := readDatadogEndpointIntoModel(ctx, client, &planData, &resp.Diagnostics); !found {
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

func (r *ExternalEndpointDatadogResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExternalEndpointDatadogResourceModel
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

	op, err := client.DeleteDBAASExternalEndpointDatadog(ctx, endpointID)
	if err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting datadog external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting datadog external endpoint: %s", err))
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": data.ID.ValueString()})
}

func (r *ExternalEndpointDatadogResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	endpointID, zone, err := parseZonedImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("import ID", fmt.Sprintf("error parsing import ID: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), endpointID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone"), zone)...)
}

func readDatadogEndpointIntoModel(ctx context.Context, client *v3.Client, data *ExternalEndpointDatadogResourceModel, diagnostics *diag.Diagnostics) bool {
	endpointID, err := v3.ParseUUID(data.ID.ValueString())
	if err != nil {
		diagnostics.AddError("parse ID", fmt.Sprintf("error parsing endpoint ID: %s", err))
		return false
	}

	endpoint, found := pollEndpoint(ctx, func() (*v3.DBAASExternalEndpointDatadogOutput, error) {
		return client.GetDBAASExternalEndpointDatadog(ctx, endpointID)
	}, diagnostics, "error reading datadog external endpoint")
	if !found {
		return false
	}

	data.Name = types.StringValue(endpoint.Name)
	if endpoint.Settings != nil {
		data.Site = types.StringValue(string(endpoint.Settings.Site))
	}
	// DatadogAPIKey is write-only - not returned by API, preserve existing value
	return true
}
