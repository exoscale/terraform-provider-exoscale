package exoscale

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/runseb/egoscale/src/egoscale"
)

func securityGroupResource() *schema.Resource {
	return &schema.Resource{
		Create: sgCreate,
		Read:   sgRead,
		Update: sgUpdate,
		Delete: sgDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:		schema.TypeString,
				Computed:	true,
			},
		},
	}
}

func sgCreate(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("Unimplemented")
}

func sgRead(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("Unimplemented")
}

func sgUpdate(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("Unimplemented")
}

func sgDelete(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("Unimplemented")
}