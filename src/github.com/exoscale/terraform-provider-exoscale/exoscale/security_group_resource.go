package exoscale

import (
	"errors"
	"fmt"
	"log"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func securityGroupResource() *schema.Resource {
	return &schema.Resource{
		Create: sgCreate,
		Read:   sgRead,
		Delete: sgDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"account": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ingress_rules": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"rule_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"cidr": {
							Type:     schema.TypeString,
							Optional: true,
							// XXX TODO
							// https://github.com/hashicorp/terraform/issues/13016
							//ConflictsWith: []string{"user_security_group_list"},
						},
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
							DefaultFunc: func() (interface{}, error) {
								return "TCP", nil
							},
						},
						"port": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"start_port": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"end_port": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"icmptype": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"icmpcode": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"user_security_group_list": {
							Type:     schema.TypeSet,
							Optional: true,
							//ConflictsWith: []string{"cidr"},
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Set: schema.HashString,
						},
					},
				},
			},
			"egress_rules": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sgid": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"cidr": {
							Type:     schema.TypeString,
							Optional: true,
							//ConflictsWith: []string{"user_security_group_list"},
						},
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
							DefaultFunc: func() (interface{}, error) {
								return "TCP", nil
							},
						},
						"port": {
							Type:     schema.TypeInt,
							Optional: true,
							//ConflictsWith: []string{"start_port", "end_port"},
						},
						"start_port": {
							Type:     schema.TypeInt,
							Optional: true,
							//ConflictsWith: []string{"start_port", "end_port"},
						},
						"end_port": {
							Type:     schema.TypeInt,
							Optional: true,
							//ConflictsWith: []string{"start_port"},
						},
						"icmptype": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"icmpcode": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"user_security_group_list": {
							Type:     schema.TypeSet,
							Optional: true,
							//ConflictsWith: []string{"cidr"},
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Set: schema.HashString,
						},
					},
				},
			},
		},
	}
}

func sgCreate(d *schema.ResourceData, meta interface{}) error {
	var i int
	client := GetComputeClient(meta)

	ingressLength := d.Get("ingress_rules.#").(int)
	egressLength := d.Get("egress_rules.#").(int)

	ingressRules := make([]egoscale.SecurityGroupRule, ingressLength)
	egressRules := make([]egoscale.SecurityGroupRule, egressLength)

	var rules = []struct {
		length int
		key    string
		rules  []egoscale.SecurityGroupRule
	}{
		{
			ingressLength,
			"ingress_rules",
			ingressRules,
		},
		{
			egressLength,
			"egress_rules",
			egressRules,
		},
	}

	for _, r := range rules {
		for i = 0; i < r.length; i++ {
			var rule egoscale.SecurityGroupRule
			var groupList []*egoscale.UserSecurityGroup

			key := fmt.Sprintf("%s.%d.", r.key, i)

			cidr := d.Get(key + "cidr").(string)

			if groups, ok := d.Get(key + "user_security_group_list").(*schema.Set); ok {
				groupList = make([]*egoscale.UserSecurityGroup, groups.Len())
				for i, group := range groups.List() {
					groupName := group.(string)

					securityGroup, err := getSecurityGroup(client, groupName)

					if err == nil {
						groupList[i] = &egoscale.UserSecurityGroup{
							Account: securityGroup.Account,
							Group:   securityGroup.Name,
						}
					} else {
						return fmt.Errorf("Security Group not found %v", groupName)
					}
				}
			}

			rule.SecurityGroupId = ""
			// CIDR vs User Security Group
			log.Printf("[DEBUG] %v vs %d (%v)", cidr, len(groupList), groupList)
			if cidr != "" && len(groupList) == 0 {
				log.Printf("[DEBUG] Use the CIDR %v", cidr)
				rule.Cidr = cidr
			} else if cidr == "" && len(groupList) > 0 {
				log.Printf("[DEBUG] Use the groupList %v", groupList)
				rule.UserSecurityGroupList = groupList
			} else {
				return errors.New("Either CIDR or User Security Group List are required")
			}
			// Sockets
			rule.Protocol = d.Get(key + "protocol").(string)
			if port, ok := d.GetOkExists(key + "port"); ok && port.(int) > 0 {
				rule.StartPort = port.(int)
				rule.EndPort = port.(int)
			} else {
				sP, startOk := d.GetOkExists(key + "start_port")
				startPort := sP.(int)
				eP, endOk := d.GetOkExists(key + "end_port")
				endPort := eP.(int)

				if startOk && endOk && startPort > 0 && endPort >= startPort {
					rule.StartPort = startPort
					rule.EndPort = endPort
				} else {
					return errors.New("Either Port or Start/End ports are required")
				}
			}
			// ICMP
			rule.IcmpType = d.Get(key + "icmptype").(int)
			rule.IcmpCode = d.Get(key + "icmpcode").(int)

			r.rules[i] = rule
		}
	}

	resp, err := client.CreateSecurityGroupWithRules(d.Get("name").(string),
		ingressRules, egressRules)
	if err != nil {
		return err
	}

	d.SetId(resp.Id)

	/* Update the sgid field for all of the ingress/egress rules */
	for i = 0; i < ingressLength; i++ {
		key := fmt.Sprintf("ingress_rules.%d.", i)
		d.Set(key+"sgid", resp.Id)
	}

	for i = 0; i < egressLength; i++ {
		key := fmt.Sprintf("egress_rules.%d.", i)
		d.Set(key+"sgid", resp.Id)
	}

	return sgRead(d, meta)
}

func sgRead(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	sgs, err := client.GetSecurityGroupsByName()
	if err != nil {
		return err
	}

	var securityGroups egoscale.SecurityGroup
	for _, v := range sgs {
		if d.Id() == v.Id {
			securityGroups = v
			break
		}
	}

	d.SetId(securityGroups.Id)
	d.Set("name", securityGroups.Name)
	d.Set("description", securityGroups.Description)
	d.Set("ingress_rules.#", len(securityGroups.IngressRules))
	d.Set("egress_rules.#", len(securityGroups.EgressRules))

	for i := 0; i < len(securityGroups.IngressRules); i++ {
		key := fmt.Sprintf("ingress_rules.%d.", i)
		d.Set(key+"sgid", securityGroups.IngressRules[i].RuleId)
		d.Set(key+"port", securityGroups.IngressRules[i].StartPort)
		d.Set(key+"cidr", securityGroups.IngressRules[i].Cidr)
		d.Set(key+"protocol", securityGroups.IngressRules[i].Protocol)
		d.Set(key+"icmpcode", securityGroups.IngressRules[i].IcmpCode)
		d.Set(key+"icmptype", securityGroups.IngressRules[i].IcmpType)
	}

	for i := 0; i < len(securityGroups.EgressRules); i++ {
		key := fmt.Sprintf("egress_rules.%d.", i)
		d.Set(key+"sgid", securityGroups.EgressRules[i].RuleId)
		d.Set(key+"port", securityGroups.EgressRules[i].StartPort)
		d.Set(key+"cidr", securityGroups.EgressRules[i].Cidr)
		d.Set(key+"protocol", securityGroups.EgressRules[i].Protocol)
		d.Set(key+"icmpcode", securityGroups.EgressRules[i].IcmpCode)
		d.Set(key+"icmptype", securityGroups.EgressRules[i].IcmpType)
	}

	return nil
}

func sgDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	err := client.DeleteSecurityGroup(d.Get("name").(string))
	if err != nil {
		return err
	}

	return nil
}
