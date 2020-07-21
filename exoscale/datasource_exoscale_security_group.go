package exoscale

import (
	"context"
	"errors"
	"fmt"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:          schema.TypeString,
				Description:   "ID of the Security Group",
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			"name": {
				Type:          schema.TypeString,
				Description:   "Name of the Security Group",
				Optional:      true,
				ConflictsWith: []string{"id"},
			},
		},

		Read: dataSourceSecurityGroupRead,
	}
}

func dataSourceSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	req := egoscale.ListSecurityGroups{}

	sgName, byName := d.GetOk("name")
	sgID, byID := d.GetOk("id")
	if !byName && !byID {
		return errors.New("either name or id must be specified")
	}

	req.SecurityGroupName = sgName.(string)

	if sgID != "" {
		if req.ID, err = egoscale.ParseUUID(sgID.(string)); err != nil {
			return fmt.Errorf("invalid value for id: %s", err)
		}
	}

	resp, err := client.GetWithContext(ctx, &req)
	if err != nil {
		return err
	}
	sg := resp.(*egoscale.SecurityGroup)

	d.SetId(sg.ID.String())

	if err := d.Set("id", d.Id()); err != nil {
		return err
	}
	if err := d.Set("name", sg.Name); err != nil {
		return err
	}

	return nil
}
