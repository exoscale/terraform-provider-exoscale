package exoscale

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/runseb/egoscale/src/egoscale"
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
			"ingressRules": &schema.Schema{
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
			"egressRules": &schema.Schema{
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

	ingressLength := d.Get("ingressRules.#").(int)
	egressLength := d.Get("egressRules.#").(int)

	ingressRules := make([]egoscale.SecurityGroupRule, ingressLength)
	egressRules := make([]egoscale.SecurityGroupRule, egressLength)

	for i = 0; i < ingressLength; i++ {
		var rule egoscale.SecurityGroupRule
		key := fmt.Sprintf("ingressRules.%d.", i)
		
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
		key := fmt.Sprintf("egressRules.%d.", i)
		
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
		key := fmt.Sprintf("ingressRules.%d.", i)
		d.Set(key + "sgid", resp.Id)
	}

	for i = 0; i < egressLength; i++ {
		key := fmt.Sprintf("egressRules.%d.", i)
		d.Set(key + "sgid", resp.Id)
	}


	return sgRead(d, meta)
}

func sgRead(d *schema.ResourceData, meta interface{}) error {
	/*
	 * We cannot retrieve the ingress/egress rules at this time, and the only
	 * thing that could possibly change on the group side is the name so do
	 * nothing for now.
	 */
	return nil
}

func sgDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	err := client.DeleteSecurityGroup(d.Get("name").(string)); if err != nil {
		return err
	}

	return nil
}