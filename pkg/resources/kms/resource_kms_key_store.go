package kms

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

var _ resource.Resource = &ResourceKMSKeyStore{}
var _ resource.ResourceWithImportState = &ResourceKMSKeyStore{}
var _ resource.ResourceWithModifyPlan = &ResourceKMSKeyStore{}

// ResourceKMSKeyStoreModel holds the Terraform state for a KMS key store.
type ResourceKMSKeyStoreModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Zone        types.String `tfsdk:"zone"`
	Description types.String `tfsdk:"description"`
	Proxy       types.Object `tfsdk:"proxy"`
	Connected   types.Bool   `tfsdk:"connected"`
	Type        types.String `tfsdk:"type"`
	Status      types.String `tfsdk:"status"`
	StatusSince types.String `tfsdk:"status_since"`
	CreatedAt   types.String `tfsdk:"created_at"`
	Health      types.Object `tfsdk:"health"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// ResourceKMSKeyStoreProxyModel holds the XKS proxy settings of a KMS key store.
type ResourceKMSKeyStoreProxyModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	AccessKey          types.String `tfsdk:"access_key"`
	SecretKeyWO        types.String `tfsdk:"secret_key_wo"`
	SecretKeyWOVersion types.Int64  `tfsdk:"secret_key_wo_version"`
}

// Types returns nested data model types to be used for conversion.
func (m ResourceKMSKeyStoreProxyModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"endpoint":              types.StringType,
		"access_key":            types.StringType,
		"secret_key_wo":         types.StringType,
		"secret_key_wo_version": types.Int64Type,
	}
}

// KMSKeyStoreHealthModel holds the latest health check of a KMS key store.
type KMSKeyStoreHealthModel struct {
	Status       types.String `tfsdk:"status"`
	StatusReason types.String `tfsdk:"status_reason"`
	CheckedAt    types.String `tfsdk:"checked_at"`
	ErrorDetail  types.String `tfsdk:"error_detail"`
}

// Types returns nested data model types to be used for conversion.
func (m KMSKeyStoreHealthModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"status":        types.StringType,
		"status_reason": types.StringType,
		"checked_at":    types.StringType,
		"error_detail":  types.StringType,
	}
}

var httpsURLRegexp = regexp.MustCompile(`^https://\S+$`)

type ResourceKMSKeyStore struct {
	client *exoscale.Client
}

func NewResourceKMSKeyStore() resource.Resource {
	return &ResourceKMSKeyStore{}
}

func (r *ResourceKMSKeyStore) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_key_store"
}

