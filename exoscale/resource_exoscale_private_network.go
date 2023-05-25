package exoscale

import (
	"context"
	"errors"
	"net"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
)

const (
	resPrivateNetworkAttrDescription = "description"
	resPrivateNetworkAttrEndIP       = "end_ip"
	resPrivateNetworkAttrName        = "name"
	resPrivateNetworkAttrNetmask     = "netmask"
	resPrivateNetworkAttrStartIP     = "start_ip"
	resPrivateNetworkAttrZone        = "zone"

	resPrivateNetworkDocHint = "(For managed Privnets) "
)

func resourcePrivateNetworkIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_private_network")
}

func resourcePrivateNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Manage Exoscale Private Networks.",
		Schema: map[string]*schema.Schema{
			resPrivateNetworkAttrDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A free-form text describing the network.",
			},
			resPrivateNetworkAttrEndIP: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsIPAddress,
				Description:  "(For managed Privnets) The first/last IPv4 addresses used by the DHCP service for dynamic leases.",
			},
			resPrivateNetworkAttrName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The private network name.",
			},
			resPrivateNetworkAttrNetmask: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsIPAddress,
				Description:  "(For managed Privnets) The network mask defining the IPv4 network allowed for static leases.",
			},
			resPrivateNetworkAttrStartIP: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsIPAddress,
				Description:  resPrivateNetworkDocHint + "The first/last IPv4 addresses used by the DHCP service for dynamic leases.",
			},
			resPrivateNetworkAttrZone: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
			},
		},

		CreateContext: resourcePrivateNetworkCreate,
		ReadContext:   resourcePrivateNetworkRead,
		UpdateContext: resourcePrivateNetworkUpdate,
		DeleteContext: resourcePrivateNetworkDelete,

		Importer: &schema.ResourceImporter{
			StateContext: zonedStateContextFunc,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(DefaultTimeout),
			Read:   schema.DefaultTimeout(DefaultTimeout),
			Update: schema.DefaultTimeout(DefaultTimeout),
			Delete: schema.DefaultTimeout(DefaultTimeout),
		},
	}
}

func resourcePrivateNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

	zone := d.Get(resPrivateNetworkAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	privateNetwork := &egoscale.PrivateNetwork{
		Name: nonEmptyStringPtr(d.Get(resPrivateNetworkAttrName).(string)),
	}

	if v, ok := d.GetOk(resPrivateNetworkAttrDescription); ok {
		privateNetwork.Description = nonEmptyStringPtr(v.(string))
	}

	if v, ok := d.GetOk(resPrivateNetworkAttrEndIP); ok {
		ip := net.ParseIP(v.(string))
		privateNetwork.EndIP = &ip
	}

	if v, ok := d.GetOk(resPrivateNetworkAttrNetmask); ok {
		ip := net.ParseIP(v.(string))
		privateNetwork.Netmask = &ip
	}

	if v, ok := d.GetOk(resPrivateNetworkAttrStartIP); ok {
		ip := net.ParseIP(v.(string))
		privateNetwork.StartIP = &ip
	}

	privateNetwork, err := client.CreatePrivateNetwork(ctx, zone, privateNetwork)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*privateNetwork.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

	return resourcePrivateNetworkRead(ctx, d, meta)
}

func resourcePrivateNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

	zone := d.Get(resPrivateNetworkAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	privateNetwork, err := client.GetPrivateNetwork(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

	return diag.FromErr(resourcePrivateNetworkApply(ctx, d, privateNetwork))
}

func resourcePrivateNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

	zone := d.Get(resPrivateNetworkAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	privateNetwork, err := client.GetPrivateNetwork(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var updated bool

	if d.HasChange(resPrivateNetworkAttrDescription) {
		v := d.Get(resPrivateNetworkAttrDescription).(string)
		privateNetwork.Description = &v
		updated = true
	}

	if d.HasChange(resPrivateNetworkAttrEndIP) {
		ip := net.ParseIP(d.Get(resPrivateNetworkAttrEndIP).(string))
		privateNetwork.EndIP = &ip
		updated = true
	}

	if d.HasChange(resPrivateNetworkAttrNetmask) {
		ip := net.ParseIP(d.Get(resPrivateNetworkAttrNetmask).(string))
		privateNetwork.Netmask = &ip
		updated = true
	}

	if d.HasChange(resPrivateNetworkAttrName) {
		v := d.Get(resPrivateNetworkAttrName).(string)
		privateNetwork.Name = &v
		updated = true
	}

	if d.HasChange(resPrivateNetworkAttrStartIP) {
		ip := net.ParseIP(d.Get(resPrivateNetworkAttrStartIP).(string))
		privateNetwork.StartIP = &ip
		updated = true
	}

	if updated {
		if err = client.UpdatePrivateNetwork(ctx, zone, privateNetwork); err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

	return resourcePrivateNetworkRead(ctx, d, meta)
}

func resourcePrivateNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

	zone := d.Get(resPrivateNetworkAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	privateNetworkID := d.Id()
	if err := client.DeletePrivateNetwork(ctx, zone, &egoscale.PrivateNetwork{ID: &privateNetworkID}); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

	return nil
}

func resourcePrivateNetworkApply(
	_ context.Context,
	d *schema.ResourceData,
	privateNetwork *egoscale.PrivateNetwork,
) error {
	if err := d.Set(resPrivateNetworkAttrDescription, defaultString(privateNetwork.Description, "")); err != nil {
		return err
	}

	if privateNetwork.EndIP != nil {
		if err := d.Set(resPrivateNetworkAttrEndIP, privateNetwork.EndIP.String()); err != nil {
			return err
		}
	}

	if privateNetwork.Netmask != nil {
		if err := d.Set(resPrivateNetworkAttrNetmask, privateNetwork.Netmask.String()); err != nil {
			return err
		}
	}

	if err := d.Set(resPrivateNetworkAttrName, *privateNetwork.Name); err != nil {
		return err
	}

	if privateNetwork.StartIP != nil {
		if err := d.Set(resPrivateNetworkAttrStartIP, privateNetwork.StartIP.String()); err != nil {
			return err
		}
	}

	return nil
}
