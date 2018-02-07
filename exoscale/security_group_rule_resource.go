package exoscale

import (
	"fmt"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func securityGroupRuleResource() *schema.Resource {
	return &schema.Resource{
		Create: createSecurityGroupRule,
		Exists: existsSecurityGroupRule,
		Read:   readSecurityGroupRule,
		Delete: deleteSecurityGroupRule,

		Importer: &schema.ResourceImporter{
			State: importSecurityGroupRule,
		},

		Schema: map[string]*schema.Schema{
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"INGRESS", "EGRESS"}, true),
			},
			"security_group_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"security_group"},
			},
			"security_group": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"security_group_id"},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"cidr": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.CIDRNetwork(0, 32),
				ConflictsWith: []string{"user_security_group"},
			},
			"protocol": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "tcp",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"TCP", "UDP", "ICMP", "AH", "ESP", "GRE"}, true),
			},
			"start_port": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(1, 65535),
				ConflictsWith: []string{"icmp_type", "icmp_code"},
			},
			"end_port": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(1, 65535),
				ConflictsWith: []string{"icmp_type", "icmp_code"},
			},
			"icmp_type": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(0, 255),
				ConflictsWith: []string{"start_port", "end_port"},
			},
			"icmp_code": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(0, 255),
				ConflictsWith: []string{"start_port", "end_port"},
			},
			"user_security_group_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cidr", "user_security_group"},
			},
			"user_security_group": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cidr", "user_security_group_id"},
			},
		},
	}
}

func createSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async
	r := &egoscale.ListSecurityGroups{}
	securityGroupID, ok := d.GetOkExists("security_group_id")
	if ok {
		r.ID = securityGroupID.(string)
	} else {
		r.SecurityGroupName = d.Get("security_group").(string)
	}

	resp, err := client.Request(r)
	if err != nil {
		return err
	}

	securityGroup := resp.(*egoscale.ListSecurityGroupsResponse).SecurityGroup[0]

	cidrList := make([]string, 0)
	groupList := make([]egoscale.UserSecurityGroup, 0)

	cidr, cidrOk := d.GetOk("cidr")
	if cidrOk {
		cidrList = append(cidrList, cidr.(string))
	} else {
		userSecurityGroupID, idOk := d.GetOk("user_security_group_id")
		userSecurityGroupName, nameOk := d.GetOk("user_security_group")

		if !idOk && !nameOk {
			return fmt.Errorf("No CIDR, User Security Group ID or Name were provided")
		}

		resp, err := client.Request(&egoscale.ListSecurityGroups{
			ID:                userSecurityGroupID.(string),
			SecurityGroupName: userSecurityGroupName.(string),
		})

		if err != nil {
			return err
		}

		groups := resp.(*egoscale.ListSecurityGroupsResponse)
		if len(groups.SecurityGroup) == 0 {
			return fmt.Errorf("No groups found for id: %s or name: %s", userSecurityGroupID, userSecurityGroupName)
		}

		group := groups.SecurityGroup[0]
		groupList = append(groupList, egoscale.UserSecurityGroup{
			Account: group.Account,
			Group:   group.Name,
		})
	}

	var req egoscale.AsyncCommand
	req = &egoscale.AuthorizeSecurityGroupIngress{
		SecurityGroupID:       securityGroup.ID,
		CidrList:              cidrList,
		Description:           d.Get("description").(string),
		Protocol:              d.Get("protocol").(string),
		EndPort:               d.Get("end_port").(int),
		StartPort:             d.Get("start_port").(int),
		IcmpType:              d.Get("icmp_type").(int),
		IcmpCode:              d.Get("icmp_code").(int),
		UserSecurityGroupList: groupList,
	}

	trafficType := strings.ToUpper(d.Get("type").(string))
	if trafficType == "EGRESS" {
		// yay! types
		req = (*egoscale.AuthorizeSecurityGroupEgress)(req.(*egoscale.AuthorizeSecurityGroupIngress))
	}

	resp, err = client.AsyncRequest(req, async)
	if err != nil {
		return err
	}

	// The rule allowed for creation produces only one rule!
	d.Set("type", trafficType)
	if trafficType == "EGRESS" {
		securityGroup = resp.(*egoscale.AuthorizeSecurityGroupEgressResponse).SecurityGroup
		d.Set("type", trafficType)
		return applySecurityGroupRule(d, securityGroup, securityGroup.EgressRule[0])
	}

	securityGroup = resp.(*egoscale.AuthorizeSecurityGroupIngressResponse).SecurityGroup
	return applySecurityGroupRule(d, securityGroup, (egoscale.EgressRule)(securityGroup.IngressRule[0]))
}

