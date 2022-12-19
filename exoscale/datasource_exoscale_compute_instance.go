package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strings"

	exo "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsComputeInstanceAttrAntiAffinityGroupIDs = "anti_affinity_group_ids"
	dsComputeInstanceAttrCreatedAt            = "created_at"
	dsComputeInstanceAttrDeployTargetID       = "deploy_target_id"
	dsComputeInstanceAttrDiskSize             = "disk_size"
	dsComputeInstanceAttrElasticIPIDs         = "elastic_ip_ids"
	dsComputeInstanceAttrID                   = "id"
	dsComputeInstanceAttrIPv6                 = "ipv6"
	dsComputeInstanceAttrIPv6Address          = "ipv6_address"
	dsComputeInstanceAttrLabels               = "labels"
	dsComputeInstanceAttrManagerID            = "manager_id"
	dsComputeInstanceAttrManagerType          = "manager_type"
	dsComputeInstanceAttrName                 = "name"
	dsComputeInstanceAttrPrivateNetworkIDs    = "private_network_ids"
	dsComputeInstanceAttrPublicIPAddress      = "public_ip_address"
	dsComputeInstanceAttrReverseDNS           = "reverse_dns"
	dsComputeInstanceAttrSSHKey               = "ssh_key"
	dsComputeInstanceAttrSecurityGroupIDs     = "security_group_ids"
	dsComputeInstanceAttrState                = "state"
	dsComputeInstanceAttrTemplateID           = "template_id"
	dsComputeInstanceAttrType                 = "type"
	dsComputeInstanceAttrUserData             = "user_data"
	dsComputeInstanceAttrZone                 = "zone"
)

// getDataSourceComputeInstanceSchema returns a schema for a single Compute instance data source.
func getDataSourceComputeInstanceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		dsComputeInstanceAttrAntiAffinityGroupIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		dsComputeInstanceAttrCreatedAt: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrDeployTargetID: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrDiskSize: {
			Type:     schema.TypeInt,
			Computed: true,
		},
		dsComputeInstanceAttrElasticIPIDs: {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		dsComputeInstanceAttrID: {
			Type:     schema.TypeString,
			Optional: true,
		},
		dsComputeInstanceAttrIPv6: {
			Type:     schema.TypeBool,
			Computed: true,
		},
		dsComputeInstanceAttrIPv6Address: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrLabels: {
			Type:     schema.TypeMap,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
		},
		dsComputeInstanceAttrManagerID: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrManagerType: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrName: {
			Type:     schema.TypeString,
			Optional: true,
		},
		dsComputeInstanceAttrPrivateNetworkIDs: {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		dsComputeInstanceAttrPublicIPAddress: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrReverseDNS: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrSSHKey: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrSecurityGroupIDs: {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		dsComputeInstanceAttrState: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrTemplateID: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrType: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrUserData: {
			Type:     schema.TypeString,
			Computed: true,
		},
		dsComputeInstanceAttrZone: {
			Type:     schema.TypeString,
			Required: true,
		},
	}
}

func dataSourceComputeInstance() *schema.Resource {
	return &schema.Resource{
		Schema: func() map[string]*schema.Schema {
			schema := getDataSourceComputeInstanceSchema()

			// adding context-aware schema settings here so getDataSourceComputeInstanceSchema can be used elsewhere
			schema[dsComputeInstanceAttrID].ConflictsWith = []string{dsComputeInstanceAttrName}
			schema[dsComputeInstanceAttrName].ConflictsWith = []string{dsComputeInstanceAttrID}
			return schema
		}(),
		ReadContext: dataSourceComputeInstanceRead,
	}
}

func dataSourceComputeInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	zone := d.Get(dsComputeInstanceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	computeInstanceID, byComputeInstanceID := d.GetOk(dsComputeInstanceAttrID)
	computeInstanceName, byComputeInstanceName := d.GetOk(dsComputeInstanceAttrName)
	if !byComputeInstanceID && !byComputeInstanceName {
		return diag.Errorf(
			"either %s or %s must be specified",
			dsComputeInstanceAttrName,
			dsComputeInstanceAttrID,
		)
	}

	computeInstance, err := client.FindInstance(
		ctx,
		zone, func() string {
			if byComputeInstanceID {
				return computeInstanceID.(string)
			}
			return computeInstanceName.(string)
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*computeInstance.ID)

	data, err := dataSourceComputeInstanceBuildData(computeInstance)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		zone,
		*computeInstance.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("unable to retrieve instance type: %s", err)
	}

	data[dsComputeInstanceAttrType] = fmt.Sprintf(
		"%s.%s",
		strings.ToLower(*instanceType.Family),
		strings.ToLower(*instanceType.Size),
	)

	rdns, err := client.GetInstanceReverseDNS(ctx, zone, *computeInstance.ID)
	if err != nil && !errors.Is(err, exoapi.ErrNotFound) {
		return diag.Errorf("unable to retrieve instance reverse-dns: %s", err)
	}
	data[dsComputeInstanceAttrReverseDNS] = rdns

	for key, value := range data {
		err := d.Set(key, value)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceComputeInstanceIDString(d),
	})

	return nil
}

