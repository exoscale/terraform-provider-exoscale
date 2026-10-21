package elasticip

import (
	"context"
	"errors"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionResource = `Manage Exoscale [Elastic IPs (EIP)](https://community.exoscale.com/product/networking/eip/).

Corresponding data source: [exoscale_elastic_ip](../data-sources/elastic_ip.md).`

var (
	_ resource.ResourceWithConfigure    = (*Resource)(nil)
	_ resource.ResourceWithImportState  = (*Resource)(nil)
	_ resource.ResourceWithUpgradeState = (*Resource)(nil)
)

type Resource struct {
	client *exoscale.Client
}

func NewResource() resource.Resource {
	return &Resource{}
}

type ResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Zone          types.String `tfsdk:"zone"`
	AddressFamily types.String `tfsdk:"address_family"`
	CIDR          types.String `tfsdk:"cidr"`
	Description   types.String `tfsdk:"description"`
	Healthcheck   types.Object `tfsdk:"healthcheck"`
	IPAddress     types.String `tfsdk:"ip_address"`
	ReverseDNS    types.String `tfsdk:"reverse_dns"`
	Labels        types.Map    `tfsdk:"labels"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies resource name.
func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_elastic_ip"
}

func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// Version 1: healthcheck moved from an SDKv2 TypeList(MaxItems: 1)
		// (array-shaped in state) to a framework SingleNestedAttribute
		// (object-shaped). See UpgradeState below.
		Version: 1,

		Description:         "Manage Exoscale Elastic IPs (EIP).",
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
			"address_family": schema.StringAttribute{
				Description:         "❗ The Elastic IP (EIP) address family ('inet4' or 'inet6'; default: 'inet4').",
				MarkdownDescription: "❗ The Elastic IP (EIP) address family (`inet4` or `inet6`; default: `inet4`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(exoscale.ElasticIPAddressfamilyInet4),
						string(exoscale.ElasticIPAddressfamilyInet6),
					),
				},
			},
			"cidr": schema.StringAttribute{
				Description:         "The Elastic IP (EIP) CIDR.",
				MarkdownDescription: "The Elastic IP (EIP) CIDR.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description:         "A free-form text describing the Elastic IP (EIP).",
				MarkdownDescription: "A free-form text describing the Elastic IP (EIP).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"healthcheck": schema.SingleNestedAttribute{
				Description:         "Healthcheck configuration for *managed* EIPs. ❗ Adding it to or removing it from an existing EIP forces the creation of a new resource (an *unmanaged* EIP can not become *managed* and vice versa).",
				MarkdownDescription: "Healthcheck configuration for *managed* EIPs. ❗ Adding it to or removing it from an existing EIP forces the creation of a new resource (an *unmanaged* EIP can not become *managed* and vice versa).",
				Optional:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = req.StateValue.IsNull() != req.PlanValue.IsNull()
						},
						"An unmanaged EIP can not become managed and vice versa.",
						"An *unmanaged* EIP can not become *managed* and vice versa.",
					),
				},
				Attributes: map[string]schema.Attribute{
					"interval": schema.Int64Attribute{
						Description:         "The healthcheck interval (seconds; must be between '5' and '300'; default: '10').",
						MarkdownDescription: "The healthcheck interval (seconds; must be between `5` and `300`; default: `10`).",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(10),
						Validators: []validator.Int64{
							int64validator.Between(5, 300),
						},
					},
					"mode": schema.StringAttribute{
						Description:         "The healthcheck mode ('tcp', 'http' or 'https'; may only be set at creation time).",
						MarkdownDescription: "The healthcheck mode (`tcp`, `http` or `https`; may only be set at creation time).",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf(
								string(exoscale.ElasticIPHealthcheckModeTCP),
								string(exoscale.ElasticIPHealthcheckModeHTTP),
								string(exoscale.ElasticIPHealthcheckModeHttps),
							),
						},
					},
					"port": schema.Int64Attribute{
						Description:         "The healthcheck target port (must be between '1' and '65535').",
						MarkdownDescription: "The healthcheck target port (must be between `1` and `65535`).",
						Required:            true,
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
					},
					"timeout": schema.Int64Attribute{
						Description:         "The time before considering a healthcheck probing failed (seconds; must be between '2' and '60'; default: '3').",
						MarkdownDescription: "The time before considering a healthcheck probing failed (seconds; must be between `2` and `60`; default: `3`).",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(3),
						Validators: []validator.Int64{
							int64validator.Between(2, 60),
						},
					},
					"strikes_fail": schema.Int64Attribute{
						Description:         "The number of failed healthcheck attempts before considering the target unhealthy (must be between '1' and '20'; default: '2').",
						MarkdownDescription: "The number of failed healthcheck attempts before considering the target unhealthy (must be between `1` and `20`; default: `2`).",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(2),
						Validators: []validator.Int64{
							int64validator.Between(1, 20),
						},
					},
					"strikes_ok": schema.Int64Attribute{
						Description:         "The number of successful healthcheck attempts before considering the target healthy (must be between '1' and '20'; default: '3').",
						MarkdownDescription: "The number of successful healthcheck attempts before considering the target healthy (must be between `1` and `20`; default: `3`).",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(3),
						Validators: []validator.Int64{
							int64validator.Between(1, 20),
						},
					},
					"tls_skip_verify": schema.BoolAttribute{
						Description:         "Disable TLS certificate verification for healthcheck in 'https' mode (boolean; default: 'false').",
						MarkdownDescription: "Disable TLS certificate verification for healthcheck in `https` mode (boolean; default: `false`).",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					// tls_sni and uri default to "", the value the SDKv2
					// stored when they were not configured.
					"tls_sni": schema.StringAttribute{
						Description:         "The healthcheck server name to present with SNI in 'https' mode.",
						MarkdownDescription: "The healthcheck server name to present with SNI in `https` mode.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
					},
					"uri": schema.StringAttribute{
						Description:         "The healthcheck target URI (required in 'http(s)' modes).",
						MarkdownDescription: "The healthcheck target URI (required in `http(s)` modes).",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
					},
				},
			},
			"ip_address": schema.StringAttribute{
				Description:         "The Elastic IP (EIP) IPv4 or IPv6 address.",
				MarkdownDescription: "The Elastic IP (EIP) IPv4 or IPv6 address.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reverse_dns": schema.StringAttribute{
				Description:         "Domain name for reverse DNS record.",
				MarkdownDescription: "Domain name for reverse DNS record.",
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

// resourceModelV0 mirrors the schema this resource had at version 0 (SDKv2).
type resourceModelV0 struct {
	ID            types.String       `tfsdk:"id"`
	Zone          types.String       `tfsdk:"zone"`
	AddressFamily types.String       `tfsdk:"address_family"`
	CIDR          types.String       `tfsdk:"cidr"`
	Description   types.String       `tfsdk:"description"`
	Healthcheck   []HealthcheckModel `tfsdk:"healthcheck"`
	IPAddress     types.String       `tfsdk:"ip_address"`
	ReverseDNS    types.String       `tfsdk:"reverse_dns"`
	Labels        types.Map          `tfsdk:"labels"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *Resource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// SDKv2 to Framework migration: healthcheck was stored as an array
		// (SDKv2 TypeList(MaxItems: 1)); it is now a single object
		// attribute. The SDKv2 also wrote "" / {} for an unset description,
		// reverse_dns and labels: they are normalised to null here, once, rather than on
		// every plan. Every other attribute is byte-compatible as-is.
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":             schema.StringAttribute{Optional: true, Computed: true},
					"zone":           schema.StringAttribute{Required: true},
					"address_family": schema.StringAttribute{Optional: true, Computed: true},
					"cidr":           schema.StringAttribute{Computed: true},
					"description":    schema.StringAttribute{Optional: true, Computed: true},
					"ip_address":     schema.StringAttribute{Computed: true},
					"reverse_dns":    schema.StringAttribute{Optional: true},
					"labels":         schema.MapAttribute{ElementType: types.StringType, Optional: true},
				},
				Blocks: map[string]schema.Block{
					"healthcheck": schema.ListNestedBlock{
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"interval":        schema.Int64Attribute{Optional: true},
								"mode":            schema.StringAttribute{Required: true},
								"port":            schema.Int64Attribute{Required: true},
								"strikes_fail":    schema.Int64Attribute{Optional: true},
								"strikes_ok":      schema.Int64Attribute{Optional: true},
								"timeout":         schema.Int64Attribute{Optional: true},
								"tls_skip_verify": schema.BoolAttribute{Optional: true},
								"tls_sni":         schema.StringAttribute{Optional: true},
								"uri":             schema.StringAttribute{Optional: true},
							},
						},
					},
					"timeouts": timeouts.BlockAll(ctx),
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorState resourceModelV0

				resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded := ResourceModel{
					ID:            priorState.ID,
					Zone:          priorState.Zone,
					AddressFamily: priorState.AddressFamily,
					CIDR:          priorState.CIDR,
					Description:   priorState.Description,
					IPAddress:     priorState.IPAddress,
					ReverseDNS:    priorState.ReverseDNS,
					Labels:        priorState.Labels,
					Timeouts:      priorState.Timeouts,
				}

				if priorState.Description.ValueString() == "" {
					upgraded.Description = types.StringNull()
				}
				if priorState.ReverseDNS.ValueString() == "" {
					upgraded.ReverseDNS = types.StringNull()
				}
				if len(priorState.Labels.Elements()) == 0 {
					upgraded.Labels = types.MapNull(types.StringType)
				}

				upgraded.Healthcheck = types.ObjectNull(healthcheckAttrTypes())
				if len(priorState.Healthcheck) > 0 {
					obj, d := types.ObjectValueFrom(ctx, healthcheckAttrTypes(), priorState.Healthcheck[0])
					resp.Diagnostics.Append(d...)
					upgraded.Healthcheck = obj
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
			},
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

	request := exoscale.CreateElasticIPRequest{
		Description: plan.Description.ValueString(),
	}

	if v := plan.AddressFamily.ValueString(); v != "" {
		request.Addressfamily = exoscale.CreateElasticIPRequestAddressfamily(v)
	}

	healthcheck, diags := healthcheckRequest(ctx, plan.Healthcheck)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	request.Healthcheck = healthcheck

	if len(plan.Labels.Elements()) > 0 {
		labels := exoscale.Labels{}
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Labels = labels
	}

	op, err := client.CreateElasticIP(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when creating Elastic IP", err.Error())
		return
	}

	op, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create Elastic IP operation failed", err.Error())
		return
	}

	id := op.Reference.ID
	plan.ID = types.StringValue(id.String())

	// The Elastic IP exists from here on: save it in the state right away, so
	// that if one of the steps below fails it is kept there (Terraform marks
	// it as tainted) rather than left behind unmanaged.
	resp.Diagnostics.Append(resp.State.Set(ctx, nullUnknown(plan))...)
	if resp.Diagnostics.HasError() {
		return
	}

	if v := plan.ReverseDNS.ValueString(); v != "" {
		op, err := client.UpdateReverseDNSElasticIP(ctx, id, exoscale.UpdateReverseDNSElasticIPRequest{DomainName: v})
		if err != nil {
			resp.Diagnostics.AddError("unable to create Reverse DNS record", err.Error())
			return
		}
		if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError("create Reverse DNS record operation failed", err.Error())
			return
		}
	}

	elasticIP, err := client.GetElasticIP(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching the created Elastic IP", err.Error())
		return
	}

	applyElasticIPComputed(&plan, elasticIP)

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
		tflog.Info(ctx, "Elastic IP has no ID, removing from state to report drift", map[string]any{})
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

	elasticIP, err := client.GetElasticIP(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching Elastic IP", err.Error())
		return
	}

	rdns, err := reverseDNS(ctx, client, id)
	if err != nil {
		resp.Diagnostics.AddError("unable to retrieve Elastic IP reverse DNS", err.Error())
		return
	}

	resp.Diagnostics.Append(applyElasticIP(ctx, &state, elasticIP, rdns)...)
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

	id, err := exoscale.ParseUUID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(plan.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	var (
		request          exoscale.UpdateElasticIPRequest
		update           bool
		resetDescription bool
	)

	// TODO(egoscale): emptied labels cannot be expressed, UpdateElasticIPRequest.Labels
	// is a plain map with omitempty and there is no reset field for labels.
	if !plan.Labels.Equal(state.Labels) && len(plan.Labels.Elements()) > 0 {
		labels := exoscale.Labels{}
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		request.Labels = labels
		update = true
	}

	if !plan.Description.Equal(state.Description) {
		if v := plan.Description.ValueString(); v != "" {
			request.Description = v
			update = true
		} else {
			resetDescription = true
		}
	}

	if !plan.Healthcheck.Equal(state.Healthcheck) {
		healthcheck, diags := healthcheckRequest(ctx, plan.Healthcheck)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if healthcheck != nil {
			request.Healthcheck = healthcheck
			update = true
		}
	}

	if update {
		op, err := client.UpdateElasticIP(ctx, id, request)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error when updating Elastic IP", err.Error())
			return
		}
		if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError("update Elastic IP operation failed", err.Error())
			return
		}
	}

	if resetDescription {
		op, err := client.ResetElasticIPField(ctx, id, exoscale.ResetElasticIPFieldFieldDescription)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error when resetting Elastic IP description", err.Error())
			return
		}
		if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError("reset Elastic IP description operation failed", err.Error())
			return
		}
	}

	if !plan.ReverseDNS.Equal(state.ReverseDNS) {
		if v := plan.ReverseDNS.ValueString(); v != "" {
			op, err := client.UpdateReverseDNSElasticIP(ctx, id, exoscale.UpdateReverseDNSElasticIPRequest{DomainName: v})
			if err != nil {
				resp.Diagnostics.AddError("unable to update Reverse DNS record", err.Error())
				return
			}
			if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
				resp.Diagnostics.AddError("update Reverse DNS record operation failed", err.Error())
				return
			}
		} else {
			if err := deleteReverseDNS(ctx, client, id); err != nil {
				resp.Diagnostics.AddError("unable to delete Reverse DNS record", err.Error())
				return
			}
		}
	}

	elasticIP, err := client.GetElasticIP(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching the updated Elastic IP", err.Error())
		return
	}

	applyElasticIPComputed(&plan, elasticIP)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "resource updated", map[string]any{"id": plan.ID})
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

	if err := detachInstances(ctx, client, id); err != nil {
		resp.Diagnostics.AddError("unable to detach Elastic IP from instances", err.Error())
		return
	}

	if err := deleteReverseDNS(ctx, client, id); err != nil {
		resp.Diagnostics.AddError("unable to delete Reverse DNS record", err.Error())
		return
	}

	op, err := client.DeleteElasticIP(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("API returned an error when deleting Elastic IP", err.Error())
		return
	}
	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete Elastic IP operation failed", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": state.ID})
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	utils.ImportStatePassthroughZonedID(ctx, req, resp)
}