func (r *ResourceKMSKeyStore) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "[BETA] Manage Exoscale KMS External Key Stores.",
		Description:         "[BETA] Manage Exoscale KMS External Key Stores.",
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
				MarkdownDescription: "❗ The name of the key store, unique within the organization in this zone.",
				Description:         "The name of the key store, unique within the organization in this zone.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
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
				MarkdownDescription: "A description of the key store.",
				Description:         "A description of the key store.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1024),
				},
			},
			"proxy": schema.SingleNestedAttribute{
				MarkdownDescription: "The customer-managed XKS proxy. A change of `endpoint` or of the credentials is only saved if the proxy passes a health check with the new settings.",
				Description:         "The customer-managed XKS proxy. A change of endpoint or of the credentials is only saved if the proxy passes a health check with the new settings.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"endpoint": schema.StringAttribute{
						MarkdownDescription: "The HTTPS base URL of the XKS API on the proxy (e.g. `https://xks.example.com/kms/xks/v1`).",
						Description:         "The HTTPS base URL of the XKS API on the proxy (e.g. https://xks.example.com/kms/xks/v1).",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(httpsURLRegexp, "must be an https:// URL"),
						},
					},
					"access_key": schema.StringAttribute{
						MarkdownDescription: "The access key used to sign requests to the proxy. Changing it requires to also change `secret_key_wo_version`.",
						Description:         "The access key used to sign requests to the proxy. Changing it requires to also change secret_key_wo_version.",
						Required:            true,
					},
					"secret_key_wo": schema.StringAttribute{
						MarkdownDescription: "The secret key used to sign requests to the proxy. Write-only: it is never stored in the Terraform state; change `secret_key_wo_version` to send a new value. Requires Terraform 1.11 or later.",
						Description:         "The secret key used to sign requests to the proxy. Write-only: it is never stored in the Terraform state; change secret_key_wo_version to send a new value. Requires Terraform 1.11 or later.",
						Required:            true,
						Sensitive:           true,
						WriteOnly:           true,
					},
					"secret_key_wo_version": schema.Int64Attribute{
						MarkdownDescription: "Any number, changed to send `secret_key_wo` (and `access_key`) to the API again.",
						Description:         "Any number, changed to send secret_key_wo (and access_key) to the API again.",
						Required:            true,
					},
				},
			},
			"connected": schema.BoolAttribute{
				MarkdownDescription: "Whether the key store is connected (default: `false`). A key store is created disconnected; KMS keys can only be created in, and used from, a connected key store.",
				Description:         "Whether the key store is connected (default: false). A key store is created disconnected; KMS keys can only be created in, and used from, a connected key store.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The key store type (`external-key-store`).",
				Description:         "The key store type (external-key-store).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The connection status of the key store (`connected` or `disconnected`).",
				Description:         "The connection status of the key store (connected or disconnected).",
				Computed:            true,
			},
			"status_since": schema.StringAttribute{
				MarkdownDescription: "When the status last changed.",
				Description:         "When the status last changed.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The creation date of the key store.",
				Description:         "The creation date of the key store.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"health": schema.SingleNestedAttribute{
				MarkdownDescription: "The latest periodic health check of the proxy. Only connected key stores are checked, every 10 minutes, and a change to the key store resets it: it is usually empty right after an apply.",
				Description:         "The latest periodic health check of the proxy. Only connected key stores are checked, every 10 minutes, and a change to the key store resets it: it is usually empty right after an apply.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						MarkdownDescription: "`healthy`, `unhealthy` or `unknown`.",
						Description:         "healthy, unhealthy or unknown.",
						Computed:            true,
					},
					"status_reason": schema.StringAttribute{
						MarkdownDescription: "The reason for the status (e.g. `active`, `timeout`, `credential_error`).",
						Description:         "The reason for the status (e.g. active, timeout, credential_error).",
						Computed:            true,
					},
					"checked_at": schema.StringAttribute{
						MarkdownDescription: "When the check completed.",
						Description:         "When the check completed.",
						Computed:            true,
					},
					"error_detail": schema.StringAttribute{
						MarkdownDescription: "A short description of the failure, when the check failed.",
						Description:         "A short description of the failure, when the check failed.",
						Computed:            true,
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceKMSKeyStore) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// ModifyPlan requires a new secret_key_wo_version when access_key changes,
// since the API only accepts the access key and the secret key together.
func (r *ResourceKMSKeyStore) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to compare on create or destroy.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan ResourceKMSKeyStoreModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Proxy.IsUnknown() || state.Proxy.IsNull() {
		return
	}

	var stateProxy, planProxy ResourceKMSKeyStoreProxyModel
	resp.Diagnostics.Append(state.Proxy.As(ctx, &stateProxy, basetypes.ObjectAsOptions{})...)
	resp.Diagnostics.Append(plan.Proxy.As(ctx, &planProxy, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	if planProxy.AccessKey.IsUnknown() || planProxy.AccessKey.Equal(stateProxy.AccessKey) {
		return
	}
	if planProxy.SecretKeyWOVersion.IsUnknown() || !planProxy.SecretKeyWOVersion.Equal(stateProxy.SecretKeyWOVersion) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("proxy").AtName("secret_key_wo_version"),
		"Changing access_key requires a new secret_key_wo_version",
		"The API only accepts the access key together with the secret key. "+
			"Change secret_key_wo_version so that both are sent.",
	)
}

