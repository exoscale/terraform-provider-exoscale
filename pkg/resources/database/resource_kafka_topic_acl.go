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

const markdownDescriptionKafkaTopicACL = "Manage a Kafka topic ACL entry of an Exoscale [Database Service (DBaaS)](https://community.exoscale.com/product/dbaas/).\n\n" +
	"The API has no update operation for ACL entries: changing any argument replaces the entry."

var _ resource.Resource = &KafkaTopicACLResource{}
var _ resource.ResourceWithImportState = &KafkaTopicACLResource{}

func NewKafkaTopicACLResource() resource.Resource {
	return &KafkaTopicACLResource{}
}

type KafkaTopicACLResource struct {
	client *exoscale.Client
}

type KafkaTopicACLResourceModel struct {
	Id         types.String `tfsdk:"id"`
	AclID      types.String `tfsdk:"acl_id"`
	Service    types.String `tfsdk:"service"`
	Username   types.String `tfsdk:"username"`
	Topic      types.String `tfsdk:"topic"`
	Permission types.String `tfsdk:"permission"`
	Zone       types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *KafkaTopicACLResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *KafkaTopicACLResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_kafka_topic_acl"
}

func (r *KafkaTopicACLResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage a Kafka topic ACL entry of an Exoscale Database Service (DBaaS).",
		MarkdownDescription: markdownDescriptionKafkaTopicACL,
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
			"topic": schema.StringAttribute{
				Description:         "❗ The Kafka topic name or pattern the entry applies to.",
				MarkdownDescription: "❗ The Kafka topic name or pattern (e.g. `orders-*`) the entry applies to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 249),
				},
			},
			"permission": schema.StringAttribute{
				Description:         "❗ The permission granted on the topic (admin, read, readwrite or write).",
				MarkdownDescription: "❗ The permission granted on the topic (`admin`, `read`, `readwrite` or `write`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(exoscale.DBAASKafkaTopicAclEntryPermissionAdmin),
						string(exoscale.DBAASKafkaTopicAclEntryPermissionRead),
						string(exoscale.DBAASKafkaTopicAclEntryPermissionReadwrite),
						string(exoscale.DBAASKafkaTopicAclEntryPermissionWrite),
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

func (r *KafkaTopicACLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KafkaTopicACLResourceModel
	ReadResource(ctx, req, resp, &data, r.client)
}

func (r *KafkaTopicACLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KafkaTopicACLResourceModel
	CreateResource(ctx, req, resp, &data, r.client)
}

func (r *KafkaTopicACLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData KafkaTopicACLResourceModel
	UpdateResource(ctx, req, resp, &stateData, &planData, r.client)
}

func (r *KafkaTopicACLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KafkaTopicACLResourceModel
	DeleteResource(ctx, req, resp, &data, r.client)
}

func (r *KafkaTopicACLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	service, aclID, zone, err := parseKafkaACLImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Import Identifier", err.Error())
		return
	}

	var data KafkaTopicACLResourceModel

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

func (data *KafkaTopicACLResourceModel) CreateResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	op, err := client.CreateDBAASKafkaTopicAclConfig(ctx, data.Service.ValueString(), exoscale.DBAASKafkaTopicAclEntry{
		Username:   data.Username.ValueString(),
		Topic:      data.Topic.ValueString(),
		Permission: exoscale.DBAASKafkaTopicAclEntryPermission(data.Permission.ValueString()),
	})
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create Kafka topic ACL entry, got error %s", err.Error()),
		)
		return
	}

	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create Kafka topic ACL entry, got error %s", err.Error()),
		)
		return
	}

	// The operation references the service, not the entry: the API rejects identical
	// entries, so the (username, topic, permission) triple identifies the new one.
	acls, _ := getKafkaACLConfig(ctx, client, data.Service.ValueString(), diagnostics)
	if diagnostics.HasError() {
		return
	}
	if acls != nil {
		for _, acl := range acls.TopicAcl {
			if acl.Username == data.Username.ValueString() &&
				acl.Topic == data.Topic.ValueString() &&
				string(acl.Permission) == data.Permission.ValueString() {
				data.AclID = types.StringValue(string(acl.ID))
				data.GenerateID()
				return
			}
		}
	}

	diagnostics.AddError("Client Error", "Unable to find newly created Kafka topic ACL entry for the service")
}

func (data *KafkaTopicACLResourceModel) ReadResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) (clearState bool) {
	acls, clearState := getKafkaACLConfig(ctx, client, data.Service.ValueString(), diagnostics)
	if clearState || acls == nil {
		return clearState
	}

	for _, acl := range acls.TopicAcl {
		if string(acl.ID) != data.AclID.ValueString() {
			continue
		}

		utils.RefreshString(&data.Username, acl.Username)
		utils.RefreshString(&data.Topic, acl.Topic)
		utils.RefreshString(&data.Permission, string(acl.Permission))

		return false
	}

	return true
}

func (data *KafkaTopicACLResourceModel) UpdateResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	// Nothing to do here: the API has no update operation for ACL entries, and every
	// argument requires a replacement
}

func (data *KafkaTopicACLResourceModel) DeleteResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	deleteKafkaACL(ctx, client, func() (*exoscale.Operation, error) {
		return client.DeleteDBAASKafkaTopicAclConfig(ctx, data.Service.ValueString(), data.AclID.ValueString())
	}, diagnostics)
}

func (data *KafkaTopicACLResourceModel) WaitForService(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	// Nothing to wait for: ACL entries can be managed as soon as the service exists,
	// including while it is still rebuilding
}

func (data *KafkaTopicACLResourceModel) GetTimeouts() timeouts.Value {
	return data.Timeouts
}

func (data *KafkaTopicACLResourceModel) SetTimeouts(t timeouts.Value) {
	data.Timeouts = t
}

func (data *KafkaTopicACLResourceModel) GetID() basetypes.StringValue {
	return data.Id
}

func (data *KafkaTopicACLResourceModel) GetZone() basetypes.StringValue {
	return data.Zone
}

func (data *KafkaTopicACLResourceModel) GenerateID() {
	data.Id = basetypes.NewStringValue(fmt.Sprintf("%s/%s", data.Service.ValueString(), data.AclID.ValueString()))
}
