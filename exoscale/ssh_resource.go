package exoscale

import (
	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func sshResource() *schema.Resource {
	return &schema.Resource{
		Create: createSSH,
		Exists: existsSSH,
		Read:   readSSH,
		Delete: deleteSSH,

		Importer: &schema.ResourceImporter{
			State: importSSH,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"public_key": {
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				Sensitive: true,
			},
			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createSSH(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	// XXX Could use CreateSSHKeyPairRequest if public_key wasn't set.
	req := &egoscale.RegisterSSHKeyPair{
		Name:      d.Get("name").(string),
		PublicKey: d.Get("public_key").(string),
	}
	resp, err := client.Request(req)
	if err != nil {
		return err
	}

	keypair := resp.(*egoscale.RegisterSSHKeyPairResponse).KeyPair
	return applySSH(d, keypair)
}

func existsSSH(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)

	resp, err := client.Request(&egoscale.ListSSHKeyPairs{
		Name: d.Id(),
	})
	if err != nil {
		return false, err
	}

	keys := resp.(*egoscale.ListSSHKeyPairsResponse)
	return keys.Count == 1, nil
}

func importSSH(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := readSSH(d, meta); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 1)
	resources[0] = d
	return resources, nil
}

func readSSH(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	resp, err := client.Request(&egoscale.ListSSHKeyPairs{
		Name: d.Id(),
	})
	if err != nil {
		return err
	}

	keys := resp.(*egoscale.ListSSHKeyPairsResponse)
	return applySSH(d, keys.SSHKeyPair[0])
}

func deleteSSH(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	req := &egoscale.DeleteSSHKeyPair{
		Name: d.Id(),
	}
	err := client.BooleanRequest(req)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func applySSH(d *schema.ResourceData, keypair egoscale.SSHKeyPair) error {
	d.SetId(keypair.Name)
	d.Set("name", keypair.Name)
	d.Set("fingerprint", keypair.Fingerprint)

	return nil
}
