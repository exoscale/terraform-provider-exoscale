package iam

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const ResourceOrgPolicyDescription = `Manage Exoscale [IAM](https://community.exoscale.com/documentation/iam/) Organization Policy.`

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceOrgPolicy{}

func NewResourceOrgPolicy() resource.Resource {
	return &ResourceOrgPolicy{}
}

// ResourceOrgPolicy defines the IAM Organization Policy resource implementation.
type ResourceOrgPolicy struct {
	client *exoscale.Client
	env    string
}

// ResourceOrgPolicyModel describes the IAM Organization Policy resource data model.
type ResourceOrgPolicyModel struct {
	ID types.String `tfsdk:"id"`

	DefaultServiceStrategy types.String `tfsdk:"default_service_strategy"`
	Services               types.Map    `tfsdk:"services"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceOrgPolicy) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_org_policy"
}

func (r *ResourceOrgPolicy) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: ResourceOrgPolicyDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"default_service_strategy": schema.StringAttribute{
				MarkdownDescription: "Default service strategy (`allow` or `deny`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("allow", "deny"),
				},
			},
			"services": schema.MapNestedAttribute{
				MarkdownDescription: "IAM policy services.",
				Required:            true,
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
										Computed:           true,
										Optional:           true,
										ElementType:        types.StringType,
										DeprecationMessage: "This field is not suported. Specify resources using CEL expressions.",
										Default:            listdefault.StaticValue(basetypes.NewListNull(types.StringType)),
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

func (r *ResourceOrgPolicy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV2
	r.env = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Environment
}

func (r *ResourceOrgPolicy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceOrgPolicyModel

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

	// Update policy
	r.update(ctx, &resp.Diagnostics, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read updated policy
	r.read(ctx, &resp.Diagnostics, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource created", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *ResourceOrgPolicy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceOrgPolicyModel

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

	r.read(ctx, &resp.Diagnostics, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource read done", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *ResourceOrgPolicy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData ResourceOrgPolicyModel

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
	// Update policy
	r.update(ctx, &resp.Diagnostics, &planData)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read updated policy
	r.read(ctx, &resp.Diagnostics, &stateData)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)

	tflog.Trace(ctx, "resource updated", map[string]interface{}{
		"id": planData.ID,
	})
}

// Delete is NOOP
func (r *ResourceOrgPolicy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *ResourceOrgPolicy) read(
	ctx context.Context,
	d *diag.Diagnostics,
	data *ResourceOrgPolicyModel,
) {
	policy, err := r.client.GetIAMOrgPolicy(ctx, config.DefaultZone)
	if err != nil {
		d.AddError(
			"Unable to get IAM Organization Policy",
			err.Error(),
		)
		return
	}

	// Org policy is unique for organization, we can use a dummy value for ID.
	data.ID = types.StringValue("1")

	data.DefaultServiceStrategy = types.StringValue(policy.DefaultServiceStrategy)

	services := map[string]PolicyServiceModel{}
	if len(policy.Services) > 0 {
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
					d.Append(dg...)
					return
				}
				serviceModel.Rules = t
			}

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

	data.Services = t
}

func (r *ResourceOrgPolicy) update(
	ctx context.Context,
	d *diag.Diagnostics,
	data *ResourceOrgPolicyModel,
) {
	policy := &exoscale.IAMPolicy{
		DefaultServiceStrategy: data.DefaultServiceStrategy.ValueString(),
		Services:               map[string]exoscale.IAMPolicyService{},
	}

	if len(data.Services.Elements()) > 0 {
		services := map[string]PolicyServiceModel{}

		dg := data.Services.ElementsAs(ctx, &services, false)
		if dg.HasError() {
			d.Append(dg...)
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
					d.Append(dg...)
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

					t.Rules = append(t.Rules, q)
				}
			}

			policy.Services[name] = t
		}
	}

	err := r.client.UpdateIAMOrgPolicy(ctx, config.DefaultZone, policy)
	if err != nil {
		d.AddError(
			"Unable to update IAM Organization Policy",
			err.Error(),
		)
		return
	}
}
