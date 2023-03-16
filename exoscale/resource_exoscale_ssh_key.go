package exoscale

import (
	"context"
	"errors"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	resSSHKeyAttrFingerprint = "fingerprint"
	resSSHKeyAttrName        = "name"
	resSSHKeyAttrPublicKey   = "public_key"
)

func resourceSSHKeyIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_ssh_key")
}

func resourceSSHKey() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			resSSHKeyAttrFingerprint: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The SSH key unique identifier.",
			},
			resSSHKeyAttrName: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The SSH key name.",
			},
			resSSHKeyAttrPublicKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The SSH *public* key that will be authorized in compute instances.",
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
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceSSHKeyIDString(d),
	})

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

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceSSHKeyIDString(d),
	})

	return resourceSSHKeyRead(ctx, d, meta)
}

func resourceSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceSSHKeyIDString(d),
	})

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

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceSSHKeyIDString(d),
	})

	return diag.FromErr(resourceSSHKeyApply(ctx, d, securityGroup))
}

func resourceSSHKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceSSHKeyIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	if err := client.DeleteSSHKey(ctx, zone, &egoscale.SSHKey{Name: nonEmptyStringPtr(d.Id())}); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceSSHKeyIDString(d),
	})

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
