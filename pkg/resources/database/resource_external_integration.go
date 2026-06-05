package database

import (
	"context"
	"errors"
	"fmt"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ExternalIntegrationResource{}
var _ resource.ResourceWithImportState = &ExternalIntegrationResource{}

func NewExternalIntegrationResource() resource.Resource {
	return &ExternalIntegrationResource{}
}

type ExternalIntegrationResource struct {
	client *v3.Client
}

type ExternalIntegrationResourceModel struct {
	ID                types.String   `tfsdk:"id"`
	SourceServiceName types.String   `tfsdk:"source_service_name"`
	DestEndpointID    types.String   `tfsdk:"dest_endpoint_id"`
	Type              types.String   `tfsdk:"type"`
	Zone              types.String   `tfsdk:"zone"`
	Description       types.String   `tfsdk:"description"`
	Status            types.String   `tfsdk:"status"`
	DestEndpointName  types.String   `tfsdk:"dest_endpoint_name"`
	SourceServiceType types.String   `tfsdk:"source_service_type"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

func (r *ExternalIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ExternalIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_external_integration"
}

func (r *ExternalIntegrationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Exoscale DBaaS [External Integrations](https://community.exoscale.com/documentation/dbaas/) — attach a DBaaS service to an external endpoint for log/metrics forwarding.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource (integration UUID).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_service_name": schema.StringAttribute{
				MarkdownDescription: "❗ The name of the DBaaS service to integrate.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dest_endpoint_id": schema.StringAttribute{
				MarkdownDescription: "❗ The UUID of the external endpoint to attach the integration to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "❗ The type of the external endpoint.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(v3.EnumExternalEndpointTypesPrometheus),
						string(v3.EnumExternalEndpointTypesDatadog),
						string(v3.EnumExternalEndpointTypesOpensearch),
						string(v3.EnumExternalEndpointTypesRsyslog),
						string(v3.EnumExternalEndpointTypesElasticsearch),
					),
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
			"description": schema.StringAttribute{
				MarkdownDescription: "Integration description.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Integration status.",
				Computed:            true,
			},
			"dest_endpoint_name": schema.StringAttribute{
				MarkdownDescription: "Name of the destination endpoint.",
				Computed:            true,
			},
			"source_service_type": schema.StringAttribute{
				MarkdownDescription: "Type of the source DBaaS service.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ExternalIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExternalIntegrationResourceModel
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

	endpointID, err := v3.ParseUUID(data.DestEndpointID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("parse endpoint ID", fmt.Sprintf("error parsing dest_endpoint_id: %s", err))
		return
	}

	op, err := client.AttachDBAASServiceToEndpoint(
		ctx,
		data.SourceServiceName.ValueString(),
		v3.AttachDBAASServiceToEndpointRequest{
			DestEndpointID: endpointID,
			Type:           v3.EnumExternalEndpointTypes(data.Type.ValueString()),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating dbaas external integration: %s", err))
		return
	}

	op, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating dbaas external integration: %s", err))
		return
	}

	data.ID = types.StringValue(op.Reference.ID.String())

	integrationID, err := v3.ParseUUID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("parse integration ID", fmt.Sprintf("error parsing integration ID: %s", err))
		return
	}

	integration, err := client.GetDBAASExternalIntegration(ctx, integrationID)
	if err != nil {
		resp.Diagnostics.AddError("read", fmt.Sprintf("error reading dbaas external integration: %s", err))
		return
	}

	readIntegrationIntoModel(integration, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, "resource created", map[string]any{"id": data.ID.ValueString()})
}

func (r *ExternalIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExternalIntegrationResourceModel
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

	integrationID, err := v3.ParseUUID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("parse ID", fmt.Sprintf("error parsing integration ID: %s", err))
		return
	}

	integration, err := client.GetDBAASExternalIntegration(ctx, integrationID)
	if errors.Is(err, v3.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("read", fmt.Sprintf("error reading dbaas external integration: %s", err))
		return
	}

	readIntegrationIntoModel(integration, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is not needed; all mutable fields use RequiresReplace.
func (r *ExternalIntegrationResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

func (r *ExternalIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExternalIntegrationResourceModel
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

	integrationID, err := v3.ParseUUID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("parse ID", fmt.Sprintf("error parsing integration ID: %s", err))
		return
	}

	op, err := client.DetachDBAASServiceFromEndpoint(
		ctx,
		data.SourceServiceName.ValueString(),
		v3.DetachDBAASServiceFromEndpointRequest{
			IntegrationID: integrationID,
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting dbaas external integration: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting dbaas external integration: %s", err))
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": data.ID.ValueString()})
}

func (r *ExternalIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	integrationID, zone, err := parseZonedImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("import ID", fmt.Sprintf("error parsing import ID: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), integrationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone"), zone)...)
}

func readIntegrationIntoModel(integration *v3.DBAASExternalIntegration, data *ExternalIntegrationResourceModel) {
	data.ID = types.StringValue(integration.IntegrationID.String())
	data.Description = types.StringValue(integration.Description)
	data.Status = types.StringValue(string(integration.Status))
	data.DestEndpointID = types.StringValue(integration.DestEndpointID)
	data.DestEndpointName = types.StringValue(integration.DestEndpointName)
	data.SourceServiceName = types.StringValue(integration.SourceServiceName)
	data.SourceServiceType = types.StringValue(string(integration.SourceServiceType))
	data.Type = types.StringValue(string(integration.Type))
}
