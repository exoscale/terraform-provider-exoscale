package exoscale

import (
	"context"
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

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},

		Schema: s,
	}
}

func createSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	resp, err := client.RequestWithContext(ctx, &egoscale.CreateSecurityGroup{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	sg := resp.(*egoscale.CreateSecurityGroupResponse).SecurityGroup

	d.SetId(sg.ID)
	if cmd := createTags(d, "tags", sg.ResourceType()); cmd != nil {
		if err := client.BooleanRequestWithContext(ctx, cmd); err != nil {
			// Attempting to destroy the freshly created security group
			e := client.BooleanRequestWithContext(ctx, &egoscale.DeleteSecurityGroup{
				Name: sg.Name,
			})

			if e != nil {
				log.Printf("[WARNING] Failure to create the tags, but the security group was created. %v", e)
			}

			return err
		}
	}

	return readSecurityGroup(d, meta)
}

func existsSecurityGroup(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)
	sg := &egoscale.SecurityGroup{
		ID: d.Id(),
	}
	if err := client.GetWithContext(ctx, sg); err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}
	return true, nil
}

func updateSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	d.Partial(true)

	requests, err := updateTags(d, "tags", new(egoscale.SecurityGroup).ResourceType())
	if err != nil {
		return err
	}

	for _, req := range requests {
		_, err := client.RequestWithContext(ctx, req)
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
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)
	sg := &egoscale.SecurityGroup{
		ID: d.Id(),
	}
	if err := client.GetWithContext(ctx, sg); err != nil {
		return handleNotFound(d, err)
	}

	return applySecurityGroup(d, sg)
}

func deleteSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)
	err := client.BooleanRequestWithContext(ctx, &egoscale.DeleteSecurityGroup{
		Name: d.Get("name").(string),
	})

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func importSecurityGroup(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// XXX d.Timeout will result in a null pointer exception
	// https://github.com/hashicorp/terraform/issues/17672
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	client := GetComputeClient(meta)

	// This permits to import a resource using the security group name rather than using the ID.
	id := d.Id()
	name := ""
	if !isUUID(id) {
		name = id
		id = ""
	}

	securityGroup := &egoscale.SecurityGroup{
		ID:   id,
		Name: name,
	}
	if err := client.GetWithContext(ctx, securityGroup); err != nil {
		return nil, err
	}

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

func applySecurityGroup(d *schema.ResourceData, securityGroup *egoscale.SecurityGroup) error {
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
