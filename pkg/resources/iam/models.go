package iam

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PolicyModel struct {
	DefaultServiceStrategy types.String `tfsdk:"default_service_strategy"`
	Services               types.Map    `tfsdk:"services"`
}

func (m PolicyModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"default_service_strategy": types.StringType,
		"services": types.MapType{
			ElemType: types.ObjectType{AttrTypes: PolicyServiceModel{}.Types()},
		},
	}
}

type PolicyServiceModel struct {
	Type  types.String `tfsdk:"type"`
	Rules types.List   `tfsdk:"rules"`
}

func (m PolicyServiceModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"type": types.StringType,
		"rules": types.ListType{
			ElemType: types.ObjectType{AttrTypes: PolicyServiceRuleModel{}.Types()},
		},
	}
}

type PolicyServiceRuleModel struct {
	Action     types.String `tfsdk:"action"`
	Expression types.String `tfsdk:"expression"`
	Resources  types.List   `tfsdk:"resources"`
}

func (m PolicyServiceRuleModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"action":     types.StringType,
		"expression": types.StringType,
		"resources": types.ListType{
			ElemType: types.StringType,
		},
	}
}
