package database

import (
	"context"
	"errors"
	"fmt"

	exoscale "github.com/exoscale/egoscale/v3"
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

const markdownDescriptionKafkaSchemaRegistryACL = "Manage a Kafka Schema Registry ACL entry of an Exoscale [Database Service (DBaaS)](https://community.exoscale.com/product/dbaas/).\n\n" +
	"The API has no update operation for ACL entries: changing any argument replaces the entry."

var _ resource.Resource = &KafkaSchemaRegistryACLResource{}
var _ resource.ResourceWithImportState = &KafkaSchemaRegistryACLResource{}

func NewKafkaSchemaRegistryACLResource() resource.Resource {
	return &KafkaSchemaRegistryACLResource{}
}

type KafkaSchemaRegistryACLResource struct {
	client *exoscale.Client
}

type KafkaSchemaRegistryACLResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Service    types.String `tfsdk:"service"`
	Username   types.String `tfsdk:"username"`
	Resource   types.String `tfsdk:"resource"`
	Permission types.String `tfsdk:"permission"`
	Zone       types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *KafkaSchemaRegistryACLResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *KafkaSchemaRegistryACLResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_kafka_schema_registry_acl"
}

func (r *KafkaSchemaRegistryACLResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage a Kafka Schema Registry ACL entry of an Exoscale Database Service (DBaaS).",
		MarkdownDescription: markdownDescriptionKafkaSchemaRegistryACL,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The ID of the ACL entry, assigned by the service.",
				MarkdownDescription: "The ID of the ACL entry, assigned by the service.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Description:         "❗ The name of the Kafka database service.",
				MarkdownDescription: "❗ The name of the Kafka database service.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description:         "❗ The Kafka username or username pattern the entry applies to.",
				MarkdownDescription: "❗ The Kafka username or username pattern (e.g. `app-*`) the entry applies to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"resource": schema.StringAttribute{
				Description:         "❗ The Schema Registry resource name or pattern the entry applies to (Config: or Subject:<name>).",
				MarkdownDescription: "❗ The Schema Registry resource name or pattern the entry applies to: `Config:` or `Subject:<name>` (e.g. `Subject:orders-*`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 249),
				},
			},
			"permission": schema.StringAttribute{
				Description:         "❗ The permission granted on the resource (schema_registry_read or schema_registry_write).",
				MarkdownDescription: "❗ The permission granted on the resource (`schema_registry_read` or `schema_registry_write`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(exoscale.DBAASKafkaSchemaRegistryAclEntryPermissionSchemaRegistryRead),
						string(exoscale.DBAASKafkaSchemaRegistryAclEntryPermissionSchemaRegistryWrite),
					),
				},
			},
			"zone": schema.StringAttribute{
				Description:         "❗ The Exoscale Zone name.",
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
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
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *KafkaSchemaRegistryACLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KafkaSchemaRegistryACLResourceModel
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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	op, err := client.CreateDBAASKafkaSchemaRegistryAclConfig(ctx, data.Service.ValueString(), exoscale.DBAASKafkaSchemaRegistryAclEntry{
		Username:   data.Username.ValueString(),
		Resource:   data.Resource.ValueString(),
		Permission: exoscale.DBAASKafkaSchemaRegistryAclEntryPermission(data.Permission.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Kafka Schema Registry ACL entry, got error %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Kafka Schema Registry ACL entry, got error %s", err))
		return
	}

	// The operation references the service, not the entry: the API rejects identical
	// entries, so the (username, resource, permission) triple identifies the new one.
	acls, err := client.GetDBAASKafkaAclConfig(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Kafka ACL configuration, got error: %s", err))
		return
	}

	for _, acl := range acls.SchemaRegistryAcl {
		if acl.Username == data.Username.ValueString() &&
			acl.Resource == data.Resource.ValueString() &&
			string(acl.Permission) == data.Permission.ValueString() {
			data.ID = types.StringValue(string(acl.ID))

			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			tflog.Trace(ctx, "resource created", map[string]any{"id": data.ID.ValueString()})
			return
		}
	}

	resp.Diagnostics.AddError("Client Error", "Unable to find newly created Kafka Schema Registry ACL entry for the service")
}

func (r *KafkaSchemaRegistryACLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KafkaSchemaRegistryACLResourceModel
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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	acls, err := client.GetDBAASKafkaAclConfig(ctx, data.Service.ValueString())
	if errors.Is(err, exoscale.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Kafka ACL configuration, got error: %s", err))
		return
	}

	for _, acl := range acls.SchemaRegistryAcl {
		if string(acl.ID) != data.ID.ValueString() {
			continue
		}

		utils.RefreshString(&data.Username, acl.Username)
		utils.RefreshString(&data.Resource, acl.Resource)
		utils.RefreshString(&data.Permission, string(acl.Permission))

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *KafkaSchemaRegistryACLResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// Nothing to do here: the API has no update operation for ACL entries, and every
	// argument requires a replacement
}

func (r *KafkaSchemaRegistryACLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KafkaSchemaRegistryACLResourceModel
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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	op, err := client.DeleteDBAASKafkaSchemaRegistryAclConfig(ctx, data.Service.ValueString(), data.ID.ValueString())
	if errors.Is(err, exoscale.ErrNotFound) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Kafka Schema Registry ACL entry, got error %s", err))
		return
	}

	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Kafka Schema Registry ACL entry, got error %s", err))
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": data.ID.ValueString()})
}

func (r *KafkaSchemaRegistryACLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	service, aclID, zone, err := parseKafkaACLImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Import Identifier", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), aclID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), service)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone"), zone)...)
}
