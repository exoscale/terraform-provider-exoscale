package iam

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const ResourceRoleDescription = `Manage Exoscale [IAM](https://community.exoscale.com/documentation/iam/) Role.
`

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceRole{}
var _ resource.ResourceWithImportState = &ResourceRole{}

func NewResourceRole() resource.Resource {
	return &ResourceRole{}
}

// ResourceRole defines the IAM Organization Policy resource implementation.
type ResourceRole struct {
	client *exoscale.Client
	env    string
}

// ResourceRoleModel describes the IAM Organization Policy resource data model.
type ResourceRoleModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	Description types.String `tfsdk:"description"`
	Editable    types.Bool   `tfsdk:"editable"`
	Labels      types.Map    `tfsdk:"labels"`
	Permissions types.List   `tfsdk:"permissions"`
	Policy      types.Object `tfsdk:"policy"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceRole) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_role"
}

func (r *ResourceRole) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: ResourceRoleDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of IAM Role.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A free-form text describing the IAM Role",
				Computed:            true,
				Optional:            true,
			},
			"editable": schema.BoolAttribute{
				MarkdownDescription: "Defines if IAM Role Policy is editable or not.",
				Computed:            true,
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "IAM Role labels.",
				Computed:            true,
				Optional:            true,
				ElementType:         types.StringType,
			},
			"permissions": schema.ListAttribute{
				MarkdownDescription: "IAM Role permissions.",
				Computed:            true,
				Optional:            true,
				ElementType:         types.StringType,
			},
			"policy": schema.SingleNestedAttribute{
				MarkdownDescription: "IAM Policy.",
				Computed:            true,
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"default_service_strategy": schema.StringAttribute{
						MarkdownDescription: "Default service strategy (`allow` or `deny`).",
						Computed:            true,
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("allow", "deny"),
						},
					},
					"services": schema.MapNestedAttribute{
						MarkdownDescription: "IAM policy services.",
						Computed:            true,
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									MarkdownDescription: "Service type (`rules`, `allow`, or `deny`).",
									Computed:            true,
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.OneOf("allow", "deny", "rules"),
									},
								},
								"rules": schema.ListNestedAttribute{
									MarkdownDescription: "List of IAM service rules (if type is `rules`).",
									Computed:            true,
									Optional:            true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"action": schema.StringAttribute{
												MarkdownDescription: "IAM policy rule action (`allow` or `deny`).",
												Computed:            true,
												Optional:            true,
												Validators: []validator.String{
													stringvalidator.OneOf("allow", "deny"),
												},
											},
											"expression": schema.StringAttribute{
												MarkdownDescription: "IAM policy rule expression.",
												Computed:            true,
												Optional:            true,
											},
											"resources": schema.ListAttribute{
												MarkdownDescription: "List of resources that IAM policy rule applies to.",
												Computed:            true,
												Optional:            true,
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

func (r *ResourceRole) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV2
	r.env = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Environment
}

func (r *ResourceRole) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceRoleModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))

	role := exoscale.IAMRole{
		Name:        data.Name.ValueStringPointer(),
		Description: data.Description.ValueStringPointer(),
		Editable:    data.Editable.ValueBoolPointer(),
	}

	if !data.Labels.IsUnknown() && len(data.Labels.Elements()) > 0 {
		labels := map[string]string{}

		dg := data.Labels.ElementsAs(ctx, &labels, false)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		role.Labels = labels
	}

	if !data.Permissions.IsUnknown() && len(data.Permissions.Elements()) > 0 {
		permissions := make([]string, 0, len(data.Permissions.Elements()))

		dg := data.Permissions.ElementsAs(ctx, &permissions, false)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		role.Permissions = permissions
	}

	if !data.Policy.IsUnknown() {
		dataPolicy := PolicyModel{}
		dg := data.Policy.As(ctx, &dataPolicy, basetypes.ObjectAsOptions{})
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		policy := &exoscale.IAMPolicy{
			DefaultServiceStrategy: dataPolicy.DefaultServiceStrategy.ValueString(),
			Services:               map[string]exoscale.IAMPolicyService{},
		}

		if len(dataPolicy.Services.Elements()) > 0 {
			services := map[string]PolicyServiceModel{}

			dg := dataPolicy.Services.ElementsAs(ctx, &services, false)
			if dg.HasError() {
				resp.Diagnostics.Append(dg...)
				return
			}

			for name, service := range services {
				t := exoscale.IAMPolicyService{}

				if !service.Type.IsUnknown() {
					t.Type = service.Type.ValueStringPointer()
				}

				if !service.Rules.IsUnknown() && len(service.Rules.Elements()) > 0 {
					t.Rules = []exoscale.IAMPolicyServiceRule{}
					rules := []PolicyServiceRuleModel{}
					dg := service.Rules.ElementsAs(ctx, &rules, false)
					if dg.HasError() {
						resp.Diagnostics.Append(dg...)
						return
					}

					for _, rule := range rules {
						q := exoscale.IAMPolicyServiceRule{}

						if !rule.Action.IsUnknown() {
							q.Action = rule.Action.ValueStringPointer()
						}

						if !rule.Expression.IsUnknown() {
							q.Expression = rule.Expression.ValueStringPointer()
						}

						if !rule.Resources.IsUnknown() && len(rule.Resources.Elements()) > 0 {
							elements := []types.String{}
							dg := service.Rules.ElementsAs(ctx, &elements, false)
							if dg.HasError() {
								resp.Diagnostics.Append(dg...)
								return
							}
							q.Resources = make([]string, 0, len(elements))
							for _, elem := range elements {
								q.Resources = append(q.Resources, elem.ValueString())
							}
						}

						t.Rules = append(t.Rules, q)
					}
				}

				policy.Services[name] = t
			}
		}

		role.Policy = policy
	}

	newRole, err := r.client.CreateIAMRole(ctx, config.DefaultZone, &role)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create IAM Role",
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(*newRole.ID)

	// Read created policy
	r.read(ctx, resp.Diagnostics, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource created", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *ResourceRole) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceRoleModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
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

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))

	r.read(ctx, resp.Diagnostics, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource read done", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *ResourceRole) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData ResourceRoleModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	// Read Terraform state data (for comparison) into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := stateData.Timeouts.Update(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))
	// Update role
	role := exoscale.IAMRole{
		ID:          planData.ID.ValueStringPointer(),
		Name:        planData.Name.ValueStringPointer(),
		Description: planData.Description.ValueStringPointer(),
		Editable:    planData.Editable.ValueBoolPointer(),
	}

	if !planData.Labels.IsUnknown() && len(planData.Labels.Elements()) > 0 {
		labels := map[string]string{}

		dg := planData.Labels.ElementsAs(ctx, &labels, false)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		role.Labels = labels
	}

	if !planData.Permissions.IsUnknown() && len(planData.Permissions.Elements()) > 0 {
		permissions := make([]string, 0, len(planData.Permissions.Elements()))

		dg := planData.Permissions.ElementsAs(ctx, &permissions, false)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		role.Permissions = permissions
	}

	err := r.client.UpdateIAMRole(ctx, config.DefaultZone, &role)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create IAM Role",
			err.Error(),
		)
		return
	}

	// Update policy
	if !planData.Policy.IsUnknown() && !planData.Policy.Equal(stateData.Policy) {
		dataPolicy := PolicyModel{}
		dg := planData.Policy.As(ctx, &dataPolicy, basetypes.ObjectAsOptions{})
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		policy := &exoscale.IAMPolicy{
			DefaultServiceStrategy: dataPolicy.DefaultServiceStrategy.ValueString(),
			Services:               map[string]exoscale.IAMPolicyService{},
		}

		if len(dataPolicy.Services.Elements()) > 0 {
			services := map[string]PolicyServiceModel{}

			dg := dataPolicy.Services.ElementsAs(ctx, &services, false)
			if dg.HasError() {
				resp.Diagnostics.Append(dg...)
				return
			}

			for name, service := range services {
				t := exoscale.IAMPolicyService{}

				if !service.Type.IsUnknown() {
					t.Type = service.Type.ValueStringPointer()
				}

				if !service.Rules.IsUnknown() && len(service.Rules.Elements()) > 0 {
					t.Rules = []exoscale.IAMPolicyServiceRule{}
					rules := []PolicyServiceRuleModel{}
					dg := service.Rules.ElementsAs(ctx, &rules, false)
					if dg.HasError() {
						resp.Diagnostics.Append(dg...)
						return
					}

					for _, rule := range rules {
						q := exoscale.IAMPolicyServiceRule{}

						if !rule.Action.IsUnknown() {
							q.Action = rule.Action.ValueStringPointer()
						}

						if !rule.Expression.IsUnknown() {
							q.Expression = rule.Expression.ValueStringPointer()
						}

						if !rule.Resources.IsUnknown() && len(rule.Resources.Elements()) > 0 {
							elements := []types.String{}
							dg := service.Rules.ElementsAs(ctx, &elements, false)
							if dg.HasError() {
								resp.Diagnostics.Append(dg...)
								return
							}
							q.Resources = make([]string, 0, len(elements))
							for _, elem := range elements {
								q.Resources = append(q.Resources, elem.ValueString())
							}
						}

						t.Rules = append(t.Rules, q)
					}
				}

				policy.Services[name] = t
			}
		}

		role.Policy = policy

		err := r.client.UpdateIAMRolePolicy(ctx, config.DefaultZone, &role)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to create IAM Role",
				err.Error(),
			)
			return
		}
	}

	// Read updated role
	r.read(ctx, resp.Diagnostics, &stateData)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)

	tflog.Trace(ctx, "resource updated", map[string]interface{}{
		"id": planData.ID,
	})
}

func (r *ResourceRole) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceRoleModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
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

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))

	err := r.client.DeleteIAMRole(
		ctx,
		config.DefaultZone,
		&exoscale.IAMRole{ID: data.ID.ValueStringPointer()},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete IAM Role",
			err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *ResourceRole) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ResourceRoleModel

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Timeouts = timeouts

	data.ID = types.StringValue(req.ID)
	data.Labels = types.MapNull(types.StringType)
	data.Permissions = types.ListNull(types.StringType)
	data.Policy = types.ObjectNull(PolicyModel{}.Types())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource imported", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *ResourceRole) read(
	ctx context.Context,
	d diag.Diagnostics,
	data *ResourceRoleModel,
) {

	role, err := r.client.GetIAMRole(ctx, config.DefaultZone, data.ID.ValueString())
	if err != nil {
		d.AddError(
			"Unable to get IAM Role",
			err.Error(),
		)
		return
	}

	data.Name = types.StringPointerValue(role.Name)
	data.Description = types.StringPointerValue(role.Description)
	data.Editable = types.BoolPointerValue(role.Editable)

	data.Labels = types.MapNull(types.StringType)
	if role.Labels != nil {
		t, dg := types.MapValueFrom(
			ctx,
			types.StringType,
			role.Labels,
		)
		if dg.HasError() {
			d.Append(dg...)
			return
		}

		data.Labels = t
	}

	data.Permissions = types.ListNull(types.StringType)
	if role.Permissions != nil {
		t, dg := types.ListValueFrom(
			ctx,
			types.StringType,
			role.Permissions,
		)
		if dg.HasError() {
			d.Append(dg...)
			return
		}

		data.Permissions = t
	}

	policy := PolicyModel{}
	if role.Policy != nil {
		policy.DefaultServiceStrategy = types.StringValue(role.Policy.DefaultServiceStrategy)
		services := map[string]PolicyServiceModel{}
		if len(role.Policy.Services) > 0 {
			for name, service := range role.Policy.Services {
				serviceModel := PolicyServiceModel{
					Type: types.StringPointerValue(service.Type),
				}

				rules := []PolicyServiceRuleModel{}
				if len(service.Rules) > 0 {
					for _, rule := range service.Rules {
						ruleModel := PolicyServiceRuleModel{
							Action:     types.StringPointerValue(rule.Action),
							Expression: types.StringPointerValue(rule.Expression),
						}

						if rule.Resources != nil {
							t, dg := types.ListValueFrom(ctx, types.StringType, rule.Resources)
							if dg.HasError() {
								d.Append(dg...)
								return
							}
							ruleModel.Resources = t
						}

						rules = append(rules, ruleModel)
					}
				}

				t, dg := types.ListValueFrom(
					ctx,
					types.ObjectType{
						AttrTypes: PolicyServiceRuleModel{}.Types(),
					},
					rules,
				)
				if dg.HasError() {
					d.Append(dg...)
					return
				}
				serviceModel.Rules = t

				services[name] = serviceModel
			}
		}

		t, dg := types.MapValueFrom(
			ctx,
			types.ObjectType{
				AttrTypes: PolicyServiceModel{}.Types(),
			},
			services,
		)
		if dg.HasError() {
			d.Append(dg...)
			return
		}

		policy.Services = t

	}

	p, dg := types.ObjectValueFrom(
		ctx,
		PolicyModel{}.Types(),
		policy,
	)

	if dg.HasError() {
		d.Append(dg...)
		return
	}
	data.Policy = p
}
