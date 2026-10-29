package nlb

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionResourceService = `Manage Exoscale [Network Load Balancer (NLB)](https://community.exoscale.com/product/networking/nlb/) Services.

Corresponding data source: [exoscale_nlb_service_list](../data-sources/nlb_service_list.md).
`

const (
	defaultServiceHealthcheckInterval = 10
	defaultServiceHealthcheckMode     = "tcp"
	defaultServiceHealthcheckRetries  = 1
	defaultServiceHealthcheckTimeout  = 5
	defaultServiceProtocol            = "tcp"
	defaultServiceStrategy            = "round-robin"
)

var _ resource.Resource = (*ResourceService)(nil)
var _ resource.ResourceWithImportState = (*ResourceService)(nil)
var _ resource.ResourceWithValidateConfig = (*ResourceService)(nil)

type ResourceService struct {
	client *exoscale.Client
}

// NewResourceService creates an instance of ResourceService.
func NewResourceService() resource.Resource {
	return &ResourceService{}
}

// ResourceServiceModel defines the exoscale_nlb_service resource data model.
type ResourceServiceModel struct {
	ID             types.String       `tfsdk:"id"`
	NLBID          types.String       `tfsdk:"nlb_id"`
	Zone           types.String       `tfsdk:"zone"`
	InstancePoolID types.String       `tfsdk:"instance_pool_id"`
	LabelSelector  types.Map          `tfsdk:"label_selector"`
	Name           types.String       `tfsdk:"name"`
	Description    types.String       `tfsdk:"description"`
	Port           types.Int64        `tfsdk:"port"`
	TargetPort     types.Int64        `tfsdk:"target_port"`
	Protocol       types.String       `tfsdk:"protocol"`
	Strategy       types.String       `tfsdk:"strategy"`
	State          types.String       `tfsdk:"state"`
	Healthcheck    []HealthcheckModel `tfsdk:"healthcheck"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies the resource name.
func (r *ResourceService) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nlb_service"
}

// Schema defines the resource attributes.
func (r *ResourceService) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale Network Load Balancer (NLB) Services.",
		MarkdownDescription: markdownDescriptionResourceService,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The ID of this resource.",
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nlb_id": schema.StringAttribute{
				Description:         "❗ The parent exoscale_nlb ID.",
				MarkdownDescription: "❗ The parent [exoscale_nlb](./nlb.md) ID.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
			"instance_pool_id": schema.StringAttribute{
				Description:         "❗ The exoscale_instance_pool (ID) to forward traffic to. Either instance_pool_id or label_selector must be specified.",
				MarkdownDescription: "❗ The [exoscale_instance_pool](./instance_pool.md) (ID) to forward traffic to. Either `instance_pool_id` or `label_selector` must be specified.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"label_selector": schema.MapAttribute{
				Description:         "Labels of instances to forward traffic to, as key/value pairs. Labels are matched exactly, including case and trailing spaces. Only instances in the same organization and zone as the NLB are candidates. An instance is a backend when it carries all the listed pairs, and the map must hold at least one pair. If instance_pool_id is also specified, the backends are the union of both. Removing the attribute removes the selector, which is rejected by the API if the service has no instance_pool_id. Either label_selector or instance_pool_id must be specified.",
				MarkdownDescription: "Labels of instances to forward traffic to, as key/value pairs. Labels are matched exactly, including case and trailing spaces. Only instances in the same organization and zone as the NLB are candidates. An instance is a backend when it carries **all** the listed pairs, and the map must hold at least one pair. If `instance_pool_id` is also specified, the backends are the union of both. Removing the attribute removes the selector, which is rejected by the API if the service has no `instance_pool_id`. Either `label_selector` or `instance_pool_id` must be specified.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The NLB service name.",
				MarkdownDescription: "The NLB service name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A free-form text describing the NLB service.",
				MarkdownDescription: "A free-form text describing the NLB service.",
				Optional:            true,
			},
			"port": schema.Int64Attribute{
				Description:         "The NLB service (TCP/UDP) port.",
				MarkdownDescription: "The NLB service (TCP/UDP) port.",
				Required:            true,
			},
			"target_port": schema.Int64Attribute{
				Description:         "The (TCP/UDP) port to forward traffic to (on target instance pool members).",
				MarkdownDescription: "The (TCP/UDP) port to forward traffic to (on target instance pool members).",
				Required:            true,
			},
			"protocol": schema.StringAttribute{
				Description:         "The protocol (tcp|udp; default: tcp).",
				MarkdownDescription: "The protocol (`tcp`|`udp`; default: `tcp`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaultServiceProtocol),
			},
			"strategy": schema.StringAttribute{
				Description:         "The strategy (round-robin|source-hash; default: round-robin).",
				MarkdownDescription: "The strategy (`round-robin`|`source-hash`; default: `round-robin`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaultServiceStrategy),
			},
			"state": schema.StringAttribute{
				Description:         "The current NLB service state.",
				MarkdownDescription: "The current NLB service state.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"healthcheck": schema.SetNestedBlock{
				Description:         "The service health checking configuration.",
				MarkdownDescription: "The service health checking configuration.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"port": schema.Int64Attribute{
							Description:         "The healthcheck port.",
							MarkdownDescription: "The healthcheck port.",
							Required:            true,
						},
						"interval": schema.Int64Attribute{
							Description:         "The healthcheck interval in seconds (default: 10).",
							MarkdownDescription: "The healthcheck interval in seconds (default: `10`).",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(defaultServiceHealthcheckInterval),
						},
						"mode": schema.StringAttribute{
							Description:         "The healthcheck mode (tcp|http|https; default: tcp).",
							MarkdownDescription: "The healthcheck mode (`tcp`|`http`|`https`; default: `tcp`).",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString(defaultServiceHealthcheckMode),
						},
						"retries": schema.Int64Attribute{
							Description:         "The healthcheck retries (default: 1).",
							MarkdownDescription: "The healthcheck retries (default: `1`).",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(defaultServiceHealthcheckRetries),
						},
						"timeout": schema.Int64Attribute{
							Description:         "The healthcheck timeout (seconds; default: 5).",
							MarkdownDescription: "The healthcheck timeout (seconds; default: `5`).",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(defaultServiceHealthcheckTimeout),
						},
						"tls_sni": schema.StringAttribute{
							Description:         "The healthcheck TLS SNI server name (only if mode is https).",
							MarkdownDescription: "The healthcheck TLS SNI server name (only if `mode` is `https`).",
							Optional:            true,
						},
						"uri": schema.StringAttribute{
							Description:         "The healthcheck URI (must be set only if mode is http(s)).",
							MarkdownDescription: "The healthcheck URI (must be set only if `mode` is `http(s)`).",
							Optional:            true,
						},
					},
				},
				Validators: []validator.Set{
					setvalidator.IsRequired(),
					setvalidator.SizeAtMost(1),
				},
			},
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceService) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// ValidateConfig rejects two configurations that the API would refuse, so that
// the problem surfaces during `terraform plan` rather than after the NLB and
// the instance pool have been created. A service needs an instance pool, a
// label selector, or both, and a label selector needs at least one label.
//
// An empty selector cannot be supported, as the API returns either label pairs
// or no selector at all. `label_selector = {}` could therefore never be read
// back, and the resource would report drift on every plan.
func (r *ResourceService) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	// The configuration can arrive with no attributes at all, its object value
	// null rather than an object whose attributes are each null. Get cannot
	// decode a null object into the model, and neither check below has anything
	// to go on, so return early.
	if req.Config.Raw.IsNull() {
		return
	}

	var data ResourceServiceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A value that is not yet known is not null, so an instance pool created in
	// the same apply counts as set, and so does an unknown selector.
	if data.InstancePoolID.IsNull() && data.LabelSelector.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("instance_pool_id"),
			"missing NLB service backend",
			"either `instance_pool_id` or `label_selector` must be set",
		)
	}

	if data.LabelSelector.IsNull() || data.LabelSelector.IsUnknown() {
		return
	}

	if len(data.LabelSelector.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("label_selector"),
			"invalid NLB service label selector",
			"the label selector requires at least one label pair: remove the attribute to remove the selector",
		)
	}
}

func (r *ResourceService) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceServiceModel

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

	nlbID, err := exoscale.ParseUUID(plan.NLBID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse NLB ID", err.Error())
		return
	}

	if len(plan.Healthcheck) != 1 {
		resp.Diagnostics.AddError("invalid healthcheck", "exactly one healthcheck block is required")
		return
	}

	// The instance pool is optional, as a service can take its backends from a
	// label selector alone. ValidateConfig makes sure that one of the two is
	// set.
	var ipool *exoscale.InstancePool
	if !plan.InstancePoolID.IsNull() {
		instancePoolID, err := exoscale.ParseUUID(plan.InstancePoolID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse instance pool ID", err.Error())
			return
		}
		ipool = &exoscale.InstancePool{ID: instancePoolID}
	}

	labelSelector, diags := labelSelectorToAPI(ctx, plan.LabelSelector)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := exoscale.AddServiceToLoadBalancerRequest{
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		Healthcheck:   plan.Healthcheck[0].toAPI(),
		InstancePool:  ipool,
		LabelSelector: labelSelector,
		Port:          plan.Port.ValueInt64(),
		TargetPort:    plan.TargetPort.ValueInt64(),
		Protocol:      exoscale.AddServiceToLoadBalancerRequestProtocol(plan.Protocol.ValueString()),
		Strategy:      exoscale.AddServiceToLoadBalancerRequestStrategy(plan.Strategy.ValueString()),
	}

	operation, err := client.AddServiceToLoadBalancer(ctx, nlbID, request)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when creating NLB service", err.Error())
		return
	}

	operation, err = client.Wait(ctx, operation, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create NLB service operation failed", err.Error())
		return
	}

	var reference exoscale.UUID
	if operation.Reference != nil {
		reference = operation.Reference.ID
	}

	service, err := findService(ctx, client, nlbID, reference, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to identify the NLB service created", err.Error())
		return
	}

	applyServiceComputed(&plan, service)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "resource created", map[string]any{"id": plan.ID})
}

func (r *ResourceService) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceServiceModel

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
		tflog.Info(ctx, "NLB service has no ID, removing from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	nlbID, serviceID, err := parseIDs(&state)
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	service, err := client.GetLoadBalancerService(ctx, nlbID, serviceID)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Either the parent NLB or the service is gone, signaling the core
			// to remove the resource from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching NLB service", err.Error())
		return
	}

	resp.Diagnostics.Append(applyService(ctx, &state, service)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Trace(ctx, "resource read done", map[string]any{"id": state.ID})
}

func (r *ResourceService) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResourceServiceModel

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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(plan.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	nlbID, serviceID, err := parseIDs(&state)
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	if len(plan.Healthcheck) != 1 {
		resp.Diagnostics.AddError("invalid healthcheck", "exactly one healthcheck block is required")
		return
	}

	// Only the attributes that changed are sent: the API leaves an omitted
	// attribute untouched, and "" empties the description.
	//
	// The healthcheck is always sent whole, so that dropping `uri`/`tls_sni`
	// from the configuration clears them instead of leaving the previous
	// values.
	var (
		update  bool
		request exoscale.UpdateLoadBalancerServiceRequest
	)

	if !plan.Name.Equal(state.Name) {
		update = true
		request.Name = plan.Name.ValueStringPointer()
	}

	if !plan.Description.Equal(state.Description) {
		update = true
		description := plan.Description.ValueString()
		request.Description = &description
	}

	if !plan.Port.Equal(state.Port) {
		update = true
		request.Port = plan.Port.ValueInt64Pointer()
	}

	if !plan.TargetPort.Equal(state.TargetPort) {
		update = true
		request.TargetPort = plan.TargetPort.ValueInt64Pointer()
	}

	if !plan.Protocol.Equal(state.Protocol) {
		update = true
		request.Protocol = exoscale.UpdateLoadBalancerServiceRequestProtocol(plan.Protocol.ValueString())
	}

	if !plan.Strategy.Equal(state.Strategy) {
		update = true
		request.Strategy = exoscale.UpdateLoadBalancerServiceRequestStrategy(plan.Strategy.ValueString())
	}

	// A selector dropped from the configuration decodes to a null map, which the
	// API reads as "keep the selector", so the removal is requested with an
	// empty map.
	if !plan.LabelSelector.Equal(state.LabelSelector) {
		selector := plan.LabelSelector
		if selector.IsNull() {
			selector = types.MapValueMust(types.StringType, map[string]attr.Value{})
		}

		labelSelector, diags := labelSelectorToAPI(ctx, selector)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		update = true
		request.LabelSelector = labelSelector
	}

	if len(state.Healthcheck) != 1 || plan.Healthcheck[0] != state.Healthcheck[0] {
		update = true
		request.Healthcheck = plan.Healthcheck[0].toAPI()
	}

	if update {
		operation, err := client.UpdateLoadBalancerService(ctx, nlbID, serviceID, request)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error when updating NLB service", err.Error())
			return
		}
		if _, err := client.Wait(ctx, operation, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError("update NLB service operation failed", err.Error())
			return
		}
	}

	// The remote is in sync with the plan: carry the planned values over to
	// the state. The state struct is what gets saved, never the plan itself.
	state.Name = plan.Name
	state.Description = plan.Description
	state.LabelSelector = plan.LabelSelector
	state.Port = plan.Port
	state.TargetPort = plan.TargetPort
	state.Protocol = plan.Protocol
	state.Strategy = plan.Strategy
	state.Healthcheck = plan.Healthcheck
	state.Timeouts = plan.Timeouts

	service, err := client.GetLoadBalancerService(ctx, nlbID, serviceID)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching the updated NLB service", err.Error())
		return
	}

	applyServiceComputed(&state, service)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Trace(ctx, "resource update done", map[string]any{"id": state.ID})
}

func (r *ResourceService) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceServiceModel

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

	nlbID, serviceID, err := parseIDs(&state)
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	operation, err := client.DeleteLoadBalancerService(ctx, nlbID, serviceID)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("API returned an error while deleting NLB service", err.Error())
		return
	}

	if _, err := client.Wait(ctx, operation, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete NLB service operation failed", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": state.ID})
}

// parseIDs extracts the parent NLB and service UUIDs out of the model.
func parseIDs(model *ResourceServiceModel) (nlbID exoscale.UUID, serviceID exoscale.UUID, err error) {
	nlbID, err = exoscale.ParseUUID(model.NLBID.ValueString())
	if err != nil {
		return nlbID, serviceID, fmt.Errorf("unable to parse NLB ID: %w", err)
	}

	serviceID, err = exoscale.ParseUUID(model.ID.ValueString())
	if err != nil {
		return nlbID, serviceID, fmt.Errorf("unable to parse NLB service ID: %w", err)
	}

	return nlbID, serviceID, nil
}

func (r *ResourceService) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	zoneParts := strings.Split(req.ID, "@")
	if len(zoneParts) != 2 || zoneParts[0] == "" || zoneParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf(`Expected import identifier with format: <NLB-ID>/<SERVICE-ID>@<ZONE>. Got: %q`, req.ID),
		)
		return
	}

	idParts := strings.SplitN(zoneParts[0], "/", 2)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf(`Expected import identifier with format: <NLB-ID>/<SERVICE-ID>@<ZONE>. Got: %q`, req.ID),
		)
		return
	}

	nlbID, err := exoscale.ParseUUID(idParts[0])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse NLB ID", err.Error())
		return
	}

	serviceID, err := exoscale.ParseUUID(idParts[1])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse NLB service ID", err.Error())
		return
	}

	zone := zoneParts[1]
	if !slices.Contains(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), serviceID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("nlb_id"), nlbID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone"), zone)...)

	tflog.Trace(ctx, "resource imported", map[string]any{"id": serviceID.String()})
}
