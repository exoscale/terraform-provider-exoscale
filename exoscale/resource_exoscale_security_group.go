package exoscale

import (
	"context"
	"log"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceSecurityGroupIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_security_group")
}

func resourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
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
			"tags": {
				Type:     schema.TypeMap,
				ForceNew: true,
				Optional: true,
				Removed:  "Tags cannot be set on security groups for the time being",
			},
		},

		Create: resourceSecurityGroupCreate,
		Read:   resourceSecurityGroupRead,
		Delete: resourceSecurityGroupDelete,
		Exists: resourceSecurityGroupExists,

		Importer: &schema.ResourceImporter{
			State: resourceSecurityGroupImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceSecurityGroupIDString(d))

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

	sg := resp.(*egoscale.SecurityGroup)

	d.SetId(sg.ID.String())

	log.Printf("[DEBUG] %s: create finished successfully", resourceSecurityGroupIDString(d))

	return resourceSecurityGroupRead(d, meta)
}

func resourceSecurityGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return false, err
	}

	sg := &egoscale.SecurityGroup{
		ID: id,
	}

	_, err = client.GetWithContext(ctx, sg)
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return true, nil
}

func resourceSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceSecurityGroupIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.GetWithContext(ctx, &egoscale.SecurityGroup{
		ID: id,
	})
	if err != nil {
		return handleNotFound(d, err)
	}

	sg := resp.(*egoscale.SecurityGroup)

	log.Printf("[DEBUG] %s: read finished successfully", resourceSecurityGroupIDString(d))

	return resourceSecurityGroupApply(d, sg)
}

func resourceSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceSecurityGroupIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	sg := &egoscale.DeleteSecurityGroup{ID: id}

	if err := client.BooleanRequestWithContext(ctx, sg); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSecurityGroupIDString(d))

	return nil
}

func resourceSecurityGroupImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup := &egoscale.SecurityGroup{}

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		securityGroup.Name = d.Id()
	} else {
		securityGroup.ID = id
	}

	resp, err := client.GetWithContext(ctx, securityGroup)
	if err != nil {
		return nil, err
	}

	sg := resp.(*egoscale.SecurityGroup)
	if err := resourceSecurityGroupApply(d, sg); err != nil {
		return nil, err
	}

	ruleLength := len(sg.EgressRule) + len(sg.IngressRule)
	resources := make([]*schema.ResourceData, 0, 1+ruleLength)
	resources = append(resources, d)

	for _, rule := range sg.EgressRule {
		resource := resourceSecurityGroupRule()
		d := resource.Data(nil)
		d.SetType("exoscale_security_group_rule")
		d.Set("type", "EGRESS") // nolint: errcheck
		err := resourceSecurityGroupRuleApply(d, sg, rule)
		if err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}
	for _, rule := range sg.IngressRule {
		resource := resourceSecurityGroupRule()
		d := resource.Data(nil)
		d.SetType("exoscale_security_group_rule")
		d.Set("type", "INGRESS") // nolint: errcheck
		err := resourceSecurityGroupRuleApply(d, sg, (egoscale.EgressRule)(rule))
		if err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}

	return resources, nil
}

func resourceSecurityGroupApply(d *schema.ResourceData, securityGroup *egoscale.SecurityGroup) error {
	d.SetId(securityGroup.ID.String())
	if err := d.Set("name", securityGroup.Name); err != nil {
		return err
	}
	if err := d.Set("description", securityGroup.Description); err != nil {
		return err
	}
	return nil
}
