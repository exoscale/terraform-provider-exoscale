package instance

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func Resource() *schema.Resource {
	s := map[string]*schema.Schema{
		AttrAntiAffinityGroupIDs: {
			Description: "A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs) to attach to the instance (may only be set at creation time).",
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			ForceNew:    true,
		},
		AttrCreatedAt: {
			Description: "The instance creation date.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrDestroyProtected: {
			Description: "Mark the instance as protected, the Exoscale API will refuse to delete the instance until the protection is removed (boolean; default: `false`).",
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
		},
		AttrDeployTargetID: {
			Description: "A deploy target ID.",
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
		},
		AttrDiskSize: {
			Description:  "The instance disk size (GiB; at least `10`). Can not be decreased after creation. **WARNING**: updating this attribute stops/restarts the instance.",
			Type:         schema.TypeInt,
			Computed:     true,
			Optional:     true,
			ValidateFunc: validation.IntAtLeast(10),
		},
		AttrElasticIPIDs: {
			Description: "A list of [exoscale_elastic_ip](./elastic_ip.md) (IDs) to attach to the instance.",
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrIPv6: {
			Description: "Enable IPv6 on the instance (boolean; default: `false`).",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
		},
		AttrIPv6Address: {
			Description: "The instance (main network interface) IPv6 address (if enabled).",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrLabels: {
			Description: "A map of key/value labels.",
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
		},
		AttrName: {
			Description: "The compute instance name.",
			Type:        schema.TypeString,
			Required:    true,
		},
		AttrPrivateNetworkIDs: {
			Description: "A list of private networks (IDs) attached to the instance. Please use the `network_interface.*.network_id` argument instead.",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Deprecated:  "Use the network_interface block instead.",
		},
		AttrNetworkInterface: {
			Description: "Private network interfaces (may be specified multiple times). Structure is documented below.",
			Type:        schema.TypeSet,
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"ip_address": {
						Description:      "The IPv4 address to request as static DHCP lease if the network interface is attached to a *managed* private network.",
						Type:             schema.TypeString,
						Optional:         true,
						Computed:         true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.IsIPv4Address),
					},
					"network_id": {
						Description: "The [exoscale_private_network](./private_network.md) (ID) to attach to the instance.",
						Type:        schema.TypeString,
						Required:    true,
					},
				},
			},
		},
		AttrPublicIPAddress: {
			Description: "The instance (main network interface) IPv4 address.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrPrivate: {
			Description: "Whether the instance is private (no public IP addresses; default: false)",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
		},
		AttrReverseDNS: {
			Description: "Domain name for reverse DNS record.",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrSSHKey: {
			Description: "The [exoscale_ssh_key](./ssh_key.md) (name) to authorize in the instance (may only be set at creation time).",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrSecurityGroupIDs: {
			Description: "A list of [exoscale_security_group](./security_group.md) (IDs) to attach to the instance.",
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrState: {
			Description: "The instance state (`running` or `stopped`; default: `running`).",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},
		AttrTemplateID: {
			Description: "The [exoscale_template](../data-sources/template.md) (ID) to use when creating the instance.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
		AttrType: {
			Description:      "The instance type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI](https://github.com/exoscale/cli/) - `exo compute instance-type list` - for the list of available types). **WARNING**: updating this attribute stops/restarts the instance.",
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: utils.ValidateComputeInstanceType,
			// Ignore case differences
			DiffSuppressFunc: utils.SuppressCaseDiff,
		},
		AttrUserData: {
			Description:      "[cloud-init](https://cloudinit.readthedocs.io/) configuration.",
			Type:             schema.TypeString,
			ValidateDiagFunc: utils.ValidateComputeUserData,
			Optional:         true,
		},
		AttrZone: {
			Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
		},
	}

	return &schema.Resource{
		Schema: s,

		Description: "Manage Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/).\n" +
			"\n" +
			"Corresponding data sources: [exoscale_compute_instance](../data-sources/compute_instance.md), [exoscale_compute_instance_list](../data-sources/compute_instance_list.md).\n" +
			"\n" +
			"After the creation, you can retrieve the password of an instance with [Exoscale CLI](https://github.com/exoscale/cli): `exo compute instance reveal-password NAME`.",

		CreateContext: rCreate,
		ReadContext:   rRead,
		UpdateContext: rUpdate,
		DeleteContext: rDelete,

		Importer: &schema.ResourceImporter{
			StateContext: utils.ZonedStateContextFunc,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(config.DefaultTimeout),
			Read:   schema.DefaultTimeout(config.DefaultTimeout),
			Update: schema.DefaultTimeout(config.DefaultTimeout),
			Delete: schema.DefaultTimeout(config.DefaultTimeout),
		},
	}
}

func rCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics { //nolint:gocyclo
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	zone := d.Get(AttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(config.GetEnvironment(meta), zone))
	defer cancel()

	client, err := config.GetClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	instance := &egoscale.Instance{
		Name:       utils.NonEmptyStringPtr(d.Get(AttrName).(string)),
		TemplateID: utils.NonEmptyStringPtr(d.Get(AttrTemplateID).(string)),
	}

	if set, ok := d.Get(AttrAntiAffinityGroupIDs).(*schema.Set); ok {
		instance.AntiAffinityGroupIDs = func() (v *[]string) {
			if l := set.Len(); l > 0 {
				list := make([]string, l)
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				v = &list
			}
			return
		}()
	}

	if v, ok := d.GetOk(AttrDeployTargetID); ok {
		s := v.(string)
		instance.DeployTargetID = &s
	}

	if v, ok := d.GetOk(AttrDiskSize); ok {
		i := int64(v.(int))
		instance.DiskSize = &i
	}

	if privateInstance, ok := d.GetOk(AttrPrivate); ok {
		privateInstanceBool := privateInstance.(bool)
		if privateInstanceBool {
			t := "none"
			instance.PublicIPAssignment = &t
		}
	}

	enableIPv6 := d.Get(AttrIPv6).(bool)
	instance.IPv6Enabled = &enableIPv6

	if l, ok := d.GetOk(AttrLabels); ok {
		labels := make(map[string]string)
		for k, v := range l.(map[string]interface{}) {
			labels[k] = v.(string)
		}
		instance.Labels = &labels
	}

	if v, ok := d.GetOk(AttrSSHKey); ok {
		s := v.(string)
		instance.SSHKey = &s
	}

	if set, ok := d.Get(AttrSecurityGroupIDs).(*schema.Set); ok {
		instance.SecurityGroupIDs = func() (v *[]string) {
			if l := set.Len(); l > 0 {
				list := make([]string, l)
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				v = &list
			}
			return
		}()
	}

	instanceType, err := client.FindInstanceType(ctx, zone, d.Get(AttrType).(string))
	if err != nil {
		return diag.Errorf("unable to retrieve instance type: %s", err)
	}
	instance.InstanceTypeID = instanceType.ID

	if v := d.Get(AttrUserData).(string); v != "" {
		userData, _, err := utils.EncodeUserData(v)
		if err != nil {
			return diag.FromErr(err)
		}
		instance.UserData = &userData
	}

	// FIXME: we have to reference the embedded egoscale/v2.Client explicitly
	//  here because there is already a CreateComputeInstance() method on the root
	//  egoscale client clashing with the v2 one. This can be changed once we
	//  use API V2-only calls.
	instance, err = client.CreateInstance(ctx, zone, instance)
	if err != nil {
		return diag.FromErr(err)
	}

	if isDestroyProtected, ok := d.GetOk(AttrDestroyProtected); ok && isDestroyProtected.(bool) {
		_, err := client.AddInstanceProtectionWithResponse(ctx, *instance.ID)
		if err != nil {
			return diag.Errorf("unable to make instance %s destroy protected: %s", *instance.ID, err)
		}
	}

	if set, ok := d.Get(AttrElasticIPIDs).(*schema.Set); ok {
		if set.Len() > 0 {
			for _, id := range set.List() {
				if err := client.AttachInstanceToElasticIP(
					ctx,
					zone,
					instance,
					&egoscale.ElasticIP{ID: utils.NonEmptyStringPtr(id.(string))},
				); err != nil {
					return diag.Errorf("unable to attach Elastic IP %s: %s", id.(string), err)
				}
			}
		}
	}

	if nifSet, ok := d.Get(AttrNetworkInterface).(*schema.Set); ok {
		for _, nif := range nifSet.List() {
			nif, err := NewNetworkInterface(nif)
			if err != nil {
				return diag.FromErr(err)
			}

			opts := []egoscale.AttachInstanceToPrivateNetworkOpt{}
			if nif.IPAddress != nil && *nif.IPAddress != "" {
				opts = append(opts, egoscale.AttachInstanceToPrivateNetworkWithIPAddress(net.ParseIP(*nif.IPAddress)))
			}

			if err := client.AttachInstanceToPrivateNetwork(
				ctx,
				zone,
				instance,
				&egoscale.PrivateNetwork{ID: &nif.NetworkID},
				opts...,
			); err != nil {
				return diag.Errorf("unable to attach Private Network %s: %s", nif.NetworkID, err)
			}
		}
	}

	if v, ok := d.GetOk(AttrReverseDNS); ok {
		rdns := v.(string)
		err := client.UpdateInstanceReverseDNS(
			ctx,
			zone,
			*instance.ID,
			rdns,
		)
		if err != nil {
			return diag.Errorf("unable to create Reverse DNS record: %s", err)
		}
	}

	if v := d.Get(AttrState).(string); v == "stopped" {
		if err := client.StopInstance(ctx, zone, instance); err != nil {
			return diag.Errorf("unable to stop instance: %s", err)
		}
	}

	d.SetId(*instance.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return rRead(ctx, d, meta)
}

func rRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	zone := d.Get(AttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(config.GetEnvironment(meta), zone))
	defer cancel()

	client, err := config.GetClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	instance, err := client.GetInstance(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return rApply(ctx, client, d, instance)
}

func rUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics { //nolint:gocyclo
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	zone := d.Get(AttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(config.GetEnvironment(meta), zone))
	defer cancel()

	client, err := config.GetClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	instance, err := client.GetInstance(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var updated bool

	if d.HasChange(AttrLabels) {
		labels := make(map[string]string)
		for k, v := range d.Get(AttrLabels).(map[string]interface{}) {
			labels[k] = v.(string)
		}
		instance.Labels = &labels
		updated = true
	}

	if d.HasChange(AttrName) {
		v := d.Get(AttrName).(string)
		instance.Name = &v
		updated = true
	}

	if d.HasChange(AttrUserData) {
		v, _, err := utils.EncodeUserData(d.Get(AttrUserData).(string))
		if err != nil {
			return diag.FromErr(err)
		}
		instance.UserData = &v
		updated = true
	}

	if updated {
		if err = client.UpdateInstance(ctx, zone, instance); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(AttrReverseDNS) {
		rdns := d.Get(AttrReverseDNS).(string)
		if rdns == "" {
			err = client.DeleteInstanceReverseDNS(
				ctx,
				zone,
				*instance.ID,
			)
		} else {
			err = client.UpdateInstanceReverseDNS(
				ctx,
				zone,
				*instance.ID,
				rdns,
			)
		}
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(AttrElasticIPIDs) {
		o, n := d.GetChange(AttrElasticIPIDs)
		old := o.(*schema.Set)
		cur := n.(*schema.Set)

		if added := cur.Difference(old); added.Len() > 0 {
			for _, id := range added.List() {
				if err := client.AttachInstanceToElasticIP(
					ctx,
					zone,
					instance,
					&egoscale.ElasticIP{ID: utils.NonEmptyStringPtr(id.(string))},
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}

		if removed := old.Difference(cur); removed.Len() > 0 {
			for _, id := range removed.List() {
				if err := client.DetachInstanceFromElasticIP(
					ctx,
					zone,
					instance,
					&egoscale.ElasticIP{ID: utils.NonEmptyStringPtr(id.(string))},
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	if d.HasChange(AttrNetworkInterface) {
		o, n := d.GetChange(AttrNetworkInterface)
		old := o.(*schema.Set)
		cur := n.(*schema.Set)

		if removed := old.Difference(cur); removed.Len() > 0 {
			for _, nif := range removed.List() {
				nif, err := NewNetworkInterface(nif)
				if err != nil {
					return diag.FromErr(err)
				}

				if err := client.DetachInstanceFromPrivateNetwork(
					ctx,
					zone,
					instance,
					&egoscale.PrivateNetwork{ID: &nif.NetworkID},
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}

		if added := cur.Difference(old); added.Len() > 0 {
			for _, nif := range added.List() {
				nif, err := NewNetworkInterface(nif)
				if err != nil {
					return diag.FromErr(err)
				}

				opts := []egoscale.AttachInstanceToPrivateNetworkOpt{}
				if nif.IPAddress != nil && *nif.IPAddress != "" {
					opts = append(opts, egoscale.AttachInstanceToPrivateNetworkWithIPAddress(net.ParseIP(*nif.IPAddress)))
				}

				if err := client.AttachInstanceToPrivateNetwork(
					ctx,
					zone,
					instance,
					&egoscale.PrivateNetwork{ID: &nif.NetworkID},
					opts...,
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	if d.HasChange(AttrSecurityGroupIDs) {
		o, n := d.GetChange(AttrSecurityGroupIDs)
		old := o.(*schema.Set)
		cur := n.(*schema.Set)

		if added := cur.Difference(old); added.Len() > 0 {
			for _, id := range added.List() {
				if err := client.AttachInstanceToSecurityGroup(
					ctx,
					zone,
					instance,
					&egoscale.SecurityGroup{ID: utils.NonEmptyStringPtr(id.(string))},
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}

		if removed := old.Difference(cur); removed.Len() > 0 {
			for _, id := range removed.List() {
				if err := client.DetachInstanceFromSecurityGroup(
					ctx,
					zone,
					instance,
					&egoscale.SecurityGroup{ID: utils.NonEmptyStringPtr(id.(string))},
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	if d.HasChanges(
		AttrState,
		AttrDiskSize,
		AttrType,
	) {
		// Check if size is below current size to prevent uneeded stop as API will prevent the scale operation
		if d.HasChange(AttrDiskSize) &&
			*instance.DiskSize > int64(d.Get(AttrDiskSize).(int)) {
			return diag.Errorf("unable to scale down the disk size, use size > %v", *instance.DiskSize)
		}

		// Compute instance scaling/disk resizing API operations requires the instance to be stopped.
		if d.Get(AttrState) == "stopped" ||
			d.HasChange(AttrDiskSize) ||
			d.HasChange(AttrType) {
			if err := client.StopInstance(ctx, zone, instance); err != nil {
				return diag.Errorf("unable to stop instance: %s", err)
			}
		}

		if d.HasChange(AttrDiskSize) {
			if err = client.ResizeInstanceDisk(
				ctx,
				zone,
				instance,
				int64(d.Get(AttrDiskSize).(int)),
			); err != nil {
				return diag.FromErr(err)
			}
		}

		if d.HasChange(AttrType) {
			instanceType, err := client.FindInstanceType(ctx, zone, d.Get(AttrType).(string))
			if err != nil {
				return diag.Errorf("unable to retrieve instance type: %s", err)
			}
			if err = client.ScaleInstance(ctx, zone, instance, instanceType); err != nil {
				return diag.FromErr(err)
			}
		}

		if d.Get(AttrState) == "running" {
			if err := client.StartInstance(ctx, zone, instance); err != nil {
				return diag.Errorf("unable to start instance: %s", err)
			}
		}
	}

	if d.HasChanges(AttrDestroyProtected) {
		if isDestroyProtected, ok := d.GetOk(AttrDestroyProtected); ok {
			if isDestroyProtected.(bool) {
				_, err := client.AddInstanceProtectionWithResponse(ctx, *instance.ID)
				if err != nil {
					return diag.Errorf("unable to make instance %s destroy protected: %s", *instance.ID, err)
				}
			} else {
				_, err := client.RemoveInstanceProtectionWithResponse(ctx, *instance.ID)
				if err != nil {
					return diag.Errorf("unable to remove destroy protection from instance %s: %s", *instance.ID, err)
				}
			}
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return rRead(ctx, d, meta)
}

func rDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	zone := d.Get(AttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(config.GetEnvironment(meta), zone))
	defer cancel()

	client, err := config.GetClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.DeleteInstanceReverseDNS(ctx, zone, d.Id()); err != nil && !errors.Is(err, exoapi.ErrNotFound) {
		return diag.FromErr(err)
	}
	err = client.DeleteInstance(
		ctx,
		zone,
		&egoscale.Instance{ID: utils.NonEmptyStringPtr(d.Id())},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return nil
}

func rApply( //nolint:gocyclo
	ctx context.Context,
	client *egoscale.Client,
	d *schema.ResourceData,
	instance *egoscale.Instance,
) diag.Diagnostics {
	zone := d.Get(AttrZone).(string)

	if instance.AntiAffinityGroupIDs != nil {
		antiAffinityGroupIDs := make([]string, len(*instance.AntiAffinityGroupIDs))
		copy(antiAffinityGroupIDs, *instance.AntiAffinityGroupIDs)

		if err := d.Set(AttrAntiAffinityGroupIDs, antiAffinityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(AttrCreatedAt, instance.CreatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(
		AttrDeployTargetID,
		utils.DefaultString(instance.DeployTargetID, ""),
	); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrDiskSize, *instance.DiskSize); err != nil {
		return diag.FromErr(err)
	}

	if instance.ElasticIPIDs != nil {
		elasticIPIDs := make([]string, len(*instance.ElasticIPIDs))
		copy(elasticIPIDs, *instance.ElasticIPIDs)

		if err := d.Set(AttrElasticIPIDs, elasticIPIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(AttrIPv6, utils.DefaultBool(instance.IPv6Enabled, false)); err != nil {
		return diag.FromErr(err)
	}

	if instance.IPv6Address != nil {
		if err := d.Set(AttrIPv6Address, instance.IPv6Address.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(AttrLabels, instance.Labels); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrName, *instance.Name); err != nil {
		return diag.FromErr(err)
	}

	if instance.PrivateNetworkIDs != nil {
		privateNetworkIDs := make([]string, len(*instance.PrivateNetworkIDs))
		networkInterfaces := make([]map[string]interface{}, len(*instance.PrivateNetworkIDs))

		for i, id := range *instance.PrivateNetworkIDs {
			privateNetwork, err := client.GetPrivateNetwork(ctx, zone, id)
			if err != nil {
				return diag.FromErr(err)
			}

			var instanceAddress *string
			for _, lease := range privateNetwork.Leases {
				if *lease.InstanceID == *instance.ID {
					address := lease.IPAddress.String()
					instanceAddress = &address
					break
				}
			}

			nif, err := NetworkInterface{id, instanceAddress}.ToInterface()
			if err != nil {
				return diag.FromErr(err)
			}

			networkInterfaces[i] = nif
			privateNetworkIDs[i] = id
		}
		if err := d.Set(AttrPrivateNetworkIDs, privateNetworkIDs); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(AttrNetworkInterface, networkInterfaces); err != nil {
			return diag.FromErr(err)
		}
	}

	if instance.PublicIPAddress != nil {
		if err := d.Set(AttrPublicIPAddress, instance.PublicIPAddress.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(AttrSSHKey, instance.SSHKey); err != nil {
		return diag.FromErr(err)
	}

	if instance.SecurityGroupIDs != nil {
		securityGroupIDs := make([]string, len(*instance.SecurityGroupIDs))
		copy(securityGroupIDs, *instance.SecurityGroupIDs)

		if err := d.Set(AttrSecurityGroupIDs, securityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(AttrState, instance.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrTemplateID, instance.TemplateID); err != nil {
		return diag.FromErr(err)
	}

	rdns, err := client.GetInstanceReverseDNS(
		ctx,
		d.Get(AttrZone).(string),
		*instance.ID,
	)
	if err != nil && !errors.Is(err, exoapi.ErrNotFound) {
		return diag.Errorf("unable to retrieve instance reverse-dns: %s", err)
	}
	if err := d.Set(AttrReverseDNS, strings.TrimSuffix(rdns, ".")); err != nil {
		return diag.FromErr(err)
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		d.Get(AttrZone).(string),
		*instance.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("unable to retrieve instance type: %s", err)
	}
	if err := d.Set(AttrType, fmt.Sprintf(
		"%s.%s",
		strings.ToLower(*instanceType.Family),
		strings.ToLower(*instanceType.Size),
	)); err != nil {
		return diag.FromErr(err)
	}

	if instance.UserData != nil {
		userData, err := utils.DecodeUserData(*instance.UserData)
		if err != nil {
			return diag.Errorf("unable to decode user data: %s", err)
		}
		if err := d.Set(AttrUserData, userData); err != nil {
			return diag.FromErr(err)
		}
	}

	if instance.PublicIPAddress != nil {
		// Connection info for the `ssh` remote-exec provisioner
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": instance.PublicIPAddress.String(),
		})
	}

	return nil
}
