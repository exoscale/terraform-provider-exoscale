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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = &ClickhouseUserResource{}
var _ resource.ResourceWithImportState = &ClickhouseUserResource{}

func NewClickhouseUserResource() resource.Resource {
	return &ClickhouseUserResource{}
}

type ClickhouseUserResource struct {
	UserResource
}

type ClickhouseUserResourceModel struct {
	UserResourceModel
	Roles    types.Set    `tfsdk:"roles"`
	UserUUID types.String `tfsdk:"user_uuid"`
}

func (r *ClickhouseUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ClickhouseUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_clickhouse_user"
}

type roleObj struct {
	UUID string `tfsdk:"uuid"`
}

func rolesObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"uuid": basetypes.StringType{},
	}}
}

func rolesToSet(ctx context.Context, roles []exoscale.DBAASClickhouseUserRole) (types.Set, diag.Diagnostics) {
	objType := rolesObjectType()

	if len(roles) == 0 {
		return types.SetNull(objType), nil
	}

	objs := make([]roleObj, 0, len(roles))
	for _, role := range roles {
		objs = append(objs, roleObj{UUID: string(role.Uuid)})
	}

	v, diags := types.SetValueFrom(ctx, objType, objs)
	return v, diags
}

func (r *ClickhouseUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage service users for a ClickHouse Exoscale [Database Services (DBaaS)](https://community.exoscale.com/documentation/dbaas/).",
		Attributes: buildUserAttributes(map[string]schema.Attribute{
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of the ClickHouse user. If not set, the API will generate one. Cannot be retrieved after creation (use the generated value or reset via update).",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"roles": schema.SetNestedAttribute{
				MarkdownDescription: "Roles granted to the user. Each role is identified by its UUID.",
				Optional:            true,
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The UUID of the role to grant.",
							Required:            true,
						},
					},
				},
			},
			"user_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the ClickHouse user (internal, used for delete).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		}),
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ClickhouseUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClickhouseUserResourceModel
	ReadResource(ctx, req, resp, &data, r.client)
}

func (r *ClickhouseUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClickhouseUserResourceModel
	CreateResource(ctx, req, resp, &data, r.client)
}

func (r *ClickhouseUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData ClickhouseUserResourceModel
	UpdateResource(ctx, req, resp, &stateData, &planData, r.client)
}

func (r *ClickhouseUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClickhouseUserResourceModel
	DeleteResource(ctx, req, resp, &data, r.client)
}

func (r *ClickhouseUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	var data ClickhouseUserResourceModel

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

func (data *ClickhouseUserResourceModel) CreateResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	createRequest := exoscale.CreateDBAASClickhouseUserRequest{
		Username: exoscale.DBAASUserUsername(data.Username.ValueString()),
	}

	if !data.Password.IsUnknown() && !data.Password.IsNull() {
		createRequest.Password = exoscale.DBAASUserPassword(data.Password.ValueString())
	}

	roles := []exoscale.DBAASClickhouseUserRoleInput{}
	if !data.Roles.IsUnknown() && !data.Roles.IsNull() {
		roleObjs := []roleObj{}
		dg := data.Roles.ElementsAs(ctx, &roleObjs, false)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}
		for _, role := range roleObjs {
			roles = append(roles, exoscale.DBAASClickhouseUserRoleInput{
				Uuid: exoscale.UUID(role.UUID),
			})
		}
	}
	if len(roles) > 0 {
		createRequest.Roles = roles
	}

	secrets, err := client.CreateDBAASClickhouseUser(ctx, data.Service.ValueString(), createRequest)
	if err != nil {
		if !errors.Is(err, exoscale.ErrConflict) {
			diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to create service user, got error %s", err.Error()),
			)
			return
		}
		// User already exists: idempotent create, adopt it below.
	} else {
		data.Password = basetypes.NewStringValue(secrets.Password)
	}

	// Get the user's UUID from the service. Retry a few times to ride out
	// eventual consistency between the create and the service listing.
	for i := 0; i < 5; i++ {
		svc, err := client.GetDBAASServiceClickhouse(ctx, data.Service.ValueString())
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service clickhouse, got error: %s", err))
			return
		}
		found := false
		for _, u := range svc.Users {
			if string(u.Username) == data.Username.ValueString() {
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// The user uuid used for delete is the ACL user uuid; the service's user
	// list omits the uuid, so resolve it from the ACL config.
	if err := readClickhouseUserRoles(ctx, data, client, diagnostics); err != nil {
		return
	}
	if data.UserUUID.IsNull() || data.UserUUID.IsUnknown() {
		aclConfig, err := client.GetDBAASClickhouseAclConfig(ctx, data.Service.ValueString())
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service clickhouse ACL config, got error: %s", err))
			return
		}
		for _, aclUser := range aclConfig.Users {
			if string(aclUser.Username) == data.Username.ValueString() {
				data.UserUUID = basetypes.NewStringValue(string(aclUser.Uuid))
				break
			}
		}
		if data.UserUUID.IsNull() || data.UserUUID.IsUnknown() {
			diagnostics.AddError("Client Error", "Unable to find newly created user for the service")
			return
		}
	}
	data.Type = basetypes.NewStringValue("clickhouse")

	// On the idempotent (already-exists) path with a configured password,
	// reset the password so state matches config.
	if err != nil && !data.Password.IsUnknown() && !data.Password.IsNull() {
		secrets, err := client.ResetDBAASClickhouseUserPassword(ctx, data.Service.ValueString(), data.Username.ValueString(), exoscale.ResetDBAASClickhouseUserPasswordRequest{
			Password: exoscale.DBAASUserPassword(data.Password.ValueString()),
		})
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset clickhouse user password: %s", err))
			return
		}
		data.Password = basetypes.NewStringValue(secrets.Password)
	}

	if !data.Roles.IsUnknown() && !data.Roles.IsNull() {
		// configured; keep config value
	} else {
		if err := readClickhouseUserRoles(ctx, data, client, diagnostics); err != nil {
			return
		}
	}
}

