package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = &ValkeyUserResource{}
var _ resource.ResourceWithImportState = &ValkeyUserResource{}

func NewValkeyUserResource() resource.Resource {
	return &ValkeyUserResource{}
}

type ValkeyUserResource struct {
	UserResource
}

type ValkeyUserResourceModel struct {
	UserResourceModel
}

func (r *ValkeyUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ValkeyUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_valkey_user"
}

func (r *ValkeyUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage service users for a Valkey Exoscale [Database Services (DBaaS)](https://community.exoscale.com/documentation/dbaas/).",
		Attributes:          commonAttributes,
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ValkeyUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ValkeyUserResourceModel
	ReadResource(ctx, req, resp, &data, r.client)
}

func (r *ValkeyUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ValkeyUserResourceModel
	CreateResource(ctx, req, resp, &data, r.client)
}

func (r *ValkeyUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData ValkeyUserResourceModel
	UpdateResource(ctx, req, resp, &stateData, &planData, r.client)
}

func (r *ValkeyUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ValkeyUserResourceModel
	DeleteResource(ctx, req, resp, &data, r.client)
}

func (r *ValkeyUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service/username@zone. Got: %q", req.ID),
		)
		return
	}

	userID := idParts[0]
	zone := idParts[1]

	id := strings.Split(userID, "/")

	if len(id) != 2 || id[0] == "" || id[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service/username@zone. Got: %q", req.ID),
		)
	}

	serviceName := id[0]
	username := id[1]

	var data ValkeyUserResourceModel

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Timeouts = timeouts

	data.Id = types.StringValue(userID)
	data.Username = types.StringValue(username)
	data.Service = types.StringValue(serviceName)
	data.Zone = types.StringValue(zone)

	ReadResourceForImport(ctx, req, resp, &data, r.client)
}

func (data *ValkeyUserResourceModel) CreateResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {

	createRequest := exoscale.CreateDBAASValkeyUserRequest{
		Username: exoscale.DBAASUserUsername(data.Username.ValueString()),
	}

	op, err := client.CreateDBAASValkeyUser(ctx, data.Service.ValueString(), createRequest)
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create service user, got error %s", err.Error()),
		)
		return
	}

	_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create service user, got error %s", err.Error()),
		)
		return
	}

	svc, err := client.GetDBAASServiceValkey(ctx, data.Service.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service valkey, got error: %s", err))
		return
	}

	for _, user := range svc.Users {
		if user.Username == data.Username.ValueString() {
			data.Type = basetypes.NewStringValue(user.Type)

			pass, err := client.RevealDBAASValkeyUserPassword(ctx, data.Service.ValueString(), data.Username.ValueString())
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reveal valkey user password, got error: %s", err))
				return
			}

			data.Password = basetypes.NewStringValue(pass.Password)

			return
		}
	}
	diagnostics.AddError("Client Error", "Unable to find newly created user for the service")
}

func (data *ValkeyUserResourceModel) DeleteResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {

	op, err := client.DeleteDBAASValkeyUser(ctx, data.Service.ValueString(), data.Username.ValueString())
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete service user, got error %s", err.Error()),
		)
		return
	}

	_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete service user, got error %s", err.Error()),
		)
		return
	}
}

func (data *ValkeyUserResourceModel) ReadResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) (clearState bool) {

	svc, err := client.GetDBAASServiceValkey(ctx, data.Service.ValueString())
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return true
		}
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service valkey user, got error: %s", err))
		return false
	}

	for _, user := range svc.Users {
		if user.Username == data.Username.ValueString() {
			data.Type = basetypes.NewStringValue(user.Type)

			pass, err := client.RevealDBAASValkeyUserPassword(ctx, data.Service.ValueString(), data.Username.ValueString())
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reveal valkey user password, got error: %s", err))
				return false
			}

			data.Password = basetypes.NewStringValue(pass.Password)

			return false
		}
	}

	return true
}

func (data *ValkeyUserResourceModel) UpdateResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	// All fields are immutable; replaces are triggered automatically.
}

func (data *ValkeyUserResourceModel) WaitForService(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	_, err := waitForDBAASServiceReadyForFn(ctx, client.GetDBAASServiceValkey, data.Service.ValueString(), func(t *exoscale.DBAASServiceValkey) bool {
		return t.State == exoscale.EnumServiceStateRunning && len(t.Users) > 0
	})

	time.Sleep(SERVICE_READY_DELAY)

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database service Valkey %s", err.Error()))
	}
}

func (data *ValkeyUserResourceModel) GetTimeouts() timeouts.Value {
	return data.Timeouts
}

func (data *ValkeyUserResourceModel) SetTimeouts(t timeouts.Value) {
	data.Timeouts = t
}

func (data *ValkeyUserResourceModel) GetID() basetypes.StringValue {
	return data.Id
}

func (data *ValkeyUserResourceModel) GetZone() basetypes.StringValue {
	return data.Zone
}

func (data *ValkeyUserResourceModel) GenerateID() {
	data.Id = basetypes.NewStringValue(fmt.Sprintf("%s/%s", data.Service.ValueString(), data.Username.ValueString()))
}