func existsSecurityGroupRule(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)

	securityGroupID := ""
	securityGroupName := ""

	if s, ok := d.GetOkExists("security_group_id"); ok {
		securityGroupID = s.(string)
	} else if n, ok := d.GetOkExists("security_group"); ok {
		securityGroupName = n.(string)
	} else {
		return false, fmt.Errorf("Missing either Security Group ID or Name")
	}

	resp, err := client.Request(&egoscale.ListSecurityGroups{
		ID:                securityGroupID,
		SecurityGroupName: securityGroupName,
	})
	if err != nil {
		return false, err
	}

	id := d.Id()
	groups := resp.(*egoscale.ListSecurityGroupsResponse)
	for _, sg := range groups.SecurityGroup {
		for _, rule := range sg.EgressRule {
			if rule.RuleID == id {
				d.Set("type", "EGRESS")
				return true, nil
			}
		}
		for _, rule := range sg.IngressRule {
			if rule.RuleID == id {
				d.Set("type", "INGRESS")
				return true, nil
			}
		}
	}

	return false, nil
}

func readSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	securityGroupID := ""
	securityGroupName := ""
	if s, ok := d.GetOkExists("security_group_id"); ok {
		securityGroupID = s.(string)
	} else if n, ok := d.GetOkExists("security_group"); ok {
		securityGroupName = n.(string)
	} else {
		return fmt.Errorf("Missing either Security Group ID or Name")
	}

	resp, err := client.Request(&egoscale.ListSecurityGroups{
		ID:                securityGroupID,
		SecurityGroupName: securityGroupName,
	})
	if err != nil {
		return err
	}

	id := d.Id()
	groups := resp.(*egoscale.ListSecurityGroupsResponse)
	for _, sg := range groups.SecurityGroup {
		for _, rule := range sg.EgressRule {
			if rule.RuleID == id {
				d.Set("type", "EGRESS")
				return applySecurityGroupRule(d, sg, rule)
			}
		}
		for _, rule := range sg.IngressRule {
			if rule.RuleID == id {
				d.Set("type", "INGRESS")
				return applySecurityGroupRule(d, sg, (egoscale.EgressRule)(rule))
			}
		}
	}

	d.SetId("")
	return nil
}

func deleteSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	id := d.Id()
	var req egoscale.AsyncCommand
	if d.Get("type").(string) == "EGRESS" {
		req = &egoscale.RevokeSecurityGroupEgress{
			ID: id,
		}
	} else {
		req = &egoscale.RevokeSecurityGroupIngress{
			ID: id,
		}
	}

	return client.BooleanAsyncRequest(req, async)
}

func importSecurityGroupRule(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := readSecurityGroupRule(d, meta); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 1)
	resources[0] = d
	return resources, nil
}

func applySecurityGroupRule(d *schema.ResourceData, group egoscale.SecurityGroup, rule egoscale.EgressRule) error {
	d.SetId(rule.RuleID)
	d.Set("cidr", rule.Cidr)
	d.Set("icmp_type", rule.IcmpType)
	d.Set("icmp_code", rule.IcmpCode)
	d.Set("start_port", rule.StartPort)
	d.Set("end_port", rule.EndPort)
	d.Set("protocol", strings.ToUpper(rule.Protocol))

	d.Set("user_security_group", rule.SecurityGroupName)

	d.Set("security_group_id", group.ID)
	d.Set("security_group", group.Name)

	return nil
}
