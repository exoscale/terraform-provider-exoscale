package instance_pool

import (
	"context"
	"errors"
	"fmt"
	"strings"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const (
	DefaultInstancePrefix = "pool"
)

func Resource() *schema.Resource {
	s := map[string]*schema.Schema{
		AttrAffinityGroupIDs: {
			Description: "A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs; may only be set at creation time).",
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrDeployTargetID: {
			Description: "A deploy target ID.",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrDescription: {
			Description: "A free-form text describing the pool.",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrDiskSize: {
			Description: "The managed instances disk size (GiB).",
			Type:        schema.TypeInt,
			Computed:    true,
			Optional:    true,
		},
		AttrElasticIPIDs: {
			Description: "A list of [exoscale_elastic_ip](./elastic_ip.md) (IDs).",
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrInstancePrefix: {
			Description: "The string used to prefix managed instances name (default: `pool`).",
			Type:        schema.TypeString,
			Optional:    true,
			Default:     DefaultInstancePrefix,
		},
		AttrInstanceType: {
			// TODO: as long as "service_offering" is still deprecated but supported,
			//  we cannot make "instance_type" required as it'd break existing configurations.
			//  As soon as the "service_offering" parameter is phased out, the schema must be changed:
			//  - Optional:true must become Required:true
			//  - Computed:true must be removed
			Description:      "The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI](https://github.com/exoscale/cli/) - `exo compute instance-type list` - for the list of available types).",
			Type:             schema.TypeString,
			Optional:         true,
			Computed:         true,
			ConflictsWith:    []string{AttrServiceOffering},
			ValidateDiagFunc: utils.ValidateComputeInstanceType,
			// Ignore case differences
			DiffSuppressFunc: utils.SuppressCaseDiff,
		},
		AttrIPv6: {
			Description: "Enable IPv6 on managed instances (boolean; default: `false`).",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
		},
		AttrKeyPair: {
			Description: "The [exoscale_ssh_key](./ssh_key.md) (name) to authorize in the managed instances.",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrLabels: {
			Description: "A map of key/value labels.",
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
		},
		AttrName: {
			Description: "The instance pool name.",
			Type:        schema.TypeString,
			Required:    true,
		},
		AttrNetworkIDs: {
			Description: "A list of [exoscale_private_network](./private_network.md) (IDs).",
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrSecurityGroupIDs: {
			Description: "A list of [exoscale_security_group](./security_groups.md) (IDs).",
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrServiceOffering: {
			Description:   "The managed instances type. Please use the `instance_type` argument instead.",
			Type:          schema.TypeString,
			Optional:      true,
			Computed:      true,
			Deprecated:    `This attribute has been replaced by "instance_type".`,
			ConflictsWith: []string{AttrInstanceType},
			ValidateFunc:  utils.ValidateLowercaseString,
		},
		AttrSize: {
			Description:  "The number of managed instances.",
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(1),
		},
		AttrState: {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		AttrTemplateID: {
			Description: "The [exoscale_compute_template](../data-sources/compute_template.md) (ID) to use when creating the managed instances.",
			Type:        schema.TypeString,
			Required:    true,
		},
		AttrUserData: {
			Description: "[cloud-init](http://cloudinit.readthedocs.io/) configuration to apply to the managed instances (no need to base64-encode or gzip it as the provider will take care of it).",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrVirtualMachines: {
			Description: "The list of managed instances (IDs). Please use the `instances.*.id` attribute instead.",
			Type:        schema.TypeSet,
			Optional:    true,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Deprecated:  "Use the instances exported attribute instead.",
		},
		AttrInstances: {
			Description: "The list of managed instances. Structure is documented below.",
			Type:        schema.TypeSet,
			Optional:    true,
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					AttrInstanceID: {
						Type:     schema.TypeString,
						Optional: true,
					},
					AttrInstanceIPv6Address: {
						Description: "The instance (main network interface) IPv6 address.",
						Type:        schema.TypeString,
						Computed:    true,
					},
					AttrInstanceName: {
						Description: "The instance name.",
						Type:        schema.TypeString,
						Optional:    true,
					},
					AttrInstancePublicIPAddress: {
						Description: "The instance (main network interface) IPv4 address.",
						Type:        schema.TypeString,
						Computed:    true,
					},
				},
			},
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

func rCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	pool := new(egoscale.InstancePool)

	if v, ok := d.GetOk(AttrDeployTargetID); ok {
		s := v.(string)
		pool.DeployTargetID = &s
	}

	if v, ok := d.GetOk(AttrDescription); ok {
		s := v.(string)
		pool.Description = &s
	}

	if v, ok := d.GetOk(AttrDiskSize); ok {
		i := int64(v.(int))
		pool.DiskSize = &i
	}

	if v, ok := d.GetOk(AttrName); ok {
		s := v.(string)
		pool.Name = &s
	}

	if v, ok := d.GetOk(AttrInstancePrefix); ok {
		s := v.(string)
		pool.InstancePrefix = &s
	}

	if v, ok := d.GetOk(AttrKeyPair); ok {
		s := v.(string)
		pool.SSHKey = &s
	}

	if l, ok := d.GetOk(AttrLabels); ok {
		labels := make(map[string]string)
		for k, v := range l.(map[string]interface{}) {
			labels[k] = v.(string)
		}
		pool.Labels = &labels
	}

	if v, ok := d.GetOk(AttrSize); ok {
		i := int64(v.(int))
		pool.Size = &i
	}

	if v, ok := d.GetOk(AttrTemplateID); ok {
		s := v.(string)
		pool.TemplateID = &s
	}

	// FIXME: once the "instance_type" attribute has been made required, this check can be removed.
	if d.Get(AttrServiceOffering).(string) == "" &&
		d.Get(AttrInstanceType).(string) == "" {
		return diag.Errorf(
			"either %s or %s must be set",
			AttrServiceOffering,
			AttrInstanceType,
		)
	}
	it := d.Get(AttrServiceOffering).(string)
	if v, ok := d.GetOk(AttrInstanceType); ok {
		it = v.(string)
	}

	instanceType, err := client.FindInstanceType(ctx, zone, it)
	if err != nil {
		return diag.Errorf("error retrieving instance type: %s", err)
	}
	pool.InstanceTypeID = instanceType.ID

	if set, ok := d.Get(AttrAffinityGroupIDs).(*schema.Set); ok {
		pool.AntiAffinityGroupIDs = func() (v *[]string) {
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

	if set, ok := d.Get(AttrSecurityGroupIDs).(*schema.Set); ok {
		pool.SecurityGroupIDs = func() (v *[]string) {
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

	if set, ok := d.Get(AttrNetworkIDs).(*schema.Set); ok {
		pool.PrivateNetworkIDs = func() (v *[]string) {
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

	if set, ok := d.Get(AttrElasticIPIDs).(*schema.Set); ok {
		pool.ElasticIPIDs = func() (v *[]string) {
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

	enableIPv6 := d.Get(AttrIPv6).(bool)
	pool.IPv6Enabled = &enableIPv6

	if v := d.Get(AttrUserData).(string); v != "" {
		userData, _, err := utils.EncodeUserData(v)
		if err != nil {
			return diag.FromErr(err)
		}
		pool.UserData = &userData
	}

	// FIXME: we have to reference the embedded egoscale/v2.Client explicitly
	//  here because there is already a CreateInstancePool() method on the root
	//  egoscale client clashing with the v2 one. This can be changed once we
	//  use API V2-only calls.
	pool, err = client.CreateInstancePool(ctx, zone, pool)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*pool.ID)

	if err := client.WaitInstancePoolConverged(ctx, zone, *pool.ID); err != nil {
		return diag.FromErr(err)
	}

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

	pool, err := client.GetInstancePool(ctx, zone, d.Id())
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

	return rApply(ctx, client, d, pool)
}

func rUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	pool, err := client.GetInstancePool(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var updated bool

	if d.HasChange(AttrAffinityGroupIDs) {
		set := d.Get(AttrAffinityGroupIDs).(*schema.Set)
		pool.AntiAffinityGroupIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(AttrDeployTargetID) {
		v := d.Get(AttrDeployTargetID).(string)
		pool.DeployTargetID = &v
		updated = true
	}

	if d.HasChange(AttrDescription) {
		v := d.Get(AttrDescription).(string)
		pool.Description = &v
		updated = true
	}

	if d.HasChange(AttrDiskSize) {
		v := int64(d.Get(AttrDiskSize).(int))
		pool.DiskSize = &v
		updated = true
	}

	if d.HasChange(AttrElasticIPIDs) {
		set := d.Get(AttrElasticIPIDs).(*schema.Set)
		pool.ElasticIPIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(AttrInstancePrefix) {
		v := d.Get(AttrInstancePrefix).(string)
		pool.InstancePrefix = &v
		updated = true
	}

	if d.HasChange(AttrIPv6) {
		v := d.Get(AttrIPv6).(bool)
		pool.IPv6Enabled = &v
		updated = true
	}

	if d.HasChange(AttrKeyPair) {
		v := d.Get(AttrKeyPair).(string)
		pool.SSHKey = &v
		updated = true
	}

	if d.HasChange(AttrLabels) {
		labels := make(map[string]string)
		for k, v := range d.Get(AttrLabels).(map[string]interface{}) {
			labels[k] = v.(string)
		}
		pool.Labels = &labels
		updated = true
	}

	if d.HasChange(AttrName) {
		v := d.Get(AttrName).(string)
		pool.Name = &v
		updated = true
	}

	if d.HasChange(AttrNetworkIDs) {
		set := d.Get(AttrNetworkIDs).(*schema.Set)
		pool.PrivateNetworkIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(AttrSecurityGroupIDs) {
		set := d.Get(AttrSecurityGroupIDs).(*schema.Set)
		pool.SecurityGroupIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(AttrInstanceType) {
		instanceType, err := client.FindInstanceType(ctx, zone, d.Get(AttrInstanceType).(string))
		if err != nil {
			return diag.Errorf("error retrieving instance type: %s", err)
		}
		pool.InstanceTypeID = instanceType.ID
		updated = true
	}

	if d.HasChange(AttrTemplateID) {
		v := d.Get(AttrTemplateID).(string)
		pool.TemplateID = &v
		updated = true
	}

	if d.HasChange(AttrUserData) {
		v, _, err := utils.EncodeUserData(d.Get(AttrUserData).(string))
		if err != nil {
			return diag.FromErr(err)
		}
		pool.UserData = &v
		updated = true
	}

	if updated {
		if err = client.UpdateInstancePool(ctx, zone, pool); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(AttrSize) {
		if err = client.ScaleInstancePool(
			ctx,
			zone,
			pool,
			int64(d.Get(AttrSize).(int)),
		); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := client.WaitInstancePoolConverged(ctx, zone, *pool.ID); err != nil {
		return diag.FromErr(err)
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

	poolID := d.Id()
	err = client.DeleteInstancePool(ctx, zone, &egoscale.InstancePool{ID: &poolID})
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return nil
}

func rApply(ctx context.Context, client *egoscale.Client, d *schema.ResourceData, pool *egoscale.InstancePool) diag.Diagnostics {
	zone := d.Get(AttrZone).(string)

	if pool.AntiAffinityGroupIDs != nil {
		antiAffinityGroupIDs := make([]string, len(*pool.AntiAffinityGroupIDs))
		copy(antiAffinityGroupIDs, *pool.AntiAffinityGroupIDs)

		if err := d.Set(AttrAffinityGroupIDs, antiAffinityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(AttrDeployTargetID, utils.DefaultString(pool.DeployTargetID, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrDescription, utils.DefaultString(pool.Description, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrDiskSize, *pool.DiskSize); err != nil {
		return diag.FromErr(err)
	}

	if pool.ElasticIPIDs != nil {
		elasticIPIDs := make([]string, len(*pool.ElasticIPIDs))
		copy(elasticIPIDs, *pool.ElasticIPIDs)

		if err := d.Set(AttrElasticIPIDs, elasticIPIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(AttrInstancePrefix, utils.DefaultString(pool.InstancePrefix, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrIPv6, utils.DefaultBool(pool.IPv6Enabled, false)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrKeyPair, pool.SSHKey); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrLabels, pool.Labels); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrName, pool.Name); err != nil {
		return diag.FromErr(err)
	}

	if pool.PrivateNetworkIDs != nil {
		if err := d.Set(AttrNetworkIDs, *pool.PrivateNetworkIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if pool.SecurityGroupIDs != nil {
		if err := d.Set(AttrSecurityGroupIDs, *pool.SecurityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		d.Get(AttrZone).(string),
		*pool.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("error retrieving instance type: %s", err)
	}
	if err := d.Set(AttrInstanceType, fmt.Sprintf(
		"%s.%s",
		strings.ToLower(*instanceType.Family),
		strings.ToLower(*instanceType.Size),
	)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(AttrServiceOffering, strings.ToLower(*instanceType.Size)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrSize, pool.Size); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrState, pool.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(AttrTemplateID, pool.TemplateID); err != nil {
		return diag.FromErr(err)
	}

	if pool.UserData != nil {
		userData, err := utils.DecodeUserData(*pool.UserData)
		if err != nil {
			return diag.Errorf("error decoding user data: %s", err)
		}
		if err := d.Set(AttrUserData, userData); err != nil {
			return diag.FromErr(err)
		}
	}

	if pool.InstanceIDs != nil {
		instanceIDs := make([]string, len(*pool.InstanceIDs))
		instanceDetails := make([]interface{}, len(*pool.InstanceIDs))

		for i, id := range *pool.InstanceIDs {
			instanceIDs[i] = id

			// instance details
			instance, err := client.GetInstance(ctx, zone, id)
			if err != nil {
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

			instanceDetails[i] = computeInstanceToResource(instance, instanceType)
		}

		if err := d.Set(AttrVirtualMachines, instanceIDs); err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(AttrInstances, instanceDetails); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func computeInstanceToResource(instance *egoscale.Instance, instanceType *egoscale.InstanceType) interface{} {
	c := make(map[string]interface{})
	c[AttrInstanceID] = instance.ID
	c[AttrInstanceIPv6Address] = utils.AddressToStringPtr(instance.IPv6Address)
	c[AttrInstanceName] = instance.Name
	c[AttrInstancePublicIPAddress] = utils.AddressToStringPtr(instance.PublicIPAddress)

	return c
}
