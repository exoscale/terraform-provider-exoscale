package exoscale

import (
	"context"
	"fmt"
	"log"
	"strings"

	exo "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
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
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		dsInstancePoolAttrDeployTargetID: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsInstancePoolAttrDescription: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsInstancePoolAttrDiskSize: {
			Type:     schema.TypeInt,
			Computed: true,
		},
		dsInstancePoolAttrElasticIPIDs: {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		dsInstancePoolAttrInstancePrefix: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsInstancePoolAttrInstanceType: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsInstancePoolAttrIPv6: {
			Type:     schema.TypeBool,
			Computed: true,
		},
		dsInstancePoolAttrKeyPair: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsInstancePoolAttrLabels: {
			Type:     schema.TypeMap,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
		},
		dsInstancePoolAttrName: {
			Type:     schema.TypeString,
			Optional: true,
		},
		dsInstancePoolAttrID: {
			Type:     schema.TypeString,
			Optional: true,
		},
		dsInstancePoolAttrNetworkIDs: {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		dsInstancePoolAttrSecurityGroupIDs: {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		dsInstancePoolAttrSize: {
			Type:     schema.TypeInt,
			Computed: true,
		},
		dsInstancePoolAttrState: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsInstancePoolAttrTemplateID: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsInstancePoolAttrUserData: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsInstancePoolAttrInstances: {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					dsInstancePoolAttrInstanceID: {
						Type:     schema.TypeString,
						Optional: true,
					},
					dsInstancePoolAttrInstanceIPv6Address: {
						Type:     schema.TypeString,
						Computed: true,
					},
					dsInstancePoolAttrInstanceName: {
						Type:     schema.TypeString,
						Optional: true,
					},
					dsInstancePoolAttrInstancePublicIPAddress: {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
		dsInstancePoolAttrZone: {
			Type:     schema.TypeString,
			Required: true,
		},
	}
}

func dataSourceInstancePool() *schema.Resource {
	return &schema.Resource{
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
	log.Printf("[DEBUG] %s: beginning read", resourceInstancePoolIDString(d))

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

	log.Printf("[DEBUG] %s: read finished successfully", resourceInstancePoolIDString(d))

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
