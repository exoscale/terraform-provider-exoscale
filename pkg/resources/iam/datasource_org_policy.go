package iam

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const DataSourceOrgPolicyDescription = `Fetch Exoscale [IAM](https://community.exoscale.com/documentation/iam/) Organization Policy.

Corresponding resource: [exoscale_iam_org_policy](../resources/iam_org_policy.md).`

var _ datasource.DataSourceWithConfigure = &DataSourceOrgPolicy{}

func NewDataSourceOrgPolicy() datasource.DataSource {
	return &DataSourceOrgPolicy{}
}

type DataSourceOrgPolicy struct {
	client *exoscale.Client
	env    string
}

type DataSourceOrgPolicyModel struct {
	ID types.String `tfsdk:"id"`

	DefaultServiceStrategy types.String `tfsdk:"default_service_strategy"`
	Services               types.Map    `tfsdk:"services"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceOrgPolicy) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_iam_org_policy"
}

func (d *DataSourceOrgPolicy) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceOrgPolicyDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"default_service_strategy": schema.StringAttribute{
				MarkdownDescription: "Default service strategy (`allow` or `deny`).",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("allow", "deny"),
				},
			},
			"services": schema.MapNestedAttribute{
				MarkdownDescription: "IAM policy services.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "Service type (`rules`, `allow`, or `deny`).",
							Computed:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("allow", "deny", "rules"),
							},
						},
						"rules": schema.ListNestedAttribute{
							MarkdownDescription: "List of IAM service rules (if type is `rules`).",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"action": schema.StringAttribute{
										MarkdownDescription: "IAM policy rule action (`allow` or `deny`).",
										Computed:            true,
										Validators: []validator.String{
											stringvalidator.OneOf("allow", "deny"),
										},
									},
									"expression": schema.StringAttribute{
										MarkdownDescription: "IAM policy rule expression.",
										Computed:            true,
									},
									"resources": schema.ListAttribute{
										Computed:           true,
										ElementType:        types.StringType,
										DeprecationMessage: "This field is no longer suported.",
									},
								},
							},
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (d *DataSourceOrgPolicy) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV2
	d.env = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Environment
}

func (d *DataSourceOrgPolicy) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceOrgPolicyModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(d.env, config.DefaultZone))

	// Org policy is unique for organization, we can use a dummy value for ID.
	data.ID = types.StringValue("1")

	policy, err := d.client.GetIAMOrgPolicy(ctx, config.DefaultZone)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get IAM Organization Policy",
			err.Error(),
		)
		return
	}

	data.DefaultServiceStrategy = types.StringValue(policy.DefaultServiceStrategy)
	if len(policy.Services) > 0 {
		services := map[string]PolicyServiceModel{}
		for name, service := range policy.Services {
			serviceModel := PolicyServiceModel{
				Type: types.StringPointerValue(service.Type),
			}

			if len(service.Rules) > 0 {
				rules := []PolicyServiceRuleModel{}
				for _, rule := range service.Rules {
					ruleModel := PolicyServiceRuleModel{
						Action:     types.StringPointerValue(rule.Action),
						Expression: types.StringPointerValue(rule.Expression),
						Resources:  types.ListNull(types.StringType),
					}

					rules = append(rules, ruleModel)
				}

				t, dg := types.ListValueFrom(
					ctx,
					types.ObjectType{
						AttrTypes: PolicyServiceRuleModel{}.Types(),
					},
					rules,
				)
				if dg.HasError() {
					resp.Diagnostics.Append(dg...)
					return
				}
				serviceModel.Rules = t
			}

			services[name] = serviceModel
		}

		t, dg := types.MapValueFrom(
			ctx,
			types.ObjectType{
				AttrTypes: PolicyServiceModel{}.Types(),
			},
			services,
		)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		data.Services = t
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
