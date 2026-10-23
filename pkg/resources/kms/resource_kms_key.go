package kms

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

var _ resource.Resource = &ResourceKMSKey{}
var _ resource.ResourceWithImportState = &ResourceKMSKey{}
var _ resource.ResourceWithValidateConfig = &ResourceKMSKey{}
var _ resource.ResourceWithModifyPlan = &ResourceKMSKey{}

// kmsKeyDeletionDelayDays is the minimum scheduled deletion delay accepted by the KMS API.
// Keys cannot be deleted immediately; they enter a pending-deletion state for at least this many days.
const kmsKeyDeletionDelayDays = 7

// ResourceKMSKeyModel holds the Terraform state for a KMS key.
type ResourceKMSKeyModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Zone        types.String `tfsdk:"zone"`
	Description types.String `tfsdk:"description"`
	MultiZone   types.Bool   `tfsdk:"multi_zone"`
	Usage       types.String `tfsdk:"usage"`
	KeySpec     types.String `tfsdk:"key_spec"`
	Source      types.String `tfsdk:"source"`
	Xks         types.Object `tfsdk:"xks"`
	Status      types.String `tfsdk:"status"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// ResourceKMSKeyXksModel holds the external key store reference of a KMS key.
type ResourceKMSKeyXksModel struct {
	KeyStoreID    types.String `tfsdk:"key_store_id"`
	ExternalKeyID types.String `tfsdk:"external_key_id"`
}

// Types returns nested data model types to be used for conversion.
func (m ResourceKMSKeyXksModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"key_store_id":    types.StringType,
		"external_key_id": types.StringType,
	}
}

var kmsKeySpecs = []string{
	string(exoscale.CreateKmsKeyRequestKeySpecAES256),
	string(exoscale.CreateKmsKeyRequestKeySpecECCNISTP256),
	string(exoscale.CreateKmsKeyRequestKeySpecECCNISTP384),
	string(exoscale.CreateKmsKeyRequestKeySpecECCNISTP521),
	string(exoscale.CreateKmsKeyRequestKeySpecECCEDWARDS25519),
	string(exoscale.CreateKmsKeyRequestKeySpecRSA3072),
	string(exoscale.CreateKmsKeyRequestKeySpecRSA4096),
	string(exoscale.CreateKmsKeyRequestKeySpecMLDSA65),
	string(exoscale.CreateKmsKeyRequestKeySpecMLDSA87),
}

// kmsSignOnlyKeySpecs are the key specs that only support the sign-verify usage.
// AES_256 only supports encrypt-decrypt, and RSA specs support both.
var kmsSignOnlyKeySpecs = []string{
	string(exoscale.CreateKmsKeyRequestKeySpecECCNISTP256),
	string(exoscale.CreateKmsKeyRequestKeySpecECCNISTP384),
	string(exoscale.CreateKmsKeyRequestKeySpecECCNISTP521),
	string(exoscale.CreateKmsKeyRequestKeySpecECCEDWARDS25519),
	string(exoscale.CreateKmsKeyRequestKeySpecMLDSA65),
	string(exoscale.CreateKmsKeyRequestKeySpecMLDSA87),
}

var externalKeyIDRegexp = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type ResourceKMSKey struct {
	client *exoscale.Client
}

func NewResourceKMSKey() resource.Resource {
	return &ResourceKMSKey{}
}

func (r *ResourceKMSKey) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_key"
}

func (r *ResourceKMSKey) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Exoscale KMS Keys.",
		Description:         "Manage Exoscale KMS Keys.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Description:         "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "❗ The name of the KMS Key.",
				Description:         "The name of the KMS Key.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Description:         "The Exoscale Zone name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "❗ A description of the KMS Key.",
				Description:         "A description of the KMS Key.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"multi_zone": schema.BoolAttribute{
				MarkdownDescription: "❗ Whether the key is replicated across multiple zones.",
				Description:         "Whether the key is replicated across multiple zones.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"usage": schema.StringAttribute{
				MarkdownDescription: "❗ The key usage purpose (`encrypt-decrypt` or `sign-verify`).",
				Description:         "The key usage purpose (encrypt-decrypt or sign-verify).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(exoscale.CreateKmsKeyRequestUsageEncryptDecrypt),
						string(exoscale.CreateKmsKeyRequestUsageSignVerify),
					),
				},
			},
			"key_spec": schema.StringAttribute{
				MarkdownDescription: "❗ The cryptographic key specification (e.g. `AES_256`).",
				Description:         "The cryptographic key specification (e.g. AES_256).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(kmsKeySpecs...),
				},
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "The source of the key material: `exoscale-kms`, or `external-key-store` when `xks` is set.",
				Description:         "The source of the key material: exoscale-kms, or external-key-store when xks is set.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"xks": schema.SingleNestedAttribute{
				MarkdownDescription: "❗ [BETA] Back the key with a key held in an External Key Store. Only `usage = \"encrypt-decrypt\"`, `key_spec = \"AES_256\"` and single-zone keys are supported.",
				Description:         "[BETA] Back the key with a key held in an External Key Store. Only usage encrypt-decrypt, key_spec AES_256 and single-zone keys are supported.",
				Optional:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"key_store_id": schema.StringAttribute{
						MarkdownDescription: "❗ The ID of the External Key Store.",
						Description:         "The ID of the External Key Store.",
						Required:            true,
					},
					"external_key_id": schema.StringAttribute{
						MarkdownDescription: "❗ The ID of the key as known to the XKS proxy.",
						Description:         "The ID of the key as known to the XKS proxy.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 128),
							stringvalidator.RegexMatches(externalKeyIDRegexp, "must only contain letters, digits, underscores, dots and hyphens"),
						},
					},
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the KMS Key.",
				Description:         "The current status of the KMS Key.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceKMSKey) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// ValidateConfig rejects settings that the API does not support:
