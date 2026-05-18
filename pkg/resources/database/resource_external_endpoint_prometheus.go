package database

import (
	"context"
	"errors"
	"fmt"
	"time"

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

var _ resource.Resource = &ExternalEndpointPrometheusResource{}
var _ resource.ResourceWithImportState = &ExternalEndpointPrometheusResource{}

func NewExternalEndpointPrometheusResource() resource.Resource {
	return &ExternalEndpointPrometheusResource{}
}

type ExternalEndpointPrometheusResource struct {
	client *v3.Client
}

type ExternalEndpointPrometheusResourceModel struct {
	ID                types.String   `tfsdk:"id"`
	Name              types.String   `tfsdk:"name"`
	Zone              types.String   `tfsdk:"zone"`
	BasicAuthUsername types.String   `tfsdk:"basic_auth_username"`
	BasicAuthPassword types.String   `tfsdk:"basic_auth_password"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

func (r *ExternalEndpointPrometheusResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ExternalEndpointPrometheusResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_external_endpoint_prometheus"
}

func (r *ExternalEndpointPrometheusResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Exoscale DBaaS [External Endpoint](https://community.exoscale.com/documentation/dbaas/) for Prometheus metrics integration.",
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
			"basic_auth_username": schema.StringAttribute{
				MarkdownDescription: "Prometheus basic auth username (5-32 characters).",
				Optional:            true,
				Computed:            true,
			},
			"basic_auth_password": schema.StringAttribute{
				MarkdownDescription: "Prometheus basic auth password (8-64 characters). Not returned by the API after creation.",
				Optional:            true,
				Sensitive:           true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ExternalEndpointPrometheusResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExternalEndpointPrometheusResourceModel
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

	payload := v3.DBAASEndpointPrometheusPayload{}
	if !data.BasicAuthUsername.IsNull() || !data.BasicAuthPassword.IsNull() {
		settings := &v3.DBAASEndpointPrometheusPayloadSettings{}
		if !data.BasicAuthUsername.IsNull() {
			settings.BasicAuthUsername = data.BasicAuthUsername.ValueString()
		}
		if !data.BasicAuthPassword.IsNull() {
			settings.BasicAuthPassword = data.BasicAuthPassword.ValueString()
		}
		payload.Settings = settings
	}

	op, err := client.CreateDBAASExternalEndpointPrometheus(ctx, data.Name.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating prometheus external endpoint: %s", err))
		return
	}

	op, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating prometheus external endpoint: %s", err))
		return
	}

	data.ID = types.StringValue(op.Reference.ID.String())

	if found := readPrometheusEndpointIntoModel(ctx, client, &data, &resp.Diagnostics); !found {
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

func (r *ExternalEndpointPrometheusResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExternalEndpointPrometheusResourceModel
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

	found := readPrometheusEndpointIntoModel(ctx, client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalEndpointPrometheusResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ExternalEndpointPrometheusResourceModel
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

	payload := v3.DBAASEndpointPrometheusPayload{}
	if !planData.BasicAuthUsername.IsNull() || !planData.BasicAuthPassword.IsNull() {
		settings := &v3.DBAASEndpointPrometheusPayloadSettings{}
		if !planData.BasicAuthUsername.IsNull() {
			settings.BasicAuthUsername = planData.BasicAuthUsername.ValueString()
		}
		if !planData.BasicAuthPassword.IsNull() {
			settings.BasicAuthPassword = planData.BasicAuthPassword.ValueString()
		}
		payload.Settings = settings
	}

	op, err := client.UpdateDBAASExternalEndpointPrometheus(ctx, endpointID, payload)
	if err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating prometheus external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating prometheus external endpoint: %s", err))
		return
	}

	planData.ID = stateData.ID
	planData.Timeouts = stateData.Timeouts

	if found := readPrometheusEndpointIntoModel(ctx, client, &planData, &resp.Diagnostics); !found {
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

func (r *ExternalEndpointPrometheusResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExternalEndpointPrometheusResourceModel
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

	op, err := client.DeleteDBAASExternalEndpointPrometheus(ctx, endpointID)
	if err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting prometheus external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting prometheus external endpoint: %s", err))
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": data.ID.ValueString()})
}

func (r *ExternalEndpointPrometheusResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	endpointID, zone, err := parseZonedImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("import ID", fmt.Sprintf("error parsing import ID: %s", err))
		return
	}

	var data ExternalEndpointPrometheusResourceModel
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

	found := readPrometheusEndpointIntoModel(ctx, client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("import", "endpoint not found")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func readPrometheusEndpointIntoModel(ctx context.Context, client *v3.Client, data *ExternalEndpointPrometheusResourceModel, diagnostics *diag.Diagnostics) bool {
	endpointID, err := v3.ParseUUID(data.ID.ValueString())
	if err != nil {
		diagnostics.AddError("parse ID", fmt.Sprintf("error parsing endpoint ID: %s", err))
		return false
	}

	var endpoint *v3.DBAASEndpointExternalPrometheusOutput
	// Poll up to 3 times for the endpoint to appear after creation
	for i := range 3 {
		endpoint, err = client.GetDBAASExternalEndpointPrometheus(ctx, endpointID)
		if err == nil {
			break
		}
		if errors.Is(err, v3.ErrNotFound) && i < 2 {
			select {
			case <-ctx.Done():
				diagnostics.AddError("context cancelled", ctx.Err().Error())
				return false
			case <-time.After(3 * time.Second):
			}
			continue
		}
		if errors.Is(err, v3.ErrNotFound) {
			return false
		}
		diagnostics.AddError("read", fmt.Sprintf("error reading prometheus external endpoint: %s", err))
		return false
	}

	data.Name = types.StringValue(endpoint.Name)
	if endpoint.Settings != nil {
		data.BasicAuthUsername = types.StringValue(endpoint.Settings.BasicAuthUsername)
	} else {
		data.BasicAuthUsername = types.StringValue("")
	}
	// BasicAuthPassword is write-only - not returned by API, preserve existing value
	return true
}