func (r *ResourceKMSKeyStore) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, data ResourceKMSKeyStoreModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	// Write-only attributes are only available in the config.
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
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

	var proxy ResourceKMSKeyStoreProxyModel
	resp.Diagnostics.Append(data.Proxy.As(ctx, &proxy, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := exoscale.CreateKeyStoreRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Type:        exoscale.CreateKeyStoreRequestTypeExternalKeyStore,
		Proxy: &exoscale.KeyStoreProxy{
			Endpoint: proxy.Endpoint.ValueString(),
			Auth: &exoscale.KeyStoreProxyAuth{
				Key:    proxy.AccessKey.ValueString(),
				Secret: proxy.SecretKeyWO.ValueString(),
			},
		},
	}

	keyStore, err := client.CreateKeyStore(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("API error creating KMS key store", apiErrorMessage(err))
		return
	}

	plan.ID = types.StringValue(keyStore.ID.String())

	// A new key store starts disconnected, and is only connected if the configuration asks for it.
	//
	// If it cannot be connected, the key store is kept and a warning is returned instead of an error:
	// an error would taint the resource, and the next apply would delete the key store and create it again.
	// connected keeps its planned value (Terraform rejects a result that differs from the plan) while status
	// reports the actual state; the next refresh then reads connected = false, and the next apply connects
	// the key store in place.
	connected := plan.Connected
	isConnected := false
	if connected.ValueBool() {
		if err := connectKeyStoreWithRetry(ctx, client, keyStore.ID); err == nil {
			isConnected = true
		} else {
			resp.Diagnostics.AddWarning(
				"KMS key store created but not connected",
				fmt.Sprintf("Connecting the key store failed: %s. "+
					"The key store has been created and stays disconnected; the next apply will try to connect it again.",
					apiErrorMessage(err)),
			)
		}
	}

	// Failing to read the key store back must not taint it either: fall back to the creation response.
	if found, diags := r.refresh(ctx, client, &plan); !found || diags.HasError() {
		resp.Diagnostics.Append(plan.setFromAPI(ctx, keyStoreFromCreateResponse(keyStore, isConnected))...)
		resp.Diagnostics.AddWarning(
			"KMS key store created but not read back",
			fmt.Sprintf("Reading the key store after its creation failed: %s. Its attributes are refreshed by the next plan.",
				diagsSummary(diags, found)),
		)
	}
	plan.Connected = connected
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// keyStoreFromCreateResponse converts the response of create-key-store, which has no health,
// into the shape of get-key-store.
func keyStoreFromCreateResponse(keyStore *exoscale.ListKeyStoresResponseEntry, connected bool) *exoscale.GetKeyStoreResponse {
	status := exoscale.GetKeyStoreResponseStatusDisconnected
	if connected {
		status = exoscale.GetKeyStoreResponseStatusConnected
	}

	return &exoscale.GetKeyStoreResponse{
		CreatedAT:   keyStore.CreatedAT,
		Description: keyStore.Description,
		ID:          keyStore.ID,
		Name:        keyStore.Name,
		Proxy:       keyStore.Proxy,
		Status:      status,
		StatusSince: keyStore.StatusSince,
		Type:        exoscale.GetKeyStoreResponseType(keyStore.Type),
	}
}

// diagsSummary describes why a refresh failed, for a warning.
func diagsSummary(diags diag.Diagnostics, found bool) string {
	if !found && !diags.HasError() {
		return "key store not found"
	}

	var msgs []string
	for _, d := range diags.Errors() {
		msgs = append(msgs, d.Detail())
	}
	return strings.Join(msgs, "; ")
}

// connectKeyStoreRetryDelays are the waits between attempts to connect a key store,
// in case its proxy fails a health check for a short time.
var connectKeyStoreRetryDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}

// transientProxyErrors are the XKS proxy errors that may clear on their own.
// Others, such as AuthenticationFailedException, need a configuration change.
var transientProxyErrors = []string{"ThrottlingException", "InternalException", "DependencyTimeoutException"}

func isTransientProxyError(err error) bool {
	if isAPIError(err, kmsErrKeyStoreProxyUnhealthy) {
		return true
	}
	if !isAPIError(err, kmsErrKeyStoreProxyRequestFailed) {
		return false
	}

	var apiErr *exoscale.APIError
	errors.As(err, &apiErr)
	resp, _ := apiErr.Response.(*exoscale.ErrorResponse)
	return resp.XksProxyError == nil || slices.Contains(transientProxyErrors, resp.XksProxyError.ErrorName)
}

// connectKeyStoreWithRetry connects a key store, retrying while its proxy fails the health check.
func connectKeyStoreWithRetry(ctx context.Context, client *exoscale.Client, id exoscale.UUID) error {
	for attempt := 0; ; attempt++ {
		_, err := client.ConnectKeyStore(ctx, id)
		if err == nil {
			return nil
		}

		if !isTransientProxyError(err) || attempt >= len(connectKeyStoreRetryDelays) {
			return err
		}

		tflog.Debug(ctx, "KMS key store proxy unhealthy, retrying connect", map[string]any{"attempt": attempt + 1})
		select {
		case <-ctx.Done():
			return err
		case <-time.After(connectKeyStoreRetryDelays[attempt]):
		}
	}
}