// applyElasticIPComputed sets the computed attributes the plan left unknown.
//
// Create and Update use this rather than a full refresh: overwriting the
// configured attributes there would make the applied state diverge from the
// plan on any API-side normalisation, which the framework rejects outright
// ("Provider produced inconsistent result after apply"). Leaving them as
// planned lets the next Read report the difference as ordinary drift.
func applyElasticIPComputed(model *ResourceModel, elasticIP *exoscale.ElasticIP) {
	if model.ID.IsUnknown() {
		model.ID = types.StringValue(elasticIP.ID.String())
	}
	if model.AddressFamily.IsUnknown() {
		model.AddressFamily = types.StringValue(string(elasticIP.Addressfamily))
	}
	if model.CIDR.IsUnknown() {
		model.CIDR = types.StringValue(elasticIP.Cidr)
	}
	if model.IPAddress.IsUnknown() {
		model.IPAddress = types.StringValue(elasticIP.IP)
	}
}

// applyElasticIP refreshes the whole model from the API. Only Read should use
// this; see applyElasticIPComputed. Attributes the API does not return (zone,
// timeouts) are left untouched.
func applyElasticIP(ctx context.Context, model *ResourceModel, elasticIP *exoscale.ElasticIP, rdns string) diag.Diagnostics {
	model.ID = types.StringValue(elasticIP.ID.String())
	model.AddressFamily = types.StringValue(string(elasticIP.Addressfamily))
	model.CIDR = types.StringValue(elasticIP.Cidr)
	model.IPAddress = types.StringValue(elasticIP.IP)

	utils.RefreshString(&model.Description, elasticIP.Description)
	utils.RefreshString(&model.ReverseDNS, rdns)

	diags := utils.RefreshLabels(ctx, &model.Labels, elasticIP.Labels)
	if diags.HasError() {
		return diags
	}

	healthcheck, dg := healthcheckObject(ctx, elasticIP.Healthcheck)
	diags.Append(dg...)
	model.Healthcheck = healthcheck

	return diags
}

