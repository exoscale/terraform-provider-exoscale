package exoscale

import (
	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func securityGroupResource() *schema.Resource {
	return &schema.Resource{
		Create: createSecurityGroup,
		Exists: existsSecurityGroup,
		Read:   readSecurityGroup,
		Delete: deleteSecurityGroup,

		Importer: &schema.ResourceImporter{
			State: importSecurityGroup,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}

func createSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	resp, err := client.Request(&egoscale.CreateSecurityGroup{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	sg := resp.(*egoscale.CreateSecurityGroupResponse).SecurityGroup
	return applySecurityGroup(d, sg)
}

func existsSecurityGroup(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)
	_, err := client.Request(&egoscale.ListSecurityGroups{
		ID: d.Id(),
	})

	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}
	return true, nil
}

func readSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	resp, err := client.Request(&egoscale.ListSecurityGroups{
		ID: d.Id(),
	})
	if err != nil {
		return handleNotFound(d, err)
	}

	groups := resp.(*egoscale.ListSecurityGroupsResponse)
	if groups.Count == 0 {
		d.SetId("")
		return nil
	}

	return applySecurityGroup(d, groups.SecurityGroup[0])
}

func deleteSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	err := client.BooleanRequest(&egoscale.DeleteSecurityGroup{
		Name: d.Get("name").(string),
	})

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func importSecurityGroup(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := GetComputeClient(meta)
	resp, err := client.Request(&egoscale.ListSecurityGroups{
		ID: d.Id(),
	})
	if err != nil {
		return nil, err
	}

	securityGroup := resp.(*egoscale.ListSecurityGroupsResponse).SecurityGroup[0]
	applySecurityGroup(d, securityGroup)

	// Create all the rulez!
	ruleLength := len(securityGroup.EgressRule) + len(securityGroup.IngressRule)
	resources := make([]*schema.ResourceData, 0, 1+ruleLength)
	resources = append(resources, d)

	for _, rule := range securityGroup.EgressRule {
		resource := securityGroupRuleResource()
		d := resource.Data(nil)
		d.SetType("exoscale_security_group_rule")
		d.Set("type", "EGRESS")
		err := applySecurityGroupRule(d, securityGroup, rule)
		if err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}
	for _, rule := range securityGroup.IngressRule {
		resource := securityGroupRuleResource()
		d := resource.Data(nil)
		d.SetType("exoscale_security_group_rule")
		d.Set("type", "INGRESS")
		err := applySecurityGroupRule(d, securityGroup, (egoscale.EgressRule)(rule))
		if err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}

	return resources, nil
}

func applySecurityGroup(d *schema.ResourceData, securityGroup egoscale.SecurityGroup) error {
	d.SetId(securityGroup.ID)
	d.Set("name", securityGroup.Name)
	d.Set("description", securityGroup.Description)

	return nil
}
