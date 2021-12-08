package exoscale

import (
	"context"
	"errors"
	"log"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	resSSHKeyAttrFingerprint = "fingerprint"
	resSSHKeyAttrName        = "name"
	resSSHKeyAttrPublicKey   = "public_key"
)

func resourceSSHKeyIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_ssh_key")
}

func resourceSSHKey() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			resSSHKeyAttrFingerprint: {
				Type:     schema.TypeString,
				Computed: true,
			},
			resSSHKeyAttrName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			resSSHKeyAttrPublicKey: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},

		CreateContext: resourceSSHKeyCreate,
		ReadContext:   resourceSSHKeyRead,
		DeleteContext: resourceSSHKeyDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSSHKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceSSHKeyIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	publicKey, ok := d.GetOk(resSSHKeyAttrPublicKey)
	if !ok {
		return diag.Errorf("a value must be provided for attribute %q", resSSHKeyAttrPublicKey)
	}

	sshKey, err := client.RegisterSSHKey(
		ctx,
		zone,
		d.Get(resSSHKeyAttrName).(string),
		publicKey.(string),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*sshKey.Name)

	log.Printf("[DEBUG] %s: create finished successfully", resourceSSHKeyIDString(d))

	return resourceSSHKeyRead(ctx, d, meta)
}

func resourceSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceSSHKeyIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup, err := client.GetSSHKey(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSSHKeyIDString(d))

	return diag.FromErr(resourceSSHKeyApply(ctx, d, securityGroup))
}

func resourceSSHKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceSSHKeyIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	if err := client.DeleteSSHKey(ctx, zone, &egoscale.SSHKey{Name: nonEmptyStringPtr(d.Id())}); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSSHKeyIDString(d))

	return nil
}

func resourceSSHKeyApply(_ context.Context, d *schema.ResourceData, sshKey *egoscale.SSHKey) error {
	if err := d.Set(resSSHKeyAttrName, *sshKey.Name); err != nil {
		return err
	}

	if err := d.Set(resSSHKeyAttrFingerprint, *sshKey.Fingerprint); err != nil {
		return err
	}

	return nil
}
