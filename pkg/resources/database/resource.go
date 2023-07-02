package database

import (
	"context"
	"fmt"
	"strings"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func NewResource() resource.Resource {
	return &Resource{}
}

// Resource defines the DBaaS Service resource implementation.
type Resource struct {
	client *exoscale.Client
	env    string
}

// ResourceModel describes the generic DBaaS Service resource data model.
type ResourceModel struct {
	Id                    types.String `tfsdk:"id"`
	CreatedAt             types.String `tfsdk:"created_at"`
	DiskSize              types.Int64  `tfsdk:"disk_size"`
	MaintenanceDOW        types.String `tfsdk:"maintenance_dow"`
	MaintenanceTime       types.String `tfsdk:"maintenance_time"`
	Name                  types.String `tfsdk:"name"`
	NodeCPUs              types.Int64  `tfsdk:"node_cpus"`
	NodeMemory            types.Int64  `tfsdk:"node_memory"`
	Nodes                 types.Int64  `tfsdk:"nodes"`
	Plan                  types.String `tfsdk:"plan"`
	State                 types.String `tfsdk:"state"`
	CA                    types.String `tfsdk:"ca_certificate"`
	TerminationProtection types.Bool   `tfsdk:"termination_protection"`
	Type                  types.String `tfsdk:"type"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
	Zone                  types.String `tfsdk:"zone"`

	Pg      *ResourcePgModel      `tfsdk:"pg"`
	Mysql   *ResourceMysqlModel   `tfsdk:"mysql"`
	Redis   *ResourceRedisModel   `tfsdk:"redis"`
	Grafana *ResourceGrafanaModel `tfsdk:"grafana"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Exoscale [Database Services (DBaaS)](https://community.exoscale.com/documentation/dbaas/).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The creation date of the database service.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disk_size": schema.Int64Attribute{
				MarkdownDescription: "The disk size of the database service.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"maintenance_dow": schema.StringAttribute{
				MarkdownDescription: "The day of week to perform the automated database service maintenance (`never`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.Expressions{
						path.MatchRoot("maintenance_time"),
					}...),
					stringvalidator.OneOf(
						"never",
						"monday",
						"tuesday",
						"wednesday",
						"thursday",
						"friday",
						"saturday",
						"sunday",
					),
				},
			},
			"maintenance_time": schema.StringAttribute{
				MarkdownDescription: "The time of day to perform the automated database service maintenance (`HH:MM:SS`)",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.Expressions{
						path.MatchRoot("maintenance_dow"),
					}...),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the database service.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"node_cpus": schema.Int64Attribute{
				MarkdownDescription: "The number of CPUs of the database service.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"node_memory": schema.Int64Attribute{
				MarkdownDescription: "The amount of memory of the database service.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"nodes": schema.Int64Attribute{
				MarkdownDescription: "The number of nodes of the database service.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"plan": schema.StringAttribute{
				MarkdownDescription: "The plan of the database service (use the [Exoscale CLI](https://github.com/exoscale/cli/) - `exo dbaas type show <TYPE>` - for reference).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The current state of the database service.",
				Computed:            true,
			},
			"ca_certificate": schema.StringAttribute{
				MarkdownDescription: "CA Certificate required to reach a DBaaS service through a TLS-protected connection.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"termination_protection": schema.BoolAttribute{
				MarkdownDescription: "The database service protection boolean flag against termination/power-off.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the database service (`kafka`, `mysql`, `opensearch`, `pg`, `redis`, `grafana`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(ServicesList...),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "The date of the latest database service update.",
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"pg":       ResourcePgSchema,
			"mysql":    ResourceMysqlSchema,
			"redis":    ResourceRedisSchema,
			"grafana":  ResourceGrafanaSchema,
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV2
	r.env = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Environment
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	data.Id = data.Name
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, data.Zone.ValueString()))

	switch data.Type.ValueString() {
	case "pg":
		r.createPg(ctx, &data, &resp.Diagnostics)
	case "mysql":
		r.createMysql(ctx, &data, &resp.Diagnostics)
	case "redis":
		r.createRedis(ctx, &data, &resp.Diagnostics)
	case "grafana":
		r.createGrafana(ctx, &data, &resp.Diagnostics)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource created", map[string]interface{}{
		"id": data.Id,
	})
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	data.Id = data.Name
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, data.Zone.ValueString()))

	switch data.Type.ValueString() {
	case "pg":
		r.readPg(ctx, &data, &resp.Diagnostics)
	case "mysql":
		r.readMysql(ctx, &data, &resp.Diagnostics)
	case "redis":
		r.readRedis(ctx, &data, &resp.Diagnostics)
	case "grafana":
		r.readGrafana(ctx, &data, &resp.Diagnostics)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource read done", map[string]interface{}{
		"id": data.Id,
	})
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData ResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	// Read Terraform state data (for comparison) into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := stateData.Timeouts.Update(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, planData.Zone.ValueString()))

	switch planData.Type.ValueString() {
	case "pg":
		r.updatePg(ctx, &stateData, &planData, &resp.Diagnostics)
	case "mysql":
		r.updateMysql(ctx, &stateData, &planData, &resp.Diagnostics)
	case "redis":
		r.updateRedis(ctx, &stateData, &planData, &resp.Diagnostics)
	case "grafana":
		r.updateGrafana(ctx, &stateData, &planData, &resp.Diagnostics)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)

	tflog.Trace(ctx, "resource updated", map[string]interface{}{
		"id": planData.Id,
	})
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.Timeouts.Delete(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, data.Zone.ValueString()))

	err := r.client.DeleteDatabaseService(ctx, data.Zone.ValueString(), &exoscale.DatabaseService{Name: data.Id.ValueStringPointer()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete database service, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]interface{}{
		"id": data.Id,
	})
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: name@zone. Got: %q", req.ID),
		)
		return
	}

	var data ResourceModel

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Timeouts = timeouts

	data.Id = types.StringValue(idParts[0])
	data.Name = types.StringValue(idParts[0])
	data.Zone = types.StringValue(idParts[1])

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, data.Zone.ValueString()))

	services, err := r.client.ListDatabaseServices(ctx, data.Zone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list Database Services: %s", err))
		return
	}

	for _, s := range services {
		if *s.Name == data.Id.ValueString() {
			data.Type = types.StringPointerValue(s.Type)
			break
		}
	}

	switch data.Type.ValueString() {
	case "pg":
		r.readPg(ctx, &data, &resp.Diagnostics)
	case "mysql":
		r.readMysql(ctx, &data, &resp.Diagnostics)
	case "redis":
		r.readRedis(ctx, &data, &resp.Diagnostics)
	case "grafana":
		r.readGrafana(ctx, &data, &resp.Diagnostics)
	default:
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Database service %q not found in zone %q", data.Id.ValueString(), data.Zone.ValueString()))
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource imported", map[string]interface{}{
		"id": data.Id,
	})
}
