package exoscale

import (
	"log"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func securityGroupResource() *schema.Resource {
	s := map[string]*schema.Schema{
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
	}

	addTags(s, "tags")

	return &schema.Resource{
		Create: createSecurityGroup,
		Exists: existsSecurityGroup,
		Update: updateSecurityGroup,
		Read:   readSecurityGroup,
		Delete: deleteSecurityGroup,

		Importer: &schema.ResourceImporter{
			State: importSecurityGroup,
		},

		Schema: s,
	}
}

func createSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	resp, err := client.Request(&egoscale.CreateSecurityGroup{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	sg := resp.(*egoscale.CreateSecurityGroupResponse).SecurityGroup

	d.SetId(sg.ID)
	if cmd := createTags(d, "tags", sg.ResourceType()); cmd != nil {
		if err := client.BooleanAsyncRequest(cmd, async); err != nil {
			// Attempting to destroy the freshly created security group
			e := client.BooleanRequest(&egoscale.DeleteSecurityGroup{
				Name: sg.Name,
			})

			if e != nil {
				log.Printf("[WARNING] Failure to create the tags, but the security group was created. %v", e)
			}

			return err
		}
	}

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

func updateSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	d.Partial(true)

	requests, err := updateTags(d, "tags", new(egoscale.SecurityGroup).ResourceType())
	if err != nil {
		return err
	}

	for _, req := range requests {
		_, err := client.AsyncRequest(req, async)
		if err != nil {
			return err
		}
	}

	err = readSecurityGroup(d, meta)
	if err != nil {
		return err
	}

	d.SetPartial("tags")
	d.Partial(false)

	return err
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

	// This permits to import a resource using the security group name rather than using the ID.
	id := d.Id()
	name := ""
	if !isUUID(id) {
		id = ""
		name = id
	}

	resp, err := client.Request(&egoscale.ListSecurityGroups{
		ID:                id,
		SecurityGroupName: name,
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

	// tags
	tags := make(map[string]interface{})
	for _, tag := range securityGroup.Tags {
		tags[tag.Key] = tag.Value
	}
	d.Set("tags", tags)

	return nil
}