// readClickhouseUserRoles resolves the roles granted to the user from the
// service role list and ACL config into data.Roles.
func readClickhouseUserRoles(ctx context.Context, data *ClickhouseUserResourceModel, client *exoscale.Client, diagnostics *diag.Diagnostics) error {
	roles, err := client.ListDBAASClickhouseRoles(ctx, data.Service.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service clickhouse roles, got error: %s", err))
		return err
	}

	roleNames := map[string]exoscale.DBAASClickhouseUserRole{}
	for _, r := range roles.Roles {
		roleNames[r.Uuid] = exoscale.DBAASClickhouseUserRole{
			Uuid: exoscale.UUID(r.Uuid),
			Name: string(r.Name),
		}
	}

	aclConfig, err := client.GetDBAASClickhouseAclConfig(ctx, data.Service.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service clickhouse ACL config, got error: %s", err))
		return err
	}

	data.Roles = types.SetNull(rolesObjectType())
	for _, aclUser := range aclConfig.Users {
		if string(aclUser.Username) == data.Username.ValueString() {
			granted := make([]exoscale.DBAASClickhouseUserRole, 0, len(aclUser.Roles))
			for _, uuid := range aclUser.Roles {
				if role, ok := roleNames[string(uuid.Uuid)]; ok {
					granted = append(granted, role)
				}
			}
			s, dg := rolesToSet(ctx, granted)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return fmt.Errorf("unable to encode roles")
			}
			data.Roles = s
			break
		}
	}
	return nil
}

func (data *ClickhouseUserResourceModel) DeleteResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	if data.UserUUID.IsNull() || data.UserUUID.IsUnknown() {
		diagnostics.AddError("Client Error", "user_uuid is not set in state; cannot delete")
		return
	}

	op, err := client.DeleteDBAASClickhouseUser(ctx, data.Service.ValueString(), exoscale.UUID(data.UserUUID.ValueString()))
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
			fmt.Sprintf("Unable to wait for service user deletion, got error %s", err.Error()),
		)
		return
	}
}

func (data *ClickhouseUserResourceModel) ReadResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) (clearState bool) {
	svc, err := client.GetDBAASServiceClickhouse(ctx, data.Service.ValueString())
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return true
		}
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service clickhouse user, got error: %s", err))
		return false
	}

	for _, user := range svc.Users {
		if string(user.Username) == data.Username.ValueString() {
			// The service user list omits the uuid on some environments;
			// fall back to the stored value / ACL uuid rather than clearing it.
			if string(user.Uuid) != "" {
				data.UserUUID = basetypes.NewStringValue(string(user.Uuid))
			}
			data.Type = basetypes.NewStringValue("clickhouse")

			if data.UserUUID.IsNull() || data.UserUUID.IsUnknown() {
				aclConfig, err := client.GetDBAASClickhouseAclConfig(ctx, data.Service.ValueString())
				if err == nil {
					for _, aclUser := range aclConfig.Users {
						if string(aclUser.Username) == data.Username.ValueString() {
							data.UserUUID = basetypes.NewStringValue(string(aclUser.Uuid))
							break
						}
					}
				}
			}

			if err := readClickhouseUserRoles(ctx, data, client, diagnostics); err != nil {
				return false
			}

			return false
		}
	}

	return true
}

func (data *ClickhouseUserResourceModel) UpdateResource(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	req := exoscale.ResetDBAASClickhouseUserPasswordRequest{}
	if !data.Password.IsUnknown() && !data.Password.IsNull() {
		req.Password = exoscale.DBAASUserPassword(data.Password.ValueString())
	}

	secrets, err := client.ResetDBAASClickhouseUserPassword(ctx, data.Service.ValueString(), data.Username.ValueString(), req)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset clickhouse user password: %s", err))
		return
	}

	data.Password = basetypes.NewStringValue(secrets.Password)
	data.Type = basetypes.NewStringValue("clickhouse")

	if data.Roles.IsUnknown() || data.Roles.IsNull() {
		if err := readClickhouseUserRoles(ctx, data, client, diagnostics); err != nil {
			return
		}
	}
}

func (data *ClickhouseUserResourceModel) WaitForService(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics) {
	_, err := waitForDBAASServiceReadyForFn(ctx, client.GetDBAASServiceClickhouse, data.Service.ValueString(), func(t *exoscale.DBAASServiceClickhouse) bool {
		return t.State == exoscale.EnumServiceStateRunning && len(t.Users) > 0
	})

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database service ClickHouse: %s", err.Error()))
	}
}

func (data *ClickhouseUserResourceModel) GetTimeouts() timeouts.Value {
	return data.Timeouts
}

func (data *ClickhouseUserResourceModel) SetTimeouts(t timeouts.Value) {
	data.Timeouts = t
}

func (data *ClickhouseUserResourceModel) GetID() basetypes.StringValue {
	return data.Id
}

func (data *ClickhouseUserResourceModel) GetZone() basetypes.StringValue {
	return data.Zone
}

func (data *ClickhouseUserResourceModel) GenerateID() {
	data.Id = basetypes.NewStringValue(fmt.Sprintf("%s/%s", data.Service.ValueString(), data.Username.ValueString()))
}
