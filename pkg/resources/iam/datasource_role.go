package iam

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const DataSourceRoleDescription = `Fetch Exoscale [IAM](https://community.exoscale.com/documentation/iam/) Role.

Corresponding resource: [exoscale_iam_role](../resources/iam_role.md).`

var _ datasource.DataSourceWithConfigure = &DataSourceRole{}

func NewDataSourceRole() datasource.DataSource {
	return &DataSourceRole{}
}

type DataSourceRole struct {
	client *exoscale.Client
	env    string
}

type DataSourceRoleModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Zone types.String `tfsdk:"zone"`

	Description types.String `tfsdk:"description"`
	Editable    types.Bool   `tfsdk:"editable"`
	Labels      types.Map    `tfsdk:"labels"`
	Permissions types.List   `tfsdk:"permissions"`
	Policy      types.Object `tfsdk:"policy"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceRole) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_iam_role"
}

func (d *DataSourceRole) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceRoleDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The role ID to match (conflicts with `name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "the role name to match (conflicts with `id`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("id"),
					}...),
				},
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A free-form text describing the IAM Role",
				Computed:            true,
			},
			"editable": schema.BoolAttribute{
				MarkdownDescription: "Defines if IAM Role Policy is editable or not.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "IAM Role labels.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"permissions": schema.ListAttribute{
				MarkdownDescription: "IAM Role permissions.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"policy": schema.SingleNestedAttribute{
				MarkdownDescription: "IAM Policy.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
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
												MarkdownDescription: "List of resources that IAM policy rule applies to.",
												Computed:            true,
												ElementType:         types.StringType,
											},
										},
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

func (d *DataSourceRole) Configure(
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

func (d *DataSourceRole) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceRoleModel

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

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(d.env, data.Zone.ValueString()))

	if name := data.Name.ValueStringPointer(); name != nil {
		roles, err := d.client.ListIAMRoles(ctx, data.Zone.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to get IAM Role",
				err.Error(),
			)
			return
		}

		for _, role := range roles {
			if *role.Name == *name {
				data.ID = types.StringPointerValue(role.ID)
				break
			}
		}
	}

	role, err := d.client.GetIAMRole(
		ctx,
		data.Zone.ValueString(),
		data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get IAM Role",
			err.Error(),
		)
		return
	}

	data.Description = types.StringPointerValue(role.Description)
	data.Editable = types.BoolPointerValue(role.Editable)

	data.Labels = types.MapNull(types.StringType)
	if len(role.Labels) > 0 {
		t, dg := types.MapValueFrom(
			ctx,
			types.StringType,
			role.Labels,
		)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		data.Labels = t
	}

	data.Permissions = types.ListNull(types.StringType)
	if len(role.Permissions) > 0 {
		t, dg := types.ListValueFrom(
			ctx,
			types.StringType,
			role.Permissions,
		)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		data.Permissions = t
	}

	if role.Policy != nil {
		policy := PolicyModel{}

		policy.DefaultServiceStrategy = types.StringValue(role.Policy.DefaultServiceStrategy)
		if len(role.Policy.Services) > 0 {
			services := map[string]PolicyServiceModel{}
			for name, service := range role.Policy.Services {
				serviceModel := PolicyServiceModel{
					Type: types.StringPointerValue(service.Type),
				}

				if len(service.Rules) > 0 {
					rules := []PolicyServiceRuleModel{}
					for _, rule := range service.Rules {
						ruleModel := PolicyServiceRuleModel{
							Action:     types.StringPointerValue(rule.Action),
							Expression: types.StringPointerValue(rule.Expression),
						}

						if rule.Resources != nil {
							t, dg := types.ListValueFrom(ctx, types.StringType, rule.Resources)
							if dg.HasError() {
								resp.Diagnostics.Append(dg...)
								return
							}
							ruleModel.Resources = t
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

			policy.Services = t
		}

		t, dg := types.ObjectValueFrom(
			ctx,
			PolicyModel{}.Types(),
			policy,
		)

		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		data.Policy = t
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
