package exoscale

import (
	"context"
	"fmt"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

// supportedProtocols contains the allowed protocols
var supportedProtocols = []string{
	"TCP", "UDP", "ICMP", "ICMPv6", "AH", "ESP", "GRE", "IPIP", "ALL",
}

func securityGroupRuleResource() *schema.Resource {
	return &schema.Resource{
		Create: createSecurityGroupRule,
		Exists: existsSecurityGroupRule,
		Read:   readSecurityGroupRule,
		Delete: deleteSecurityGroupRule,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
				Default:      "TCP",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(supportedProtocols, true),
			},
			"start_port": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(0, 65535),
				ConflictsWith: []string{"icmp_type", "icmp_code"},
			},
			"end_port": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(0, 65535),
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

	sg, err := inferSecurityGroup(d)
	if err != nil {
		return err
	}

	resp, err := client.GetWithContext(ctx, sg)
	if err != nil {
		return err
	}

	securityGroup := resp.(*egoscale.SecurityGroup)

	cidrList := make([]egoscale.CIDR, 0)
	groupList := make([]egoscale.UserSecurityGroup, 0)

	cidr, cidrOk := d.GetOk("cidr")
	if cidrOk {
		c, err := egoscale.ParseCIDR(cidr.(string))
		if err != nil {
			return err
		}
		cidrList = append(cidrList, *c)
	} else {
		userSecurityGroupID := d.Get("user_security_group_id").(string)
		userSecurityGroupName := d.Get("user_security_group").(string)

		if userSecurityGroupID == "" && userSecurityGroupName == "" {
			return fmt.Errorf("No CIDR, User Security Group ID or Name were provided")
		}

		group := &egoscale.SecurityGroup{
			Name: userSecurityGroupName,
		}

		if userSecurityGroupID != "" {
			id, err := egoscale.ParseUUID(userSecurityGroupID)
			if err != nil {
				return err
			}
			group.ID = id
		}

		resp, err := client.GetWithContext(ctx, group)
		if err != nil {
			return err
		}

		g := resp.(*egoscale.SecurityGroup)
		groupList = append(groupList, g.UserSecurityGroup())
	}

	var req egoscale.Command // nolint: megacheck
	req = &egoscale.AuthorizeSecurityGroupIngress{
		SecurityGroupID:       securityGroup.ID,
		CIDRList:              cidrList,
		Description:           d.Get("description").(string),
		Protocol:              d.Get("protocol").(string),
		EndPort:               (uint16)(d.Get("end_port").(int)),
		StartPort:             (uint16)(d.Get("start_port").(int)),
		IcmpType:              (uint8)(d.Get("icmp_type").(int)),
		IcmpCode:              (uint8)(d.Get("icmp_code").(int)),
		UserSecurityGroupList: groupList,
	}

	trafficType := strings.ToUpper(d.Get("type").(string))
	if trafficType == "EGRESS" {
		// yay! types
		req = (*egoscale.AuthorizeSecurityGroupEgress)(req.(*egoscale.AuthorizeSecurityGroupIngress))
	}

	resp, err = client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	sg = resp.(*egoscale.SecurityGroup)

	// The rule allowed for creation produces only one rule!
	if err := d.Set("type", trafficType); err != nil {
		return err
	}
	if trafficType == "EGRESS" {
		if len(sg.EgressRule) != 1 {
			return fmt.Errorf("no security group rules were created, aborting")
		}

		return applySecurityGroupRule(d, securityGroup, sg.EgressRule[0])
	}

	if len(sg.IngressRule) != 1 {
		return fmt.Errorf("no security group rules were created, aborting")
	}

	return applySecurityGroupRule(d, securityGroup, (egoscale.EgressRule)(sg.IngressRule[0]))
}

