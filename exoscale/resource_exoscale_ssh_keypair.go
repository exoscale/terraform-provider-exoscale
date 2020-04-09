package exoscale

import (
	"context"
	"log"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceSSHKeypairIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_ssh_keypair")
}

func resourceSSHKeypair() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"public_key": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"private_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},

		Create: resourceSSHKeypairCreate,
		Read:   resourceSSHKeypairRead,
		Delete: resourceSSHKeypairDelete,
		Exists: resourceSSHKeypairExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSSHKeypairCreate(d *schema.ResourceData, meta interface{}) error {
	var keypair *egoscale.SSHKeyPair

	log.Printf("[DEBUG] %s: beginning create", resourceSSHKeypairIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	name := d.Get("name").(string)
	publicKey, publicKeyOk := d.GetOk("public_key")
	if publicKeyOk {
		resp, err := client.RequestWithContext(ctx, &egoscale.RegisterSSHKeyPair{
			Name:      name,
			PublicKey: publicKey.(string),
		})
		if err != nil {
			return err
		}
		keypair = resp.(*egoscale.SSHKeyPair)
	} else {
		resp, err := client.RequestWithContext(ctx, &egoscale.CreateSSHKeyPair{Name: name})
		if err != nil {
			return err
		}
		keypair = resp.(*egoscale.SSHKeyPair)

		// We have to set this attribute now instead of later in resourceSSHKeypairApply, because once we go
		// through resourceSSHKeypairRead we'll have lost the information.
		if err := d.Set("private_key", keypair.PrivateKey); err != nil {
			return err
		}
	}

	d.SetId(keypair.Name)

	log.Printf("[DEBUG] %s: create finished successfully", resourceSSHKeypairIDString(d))

	return resourceSSHKeypairRead(d, meta)
}

func resourceSSHKeypairExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	key := &egoscale.SSHKeyPair{
		Name: d.Id(),
	}

	_, err := client.GetWithContext(ctx, key)
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return true, nil
}

func resourceSSHKeypairRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceSSHKeypairIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	key := &egoscale.SSHKeyPair{Name: d.Id()}

	resp, err := client.GetWithContext(ctx, key)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSSHKeypairIDString(d))

	return resourceSSHKeypairApply(d, resp.(*egoscale.SSHKeyPair))
}

func resourceSSHKeypairDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceSSHKeypairIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	key := &egoscale.SSHKeyPair{Name: d.Id()}

	if err := client.DeleteWithContext(ctx, key); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSSHKeypairIDString(d))

	return nil
}

func resourceSSHKeypairApply(d *schema.ResourceData, keypair *egoscale.SSHKeyPair) error {
	if err := d.Set("name", keypair.Name); err != nil {
		return err
	}

	if err := d.Set("fingerprint", keypair.Fingerprint); err != nil {
		return err
	}

	return nil
}
