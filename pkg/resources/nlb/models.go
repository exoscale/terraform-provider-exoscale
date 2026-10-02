package nlb

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// HealthcheckModel maps the `healthcheck` block of exoscale_nlb_service.
type HealthcheckModel struct {
	Interval types.Int64  `tfsdk:"interval"`
	Mode     types.String `tfsdk:"mode"`
	Port     types.Int64  `tfsdk:"port"`
	Retries  types.Int64  `tfsdk:"retries"`
	Timeout  types.Int64  `tfsdk:"timeout"`
	TLSSNI   types.String `tfsdk:"tls_sni"`
	URI      types.String `tfsdk:"uri"`
}

// Types returns nested data model types to be used for conversion.
func (m HealthcheckModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"interval": types.Int64Type,
		"mode":     types.StringType,
		"port":     types.Int64Type,
		"retries":  types.Int64Type,
		"timeout":  types.Int64Type,
		"tls_sni":  types.StringType,
		"uri":      types.StringType,
	}
}

// toAPI converts the healthcheck block into its API representation. The whole
// struct is always sent, so that dropping `uri`/`tls_sni` from the
// configuration actually clears them on the API side.
func (m HealthcheckModel) toAPI() *exoscale.LoadBalancerServiceHealthcheck {
	return &exoscale.LoadBalancerServiceHealthcheck{
		Interval: m.Interval.ValueInt64(),
		Mode:     exoscale.LoadBalancerServiceHealthcheckMode(m.Mode.ValueString()),
		Port:     m.Port.ValueInt64(),
		Retries:  m.Retries.ValueInt64(),
		Timeout:  m.Timeout.ValueInt64(),
		TlsSNI:   m.TLSSNI.ValueString(),
		URI:      m.URI.ValueString(),
	}
}

// healthcheckFromAPI maps an API healthcheck onto the block model.
func healthcheckFromAPI(hc *exoscale.LoadBalancerServiceHealthcheck) HealthcheckModel {
	if hc == nil {
		return HealthcheckModel{}
	}

	return HealthcheckModel{
		Interval: types.Int64Value(hc.Interval),
		Mode:     types.StringValue(string(hc.Mode)),
		Port:     types.Int64Value(hc.Port),
		Retries:  types.Int64Value(hc.Retries),
		Timeout:  types.Int64Value(hc.Timeout),
		TLSSNI:   utils.OptionalString(hc.TlsSNI),
		URI:      utils.OptionalString(hc.URI),
	}
}

// applyNLBComputed fills in the computed attributes of the model from the API.
//
// Create and Update use this rather than a full refresh: overwriting the
// configured attributes there would make the applied state diverge from the
// plan on any API-side normalisation, which the framework rejects outright
// ("Provider produced inconsistent result after apply"). Leaving them as
// planned lets the next Read report the difference as ordinary drift.
func applyNLBComputed(model *ResourceModel, nlb *exoscale.LoadBalancer) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(nlb.ID.String())
	model.CreatedAt = types.StringValue(nlb.CreatedAT.String())
	model.State = types.StringValue(string(nlb.State))

	model.IPAddress = types.StringNull()
	if len(nlb.IP) > 0 {
		model.IPAddress = types.StringValue(nlb.IP.String())
	}

	// The API returns the full service objects, the schema only tracks their IDs.
	services := make([]attr.Value, 0, len(nlb.Services))
	for _, service := range nlb.Services {
		services = append(services, types.StringValue(service.ID.String()))
	}
	set, dg := types.SetValue(types.StringType, services)
	diags.Append(dg...)
	if diags.HasError() {
		return diags
	}
	model.Services = set

	return diags
}

// applyNLB refreshes the whole model from the API. Only Read should use this;
// see applyNLBComputed. Attributes the API does not return (zone, timeouts)
// are left untouched.
func applyNLB(ctx context.Context, model *ResourceModel, nlb *exoscale.LoadBalancer) diag.Diagnostics {
	diags := applyNLBComputed(model, nlb)
	if diags.HasError() {
		return diags
	}

	model.Name = types.StringValue(nlb.Name)
	model.Description = utils.OptionalString(nlb.Description)

	model.Labels = types.MapNull(types.StringType)
	if len(nlb.Labels) > 0 {
		labels, dg := types.MapValueFrom(ctx, types.StringType, nlb.Labels)
		diags.Append(dg...)
		if diags.HasError() {
			return diags
		}
		model.Labels = labels
	}

	return diags
}

// applyServiceComputed fills in the computed attributes of the service model.
// See applyNLBComputed for why Create and Update do not refresh everything.
func applyServiceComputed(model *ResourceServiceModel, service *exoscale.LoadBalancerService) {
	model.ID = types.StringValue(service.ID.String())
	model.State = types.StringValue(string(service.State))
}

// applyService refreshes the whole service model from the API. Only Read
// should use this; see applyServiceComputed.
func applyService(model *ResourceServiceModel, service *exoscale.LoadBalancerService) {
	applyServiceComputed(model, service)

	model.Name = types.StringValue(service.Name)
	model.Description = utils.OptionalString(service.Description)
	model.Port = types.Int64Value(service.Port)
	model.TargetPort = types.Int64Value(service.TargetPort)
	model.Protocol = types.StringValue(string(service.Protocol))
	model.Strategy = types.StringValue(string(service.Strategy))

	model.InstancePoolID = types.StringNull()
	if service.InstancePool != nil {
		model.InstancePoolID = types.StringValue(service.InstancePool.ID.String())
	}

	model.Healthcheck = []HealthcheckModel{healthcheckFromAPI(service.Healthcheck)}
}

// findService resolves the service created by AddServiceToLoadBalancer.
//
// The returned operation reference is not guaranteed to carry the new service
// ID: for the analogous AddRuleToSecurityGroup the reference holds the parent
// Security Group ID instead. Match on the reference first, then fall back to
// the service name, which is unique within a Load Balancer.
func findService(
	ctx context.Context,
	client *exoscale.Client,
	nlbID exoscale.UUID,
	reference exoscale.UUID,
	name string,
) (*exoscale.LoadBalancerService, error) {
	nlb, err := client.GetLoadBalancer(ctx, nlbID)
	if err != nil {
		return nil, err
	}

	for i := range nlb.Services {
		if nlb.Services[i].ID == reference {
			return &nlb.Services[i], nil
		}
	}

	for i := range nlb.Services {
		if nlb.Services[i].Name == name {
			return &nlb.Services[i], nil
		}
	}

	return nil, exoscale.ErrNotFound
}