// sign-verify keys with the default AES_256 key_spec, and anything else than single-zone
// AES_256 encrypt-decrypt keys when backed by an External Key Store.
func (r *ResourceKMSKey) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ResourceKMSKeyModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// key_spec defaults to AES_256, which only supports encrypt-decrypt.
	if data.Usage.ValueString() == string(exoscale.CreateKmsKeyRequestUsageSignVerify) && !data.KeySpec.IsUnknown() &&
		(data.KeySpec.IsNull() || data.KeySpec.ValueString() == string(exoscale.CreateKmsKeyRequestKeySpecAES256)) {
		resp.Diagnostics.AddAttributeError(
			path.Root("key_spec"),
			"Unsupported key_spec for sign-verify key",
			fmt.Sprintf("Keys with usage %q need an asymmetric key_spec; %q (the default) only supports %q.",
				exoscale.CreateKmsKeyRequestUsageSignVerify,
				exoscale.CreateKmsKeyRequestKeySpecAES256,
				exoscale.CreateKmsKeyRequestUsageEncryptDecrypt,
			),
		)
	}

	// usage defaults to encrypt-decrypt, which ECC and ML-DSA keys do not support.
	if slices.Contains(kmsSignOnlyKeySpecs, data.KeySpec.ValueString()) && !data.Usage.IsUnknown() &&
		(data.Usage.IsNull() || data.Usage.ValueString() == string(exoscale.CreateKmsKeyRequestUsageEncryptDecrypt)) {
		resp.Diagnostics.AddAttributeError(
			path.Root("usage"),
			"Unsupported usage for key_spec",
			fmt.Sprintf("Keys with key_spec %q only support usage %q.",
				data.KeySpec.ValueString(),
				exoscale.CreateKmsKeyRequestUsageSignVerify,
			),
		)
	}

	if data.Xks.IsNull() {
		return
	}

	if !data.Usage.IsNull() && !data.Usage.IsUnknown() &&
		data.Usage.ValueString() != string(exoscale.CreateKmsKeyRequestUsageEncryptDecrypt) {
		resp.Diagnostics.AddAttributeError(
			path.Root("usage"),
			"Unsupported usage for External Key Store key",
			fmt.Sprintf("Keys with xks only support usage %q.", exoscale.CreateKmsKeyRequestUsageEncryptDecrypt),
		)
	}

	if !data.KeySpec.IsNull() && !data.KeySpec.IsUnknown() &&
		data.KeySpec.ValueString() != string(exoscale.CreateKmsKeyRequestKeySpecAES256) {
		resp.Diagnostics.AddAttributeError(
			path.Root("key_spec"),
			"Unsupported key_spec for External Key Store key",
			fmt.Sprintf("Keys with xks only support key_spec %q.", exoscale.CreateKmsKeyRequestKeySpecAES256),
		)
	}

	if !data.MultiZone.IsNull() && !data.MultiZone.IsUnknown() && data.MultiZone.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("multi_zone"),
			"Unsupported multi_zone for External Key Store key",
			"Keys with xks cannot be multi-zone.",
		)
	}
}

