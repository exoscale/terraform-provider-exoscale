package database

import (
	"context"
	"fmt"

	exoscale "github.com/exoscale/egoscale/v3"
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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	Id         types.String `tfsdk:"id"`
	AclID      types.String `tfsdk:"acl_id"`
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
				Description:         "The ID of this resource, computed as service/acl_id.",
				MarkdownDescription: "The ID of this resource, computed as `service/acl_id`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"acl_id": schema.StringAttribute{
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

func (r *KafkaSchemaRegistryACLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KafkaSchemaRegistryACLResourceModel
	ReadResource(ctx, req, resp, &data, r.client)
}

func (r *KafkaSchemaRegistryACLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KafkaSchemaRegistryACLResourceModel
	CreateResource(ctx, req, resp, &data, r.client)
}

func (r *KafkaSchemaRegistryACLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData KafkaSchemaRegistryACLResourceModel
	UpdateResource(ctx, req, resp, &stateData, &planData, r.client)
}

func (r *KafkaSchemaRegistryACLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KafkaSchemaRegistryACLResourceModel
	DeleteResource(ctx, req, resp, &data, r.client)
}

func (r *KafkaSchemaRegistryACLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	service, aclID, zone, err := parseKafkaACLImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Import Identifier", err.Error())
		return
	}

	var data KafkaSchemaRegistryACLResourceModel

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Timeouts = timeouts

	data.Service = types.StringValue(service)
	data.AclID = types.StringValue(aclID)
	data.Zone = types.StringValue(zone)

	ReadResourceForImport(ctx, req, resp, &data, r.client)
}

func (data *KafkaSchemaRegistryACLResourceModel) CreateResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	op, err := client.CreateDBAASKafkaSchemaRegistryAclConfig(ctx, data.Service.ValueString(), exoscale.DBAASKafkaSchemaRegistryAclEntry{
		Username:   data.Username.ValueString(),
		Resource:   data.Resource.ValueString(),
		Permission: exoscale.DBAASKafkaSchemaRegistryAclEntryPermission(data.Permission.ValueString()),
	})
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create Kafka Schema Registry ACL entry, got error %s", err.Error()),
		)
		return
	}

	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create Kafka Schema Registry ACL entry, got error %s", err.Error()),
		)
		return
	}

	// The operation references the service, not the entry: the API rejects identical
	// entries, so the (username, resource, permission) triple identifies the new one.
	acls, _ := getKafkaACLConfig(ctx, client, data.Service.ValueString(), diagnostics)
	if diagnostics.HasError() {
		return
	}
	if acls != nil {
		for _, acl := range acls.SchemaRegistryAcl {
			if acl.Username == data.Username.ValueString() &&
				acl.Resource == data.Resource.ValueString() &&
				string(acl.Permission) == data.Permission.ValueString() {
				data.AclID = types.StringValue(string(acl.ID))
				data.GenerateID()
				return
			}
		}
	}

	diagnostics.AddError("Client Error", "Unable to find newly created Kafka Schema Registry ACL entry for the service")
}

func (data *KafkaSchemaRegistryACLResourceModel) ReadResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) (clearState bool) {
	acls, clearState := getKafkaACLConfig(ctx, client, data.Service.ValueString(), diagnostics)
	if clearState || acls == nil {
		return clearState
	}

	for _, acl := range acls.SchemaRegistryAcl {
		if string(acl.ID) != data.AclID.ValueString() {
			continue
		}

		utils.RefreshString(&data.Username, acl.Username)
		utils.RefreshString(&data.Resource, acl.Resource)
		utils.RefreshString(&data.Permission, string(acl.Permission))

		return false
	}

	return true
}

func (data *KafkaSchemaRegistryACLResourceModel) UpdateResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	// Nothing to do here: the API has no update operation for ACL entries, and every
	// argument requires a replacement
}

func (data *KafkaSchemaRegistryACLResourceModel) DeleteResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	deleteKafkaACL(ctx, client, func() (*exoscale.Operation, error) {
		return client.DeleteDBAASKafkaSchemaRegistryAclConfig(ctx, data.Service.ValueString(), data.AclID.ValueString())
	}, diagnostics)
}

func (data *KafkaSchemaRegistryACLResourceModel) WaitForService(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	// Nothing to wait for: ACL entries can be managed as soon as the service exists,
	// including while it is still rebuilding
}

func (data *KafkaSchemaRegistryACLResourceModel) GetTimeouts() timeouts.Value {
	return data.Timeouts
}

func (data *KafkaSchemaRegistryACLResourceModel) SetTimeouts(t timeouts.Value) {
	data.Timeouts = t
}

func (data *KafkaSchemaRegistryACLResourceModel) GetID() basetypes.StringValue {
	return data.Id
}

func (data *KafkaSchemaRegistryACLResourceModel) GetZone() basetypes.StringValue {
	return data.Zone
}

func (data *KafkaSchemaRegistryACLResourceModel) GenerateID() {
	data.Id = basetypes.NewStringValue(fmt.Sprintf("%s/%s", data.Service.ValueString(), data.AclID.ValueString()))
}
