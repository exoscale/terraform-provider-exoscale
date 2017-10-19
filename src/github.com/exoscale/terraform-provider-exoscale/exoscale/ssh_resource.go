package exoscale

import (
	"log"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pyr/egoscale/src/egoscale"
)

func sshResource() *schema.Resource {
	return &schema.Resource{
		Create: sshCreate,
		Read:   sshRead,
		Update: nil,
		Delete: sshDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:		schema.TypeString,
				Computed:	true,
			},
			"name": &schema.Schema{
				Type:		schema.TypeString,
				Required:	true,
				ForceNew:	true,
			},
			"key": &schema.Schema{
				Type:		schema.TypeString,
				Required:	true,
				ForceNew:	true,
			},
			"fingerprint": &schema.Schema{
				Type:     	schema.TypeString,
				Computed:	true,
			},
		},
	}
}

func sshCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	name := d.Get("name").(string)
	found, err := findKey(name, client)
	if err != nil {
		return err
	}

	if found != "" {
		log.Printf("Found keypair by name: %s\n", name)
		return nil
	}

	_, err = client.RegisterKeypair(name, d.Get("key").(string))
	if err != nil {
		return err
	}

	d.SetId(name)
	log.Printf("Created keypair by name: %s\n", name)

	return sshRead(d, meta)
}

func sshRead(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	name := d.Id()
	fingerprint, err := findKey(name, client)
	if err != nil {
		return err
	}

	d.Set("fingerprint", fingerprint)

	return nil
}

func sshDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	name := d.Id()
	found, err := findKey(name, client)
	if err != nil {
		return err
	}

	log.Printf("deleting key: %s (%s)", name, found)

	if found == "" {
		return fmt.Errorf("Key %s does not exist", name)
	}

	_, err = client.DeleteKeypair(name)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func findKey(name string, client *egoscale.Client) (string, error) {
	keys, err := client.GetKeypairs()
	if err != nil {
		return "", err
	}

	for _, k := range keys {
		if k.Name == name {
			return k.Fingerprint, nil
		}
	}

	return "", nil
}
