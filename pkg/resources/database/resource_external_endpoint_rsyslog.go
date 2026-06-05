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

var _ resource.Resource = &ExternalEndpointRsyslogResource{}
var _ resource.ResourceWithImportState = &ExternalEndpointRsyslogResource{}

func NewExternalEndpointRsyslogResource() resource.Resource {
	return &ExternalEndpointRsyslogResource{}
}

type ExternalEndpointRsyslogResource struct {
	client *v3.Client
}

type ExternalEndpointRsyslogResourceModel struct {
	ID             types.String   `tfsdk:"id"`
	Name           types.String   `tfsdk:"name"`
	Zone           types.String   `tfsdk:"zone"`
	Server         types.String   `tfsdk:"server"`
	Port           types.Int64    `tfsdk:"port"`
	TLS            types.Bool     `tfsdk:"tls"`
	Format         types.String   `tfsdk:"format"`
	Logline        types.String   `tfsdk:"logline"`
	MaxMessageSize types.Int64    `tfsdk:"max_message_size"`
	SD             types.String   `tfsdk:"sd"`
	CA             types.String   `tfsdk:"ca"`
	Cert           types.String   `tfsdk:"cert"`
	Key            types.String   `tfsdk:"key"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func (r *ExternalEndpointRsyslogResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ExternalEndpointRsyslogResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_external_endpoint_rsyslog"
}

func (r *ExternalEndpointRsyslogResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Exoscale DBaaS [External Endpoint](https://community.exoscale.com/documentation/dbaas/) for Rsyslog logs integration.",
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
			"server": schema.StringAttribute{
				MarkdownDescription: "Rsyslog server IP address or hostname.",
				Required:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Rsyslog server port (1-65535).",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"tls": schema.BoolAttribute{
				MarkdownDescription: "Require TLS.",
				Required:            true,
			},
			"format": schema.StringAttribute{
				MarkdownDescription: "Syslog message format (`rfc3164`, `rfc5424`, `custom`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(v3.EnumRsyslogFormatRfc3164),
						string(v3.EnumRsyslogFormatRfc5424),
						string(v3.EnumRsyslogFormatCustom),
					),
				},
			},
			"logline": schema.StringAttribute{
				MarkdownDescription: "Custom syslog message format (required when format is `custom`).",
				Optional:            true,
				Computed:            true,
			},
			"max_message_size": schema.Int64Attribute{
				MarkdownDescription: "Rsyslog max message size (2048-2147483647).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(2048, 2147483647),
				},
			},
			"sd": schema.StringAttribute{
				MarkdownDescription: "Structured data block for log message.",
				Optional:            true,
				Computed:            true,
			},
			"ca": schema.StringAttribute{
				MarkdownDescription: "PEM encoded CA certificate.",
				Optional:            true,
				Sensitive:           true,
			},
			"cert": schema.StringAttribute{
				MarkdownDescription: "PEM encoded client certificate.",
				Optional:            true,
				Sensitive:           true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "PEM encoded client key.",
				Optional:            true,
				Sensitive:           true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ExternalEndpointRsyslogResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExternalEndpointRsyslogResourceModel
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

	tls := data.TLS.ValueBool()
	settings := &v3.DBAASEndpointRsyslogInputCreateSettings{
		Server: data.Server.ValueString(),
		Port:   data.Port.ValueInt64(),
		Tls:    &tls,
		Format: v3.EnumRsyslogFormat(data.Format.ValueString()),
	}
	if !data.Logline.IsNull() {
		settings.Logline = data.Logline.ValueString()
	}
	if !data.MaxMessageSize.IsNull() && !data.MaxMessageSize.IsUnknown() {
		settings.MaxMessageSize = data.MaxMessageSize.ValueInt64()
	}
	if !data.SD.IsNull() {
		settings.SD = data.SD.ValueString()
	}
	if !data.CA.IsNull() {
		settings.CA = data.CA.ValueString()
	}
	if !data.Cert.IsNull() {
		settings.Cert = data.Cert.ValueString()
	}
	if !data.Key.IsNull() {
		settings.Key = data.Key.ValueString()
	}

	op, err := client.CreateDBAASExternalEndpointRsyslog(ctx, data.Name.ValueString(), v3.DBAASEndpointRsyslogInputCreate{Settings: settings})
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating rsyslog external endpoint: %s", err))
		return
	}

	op, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create", fmt.Sprintf("error creating rsyslog external endpoint: %s", err))
		return
	}

	data.ID = types.StringValue(op.Reference.ID.String())

	if found := readRsyslogEndpointIntoModel(ctx, client, &data, &resp.Diagnostics); !found {
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

func (r *ExternalEndpointRsyslogResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExternalEndpointRsyslogResourceModel
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

	found := readRsyslogEndpointIntoModel(ctx, client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalEndpointRsyslogResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ExternalEndpointRsyslogResourceModel
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

	tls := planData.TLS.ValueBool()
	settings := &v3.DBAASEndpointRsyslogInputUpdateSettings{
		Server: planData.Server.ValueString(),
		Port:   planData.Port.ValueInt64(),
		Tls:    &tls,
		Format: v3.EnumRsyslogFormat(planData.Format.ValueString()),
	}
	if !planData.Logline.IsNull() {
		settings.Logline = planData.Logline.ValueString()
	}
	if !planData.MaxMessageSize.IsNull() && !planData.MaxMessageSize.IsUnknown() {
		settings.MaxMessageSize = planData.MaxMessageSize.ValueInt64()
	}
	if !planData.SD.IsNull() {
		settings.SD = planData.SD.ValueString()
	}
	if !planData.CA.IsNull() {
		settings.CA = planData.CA.ValueString()
	}
	if !planData.Cert.IsNull() {
		settings.Cert = planData.Cert.ValueString()
	}
	if !planData.Key.IsNull() {
		settings.Key = planData.Key.ValueString()
	}

	op, err := client.UpdateDBAASExternalEndpointRsyslog(ctx, endpointID, v3.DBAASEndpointRsyslogInputUpdate{Settings: settings})
	if err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating rsyslog external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("update", fmt.Sprintf("error updating rsyslog external endpoint: %s", err))
		return
	}

	planData.ID = stateData.ID
	planData.Timeouts = stateData.Timeouts

	if found := readRsyslogEndpointIntoModel(ctx, client, &planData, &resp.Diagnostics); !found {
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

func (r *ExternalEndpointRsyslogResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExternalEndpointRsyslogResourceModel
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

	op, err := client.DeleteDBAASExternalEndpointRsyslog(ctx, endpointID)
	if err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting rsyslog external endpoint: %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete", fmt.Sprintf("error deleting rsyslog external endpoint: %s", err))
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": data.ID.ValueString()})
}

func (r *ExternalEndpointRsyslogResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	endpointID, zone, err := parseZonedImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("import ID", fmt.Sprintf("error parsing import ID: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), endpointID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone"), zone)...)
}

func readRsyslogEndpointIntoModel(ctx context.Context, client *v3.Client, data *ExternalEndpointRsyslogResourceModel, diagnostics *diag.Diagnostics) bool {
	endpointID, err := v3.ParseUUID(data.ID.ValueString())
	if err != nil {
		diagnostics.AddError("parse ID", fmt.Sprintf("error parsing endpoint ID: %s", err))
		return false
	}

	endpoint, found := pollEndpoint(ctx, func() (*v3.DBAASExternalEndpointRsyslogOutput, error) {
		return client.GetDBAASExternalEndpointRsyslog(ctx, endpointID)
	}, diagnostics, "error reading rsyslog external endpoint")
	if !found {
		return false
	}

	data.Name = types.StringValue(endpoint.Name)
	if endpoint.Settings != nil {
		data.Server = types.StringValue(endpoint.Settings.Server)
		data.Port = types.Int64Value(endpoint.Settings.Port)
		if endpoint.Settings.Tls != nil {
			data.TLS = types.BoolValue(*endpoint.Settings.Tls)
		}
		data.Format = types.StringValue(string(endpoint.Settings.Format))
		data.Logline = types.StringValue(endpoint.Settings.Logline)
		data.MaxMessageSize = types.Int64Value(endpoint.Settings.MaxMessageSize)
		data.SD = types.StringValue(endpoint.Settings.SD)
	}
	// CA, Cert, Key are write-only (in secrets), preserve existing values
	return true
}