func existsSecurityGroupRule(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	sg, err := inferSecurityGroup(d)
	if err != nil {
		return false, err
	}

	var ingressRule egoscale.IngressRule
	var egressRule egoscale.EgressRule

	id := d.Id()
	req, err := sg.ListRequest()
	if err != nil {
		return false, err
	}
	client.PaginateWithContext(ctx, req, func(i interface{}, e error) bool {
		if e != nil {
			err = e
			return false
		}

		s, ok := i.(*egoscale.SecurityGroup)
		if !ok {
			err = fmt.Errorf("type SecurityGroup was expected got %T", i)
			return false
		}

		for _, rule := range s.EgressRule {
			if rule.RuleID.String() == id {
				sg = s
				egressRule = rule
				return false
			}
		}
		for _, rule := range s.IngressRule {
			if rule.RuleID.String() == id {
				sg = s
				ingressRule = rule
				return false
			}
		}

		return true
	})

	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	if egressRule.RuleID != nil || ingressRule.RuleID != nil {
		return true, nil
	}

	return false, nil
}

func readSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	sg, err := inferSecurityGroup(d)
	if err != nil {
		return err
	}

	var ingressRule egoscale.IngressRule
	var egressRule egoscale.EgressRule

	id := d.Id()
	req, err := sg.ListRequest()
	if err != nil {
		return err
	}
	client.PaginateWithContext(ctx, req, func(i interface{}, e error) bool {
		if e != nil {
			err = e
			return false
		}

		s, ok := i.(*egoscale.SecurityGroup)
		if !ok {
			err = fmt.Errorf("type SecurityGroup was expected got %T", i)
			return false
		}

		for _, rule := range s.EgressRule {
			if rule.RuleID.String() == id {
				sg = s
				egressRule = rule
				return false
			}
		}
		for _, rule := range s.IngressRule {
			if rule.RuleID.String() == id {
				sg = s
				ingressRule = rule
				return false
			}
		}

		return true
	})

	if err != nil {
		return handleNotFound(d, err)
	}

	if egressRule.RuleID != nil {
		d.Set("type", "EGRESS") // nolint: errcheck
		return applySecurityGroupRule(d, sg, egressRule)
	}

	if ingressRule.RuleID != nil {
		d.Set("type", "INGRESS") // nolint: errcheck
		return applySecurityGroupRule(d, sg, (egoscale.EgressRule)(ingressRule))
	}

	d.SetId("")
	return nil
}

func deleteSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

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

func applySecurityGroupRule(d *schema.ResourceData, group *egoscale.SecurityGroup, rule egoscale.EgressRule) error {
	d.SetId(rule.RuleID.String())
	cidr := ""
	if rule.CIDR != nil {
		cidr = rule.CIDR.String()
	}

	if err := d.Set("cidr", cidr); err != nil {
		return err
	}
	if err := d.Set("icmp_type", rule.IcmpType); err != nil {
		return err
	}
	if err := d.Set("icmp_code", rule.IcmpCode); err != nil {
		return err
	}
	if err := d.Set("start_port", rule.StartPort); err != nil {
		return err
	}
	if err := d.Set("end_port", rule.EndPort); err != nil {
		return err
	}
	protocol := strings.ToUpper(rule.Protocol)
	protocol = strings.Replace(protocol, "V6", "v6", -1)
	if err := d.Set("protocol", protocol); err != nil {
		return err
	}
	if err := d.Set("user_security_group", rule.SecurityGroupName); err != nil {
		return err
	}

	if err := d.Set("security_group_id", group.ID.String()); err != nil {
		return err
	}
	if err := d.Set("security_group", group.Name); err != nil {
		return err
	}

	return nil
}

func inferSecurityGroup(d *schema.ResourceData) (*egoscale.SecurityGroup, error) {
	var securityGroupID *egoscale.UUID
	var securityGroupName string

	if s, ok := d.GetOkExists("security_group_id"); ok {
		var err error
		securityGroupID, err = egoscale.ParseUUID(s.(string))
		if err != nil {
			return nil, err
		}
	} else if n, ok := d.GetOkExists("security_group"); ok {
		securityGroupName = n.(string)
	}

	return &egoscale.SecurityGroup{
		ID:   securityGroupID,
		Name: securityGroupName,
	}, nil
}