// dataSourceComputeInstanceBuildData builds terraform data object from egoscale API struct.
func dataSourceComputeInstanceBuildData(instance *exo.Instance) (map[string]interface{}, error) {
	data := map[string]interface{}{}

	data[dsComputeInstanceAttrDeployTargetID] = instance.DeployTargetID
	data[dsComputeInstanceAttrDiskSize] = instance.DiskSize
	data[dsComputeInstanceAttrID] = instance.ID
	data[dsComputeInstanceAttrName] = instance.Name
	data[dsComputeInstanceAttrSSHKey] = instance.SSHKey
	data[dsComputeInstanceAttrState] = instance.State
	data[dsComputeInstanceAttrTemplateID] = instance.TemplateID
	data[dsComputeInstanceAttrZone] = instance.Zone

	data[dsComputeInstanceAttrIPv6] = defaultBool(instance.IPv6Enabled, false)

	if instance.ElasticIPIDs != nil {
		data[dsComputeInstanceAttrElasticIPIDs] = *instance.ElasticIPIDs
	}
	if instance.AntiAffinityGroupIDs != nil {
		data[dsComputeInstanceAttrAntiAffinityGroupIDs] = *instance.AntiAffinityGroupIDs
	}
	if instance.Labels != nil {
		data[dsComputeInstanceAttrLabels] = *instance.Labels
	}
	if instance.PrivateNetworkIDs != nil {
		data[dsComputeInstanceAttrPrivateNetworkIDs] = *instance.PrivateNetworkIDs
	}
	if instance.SecurityGroupIDs != nil {
		data[dsComputeInstanceAttrSecurityGroupIDs] = *instance.SecurityGroupIDs
	}

	if instance.Manager != nil {
		data[dsComputeInstanceAttrManagerID] = instance.Manager.ID
		data[dsComputeInstanceAttrManagerType] = instance.Manager.Type
	}

	if instance.CreatedAt != nil {
		data[dsComputeInstanceAttrCreatedAt] = instance.CreatedAt.String()
	}

	if instance.IPv6Address != nil {
		data[dsComputeInstanceAttrIPv6Address] = instance.IPv6Address.String()
	}

	if instance.PublicIPAddress != nil {
		data[dsComputeInstanceAttrPublicIPAddress] = instance.PublicIPAddress.String()
	}

	if instance.UserData != nil {
		userData, err := decodeUserData(*instance.UserData)
		if err != nil {
			return nil, fmt.Errorf("unable to decode user data: %w", err)
		}
		data[dsComputeInstanceAttrUserData] = userData
	}

	return data, nil
}
