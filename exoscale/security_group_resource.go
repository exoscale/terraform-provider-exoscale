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

	return applySecurityGroup(resp.(*egoscale.CreateSecurityGroupResponse).SecurityGroup, d)
}

func existsSecurityGroup(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)
	_, err := client.Request(&egoscale.ListSecurityGroups{
		ID: d.Id(),
	})
	// The CS API returns an error if it doesn't exist
	if err != nil {
		if r, ok := err.(*egoscale.ErrorResponse); ok {
			if r.ErrorCode == 431 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

func readSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	resp, err := client.Request(&egoscale.ListSecurityGroups{
		ID: d.Id(),
	})
	if err != nil {
		// Check for already delete security group
		if r, ok := err.(*egoscale.ErrorResponse); ok {
			if r.ErrorCode == 431 {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	groups := resp.(*egoscale.ListSecurityGroupsResponse)
	if groups.Count == 0 {
		d.SetId("")
		return nil
	}

	return applySecurityGroup(groups.SecurityGroup[0], d)
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
	applySecurityGroup(securityGroup, d)

	// Create all the rulez!
	ruleLength := len(securityGroup.EgressRule) + len(securityGroup.IngressRule)
	resources := make([]*schema.ResourceData, 0, 1+ruleLength)
	resources = append(resources, d)

	for _, rule := range securityGroup.EgressRule {
		resource := securityGroupRuleResource()
		d := resource.Data(nil)
		d.SetType("exoscale_security_group_rule")
		d.Set("type", "EGRESS")
		err := applySecurityGroupRule(securityGroup, rule, d)
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
		err := applySecurityGroupRule(securityGroup, (*egoscale.EgressRule)(rule), d)
		if err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}

	return resources, nil
}

func applySecurityGroup(securityGroup *egoscale.SecurityGroup, d *schema.ResourceData) error {
	d.SetId(securityGroup.ID)
	d.Set("name", securityGroup.Name)
	d.Set("description", securityGroup.Description)

	return nil
}
