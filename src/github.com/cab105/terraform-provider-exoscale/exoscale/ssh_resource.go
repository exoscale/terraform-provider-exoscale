package exoscale

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	//"github.com/runseb/egoscale/src/egoscale"
)

func sshResource() *schema.Resource {
	return &schema.Resource{
		Create: 	sshCreate,
		Read: 		sshRead,
		Delete: 	sshDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:		schema.TypeString,
				Required:	true,
			},
		},
	}
}

func sshCreate(d *schema.ResourceData, meta interface{}) error {
	fmt.Printf("sshCreated called")
	return nil
}

func sshRead(d *schema.ResourceData, meta interface{}) error {
	fmt.Printf("sshRead called")
	return nil
}

func sshDelete(d *schema.ResourceData, meta interface{}) error {
	fmt.Printf("sshDelete called")
	return nil
}