package nlb

import (
	"context"
	"errors"
	"slices"
	"strings"

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

const markdownDescriptionResource = `Manage Exoscale [Network Load Balancers (NLB)](https://community.exoscale.com/product/networking/nlb/).

Corresponding data source: [exoscale_nlb](../data-sources/nlb.md).
`

var _ resource.Resource = (*Resource)(nil)
var _ resource.ResourceWithImportState = (*Resource)(nil)

type Resource struct {
	client *exoscale.Client
}

// NewResource creates an instance of Resource.
func NewResource() resource.Resource {
	return &Resource{}
}

// ResourceModel defines the exoscale_nlb resource data model.
type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Labels      types.Map    `tfsdk:"labels"`
	Zone        types.String `tfsdk:"zone"`
	CreatedAt   types.String `tfsdk:"created_at"`
	IPAddress   types.String `tfsdk:"ip_address"`
	Services    types.Set    `tfsdk:"services"`
	State       types.String `tfsdk:"state"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies the resource name.
func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nlb"
}

// Schema defines the resource attributes.
func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale Network Load Balancers (NLB).",
		MarkdownDescription: markdownDescriptionResource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The ID of this resource.",
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The network load balancer (NLB) name.",
				MarkdownDescription: "The network load balancer (NLB) name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A free-form text describing the NLB.",
				MarkdownDescription: "A free-form text describing the NLB.",
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"zone": schema.StringAttribute{
				Description:         "❗ The Exoscale zone name.",
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"created_at": schema.StringAttribute{
				Description:         "The NLB creation date.",
				MarkdownDescription: "The NLB creation date.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_address": schema.StringAttribute{
				Description:         "The NLB IPv4 address.",
				MarkdownDescription: "The NLB IPv4 address.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"services": schema.SetAttribute{
				Description:         "The list of the exoscale_nlb_service (IDs).",
				MarkdownDescription: "The list of the [exoscale_nlb_service](./nlb_service.md) (IDs).",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The current NLB state.",
				MarkdownDescription: "The current NLB state.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceModel

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

	request := exoscale.CreateLoadBalancerRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}
	if len(plan.Labels.Elements()) > 0 {
		labels := exoscale.Labels{}
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Labels = labels
	}

	operation, err := client.CreateLoadBalancer(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when creating NLB", err.Error())
		return
	}

	operation, err = client.Wait(ctx, operation, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create NLB operation failed", err.Error())
		return
	}

	nlb, err := client.GetLoadBalancer(ctx, operation.Reference.ID)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching the created NLB", err.Error())
		return
	}

	resp.Diagnostics.Append(applyNLBComputed(&plan, nlb)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "resource created", map[string]any{"id": plan.ID})
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceModel

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

	if state.ID.ValueString() == "" {
		tflog.Info(ctx, "NLB has no ID, removing from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	nlb, err := client.GetLoadBalancer(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching NLB", err.Error())
		return
	}

	resp.Diagnostics.Append(applyNLB(ctx, &state, nlb)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Trace(ctx, "resource read done", map[string]any{"id": state.ID})
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(plan.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	// Every attribute of the update payload is optional API-side, so only the
	// ones that actually changed are sent. `update` tracks whether anything
	// will actually be serialised: every field is `omitempty`, and the API
	// rejects a request that ends up with an empty body. Emptying an attribute
	// is therefore never an update -- it is either a reset call (labels) or
	// not expressible at all (description, see below).
	var (
		update  bool
		request exoscale.UpdateLoadBalancerRequest
	)

	if !plan.Name.Equal(state.Name) {
		update = true
		request.Name = plan.Name.ValueString()
	}

	// TODO(egoscale): an emptied description cannot be expressed.
	// UpdateLoadBalancerRequest.Description is a plain `string` with
	// `omitempty` (UpdateVpcRequest.Description is a *string), so "" is
	// dropped from the payload and the API keeps the previous value; the
	// DELETE /load-balancer/{id}/description reset endpoint answers 5xx.
	// State keeps the planned value, so the next Read reports it as drift.
	// Remove this note once the field is nullable upstream.
	if !plan.Description.Equal(state.Description) && plan.Description.ValueString() != "" {
		update = true
		request.Description = plan.Description.ValueString()
	}

	if !plan.Labels.Equal(state.Labels) && len(plan.Labels.Elements()) > 0 {
		labels := exoscale.Labels{}
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		update = true
		request.Labels = labels
	}

	if update {
		operation, err := client.UpdateLoadBalancer(ctx, id, request)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error when updating NLB", err.Error())
			return
		}
		if _, err := client.Wait(ctx, operation, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError("update NLB operation failed", err.Error())
			return
		}
	}

	// Every field of UpdateLoadBalancerRequest is `omitempty`, so emptying an
	// attribute cannot be expressed as an update: it needs a dedicated reset.
	if len(plan.Labels.Elements()) == 0 && len(state.Labels.Elements()) > 0 {
		if err := r.resetField(ctx, client, id, exoscale.ResetLoadBalancerFieldFieldLabels); err != nil {
			resp.Diagnostics.AddError("unable to reset NLB labels", err.Error())
			return
		}
	}

	nlb, err := client.GetLoadBalancer(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching the updated NLB", err.Error())
		return
	}

	resp.Diagnostics.Append(applyNLBComputed(&plan, nlb)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "resource update done", map[string]any{"id": plan.ID})
}

// resetField clears an NLB attribute that cannot be emptied through an update.
func (r *Resource) resetField(
	ctx context.Context,
	client *exoscale.Client,
	id exoscale.UUID,
	field exoscale.ResetLoadBalancerFieldField,
) error {
	operation, err := client.ResetLoadBalancerField(ctx, id, field)
	if err != nil {
		return err
	}

	_, err = client.Wait(ctx, operation, exoscale.OperationStateSuccess)

	return err
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceModel

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

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	operation, err := client.DeleteLoadBalancer(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("API returned an error while deleting NLB", err.Error())
		return
	}

	if _, err := client.Wait(ctx, operation, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete NLB operation failed", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": state.ID})
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			`Expected import identifier with format: id@zone. Got: "`+req.ID+`"`,
		)
		return
	}

	id, err := exoscale.ParseUUID(idParts[0])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	zone := idParts[1]
	if !slices.Contains(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
		return
	}

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var t timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &t)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &ResourceModel{
		ID:       types.StringValue(id.String()),
		Zone:     types.StringValue(zone),
		Labels:   types.MapNull(types.StringType),
		Services: types.SetNull(types.StringType),
		Timeouts: t,
	})...)

	tflog.Trace(ctx, "resource imported", map[string]any{"id": id.String()})
}
