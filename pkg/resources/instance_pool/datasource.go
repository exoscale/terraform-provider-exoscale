package instance_pool

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	exo "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

// DataSourceSchema returns a schema for a single instance pool data source.
func DataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		AttrAffinityGroupIDs: {
			Description: "The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrDeployTargetID: {
			Description: "The deploy target ID.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrDescription: {
			Description: "The instance pool description.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrDiskSize: {
			Description: "The managed instances disk size.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		AttrElasticIPIDs: {
			Description: "The list of attached [exoscale_elastic_ip](../resources/elastic_ip.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrInstancePrefix: {
			Description: "The string used to prefix the managed instances name.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrInstanceType: {
			Description: "The managed instances type.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrIPv6: {
			Description: "Whether IPv6 is enabled on managed instances.",
			Type:        schema.TypeBool,
			Computed:    true,
		},
		AttrKeyPair: {
			Description: "The [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the managed instances.",
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
			Description: "The pool name to match (conflicts with `id`).",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrID: {
			Description: "The instance pool ID to match (conflicts with `name`).",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrNetworkIDs: {
			Description: "The list of attached [exoscale_private_network](../resources/private_network.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrSecurityGroupIDs: {
			Description: "The list of attached [exoscale_security_group](../resources/security_group.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrSize: {
			Description: "The number managed instances.",
			Type:        schema.TypeInt,
			Computed:    true,
		},
		AttrState: {
			Description: "The pool state.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrTemplateID: {
			Description: "The managed instances [exoscale_compute_template](./compute_template.md) ID.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrUserData: {
			Description: "[cloud-init](http://cloudinit.readthedocs.io/en/latest/) configuration.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrInstances: {
			Description: "The list of managed instances. Structure is documented below.",
			Type:        schema.TypeSet,
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					AttrInstanceID: {
						Description: "The compute instance ID.",
						Type:        schema.TypeString,
						Optional:    true,
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
		},
	}
}

func DataSource() *schema.Resource {
	return &schema.Resource{
		Description: `Fetch Exoscale [Instance Pools](https://community.exoscale.com/documentation/compute/instance-pools/) data.

Corresponding resource: [exoscale_instance_pool](../resources/instance_pool.md).`,
		Schema: func() map[string]*schema.Schema {
			schema := DataSourceSchema()

			// adding context-aware schema settings here so getDataSourceInstancePoolSchema can be used in list method
			schema[AttrID].ConflictsWith = []string{AttrName}
			schema[AttrName].ConflictsWith = []string{AttrID}
			return schema
		}(),
		ReadContext: dsRead,
	}
}

func dsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	id, byID := d.GetOk(AttrID)
	name, byName := d.GetOk(AttrName)
	if !byID && !byName {
		return diag.Errorf(
			"either %s or %s must be specified",
			AttrName,
			AttrID,
		)
	}

	pool, err := client.FindInstancePool(
		ctx,
		zone, func() string {
			if byID {
				return id.(string)
			}
			return name.(string)
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*pool.ID)

	data, err := dsBuildData(pool)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		zone,
		*pool.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("error retrieving instance type: %s", err)
	}

	data[AttrInstanceType] = fmt.Sprintf(
		"%s.%s",
		strings.ToLower(*instanceType.Family),
		strings.ToLower(*instanceType.Size),
	)

	if pool.InstanceIDs != nil {
		instancesData := make([]interface{}, len(*pool.InstanceIDs))
		for i, id := range *pool.InstanceIDs {
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
				AttrInstanceID:              id,
				AttrInstanceIPv6Address:     ipv6,
				AttrInstanceName:            instance.Name,
				AttrInstancePublicIPAddress: publicIp,
			}
		}

		data[AttrInstances] = instancesData
	}

	for key, value := range data {
		err := d.Set(key, value)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return nil
}

// dsBuildData builds terraform data object from egoscale API struct.
func dsBuildData(pool *exo.InstancePool) (map[string]interface{}, error) {
	data := map[string]interface{}{}

	data[AttrDeployTargetID] = pool.DeployTargetID
	data[AttrDescription] = utils.DefaultString(pool.Description, "")
	data[AttrDiskSize] = pool.DiskSize
	data[AttrID] = pool.ID
	data[AttrInstancePrefix] = utils.DefaultString(pool.InstancePrefix, "")
	data[AttrIPv6] = utils.DefaultBool(pool.IPv6Enabled, false)
	data[AttrKeyPair] = pool.SSHKey
	data[AttrLabels] = pool.Labels
	data[AttrName] = pool.Name
	data[AttrSize] = pool.Size
	data[AttrState] = pool.State
	data[AttrTemplateID] = pool.TemplateID
	data[AttrZone] = pool.Zone

	if pool.AntiAffinityGroupIDs != nil {
		data[AttrAffinityGroupIDs] = *pool.AntiAffinityGroupIDs
	}

	if pool.ElasticIPIDs != nil {
		data[AttrElasticIPIDs] = *pool.ElasticIPIDs
	}

	if pool.PrivateNetworkIDs != nil {
		data[AttrNetworkIDs] = *pool.PrivateNetworkIDs
	}

	if pool.SecurityGroupIDs != nil {
		data[AttrSecurityGroupIDs] = *pool.SecurityGroupIDs
	}

	if pool.UserData != nil {
		userData, err := utils.DecodeUserData(*pool.UserData)
		if err != nil {
			return nil, fmt.Errorf("error decoding user data: %w", err)
		}
		data[AttrUserData] = userData
	}

	return data, nil
}