// nullUnknown returns the model with the values still unknown set to null, so
// that it can be saved in the state.
func nullUnknown(model ResourceModel) *ResourceModel {
	for _, v := range []*types.String{
		&model.AddressFamily,
		&model.CIDR,
		&model.IPAddress,
	} {
		if v.IsUnknown() {
			*v = types.StringNull()
		}
	}

	return &model
}

// deleteReverseDNS deletes the Elastic IP reverse DNS record, if any.
func deleteReverseDNS(ctx context.Context, client *exoscale.Client, id exoscale.UUID) error {
	op, err := client.DeleteReverseDNSElasticIP(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return nil
		}
		return err
	}

	_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	return err
}

// detachInstances detaches the Elastic IP from every instance it is attached
// to, which the API requires before deleting it.
func detachInstances(ctx context.Context, client *exoscale.Client, id exoscale.UUID) error {
	instances, err := client.ListInstances(ctx)
	if err != nil {
		return err
	}

	for _, listed := range instances.Instances {
		instance, err := client.GetInstance(ctx, listed.ID)
		if err != nil {
			if errors.Is(err, exoscale.ErrNotFound) {
				// The instance was removed between the ListInstances and
				// GetInstance calls: nothing left to detach.
				continue
			}
			return err
		}

		attached := false
		for _, elasticIP := range instance.ElasticIPS {
			if elasticIP.ID == id {
				attached = true
				break
			}
		}
		if !attached {
			continue
		}

		tflog.Debug(ctx, "detaching Elastic IP from instance", map[string]any{
			"instance_id":   instance.ID,
			"elastic_ip_id": id,
		})

		op, err := client.DetachInstanceFromElasticIP(ctx, id, exoscale.DetachInstanceFromElasticIPRequest{
			Instance: &exoscale.InstanceTarget{ID: instance.ID},
		})
		if err != nil {
			return err
		}
		if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
			return err
		}
	}

	return nil
}
