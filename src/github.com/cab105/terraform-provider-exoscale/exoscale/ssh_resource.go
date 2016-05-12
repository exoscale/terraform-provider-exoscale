package exoscale

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/runseb/egoscale/src/egoscale"
)

func sshResource() *schema.Resource {
	return &schema.Resource{
		Create: sshCreate,
		Read:   sshRead,
		Update: sshUpdate,
		Delete: sshDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
		fmt.Printf("Found keypair by name: %s\n", name)
		return nil
	}

	if d.Get("key").(string) != "" {
		_, err = client.RegisterKeypair(name, d.Get("key").(string))
		if err != nil {
			return err
		}

	} else {
		_, err = client.CreateKeypair(name)
		if err != nil {
			return err
		}
	}

	fingerprint, err := findKey(name, client); if err != nil {
		return err
	}

	d.Set("fingerprint", fingerprint)

	fmt.Printf("Created keypair by name: %s\n", name)

	return nil
}

func sshRead(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	name := d.Get("name").(string)
	fingerprint, err := findKey(name, client)
	if err != nil {
		return err
	}

	d.Set("fingerprint", fingerprint)

	return nil
}

func sshDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	name := d.Get("name").(string)
	found, err := findKey(name, client)
	if err != nil {
		return err
	}

	if found != "" {
		return fmt.Errorf("Key %s does not exist", name)
	}

	resp, err := client.DeleteKeypair(name)
	if err != nil {
		return err
	}

	fmt.Printf("Response: %s\n", resp)
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

func sshUpdate(d *schema.ResourceData, meta interface{}) error {
	/* Required, but is a no-op for now */
	return nil
}
