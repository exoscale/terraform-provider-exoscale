package security_group

import (
	"context"
	"errors"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const ResourceRuleDescription = `Manage [Exoscale Security Groups](https://community.exoscale.com/product/compute/instances/quick-start/#firewall-rules---security-groups) rules.

Parent resource: [exoscale_security_group_rule](./security_group.md).`

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceRule{}
var _ resource.ResourceWithImportState = &ResourceRule{}

type ResourceRule struct {
	client *exoscale.Client
}

// NewResourceRule creates instance of ResourceRule.
func NewResourceRule() resource.Resource {
	return &ResourceRule{}
}

// ResourceRuleModel defines the resource data model.
type ResourceRuleModel struct {
	ID                  types.String `tfsdk:"id"`
	SecurityGroupID     types.String `tfsdk:"security_group_id"`
	Description         types.String `tfsdk:"description"`
	Type                types.String `tfsdk:"type"`
	Protocol            types.String `tfsdk:"protocol"`
	CIDR                types.String `tfsdk:"cidr"`
	PublicSecurityGroup types.String `tfsdk:"public_security_group"`
	UserSecurityGroupID types.String `tfsdk:"user_security_group_id"`
	StartPort           types.Int64  `tfsdk:"start_port"`
	EndPort             types.Int64  `tfsdk:"end_port"`
	ICMPType            types.Int64  `tfsdk:"icmp_type"`
	ICMPCode            types.Int64  `tfsdk:"icmp_code"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewResourceRuleModel() ResourceRuleModel {
	return ResourceRuleModel{
		ID:                  types.StringNull(),
		SecurityGroupID:     types.StringNull(),
		Description:         types.StringNull(),
		Type:                types.StringNull(),
		Protocol:            types.StringNull(),
		CIDR:                types.StringNull(),
		PublicSecurityGroup: types.StringNull(),
		UserSecurityGroupID: types.StringNull(),
		StartPort:           types.Int64Null(),
		EndPort:             types.Int64Null(),
		ICMPType:            types.Int64Null(),
		ICMPCode:            types.Int64Null(),
		Timeouts:            timeouts.Value{},
	}
}

// Metadata specifies resource name.
func (r *ResourceRule) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_security_group_rule"
}

// Schema defines resource attributes.
func (r *ResourceRule) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Manage Security Groups",
		MarkdownDescription: ResourceRuleDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "rule ID",
				MarkdownDescription: "The ID of the Security Group rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description:         "rule description",
				MarkdownDescription: "❗ A free-form text describing the the Security Group rule.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"security_group_id": schema.StringAttribute{
				Description:         "the parent security group ID",
				MarkdownDescription: "❗ The parent [exoscale_security_group](./security_group.md) ID.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description:         "the traffic direction",
				MarkdownDescription: "❗ The traffic direction to match (`INGRESS` or `EGRESS`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						string(exoscale.AddRuleToSecurityGroupRequestFlowDirectionIngress),
						string(exoscale.AddRuleToSecurityGroupRequestFlowDirectionEgress),
					),
				},
			},
			"protocol": schema.StringAttribute{
				Description:         "the network protocol to match",
				MarkdownDescription: "❗ The network protocol to match (`TCP`, `UDP`, `ICMP`, `ICMPv6`, `AH`, `ESP`, `GRE` or `IPIP`)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("TCP"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						string(exoscale.AddRuleToSecurityGroupRequestProtocolAh),
						string(exoscale.AddRuleToSecurityGroupRequestProtocolEsp),
						string(exoscale.AddRuleToSecurityGroupRequestProtocolGre),
						string(exoscale.AddRuleToSecurityGroupRequestProtocolICMP),
						string(exoscale.AddRuleToSecurityGroupRequestProtocolIcmpv6),
						string(exoscale.AddRuleToSecurityGroupRequestProtocolIpip),
						string(exoscale.AddRuleToSecurityGroupRequestProtocolTCP),
						string(exoscale.AddRuleToSecurityGroupRequestProtocolUDP),
					),
				},
			},
			"cidr": schema.StringAttribute{
				Description:         "source / destination IP subnet",
				MarkdownDescription: "❗ An (`INGRESS`) source / (`EGRESS`) destination IP subnet (in [CIDR notation](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notation)) to match (conflicts with `public_security_group`/`user_security_group_id`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("public_security_group"),
						path.MatchRoot("user_security_group_id"),
					}...),
				},
			},
			"public_security_group": schema.StringAttribute{
				Description:         "source / destination public security group name",
				MarkdownDescription: "❗ An (`INGRESS`) source / (`EGRESS`) destination public security group name to match (conflicts with `cidr`/`user_security_group_id`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("cidr"),
						path.MatchRoot("user_security_group_id"),
					}...),
				},
			},
			"user_security_group_id": schema.StringAttribute{
				Description:         "source / destination user security group ID",
				MarkdownDescription: "❗ An (`INGRESS`) source / (`EGRESS`) user security group ID to match (conflicts with `cidr`/`public_security_group`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("cidr"),
						path.MatchRoot("public_security_group"),
					}...),
				},
			},
			"start_port": schema.Int64Attribute{
				Description:         "start port",
				MarkdownDescription: "❗A start port number in the `TCP`/`UDP` port range to match (conflicts with `icmp_type`/`icmp_code`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplaceIfConfigured(),
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
					int64validator.AlsoRequires(path.Expressions{
						path.MatchRoot("end_port"),
					}...),
					int64validator.ConflictsWith(path.Expressions{
						path.MatchRoot("icmp_type"),
						path.MatchRoot("icmp_code"),
					}...),
				},
			},
			"end_port": schema.Int64Attribute{
				Description:         "end port",
				MarkdownDescription: "❗The end port number in the `TCP`/`UDP` port range to match (conflicts with `icmp_type`/`icmp_code`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplaceIfConfigured(),
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
					int64validator.AtLeastSumOf(path.Expressions{
						path.MatchRoot("start_port"),
					}...),
					int64validator.AlsoRequires(path.Expressions{
						path.MatchRoot("start_port"),
					}...),
					int64validator.ConflictsWith(path.Expressions{
						path.MatchRoot("icmp_type"),
						path.MatchRoot("icmp_code"),
					}...),
				},
			},
			"icmp_type": schema.Int64Attribute{
				Description:         "ICMP/ICMPv6 type",
				MarkdownDescription: "❗An ICMP/ICMPv6 [type](https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol#Control_messages) to match.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplaceIfConfigured(),
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(-1, 254),
					int64validator.AlsoRequires(path.Expressions{
						path.MatchRoot("icmp_code"),
					}...),
					int64validator.ConflictsWith(path.Expressions{
						path.MatchRoot("start_port"),
						path.MatchRoot("end_port"),
					}...),
				},
			},
			"icmp_code": schema.Int64Attribute{
				Description:         "ICMP/ICMPv6 code",
				MarkdownDescription: "❗An ICMP/ICMPv6 [code](https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol#Control_messages) to match.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplaceIfConfigured(),
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(-1, 254),
					int64validator.AlsoRequires(path.Expressions{
						path.MatchRoot("icmp_type"),
					}...),
					int64validator.ConflictsWith(path.Expressions{
						path.MatchRoot("start_port"),
						path.MatchRoot("end_port"),
					}...),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

// Configure sets up resource dependencies.
func (r *ResourceRule) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// Create resources by receiving Terraform configuration and plan data, performing creation logic, and saving Terraform state data.
func (r *ResourceRule) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ResourceRuleModel

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

	parentID, err := exoscale.ParseUUID(plan.SecurityGroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse parent security group ID",
			err.Error(),
		)
		return
	}
	if _, err := r.client.GetSecurityGroup(ctx, parentID); err != nil {
		resp.Diagnostics.AddError(
			"API returned error reading parent security group",
			err.Error(),
		)

		return
	}

	lowercaser := cases.Lower(language.Und)

	ruleReq := exoscale.AddRuleToSecurityGroupRequest{
		FlowDirection: exoscale.AddRuleToSecurityGroupRequestFlowDirection(
			lowercaser.String(plan.Type.ValueString())),
		Protocol: exoscale.AddRuleToSecurityGroupRequestProtocol(
			lowercaser.String(plan.Protocol.ValueString())),
		Description: plan.Description.ValueString(),
	}

	switch {
	case !plan.CIDR.IsUnknown():
		ruleReq.Network = plan.CIDR.ValueString()
	case !plan.PublicSecurityGroup.IsUnknown():
		name := plan.PublicSecurityGroup.ValueString()
		sgs, err := r.client.ListSecurityGroups(
			ctx,
			exoscale.ListSecurityGroupsWithVisibility(
				exoscale.ListSecurityGroupsVisibilityPublic),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"API returned error listing public security groups",
				err.Error(),
			)

			return
		}
		_, err = sgs.FindSecurityGroup(name)
		if err != nil {
			resp.Diagnostics.AddError(
				"public security group not found",
				err.Error(),
			)

			return
		}
		ruleReq.SecurityGroup = &exoscale.SecurityGroupResource{
			Name:       name,
			Visibility: exoscale.SecurityGroupResourceVisibilityPublic,
		}
	case !plan.UserSecurityGroupID.IsUnknown():
		id, err := exoscale.ParseUUID(plan.UserSecurityGroupID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"unable to parse user security group ID",
				err.Error(),
			)
			return
		}
		sg, err := r.client.GetSecurityGroup(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				"API returned error reading user security group",
				err.Error(),
			)
			return
		}
		ruleReq.SecurityGroup = &exoscale.SecurityGroupResource{
			ID:         sg.ID,
			Name:       sg.Name,
			Visibility: exoscale.SecurityGroupResourceVisibilityPrivate,
		}
	default: // validation must prevent reaching here
		resp.Diagnostics.AddError(
			"missing required field",
			"requires cidr, public_security_group or user_security_group_id",
		)
		return
	}

	switch {
	case !plan.StartPort.IsUnknown() && !plan.EndPort.IsUnknown():
		ruleReq.StartPort = plan.StartPort.ValueInt64()
		ruleReq.EndPort = plan.EndPort.ValueInt64()
	case !plan.ICMPType.IsUnknown() && !plan.ICMPCode.IsUnknown():
		ruleReq.ICMP = &exoscale.AddRuleToSecurityGroupRequestICMP{
			Type: plan.ICMPType.ValueInt64Pointer(),
			Code: plan.ICMPCode.ValueInt64Pointer(),
		}
	default: // validation must prevent reaching here
		resp.Diagnostics.AddError(
			"missing required field",
			"requires start_port/end_port or icmp_type/icmp_code",
		)
		return
	}

	op, err := r.client.AddRuleToSecurityGroup(
		ctx,
		parentID,
		ruleReq,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"API returned error when adding rule to security group rule",
			err.Error(),
		)
		return
	}
	if _, err := r.client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError(
			"create security group operation failed",
			err.Error(),
		)
		return
	}

	// op.Reference.ID is not rule ID but SG ID,
	// we need to fetch rule ID from SG.
	rule := exoscale.SecurityGroupRule{
		Description:   ruleReq.Description,
		FlowDirection: exoscale.SecurityGroupRuleFlowDirection(ruleReq.FlowDirection),
		Protocol:      exoscale.SecurityGroupRuleProtocol(ruleReq.Protocol),
		Network:       ruleReq.Network,
		SecurityGroup: ruleReq.SecurityGroup,
		StartPort:     ruleReq.StartPort,
		EndPort:       ruleReq.EndPort,
	}
	if ruleReq.ICMP != nil {
		rule.ICMP = &exoscale.SecurityGroupRuleICMP{
			Type: plan.ICMPType.ValueInt64(),
			Code: plan.ICMPCode.ValueInt64(),
		}
	}
	if t, err := r.FindRemoteRule(ctx, parentID, rule); err != nil {
		resp.Diagnostics.AddError(
			"API returned error when reading security group",
			err.Error(),
		)
		return
	} else {
		plan.ID = types.StringValue(t.ID.String())
	}

	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}
	if plan.CIDR.IsUnknown() {
		plan.CIDR = types.StringNull()
	}
	if plan.PublicSecurityGroup.IsUnknown() {
		plan.PublicSecurityGroup = types.StringNull()
	}
	if plan.UserSecurityGroupID.IsUnknown() {
		plan.UserSecurityGroupID = types.StringNull()
	}
	if plan.StartPort.IsUnknown() || plan.EndPort.IsUnknown() {
		plan.StartPort = types.Int64Null()
		plan.EndPort = types.Int64Null()
	}
	if plan.ICMPType.IsUnknown() || plan.ICMPCode.IsUnknown() {
		plan.ICMPType = types.Int64Null()
		plan.ICMPCode = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read (refresh) resources by receiving Terraform prior state data, performing read logic, and saving refreshed Terraform state data.
func (r *ResourceRule) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state ResourceRuleModel

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
		tflog.Info(
			ctx,
			"rule has no ID, deleting from state to report drift",
			map[string]any{},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	parentID, err := exoscale.ParseUUID(state.SecurityGroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}
	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}
	rule, err := r.FindRemoteRule(ctx, parentID, exoscale.SecurityGroupRule{ID: id})
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			tflog.Info(
				ctx,
				"remote rule not found, deleting from state to report drift",
				map[string]any{},
			)
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"API returned error reading security group rule",
				err.Error(),
			)
		}

		return
	}

	state.Description = types.StringValue(rule.Description)

	// Type and Protocol are case insensitive, compare and keep same case
	lowercaser := cases.Lower(language.Und)
	if lowercaser.String(state.Type.ValueString()) != string(rule.FlowDirection) {
		state.Type = types.StringValue(string(rule.FlowDirection))
	}
	if lowercaser.String(state.Protocol.ValueString()) != string(rule.Protocol) {
		state.Protocol = types.StringValue(string(rule.Protocol))
	}

	if rule.Network != "" {
		state.CIDR = types.StringValue(rule.Network)
	} else if rule.SecurityGroup != nil {
		if rule.SecurityGroup.ID != "" {
			state.UserSecurityGroupID = types.StringValue(string(rule.SecurityGroup.ID))
		} else if rule.SecurityGroup.Name != "" {
			state.PublicSecurityGroup = types.StringValue(string(rule.SecurityGroup.Name))
		}
	}
	if rule.StartPort > 0 && rule.EndPort > 0 {
		state.StartPort = types.Int64Value(rule.StartPort)
		state.EndPort = types.Int64Value(rule.EndPort)
	}
	if rule.ICMP != nil {
		state.ICMPType = types.Int64Value(rule.ICMP.Type)
		state.ICMPCode = types.Int64Value(rule.ICMP.Code)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update resources in-place by receiving Terraform prior state, configuration, and plan data, performing update logic, and saving updated Terraform state data.
func (r *ResourceRule) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Nothing to do as all SG rule attributes require replace.
}

// Delete resources by receiving Terraform prior state data and performing deletion logic.
func (r *ResourceRule) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ResourceRuleModel

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

	parentID, err := exoscale.ParseUUID(state.SecurityGroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}
	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}

	op, err := r.client.DeleteRuleFromSecurityGroup(ctx, parentID, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			tflog.Info(
				ctx,
				"delete rule returned 404, nothing to do",
				map[string]any{},
			)
			return
		}
		resp.Diagnostics.AddError(
			"API returned error when deleting security group",
			err.Error(),
		)
		return
	}
	_, err = r.client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError(
			"delete security group operation failed",
			err.Error(),
		)
		return
	}
}

// ImportState lets Terraform begin managing existing infrastructure resources.
func (r *ResourceRule) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			"Requires format: security_group_id@rule_id",
		)
		return
	}
	resp.Diagnostics.Append(
		resp.State.SetAttribute(
			ctx,
			path.Root("security_group_id"), idParts[0])...)
	resp.Diagnostics.Append(
		resp.State.SetAttribute(
			ctx,
			path.Root("id"), idParts[1])...)
}

// FindRemoteRule returns a Rule from API.
// There is no direct API call to fetch the Rule and adding a Rule returns SG ID instead.
// Rule needs to be extracted from SG struct which contains a list of all child Rules.
// This function receives SG ID and partial Rule as input and returns a complete Rule.
// If input rule has ID, we match it in SG output and return it.
// If ID is empty we try to match every other parameter instead.
func (r *ResourceRule) FindRemoteRule(
	ctx context.Context,
	sgID exoscale.UUID,
	rule exoscale.SecurityGroupRule,
) (*exoscale.SecurityGroupRule, error) {
	sg, err := r.client.GetSecurityGroup(ctx, sgID)
	if err != nil {
		return nil, err
	}

	for _, item := range sg.Rules {
		if rule.ID != "" && item.ID == rule.ID {
			return &item, nil
		}
		if item.Description == rule.Description &&
			item.FlowDirection == rule.FlowDirection &&
			item.Protocol == rule.Protocol &&
			item.Network == rule.Network &&
			item.StartPort == rule.StartPort &&
			item.EndPort == rule.EndPort &&
			((item.SecurityGroup == nil && rule.SecurityGroup == nil) ||
				(item.SecurityGroup != nil &&
					rule.SecurityGroup != nil &&
					item.SecurityGroup.ID == rule.SecurityGroup.ID &&
					item.SecurityGroup.Name == rule.SecurityGroup.Name)) &&
			((item.ICMP == nil && rule.ICMP == nil) ||
				(item.ICMP != nil && rule.ICMP != nil && *item.ICMP == *rule.ICMP)) {
			return &item, nil
		}
	}

	return nil, exoscale.ErrNotFound
}