// ModifyPlan prevents replacing a key without changing its name: Delete only schedules the deletion,
// and the name stays reserved by the pending-deletion key, so creating its replacement would fail
// after the current key has already been scheduled for deletion.
func (r *ResourceKMSKey) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to replace on create or destroy.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan, data ResourceKMSKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Name.Equal(state.Name) {
		return
	}

	// Every attribute but timeouts requires a replacement (see Update): mirror their plan modifiers.
	var changed []string
	if !plan.Zone.IsUnknown() && !plan.Zone.Equal(state.Zone) {
		changed = append(changed, "zone")
	}
	if !plan.Xks.IsUnknown() && !plan.Xks.Equal(state.Xks) {
		changed = append(changed, "xks")
	}
	for name, v := range map[string]struct{ config, plan, state attr.Value }{
		"description": {data.Description, plan.Description, state.Description},
		"multi_zone":  {data.MultiZone, plan.MultiZone, state.MultiZone},
		"usage":       {data.Usage, plan.Usage, state.Usage},
		"key_spec":    {data.KeySpec, plan.KeySpec, state.KeySpec},
	} {
		if !v.config.IsNull() && !v.plan.IsUnknown() && !v.plan.Equal(v.state) {
			changed = append(changed, name)
		}
	}
	if len(changed) == 0 {
		return
	}
	slices.Sort(changed)

	resp.Diagnostics.AddAttributeError(
		path.Root("name"),
		"KMS key cannot be replaced under the same name",
		fmt.Sprintf(
			"Changing %s requires a new KMS key. The current key is only scheduled for deletion (at least %d days) "+
				"and keeps the name %q until then, so its replacement cannot be created with the same name. "+
				"Revert the change, or also change the name to create a new key. "+
				"Data encrypted with the current key can only be decrypted with that key.",
			strings.Join(changed, ", "), kmsKeyDeletionDelayDays, state.Name.ValueString(),
		),
	)
}

