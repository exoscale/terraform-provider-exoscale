package exoscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	resComputeInstanceAttrAntiAffinityGroupIDs = "anti_affinity_group_ids"
	resComputeInstanceAttrCreatedAt            = "created_at"
	resComputeInstanceAttrDeployTargetID       = "deploy_target_id"
	resComputeInstanceAttrDiskSize             = "disk_size"
	resComputeInstanceAttrElasticIPIDs         = "elastic_ip_ids"
	resComputeInstanceAttrIPv6                 = "ipv6"
	resComputeInstanceAttrIPv6Address          = "ipv6_address"
	resComputeInstanceAttrLabels               = "labels"
	resComputeInstanceAttrName                 = "name"
	resComputeInstanceAttrNetworkInterface     = "network_interface"
	resComputeInstanceAttrPrivateNetworkIDs    = "private_network_ids"
	resComputeInstanceAttrPublicIPAddress      = "public_ip_address"
	resComputeInstanceAttrReverseDNS           = "reverse_dns"
	resComputeInstanceAttrSSHKey               = "ssh_key"
	resComputeInstanceAttrSecurityGroupIDs     = "security_group_ids"
	resComputeInstanceAttrState                = "state"
	resComputeInstanceAttrTemplateID           = "template_id"
	resComputeInstanceAttrType                 = "type"
	resComputeInstanceAttrUserData             = "user_data"
	resComputeInstanceAttrZone                 = "zone"
)

func resourceComputeInstanceIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_instance_pool")
}

