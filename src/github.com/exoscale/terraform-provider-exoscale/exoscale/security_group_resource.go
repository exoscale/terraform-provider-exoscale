package exoscale

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pyr/egoscale/src/egoscale"
)

func securityGroupResource() *schema.Resource {
	return &schema.Resource{
		Create: sgCreate,
		Read:   sgRead,
		Delete: sgDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:		schema.TypeString,
				Computed:	true,
			},
			"name": &schema.Schema{
				Type:		schema.TypeString,
				ForceNew:	true,
				Required:	true,
			},
			"ingress_rules": &schema.Schema{
				Type:		schema.TypeList,
				Optional:	true,
				ForceNew:   true,
				Elem:		&schema.Resource{
					Schema: map[string]*schema.Schema{
						"sgid": &schema.Schema{
							Type:		schema.TypeString,
							Computed:	true,
						},
						"cidr": &schema.Schema{
							Type:		schema.TypeString,
							Required:	true,
						},
						"protocol": &schema.Schema{
							Type:		schema.TypeString,
							Required:	true,
						},
						"port": &schema.Schema{
							Type:		schema.TypeInt,
							Optional:	true,
						},
						"icmptype": &schema.Schema{
							Type:		schema.TypeInt,
							Optional:	true,
						},
						"icmpcode": &schema.Schema{
							Type:		schema.TypeInt,
							Optional:	true,
						},
					},
				},
			},
			"egress_rules": &schema.Schema{
				Type:		schema.TypeList,
				Optional:	true,
				ForceNew:	true,
				Elem:		&schema.Resource{
					Schema: map[string]*schema.Schema{
						"sgid": &schema.Schema{
							Type:		schema.TypeString,
							Computed:	true,
						},
						"cidr": &schema.Schema{
							Type:		schema.TypeString,
							Required:	true,
						},
						"protocol": &schema.Schema{
							Type:		schema.TypeString,
							Required:	true,
						},
						"port": &schema.Schema{
							Type:		schema.TypeInt,
							Optional:	true,
						},
						"icmptype": &schema.Schema{
							Type:		schema.TypeInt,
							Optional:	true,
						},
						"icmpcode": &schema.Schema{
							Type:		schema.TypeInt,
							Optional:	true,
						},
					},
				},
			},
		},
	}
}

func sgCreate(d *schema.ResourceData, meta interface{}) error {
	var i int
	client := GetClient(ComputeEndpoint, meta)

	ingressLength := d.Get("ingress_rules.#").(int)
	egressLength := d.Get("egress_rules.#").(int)

	ingressRules := make([]egoscale.SecurityGroupRule, ingressLength)
	egressRules := make([]egoscale.SecurityGroupRule, egressLength)

	for i = 0; i < ingressLength; i++ {
		var rule egoscale.SecurityGroupRule
		key := fmt.Sprintf("ingress_rules.%d.", i)

		rule.SecurityGroupId = ""
		rule.Cidr = d.Get(key + "cidr").(string)
		rule.Protocol = d.Get(key + "protocol").(string)
		rule.Port = d.Get(key + "port").(int)
		rule.IcmpType = d.Get(key + "icmptype").(int)
		rule.IcmpCode = d.Get(key + "icmpcode").(int)
		ingressRules[i] = rule
	}

	for i = 0; i < egressLength; i++ {
		var rule egoscale.SecurityGroupRule
		key := fmt.Sprintf("egress_rules.%d.", i)

		rule.SecurityGroupId = ""
		rule.Cidr = d.Get(key + "cidr").(string)
		rule.Protocol = d.Get(key + "protocol").(string)
		rule.Port = d.Get(key + "port").(int)
		rule.IcmpType = d.Get(key + "icmptype").(int)
		rule.IcmpCode = d.Get(key + "icmpcode").(int)
		egressRules[i] = rule
	}

	resp, err := client.CreateSecurityGroupWithRules(d.Get("name").(string),
		ingressRules, egressRules); if err != nil {
		return err
	}

	d.SetId(resp.Id)

	/* Update the sgid field for all of the ingress/egress rules */
	for i = 0; i < ingressLength; i++ {
		key := fmt.Sprintf("ingress_rules.%d.", i)
		d.Set(key + "sgid", resp.Id)
	}

	for i = 0; i < egressLength; i++ {
		key := fmt.Sprintf("egress_rules.%d.", i)
		d.Set(key + "sgid", resp.Id)
	}


	return sgRead(d, meta)
}

func sgRead(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	sgs, err := client.GetSecurityGroups()
	if err != nil {
		return err
	}

	var securityGroups egoscale.SecurityGroup
	for _,v := range sgs {
		if d.Id() == v.Id {
			securityGroups = v
			break;
		}
	}

	d.Set("id", securityGroups.Id)
	d.Set("name", securityGroups.Name)
	d.Set("description", securityGroups.Description)
	d.Set("ingress_rules.#", len(securityGroups.IngressRules))
	d.Set("egress_rules.#", len(securityGroups.EgressRules))

	for i := 0; i < len(securityGroups.IngressRules); i++ {
		key := fmt.Sprintf("ingress_rules.%d.", i)
		d.Set(key + "sgid", securityGroups.IngressRules[i].RuleId)
		d.Set(key + "port", securityGroups.IngressRules[i].StartPort)
		d.Set(key + "cidr", securityGroups.IngressRules[i].Cidr)
		d.Set(key + "protocol", securityGroups.IngressRules[i].Protocol)
		d.Set(key + "icmpcode", securityGroups.IngressRules[i].IcmpCode)
		d.Set(key + "icmptype", securityGroups.IngressRules[i].IcmpType)
	}

	for i := 0; i < len(securityGroups.EgressRules); i++ {
		key := fmt.Sprintf("egress_rules.%d.", i)
		d.Set(key + "sgid", securityGroups.EgressRules[i].RuleId)
		d.Set(key + "port", securityGroups.EgressRules[i].StartPort)
		d.Set(key + "cidr", securityGroups.EgressRules[i].Cidr)
		d.Set(key + "protocol", securityGroups.EgressRules[i].Protocol)
		d.Set(key + "icmpcode", securityGroups.EgressRules[i].IcmpCode)
		d.Set(key + "icmptype", securityGroups.EgressRules[i].IcmpType)
	}



	return nil
}

func sgDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	err := client.DeleteSecurityGroup(d.Get("name").(string)); if err != nil {
		return err
	}

	return nil
}