func (r *ResourceKMSKeyStore) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceKMSKeyStoreModel

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

	found, diags := r.refresh(ctx, client, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		tflog.Info(ctx, "KMS key store not found, removing from state", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceKMSKeyStore) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan, data ResourceKMSKeyStoreModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	// Write-only attributes are only available in the config.
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, config.DefaultTimeout)
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

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse resource ID", err.Error())
		return
	}

	var stateProxy, planProxy, configProxy ResourceKMSKeyStoreProxyModel
	resp.Diagnostics.Append(state.Proxy.As(ctx, &stateProxy, basetypes.ObjectAsOptions{})...)
	resp.Diagnostics.Append(plan.Proxy.As(ctx, &planProxy, basetypes.ObjectAsOptions{})...)
	resp.Diagnostics.Append(data.Proxy.As(ctx, &configProxy, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := exoscale.UpdateKeyStoreRequest{}
	updated := false

	if !plan.Description.Equal(state.Description) {
		updateReq.Description = plan.Description.ValueString()
		updated = true
	}

	proxyUpdate := exoscale.UpdateKeyStoreProxy{}
	if !planProxy.Endpoint.Equal(stateProxy.Endpoint) {
		proxyUpdate.Endpoint = planProxy.Endpoint.ValueString()
		updated = true
	}
	// The access key is only accepted together with the secret key, which can only be sent on a version change.
	if !planProxy.SecretKeyWOVersion.Equal(stateProxy.SecretKeyWOVersion) {
		proxyUpdate.Auth = &exoscale.KeyStoreProxyAuth{
			Key:    planProxy.AccessKey.ValueString(),
			Secret: configProxy.SecretKeyWO.ValueString(),
		}
		updated = true
	}
	if proxyUpdate.Endpoint != "" || proxyUpdate.Auth != nil {
		updateReq.Proxy = &proxyUpdate
	}

	// Use the actual status rather than state.Connected, which keeps the planned value
	// when the key store could not be connected at creation.
	isConnected := state.Status.ValueString() == string(exoscale.GetKeyStoreResponseStatusConnected)
	wantConnected := plan.Connected.ValueBool()

	// The endpoint can only be changed while the key store is disconnected: disconnect it first, and
	// reconnect it afterwards if it must stay connected. Its keys cannot be used in the meantime.
	// A key store that must end up disconnected is disconnected first too, so that the update applies to it.
	disconnectedForUpdate := false
	if isConnected && (proxyUpdate.Endpoint != "" || !wantConnected) {
		if _, err := client.DisconnectKeyStore(ctx, id); err != nil {
			resp.Diagnostics.AddError("API error disconnecting KMS key store", apiErrorMessage(err))
			return
		}
		isConnected = false
		disconnectedForUpdate = wantConnected
	}

	if updated {
		if _, err := client.UpdateKeyStore(ctx, id, updateReq); err != nil {
			resp.Diagnostics.AddError("API error updating KMS key store", apiErrorMessage(err))

			// Reconnect with the previous settings, so that the failed update does not leave the keys unusable.
			if disconnectedForUpdate {
				if err := connectKeyStoreWithRetry(ctx, client, id); err != nil {
					resp.Diagnostics.AddError("API error reconnecting KMS key store", apiErrorMessage(err))
				}
			}
			r.saveActualState(ctx, client, &state, &resp.State, &resp.Diagnostics)
			return
		}
	}

	if wantConnected && !isConnected {
		if err := connectKeyStoreWithRetry(ctx, client, id); err != nil {
			resp.Diagnostics.AddError("API error connecting KMS key store", apiErrorMessage(err))
			// Keep the successful update in the state, with the actual (disconnected) status.
			r.saveActualState(ctx, client, &plan, &resp.State, &resp.Diagnostics)
			return
		}
	}

	if found, diags := r.refresh(ctx, client, &plan); !found && !diags.HasError() {
		resp.Diagnostics.AddError("KMS key store not found", fmt.Sprintf("Key store %s disappeared before it could be read.", plan.ID.ValueString()))
	} else {
		resp.Diagnostics.Append(diags...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// saveActualState saves the key store as the API reports it, after an update failed half-way.
// If it cannot be read, the state is left as Terraform had it before the update.
func (r *ResourceKMSKeyStore) saveActualState(ctx context.Context, client *exoscale.Client, m *ResourceKMSKeyStoreModel, s *tfsdk.State, d *diag.Diagnostics) {
	if found, diags := r.refresh(ctx, client, m); !found || diags.HasError() {
		return
	}
	d.Append(s.Set(ctx, m)...)
}

func (r *ResourceKMSKeyStore) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceKMSKeyStoreModel

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

	if _, err := client.DeleteKeyStore(ctx, id); err != nil {
		if isNotFound(err, kmsErrKeyStoreNotFound) {
			return
		}

		if isAPIError(err, kmsErrKeyStoreIsReferenced) {
			resp.Diagnostics.AddError(
				"KMS key store is still referenced by KMS keys",
				"The key store can only be deleted once no KMS key references it. "+
					fmt.Sprintf("Keys scheduled for deletion keep referencing it until they are deleted, at least %d days after being destroyed. ", kmsKeyDeletionDelayDays)+
					"Run the destroy again after that, or cancel the deletion of the keys to keep using them.",
			)
			return
		}

		resp.Diagnostics.AddError("API error deleting KMS key store", apiErrorMessage(err))
	}
}

func (r *ResourceKMSKeyStore) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &ResourceKMSKeyStoreModel{
		ID:       types.StringValue(idParts[0]),
		Zone:     types.StringValue(idParts[1]),
		Proxy:    types.ObjectNull(ResourceKMSKeyStoreProxyModel{}.Types()),
		Health:   types.ObjectNull(KMSKeyStoreHealthModel{}.Types()),
		Timeouts: t,
	})...)
}