func resourceComputeInstance() *schema.Resource {
	s := map[string]*schema.Schema{
		resComputeInstanceAttrAntiAffinityGroupIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
			ForceNew: true,
		},
		resComputeInstanceAttrCreatedAt: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resComputeInstanceAttrDeployTargetID: {
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
		},
		resComputeInstanceAttrDiskSize: {
			Type:         schema.TypeInt,
			Computed:     true,
			Optional:     true,
			ValidateFunc: validation.IntAtLeast(10),
		},
		resComputeInstanceAttrElasticIPIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resComputeInstanceAttrIPv6: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		resComputeInstanceAttrIPv6Address: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resComputeInstanceAttrLabels: {
			Type:     schema.TypeMap,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
		},
		resComputeInstanceAttrName: {
			Type:     schema.TypeString,
			Required: true,
		},
		resComputeInstanceAttrPrivateNetworkIDs: {
			Type:       schema.TypeSet,
			Computed:   true,
			Set:        schema.HashString,
			Elem:       &schema.Schema{Type: schema.TypeString},
			Deprecated: "Use the network_interface block instead.",
		},
		resComputeInstanceAttrNetworkInterface: {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"ip_address": {
						Type:             schema.TypeString,
						Optional:         true,
						Description:      "IP address",
						ValidateDiagFunc: validation.ToDiagFunc(validation.IsIPv4Address),
					},
					"network_id": {
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
		resComputeInstanceAttrPublicIPAddress: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resComputeInstanceAttrReverseDNS: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resComputeInstanceAttrSSHKey: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resComputeInstanceAttrSecurityGroupIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resComputeInstanceAttrState: {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		resComputeInstanceAttrTemplateID: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		resComputeInstanceAttrType: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validateComputeInstanceType,
			// Ignore case differences
			DiffSuppressFunc: suppressCaseDiff,
		},
		resComputeInstanceAttrUserData: {
			Type:             schema.TypeString,
			ValidateDiagFunc: validateComputeUserData,
			Optional:         true,
		},
		resComputeInstanceAttrZone: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}

	return &schema.Resource{
		Schema: s,

		CreateContext: resourceComputeInstanceCreate,
		ReadContext:   resourceComputeInstanceRead,
		UpdateContext: resourceComputeInstanceUpdate,
		DeleteContext: resourceComputeInstanceDelete,

		Importer: &schema.ResourceImporter{
			StateContext: zonedStateContextFunc,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

type instanceNetworkInterface struct {
	NetworkID string  `json:"network_id"`
	IPAddress *string `json:"ip_address"`
}

func newInstanceNetworkInterfaceFromInterface(rawStatePart interface{}) (*instanceNetworkInterface, error) {
	serializedRule, err := json.Marshal(rawStatePart)
	if err != nil {
		return nil, err
	}

	networkInterface := instanceNetworkInterface{}
	if err := json.Unmarshal(serializedRule, &networkInterface); err != nil {
		tflog.Warn(context.Background(), err.Error())
		return nil, err
	}

	return &networkInterface, nil
}

func (n instanceNetworkInterface) toInterface() (map[string]interface{}, error) {
	serialized, err := json.Marshal(n)
	if err != nil {
		return nil, err
	}

	var networkInterface map[string]interface{}
	if err := json.Unmarshal(serialized, &networkInterface); err != nil {
		return nil, err
	}

	return networkInterface, nil
}

func resourceComputeInstanceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	zone := d.Get(resComputeInstanceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	computeInstance := &egoscale.Instance{
		Name:       nonEmptyStringPtr(d.Get(resComputeInstanceAttrName).(string)),
		TemplateID: nonEmptyStringPtr(d.Get(resComputeInstanceAttrTemplateID).(string)),
	}

	if set, ok := d.Get(resComputeInstanceAttrAntiAffinityGroupIDs).(*schema.Set); ok {
		computeInstance.AntiAffinityGroupIDs = func() (v *[]string) {
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

	if v, ok := d.GetOk(resComputeInstanceAttrDeployTargetID); ok {
		s := v.(string)
		computeInstance.DeployTargetID = &s
	}

	if v, ok := d.GetOk(resComputeInstanceAttrDiskSize); ok {
		i := int64(v.(int))
		computeInstance.DiskSize = &i
	}

	enableIPv6 := d.Get(resComputeInstanceAttrIPv6).(bool)
	computeInstance.IPv6Enabled = &enableIPv6

	if l, ok := d.GetOk(resComputeInstanceAttrLabels); ok {
		labels := make(map[string]string)
		for k, v := range l.(map[string]interface{}) {
			labels[k] = v.(string)
		}
		computeInstance.Labels = &labels
	}

	if v, ok := d.GetOk(resComputeInstanceAttrSSHKey); ok {
		s := v.(string)
		computeInstance.SSHKey = &s
	}

	if set, ok := d.Get(resComputeInstanceAttrSecurityGroupIDs).(*schema.Set); ok {
		computeInstance.SecurityGroupIDs = func() (v *[]string) {
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

	instanceType, err := client.FindInstanceType(ctx, zone, d.Get(resComputeInstanceAttrType).(string))
	if err != nil {
		return diag.Errorf("unable to retrieve instance type: %s", err)
	}
	computeInstance.InstanceTypeID = instanceType.ID

	if v := d.Get(resComputeInstanceAttrUserData).(string); v != "" {
		userData, _, err := encodeUserData(v)
		if err != nil {
			return diag.FromErr(err)
		}
		computeInstance.UserData = &userData
	}

	// FIXME: we have to reference the embedded egoscale/v2.Client explicitly
	//  here because there is already a CreateComputeInstance() method on the root
	//  egoscale client clashing with the v2 one. This can be changed once we
	//  use API V2-only calls.
	computeInstance, err = client.Client.CreateInstance(ctx, zone, computeInstance)
	if err != nil {
		return diag.FromErr(err)
	}

	if set, ok := d.Get(resComputeInstanceAttrElasticIPIDs).(*schema.Set); ok {
		if set.Len() > 0 {
			for _, id := range set.List() {
				if err := client.AttachInstanceToElasticIP(
					ctx,
					zone,
					computeInstance,
					&egoscale.ElasticIP{ID: nonEmptyStringPtr(id.(string))},
				); err != nil {
					return diag.Errorf("unable to attach Elastic IP %s: %s", id.(string), err)
				}
			}
		}
	}

	if networkInterfaceSet, ok := d.Get(resComputeInstanceAttrNetworkInterface).(*schema.Set); ok {
		for _, networkInterface := range networkInterfaceSet.List() {
			networkInterface, err := newInstanceNetworkInterfaceFromInterface(networkInterface)
			if err != nil {
				return diag.FromErr(err)
			}

			opts := []egoscale.AttachInstanceToPrivateNetworkOpt{}
			if networkInterface.IPAddress != nil && *networkInterface.IPAddress != "" {
				opts = append(opts, egoscale.AttachInstanceToPrivateNetworkWithIPAddress(net.ParseIP(*networkInterface.IPAddress)))
			}

			if err := client.AttachInstanceToPrivateNetwork(
				ctx,
				zone,
				computeInstance,
				&egoscale.PrivateNetwork{ID: &networkInterface.NetworkID},
				opts...,
			); err != nil {
				return diag.Errorf("unable to attach Private Network %s: %s", networkInterface.NetworkID, err)
			}
		}
	}

	if v, ok := d.GetOk(resComputeInstanceAttrReverseDNS); ok {
		rdns := v.(string)
		err := client.UpdateInstanceReverseDNS(
			ctx,
			zone,
			*computeInstance.ID,
			rdns,
		)
		if err != nil {
			return diag.Errorf("unable to create Reverse DNS record: %s", err)
		}
	}

	if v := d.Get(resComputeInstanceAttrState).(string); v == "stopped" {
		if err := client.StopInstance(ctx, zone, computeInstance); err != nil {
			return diag.Errorf("unable to stop instance: %s", err)
		}
	}

	d.SetId(*computeInstance.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	return resourceComputeInstanceRead(ctx, d, meta)
}

func resourceComputeInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	zone := d.Get(resComputeInstanceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	computeInstance, err := client.GetInstance(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	return resourceComputeInstanceApply(ctx, GetComputeClient(meta).Client, d, computeInstance)
}

func resourceComputeInstanceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	zone := d.Get(resComputeInstanceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	computeInstance, err := client.GetInstance(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var updated bool

	if d.HasChange(resComputeInstanceAttrLabels) {
		labels := make(map[string]string)
		for k, v := range d.Get(resComputeInstanceAttrLabels).(map[string]interface{}) {
			labels[k] = v.(string)
		}
		computeInstance.Labels = &labels
		updated = true
	}

	if d.HasChange(resComputeInstanceAttrName) {
		v := d.Get(resComputeInstanceAttrName).(string)
		computeInstance.Name = &v
		updated = true
	}

	if d.HasChange(resComputeInstanceAttrUserData) {
		v, _, err := encodeUserData(d.Get(resComputeInstanceAttrUserData).(string))
		if err != nil {
			return diag.FromErr(err)
		}
		computeInstance.UserData = &v
		updated = true
	}

	if updated {
		if err = client.UpdateInstance(ctx, zone, computeInstance); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(resComputeInstanceAttrReverseDNS) {
		rdns := d.Get(resComputeInstanceAttrReverseDNS).(string)
		if rdns == "" {
			err = client.DeleteInstanceReverseDNS(
				ctx,
				zone,
				*computeInstance.ID,
			)
		} else {
			err = client.UpdateInstanceReverseDNS(
				ctx,
				zone,
				*computeInstance.ID,
				rdns,
			)
		}
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(resComputeInstanceAttrElasticIPIDs) {
		o, n := d.GetChange(resComputeInstanceAttrElasticIPIDs)
		old := o.(*schema.Set)
		cur := n.(*schema.Set)

		if added := cur.Difference(old); added.Len() > 0 {
			for _, id := range added.List() {
				if err := client.AttachInstanceToElasticIP(
					ctx,
					zone,
					computeInstance,
					&egoscale.ElasticIP{ID: nonEmptyStringPtr(id.(string))},
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
					computeInstance,
					&egoscale.ElasticIP{ID: nonEmptyStringPtr(id.(string))},
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	if d.HasChange(resComputeInstanceAttrNetworkInterface) {
		o, n := d.GetChange(resComputeInstanceAttrNetworkInterface)
		old := o.(*schema.Set)
		cur := n.(*schema.Set)

		if removed := old.Difference(cur); removed.Len() > 0 {
			for _, networkInterface := range removed.List() {
				networkInterface, err := newInstanceNetworkInterfaceFromInterface(networkInterface)
				if err != nil {
					return diag.FromErr(err)
				}

				if err := client.DetachInstanceFromPrivateNetwork(
					ctx,
					zone,
					computeInstance,
					&egoscale.PrivateNetwork{ID: &networkInterface.NetworkID},
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}

		if added := cur.Difference(old); added.Len() > 0 {
			for _, networkInterface := range added.List() {
				networkInterface, err := newInstanceNetworkInterfaceFromInterface(networkInterface)
				if err != nil {
					return diag.FromErr(err)
				}

				opts := []egoscale.AttachInstanceToPrivateNetworkOpt{}
				if networkInterface.IPAddress != nil && *networkInterface.IPAddress != "" {
					opts = append(opts, egoscale.AttachInstanceToPrivateNetworkWithIPAddress(net.ParseIP(*networkInterface.IPAddress)))
				}

				if err := client.AttachInstanceToPrivateNetwork(
					ctx,
					zone,
					computeInstance,
					&egoscale.PrivateNetwork{ID: &networkInterface.NetworkID},
					opts...,
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	if d.HasChange(resComputeInstanceAttrSecurityGroupIDs) {
		o, n := d.GetChange(resComputeInstanceAttrSecurityGroupIDs)
		old := o.(*schema.Set)
		cur := n.(*schema.Set)

		if added := cur.Difference(old); added.Len() > 0 {
			for _, id := range added.List() {
				if err := client.AttachInstanceToSecurityGroup(
					ctx,
					zone,
					computeInstance,
					&egoscale.SecurityGroup{ID: nonEmptyStringPtr(id.(string))},
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
					computeInstance,
					&egoscale.SecurityGroup{ID: nonEmptyStringPtr(id.(string))},
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	if d.HasChanges(
		resComputeInstanceAttrState,
		resComputeInstanceAttrDiskSize,
		resComputeInstanceAttrType,
	) {
		// Compute instance scaling/disk resizing API operations requires the instance to be stopped.
		if d.Get(resComputeInstanceAttrState) == "stopped" ||
			d.HasChange(resComputeInstanceAttrDiskSize) ||
			d.HasChange(resComputeInstanceAttrType) {
			if err := client.StopInstance(ctx, zone, computeInstance); err != nil {
				return diag.Errorf("unable to stop instance: %s", err)
			}
		}

		if d.HasChange(resComputeInstanceAttrDiskSize) {
			if err = client.ResizeInstanceDisk(
				ctx,
				zone,
				computeInstance,
				int64(d.Get(resComputeInstanceAttrDiskSize).(int)),
			); err != nil {
				return diag.FromErr(err)
			}
		}

		if d.HasChange(resComputeInstanceAttrType) {
			instanceType, err := client.FindInstanceType(ctx, zone, d.Get(resComputeInstanceAttrType).(string))
			if err != nil {
				return diag.Errorf("unable to retrieve instance type: %s", err)
			}
			if err = client.ScaleInstance(ctx, zone, computeInstance, instanceType); err != nil {
				return diag.FromErr(err)
			}
		}

		if d.Get(resComputeInstanceAttrState) == "running" {
			if err := client.StartInstance(ctx, zone, computeInstance); err != nil {
				return diag.Errorf("unable to start instance: %s", err)
			}
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	return resourceComputeInstanceRead(ctx, d, meta)
}

func resourceComputeInstanceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	zone := d.Get(resComputeInstanceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)
	if err := client.DeleteInstanceReverseDNS(ctx, zone, d.Id()); err != nil && !errors.Is(err, exoapi.ErrNotFound) {
		return diag.FromErr(err)
	}
	err := client.DeleteInstance(ctx, zone, &egoscale.Instance{ID: nonEmptyStringPtr(d.Id())})
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	return nil
}

func resourceComputeInstanceApply(
	ctx context.Context,
	client *egoscale.Client,
	d *schema.ResourceData,
	computeInstance *egoscale.Instance,
) diag.Diagnostics {
	zone := d.Get(resComputeInstanceAttrZone).(string)

	if computeInstance.AntiAffinityGroupIDs != nil {
		antiAffinityGroupIDs := make([]string, len(*computeInstance.AntiAffinityGroupIDs))
		for i, id := range *computeInstance.AntiAffinityGroupIDs {
			antiAffinityGroupIDs[i] = id
		}
		if err := d.Set(resComputeInstanceAttrAntiAffinityGroupIDs, antiAffinityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resComputeInstanceAttrCreatedAt, computeInstance.CreatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(
		resComputeInstanceAttrDeployTargetID,
		defaultString(computeInstance.DeployTargetID, ""),
	); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resComputeInstanceAttrDiskSize, *computeInstance.DiskSize); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.ElasticIPIDs != nil {
		elasticIPIDs := make([]string, len(*computeInstance.ElasticIPIDs))
		for i, id := range *computeInstance.ElasticIPIDs {
			elasticIPIDs[i] = id
		}
		if err := d.Set(resComputeInstanceAttrElasticIPIDs, elasticIPIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resComputeInstanceAttrIPv6, defaultBool(computeInstance.IPv6Enabled, false)); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.IPv6Address != nil {
		if err := d.Set(resComputeInstanceAttrIPv6Address, computeInstance.IPv6Address.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resComputeInstanceAttrLabels, computeInstance.Labels); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resComputeInstanceAttrName, *computeInstance.Name); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.PrivateNetworkIDs != nil {
		privateNetworkIDs := make([]string, len(*computeInstance.PrivateNetworkIDs))
		networkInterfaces := make([]map[string]interface{}, len(*computeInstance.PrivateNetworkIDs))

		for i, id := range *computeInstance.PrivateNetworkIDs {
			privateNetwork, err := client.GetPrivateNetwork(ctx, zone, id)
			if err != nil {
				return diag.FromErr(err)
			}

			var instanceAddress *string
			for _, lease := range privateNetwork.Leases {
				if *lease.InstanceID == *computeInstance.ID {
					address := lease.IPAddress.String()
					instanceAddress = &address
					break
				}
			}

			networkInterface, err := instanceNetworkInterface{id, instanceAddress}.toInterface()
			if err != nil {
				return diag.FromErr(err)
			}

			networkInterfaces[i] = networkInterface
			privateNetworkIDs[i] = id
		}
		if err := d.Set(resComputeInstanceAttrPrivateNetworkIDs, privateNetworkIDs); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(resComputeInstanceAttrNetworkInterface, networkInterfaces); err != nil {
			return diag.FromErr(err)
		}
	}

	if computeInstance.PublicIPAddress != nil {
		if err := d.Set(resComputeInstanceAttrPublicIPAddress, computeInstance.PublicIPAddress.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resComputeInstanceAttrSSHKey, computeInstance.SSHKey); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.SecurityGroupIDs != nil {
		securityGroupIDs := make([]string, len(*computeInstance.SecurityGroupIDs))
		for i, id := range *computeInstance.SecurityGroupIDs {
			securityGroupIDs[i] = id
		}
		if err := d.Set(resComputeInstanceAttrSecurityGroupIDs, securityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resComputeInstanceAttrState, computeInstance.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resComputeInstanceAttrTemplateID, computeInstance.TemplateID); err != nil {
		return diag.FromErr(err)
	}

	rdns, err := client.GetInstanceReverseDNS(
		ctx,
		d.Get(resComputeInstanceAttrZone).(string),
		*computeInstance.ID,
	)
	if err != nil && !errors.Is(err, exoapi.ErrNotFound) {
		return diag.Errorf("unable to retrieve instance reverse-dns: %s", err)
	}
	if err := d.Set(resComputeInstanceAttrReverseDNS, strings.TrimSuffix(rdns, ".")); err != nil {
		return diag.FromErr(err)
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		d.Get(resComputeInstanceAttrZone).(string),
		*computeInstance.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("unable to retrieve instance type: %s", err)
	}
	if err := d.Set(resComputeInstanceAttrType, fmt.Sprintf(
		"%s.%s",
		strings.ToLower(*instanceType.Family),
		strings.ToLower(*instanceType.Size),
	)); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.UserData != nil {
		userData, err := decodeUserData(*computeInstance.UserData)
		if err != nil {
			return diag.Errorf("unable to decode user data: %s", err)
		}
		if err := d.Set(resComputeInstanceAttrUserData, userData); err != nil {
			return diag.FromErr(err)
		}
	}

	// Connection info for the `ssh` remote-exec provisioner
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": computeInstance.PublicIPAddress.String(),
	})

	return nil
}
