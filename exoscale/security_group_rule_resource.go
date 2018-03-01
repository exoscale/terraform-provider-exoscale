package exoscale

import (
	"context"
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

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
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
				ValidateFunc:  validation.CIDRNetwork(0, 128),
				ConflictsWith: []string{"user_security_group"},
			},
			"protocol": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "tcp",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"TCP", "UDP", "ICMP", "ICMPv6", "AH", "ESP", "GRE"}, true),
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
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup := &egoscale.SecurityGroup{}
	securityGroupID, ok := d.GetOkExists("security_group_id")
	if ok {
		securityGroup.ID = securityGroupID.(string)
	} else {
		securityGroup.Name = d.Get("security_group").(string)
	}

	if err := client.GetWithContext(ctx, securityGroup); err != nil {
		return err
	}

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

		group := &egoscale.SecurityGroup{
			ID:   userSecurityGroupID.(string),
			Name: userSecurityGroupName.(string),
		}

		if err := client.GetWithContext(ctx, group); err != nil {
			return err
		}

		groupList = append(groupList, egoscale.UserSecurityGroup{
			Account: group.Account,
			Group:   group.Name,
		})
	}

	var req egoscale.Command
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

	resp, err := client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	// The rule allowed for creation produces only one rule!
	d.Set("type", trafficType)
	if trafficType == "EGRESS" {
		sg := resp.(*egoscale.AuthorizeSecurityGroupEgressResponse).SecurityGroup
		d.Set("type", trafficType)
		return applySecurityGroupRule(d, securityGroup, sg.EgressRule[0])
	}

	sg := resp.(*egoscale.AuthorizeSecurityGroupIngressResponse).SecurityGroup
	return applySecurityGroupRule(d, securityGroup, (egoscale.EgressRule)(sg.IngressRule[0]))
}

func existsSecurityGroupRule(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

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

	sg := &egoscale.SecurityGroup{
		ID:   securityGroupID,
		Name: securityGroupName,
	}
	if err := client.GetWithContext(ctx, sg); err != nil {
		return false, err
	}

	switch d.Get("type") {
	case "EGRESS":
		for _, rule := range sg.EgressRule {
			if rule.RuleID == d.Id() {
				return true, nil
			}
		}
	case "INGRESS":
		for _, rule := range sg.IngressRule {
			if rule.RuleID == d.Id() {
				return true, nil
			}
		}
	}
	return false, nil
}

func readSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

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

	sg := &egoscale.SecurityGroup{
		ID:   securityGroupID,
		Name: securityGroupName,
	}
	if err := client.GetWithContext(ctx, sg); err != nil {
		return err
	}

	id := d.Id()
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

	d.SetId("")
	return nil
}

func deleteSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id := d.Id()
	var req egoscale.Command
	if d.Get("type").(string) == "EGRESS" {
		req = &egoscale.RevokeSecurityGroupEgress{
			ID: id,
		}
	} else {
		req = &egoscale.RevokeSecurityGroupIngress{
			ID: id,
		}
	}

	return client.BooleanRequestWithContext(ctx, req)
}

func importSecurityGroupRule(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := readSecurityGroupRule(d, meta); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 1)
	resources[0] = d
	return resources, nil
}

func applySecurityGroupRule(d *schema.ResourceData, group *egoscale.SecurityGroup, rule egoscale.EgressRule) error {
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
