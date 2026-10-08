package elasticip

import (
	"context"
	"errors"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// HealthcheckModel is the `healthcheck` nested object, shared by the resource
// and the data source.
type HealthcheckModel struct {
	Interval      types.Int64  `tfsdk:"interval"`
	Mode          types.String `tfsdk:"mode"`
	Port          types.Int64  `tfsdk:"port"`
	StrikesFail   types.Int64  `tfsdk:"strikes_fail"`
	StrikesOK     types.Int64  `tfsdk:"strikes_ok"`
	Timeout       types.Int64  `tfsdk:"timeout"`
	TLSSkipVerify types.Bool   `tfsdk:"tls_skip_verify"`
	TLSSNI        types.String `tfsdk:"tls_sni"`
	URI           types.String `tfsdk:"uri"`
}

func healthcheckAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interval":        types.Int64Type,
		"mode":            types.StringType,
		"port":            types.Int64Type,
		"strikes_fail":    types.Int64Type,
		"strikes_ok":      types.Int64Type,
		"timeout":         types.Int64Type,
		"tls_skip_verify": types.BoolType,
		"tls_sni":         types.StringType,
		"uri":             types.StringType,
	}
}

// healthcheckObject converts the API healthcheck to its Terraform value: null
// for an unmanaged Elastic IP. Empty strings are kept, as the SDKv2 exposed
// them.
func healthcheckObject(ctx context.Context, healthcheck *exoscale.ElasticIPHealthcheck) (types.Object, diag.Diagnostics) {
	if healthcheck == nil {
		return types.ObjectNull(healthcheckAttrTypes()), nil
	}

	tlsSkipVerify := false
	if healthcheck.TlsSkipVerify != nil {
		tlsSkipVerify = *healthcheck.TlsSkipVerify
	}

	return types.ObjectValueFrom(ctx, healthcheckAttrTypes(), HealthcheckModel{
		Interval:      types.Int64Value(healthcheck.Interval),
		Mode:          types.StringValue(string(healthcheck.Mode)),
		Port:          types.Int64Value(healthcheck.Port),
		StrikesFail:   types.Int64Value(healthcheck.StrikesFail),
		StrikesOK:     types.Int64Value(healthcheck.StrikesOk),
		Timeout:       types.Int64Value(healthcheck.Timeout),
		TLSSkipVerify: types.BoolValue(tlsSkipVerify),
		TLSSNI:        types.StringValue(healthcheck.TlsSNI),
		URI:           types.StringValue(healthcheck.URI),
	})
}

// healthcheckRequest converts the planned healthcheck to its API value: nil
// when there is none.
func healthcheckRequest(ctx context.Context, object types.Object) (*exoscale.ElasticIPHealthcheck, diag.Diagnostics) {
	if object.IsNull() || object.IsUnknown() {
		return nil, nil
	}

	var healthcheck HealthcheckModel
	diags := object.As(ctx, &healthcheck, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	request := &exoscale.ElasticIPHealthcheck{
		Interval:    healthcheck.Interval.ValueInt64(),
		Mode:        exoscale.ElasticIPHealthcheckMode(healthcheck.Mode.ValueString()),
		Port:        healthcheck.Port.ValueInt64(),
		StrikesFail: healthcheck.StrikesFail.ValueInt64(),
		StrikesOk:   healthcheck.StrikesOK.ValueInt64(),
		Timeout:     healthcheck.Timeout.ValueInt64(),
		TlsSNI:      healthcheck.TLSSNI.ValueString(),
		URI:         healthcheck.URI.ValueString(),
	}
	if healthcheck.TLSSkipVerify.ValueBool() {
		tlsSkipVerify := true
		request.TlsSkipVerify = &tlsSkipVerify
	}

	return request, diags
}

// reverseDNS returns the Elastic IP reverse DNS domain name, without its
// trailing dot, or "" when there is none.
func reverseDNS(ctx context.Context, client *exoscale.Client, id exoscale.UUID) (string, error) {
	rdns, err := client.GetReverseDNSElasticIP(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSuffix(string(rdns.DomainName), "."), nil
}
