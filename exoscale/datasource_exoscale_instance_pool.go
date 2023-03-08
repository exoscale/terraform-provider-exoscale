package exoscale

import (
	"context"
	"fmt"
	"strings"

	exo "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsInstancePoolAttrAffinityGroupIDs        = "affinity_group_ids"
	dsInstancePoolAttrDeployTargetID          = "deploy_target_id"
	dsInstancePoolAttrDescription             = "description"
	dsInstancePoolAttrDiskSize                = "disk_size"
	dsInstancePoolAttrElasticIPIDs            = "elastic_ip_ids"
	dsInstancePoolAttrInstancePrefix          = "instance_prefix"
	dsInstancePoolAttrInstanceType            = "instance_type"
	dsInstancePoolAttrIPv6                    = "ipv6"
	dsInstancePoolAttrKeyPair                 = "key_pair"
	dsInstancePoolAttrLabels                  = "labels"
	dsInstancePoolAttrID                      = "id"
	dsInstancePoolAttrName                    = "name"
	dsInstancePoolAttrNetworkIDs              = "network_ids"
	dsInstancePoolAttrSecurityGroupIDs        = "security_group_ids"
	dsInstancePoolAttrSize                    = "size"
	dsInstancePoolAttrState                   = "state"
	dsInstancePoolAttrTemplateID              = "template_id"
	dsInstancePoolAttrUserData                = "user_data"
	dsInstancePoolAttrInstances               = "instances"
	dsInstancePoolAttrInstanceID              = "id"
	dsInstancePoolAttrInstanceIPv6Address     = "ipv6_address"
	dsInstancePoolAttrInstanceName            = "name"
	dsInstancePoolAttrInstancePublicIPAddress = "public_ip_address"
	dsInstancePoolAttrZone                    = "zone"
)

// getDataSourceInstancePoolSchema returns a schema for a single instance pool data source.
func getDataSourceInstancePoolSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		dsInstancePoolAttrAffinityGroupIDs: {
			Description: "The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		dsInstancePoolAttrDeployTargetID: {
			Description: "The deploy target ID.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		dsInstancePoolAttrDescription: {
			Description: "The instance pool description.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		dsInstancePoolAttrDiskSize: {
			Description: "The managed instances disk size.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		dsInstancePoolAttrElasticIPIDs: {
			Description: "The list of attached [exoscale_elastic_ip](../resources/elastic_ip.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		dsInstancePoolAttrInstancePrefix: {
			Description: "The string used to prefix the managed instances name.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		dsInstancePoolAttrInstanceType: {
			Description: "The managed instances type.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		dsInstancePoolAttrIPv6: {
			Description: "Whether IPv6 is enabled on managed instances.",
			Type:        schema.TypeBool,
			Computed:    true,
		},
		dsInstancePoolAttrKeyPair: {
			Description: "The [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the managed instances.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		dsInstancePoolAttrLabels: {
			Description: "A map of key/value labels.",
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
		},
		dsInstancePoolAttrName: {
			Description: "The pool name to match (conflicts with `id`).",
			Type:        schema.TypeString,
			Optional:    true,
		},
		dsInstancePoolAttrID: {
			Description: "The instance pool ID to match (conflicts with `name`).",
			Type:        schema.TypeString,
			Optional:    true,
		},
		dsInstancePoolAttrNetworkIDs: {
			Description: "The list of attached [exoscale_private_network](../resources/private_network.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		dsInstancePoolAttrSecurityGroupIDs: {
			Description: "The list of attached [exoscale_security_group](../resources/security_group.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		dsInstancePoolAttrSize: {
			Description: "The number managed instances.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		dsInstancePoolAttrState: {
			Description: "The pool state.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		dsInstancePoolAttrTemplateID: {
			Description: "The managed instances [exoscale_compute_template](./compute_template.md) ID.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		dsInstancePoolAttrUserData: {
			Description: "[cloud-init](http://cloudinit.readthedocs.io/en/latest/) configuration.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		dsInstancePoolAttrInstances: {
			Description: "The list of managed instances. Structure is documented below.",
			Type:        schema.TypeSet,
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					dsInstancePoolAttrInstanceID: {
						Description: "The compute instance ID.",
						Type:        schema.TypeString,
						Optional:    true,
					},
					dsInstancePoolAttrInstanceIPv6Address: {
						Description: "The instance (main network interface) IPv6 address.",
						Type:        schema.TypeString,
						Computed:    true,
					},
					dsInstancePoolAttrInstanceName: {
						Description: "The instance name.",
						Type:        schema.TypeString,
						Optional:    true,
					},
					dsInstancePoolAttrInstancePublicIPAddress: {
						Description: "The instance (main network interface) IPv4 address.",
						Type:        schema.TypeString,
						Computed:    true,
					},
				},
			},
		},
		dsInstancePoolAttrZone: {
			Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
			Type:        schema.TypeString,
			Required:    true,
		},
	}
}

