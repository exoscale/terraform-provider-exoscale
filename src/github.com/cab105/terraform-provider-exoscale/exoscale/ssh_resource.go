package exoscale

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/runseb/egoscale/src/egoscale"
)

func sshResource() *schema.Resource {
	return &schema.Resource{
		Create: 	sshCreate,
		Read: 		sshRead,
		Update:		sshUpdate,
		Delete: 	sshDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:		schema.TypeString,
				Required:	true,
			},
			"found": &schema.Schema{
				Type:		schema.TypeBool,
				Computed:	true,
			},
		},
	}
}

func sshCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	name := d.Get("name").(string)
	d.Set("found", false)
	found, err := findKey(name, client); if err != nil {
		return err
	}

	if found {
		fmt.Printf("Found keypair by name: %s\n", name)
		d.Set("found", true)
		return nil
	}

	_, err = client.CreateKeypair(name); if err != nil {
		return err
	}

	fmt.Printf("Created keypair by name: %s\n", name)
	d.Set("found", true)

	return nil
}

func sshRead(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	name := d.Get("name").(string)
	found, err := findKey(name, client); if err != nil {
		return err
	}

	d.Set("found", found)

	return nil
}

func sshDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	name := d.Get("name").(string)
	found, err := findKey(name, client); if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("Key %s does not exist", name)
	}

	resp, err := client.DeleteKeypair(name); if err != nil {
		return err
	}

	fmt.Printf("Response: %s\n", resp)
	return nil
}

func findKey(name string, client *egoscale.Client) (bool, error) {
	keys, err := client.GetKeypairs(); if err != nil {
		return false, err
	}

	for _, k := range keys {
		if k == name {
			return true, nil
		}
	}

	return false, nil
}

func sshUpdate(d *schema.ResourceData, meta interface{}) error {
	/* Required, but is a no-op for now */
	return nil
}