func (r *ResourceKMSKey) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceKMSKeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(plan.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	createReq := exoscale.CreateKmsKeyRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if !plan.MultiZone.IsUnknown() {
		v := plan.MultiZone.ValueBool()
		createReq.MultiZone = &v
	}

	if !plan.Usage.IsUnknown() {
		createReq.Usage = exoscale.CreateKmsKeyRequestUsage(plan.Usage.ValueString())
	}

	if !plan.KeySpec.IsUnknown() {
		createReq.KeySpec = exoscale.CreateKmsKeyRequestKeySpec(plan.KeySpec.ValueString())
	}

	createReq.Source = exoscale.CreateKmsKeyRequestSourceExoscaleKms
	if !plan.Xks.IsNull() {
		var xks ResourceKMSKeyXksModel
		resp.Diagnostics.Append(plan.Xks.As(ctx, &xks, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		keyStoreID, err := exoscale.ParseUUID(xks.KeyStoreID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse key store ID", err.Error())
			return
		}

		createReq.Source = exoscale.CreateKmsKeyRequestSourceExternalKeyStore
		createReq.Xks = &exoscale.XksKey{
			XksID:         keyStoreID,
			ExternalKeyID: xks.ExternalKeyID.ValueString(),
		}
	}

	key, err := client.CreateKmsKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("API error creating KMS key", apiErrorMessage(err))
		return
	}

	plan.ID = types.StringValue(key.ID.String())
	plan.Status = types.StringValue(string(key.Status))
	plan.Source = types.StringValue(string(key.Source))

	if plan.Usage.IsUnknown() {
		plan.Usage = types.StringValue(key.Usage)
	}
	if plan.MultiZone.IsUnknown() {
		plan.MultiZone = types.BoolValue(*key.MultiZone)
	}
	if plan.Description.IsUnknown() {
		plan.Description = types.StringValue(key.Description)
	}
	if plan.KeySpec.IsUnknown() {
		plan.KeySpec = types.StringValue(key.KeySpec)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceKMSKey) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceKMSKeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse resource ID", err.Error())
		return
	}

	key, err := client.GetKmsKey(ctx, id)
	if err != nil {
		if isNotFound(err, kmsErrKeyNotFound) {
			tflog.Info(ctx, "KMS key not found, removing from state", map[string]any{})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API error reading KMS key", apiErrorMessage(err))
		return
	}

	// pending-deletion means the key is scheduled for deletion. treating it as gone.
	if key.Status == exoscale.GetKmsKeyResponseStatusPendingDeletion {
		tflog.Info(ctx, "KMS key is pending deletion, removing from state", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(key.Name)
	state.Status = types.StringValue(string(key.Status))
	state.Usage = types.StringValue(key.Usage)
	state.MultiZone = types.BoolValue(*key.MultiZone)
	state.Description = types.StringValue(key.Description)
	state.KeySpec = types.StringValue(key.KeySpec)
	state.Source = types.StringValue(string(key.Source))

	var prevXks ResourceKMSKeyXksModel
	if !state.Xks.IsNull() && !state.Xks.IsUnknown() {
		resp.Diagnostics.Append(state.Xks.As(ctx, &prevXks, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	state.Xks = types.ObjectNull(ResourceKMSKeyXksModel{}.Types())
	if key.Xks != nil {
		keyStoreID := key.Xks.XksID.String()
		// The API returns the UUID in lower case: keep the configured spelling to avoid a replacement.
		if strings.EqualFold(prevXks.KeyStoreID.ValueString(), keyStoreID) {
			keyStoreID = prevXks.KeyStoreID.ValueString()
		}

		xks, diags := types.ObjectValueFrom(ctx, ResourceKMSKeyXksModel{}.Types(), ResourceKMSKeyXksModel{
			KeyStoreID:    types.StringValue(keyStoreID),
			ExternalKeyID: types.StringValue(key.Xks.ExternalKeyID),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Xks = xks
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update makes no API call: all attributes but timeouts use RequiresReplace,
// so only the new timeouts have to be saved.
func (r *ResourceKMSKey) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceKMSKeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceKMSKey) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceKMSKeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse resource ID", err.Error())
		return
	}

	_, err = client.ScheduleKmsKeyDeletion(ctx, id, exoscale.ScheduleKmsKeyDeletionRequest{
		DelayDays: kmsKeyDeletionDelayDays,
	})
	if err != nil {
		// The key is already scheduled for deletion, which is all Delete does: nothing left to do.
		if isNotFound(err, kmsErrKeyNotFound) || isAPIError(err, kmsErrKeyPendingDeletion) {
			return
		}
		resp.Diagnostics.AddError("API error scheduling KMS key deletion", apiErrorMessage(err))
		return
	}
}

func (r *ResourceKMSKey) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf("Expected import identifier with format: id@zone. Got: %q", req.ID),
		)
		return
	}

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var t timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &t)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &ResourceKMSKeyModel{
		ID:       types.StringValue(idParts[0]),
		Zone:     types.StringValue(idParts[1]),
		Xks:      types.ObjectNull(ResourceKMSKeyXksModel{}.Types()),
		Timeouts: t,
	})...)
}