func dataSourceInstancePool() *schema.Resource {
	return &schema.Resource{
		Description: `Fetch Exoscale [Instance Pools](https://community.exoscale.com/documentation/compute/instance-pools/) data.

Corresponding resource: [exoscale_instance_pool](../resources/instance_pool.md).`,
		Schema: func() map[string]*schema.Schema {
			schema := getDataSourceInstancePoolSchema()

			// adding context-aware schema settings here so getDataSourceInstancePoolSchema can be used in list method
			schema[dsInstancePoolAttrID].ConflictsWith = []string{dsInstancePoolAttrName}
			schema[dsInstancePoolAttrName].ConflictsWith = []string{dsInstancePoolAttrID}
			return schema
		}(),
		ReadContext: dataSourceInstancePoolRead,
	}
}

func dataSourceInstancePoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceInstancePoolIDString(d),
	})

	zone := d.Get(dsInstancePoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	instancePoolID, byInstancePoolID := d.GetOk(dsInstancePoolAttrID)
	instancePoolName, byInstancePoolName := d.GetOk(dsInstancePoolAttrName)
	if !byInstancePoolID && !byInstancePoolName {
		return diag.Errorf(
			"either %s or %s must be specified",
			dsInstancePoolAttrName,
			dsInstancePoolAttrID,
		)
	}

	instancePool, err := client.FindInstancePool(
		ctx,
		zone, func() string {
			if byInstancePoolID {
				return instancePoolID.(string)
			}
			return instancePoolName.(string)
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*instancePool.ID)

	data, err := dataSourceInstancePoolBuildData(instancePool)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		zone,
		*instancePool.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("error retrieving instance type: %s", err)
	}

	data[dsInstancePoolAttrInstanceType] = fmt.Sprintf(
		"%s.%s",
		strings.ToLower(*instanceType.Family),
		strings.ToLower(*instanceType.Size),
	)

	if instancePool.InstanceIDs != nil {
		instancesData := make([]interface{}, len(*instancePool.InstanceIDs))
		for i, id := range *instancePool.InstanceIDs {
			instance, err := client.GetInstance(ctx, zone, id)
			if err != nil {
				return diag.FromErr(err)
			}

			var ipv6, publicIp string
			if instance.IPv6Address != nil {
				ipv6 = instance.IPv6Address.String()
			}
			if instance.PublicIPAddress != nil {
				publicIp = instance.PublicIPAddress.String()
			}

			instancesData[i] = map[string]interface{}{
				dsInstancePoolAttrInstanceID:              id,
				dsInstancePoolAttrInstanceIPv6Address:     ipv6,
				dsInstancePoolAttrInstanceName:            instance.Name,
				dsInstancePoolAttrInstancePublicIPAddress: publicIp,
			}
		}

		data[dsInstancePoolAttrInstances] = instancesData
	}

	for key, value := range data {
		err := d.Set(key, value)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceInstancePoolIDString(d),
	})

	return nil
}

// dataSourceInstancePoolBuildData builds terraform data object from egoscale API struct.
func dataSourceInstancePoolBuildData(pool *exo.InstancePool) (map[string]interface{}, error) {
	data := map[string]interface{}{}

	data[dsInstancePoolAttrDeployTargetID] = pool.DeployTargetID
	data[dsInstancePoolAttrDescription] = defaultString(pool.Description, "")
	data[dsInstancePoolAttrDiskSize] = pool.DiskSize
	data[dsInstancePoolAttrID] = pool.ID
	data[dsInstancePoolAttrInstancePrefix] = defaultString(pool.InstancePrefix, "")
	data[dsInstancePoolAttrIPv6] = defaultBool(pool.IPv6Enabled, false)
	data[dsInstancePoolAttrKeyPair] = pool.SSHKey
	data[dsInstancePoolAttrLabels] = pool.Labels
	data[dsInstancePoolAttrName] = pool.Name
	data[dsInstancePoolAttrSize] = pool.Size
	data[dsInstancePoolAttrState] = pool.State
	data[dsInstancePoolAttrTemplateID] = pool.TemplateID
	data[dsInstancePoolAttrZone] = pool.Zone

	if pool.AntiAffinityGroupIDs != nil {
		data[dsInstancePoolAttrAffinityGroupIDs] = *pool.AntiAffinityGroupIDs
	}

	if pool.ElasticIPIDs != nil {
		data[dsInstancePoolAttrElasticIPIDs] = *pool.ElasticIPIDs
	}

	if pool.PrivateNetworkIDs != nil {
		data[dsInstancePoolAttrNetworkIDs] = *pool.PrivateNetworkIDs
	}

	if pool.SecurityGroupIDs != nil {
		data[dsInstancePoolAttrSecurityGroupIDs] = *pool.SecurityGroupIDs
	}

	if pool.UserData != nil {
		userData, err := decodeUserData(*pool.UserData)
		if err != nil {
			return nil, fmt.Errorf("error decoding user data: %w", err)
		}
		data[dsInstancePoolAttrUserData] = userData
	}

	return data, nil
}