// refresh reads the key store from the API into the model.
// It returns false, without diagnostics, when the key store no longer exists.
func (r *ResourceKMSKeyStore) refresh(ctx context.Context, client *exoscale.Client, m *ResourceKMSKeyStoreModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	id, err := exoscale.ParseUUID(m.ID.ValueString())
	if err != nil {
		diags.AddError("unable to parse resource ID", err.Error())
		return false, diags
	}

	keyStore, err := client.GetKeyStore(ctx, id)
	if err != nil {
		if isNotFound(err, kmsErrKeyStoreNotFound) {
			return false, diags
		}
		diags.AddError("API error reading KMS key store", apiErrorMessage(err))
		return false, diags
	}

	diags.Append(m.setFromAPI(ctx, keyStore)...)
	return true, diags
}

// setFromAPI copies a key store returned by the API into the model.
// The write-only secret is never returned and its version only exists in Terraform, so both are kept as is.
func (m *ResourceKMSKeyStoreModel) setFromAPI(ctx context.Context, keyStore *exoscale.GetKeyStoreResponse) diag.Diagnostics {
	var diags diag.Diagnostics

	m.Name = types.StringValue(keyStore.Name)
	m.Description = types.StringValue(keyStore.Description)
	m.Type = types.StringValue(string(keyStore.Type))
	m.Status = types.StringValue(string(keyStore.Status))
	m.Connected = types.BoolValue(keyStore.Status == exoscale.GetKeyStoreResponseStatusConnected)
	m.StatusSince = types.StringValue(keyStore.StatusSince.Format(time.RFC3339))
	m.CreatedAt = types.StringValue(keyStore.CreatedAT.Format(time.RFC3339))

	proxy := ResourceKMSKeyStoreProxyModel{
		SecretKeyWO:        types.StringNull(),
		SecretKeyWOVersion: types.Int64Null(),
	}
	if !m.Proxy.IsNull() && !m.Proxy.IsUnknown() {
		diags.Append(m.Proxy.As(ctx, &proxy, basetypes.ObjectAsOptions{})...)
		proxy.SecretKeyWO = types.StringNull()
	}
	proxy.Endpoint = types.StringNull()
	proxy.AccessKey = types.StringNull()
	if keyStore.Proxy != nil {
		proxy.Endpoint = types.StringValue(keyStore.Proxy.Endpoint)
		if keyStore.Proxy.Auth != nil {
			proxy.AccessKey = types.StringValue(keyStore.Proxy.Auth.Key)
		}
	}
	proxyObj, d := types.ObjectValueFrom(ctx, ResourceKMSKeyStoreProxyModel{}.Types(), proxy)
	diags.Append(d...)
	m.Proxy = proxyObj

	health, d := keyStoreHealthValue(ctx, keyStore.Health)
	diags.Append(d...)
	m.Health = health

	return diags
}

// keyStoreHealthValue converts the latest health check of a key store, if any, to a Terraform object.
func keyStoreHealthValue(ctx context.Context, h *exoscale.KeyStoreHealth) (types.Object, diag.Diagnostics) {
	if h == nil {
		return types.ObjectNull(KMSKeyStoreHealthModel{}.Types()), nil
	}

	health := KMSKeyStoreHealthModel{
		Status:       types.StringValue(string(h.Status)),
		StatusReason: types.StringValue(h.StatusReason),
		CheckedAt:    types.StringNull(),
		ErrorDetail:  types.StringNull(),
	}
	if h.ErrorDetail != "" {
		health.ErrorDetail = types.StringValue(h.ErrorDetail)
	}
	if !h.CheckedAT.IsZero() {
		health.CheckedAt = types.StringValue(h.CheckedAT.Format(time.RFC3339))
	}

	return types.ObjectValueFrom(ctx, KMSKeyStoreHealthModel{}.Types(), health)
}
