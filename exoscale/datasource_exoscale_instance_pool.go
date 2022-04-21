package exoscale

import (
	"context"
	"fmt"
	"log"
	"strings"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsInstancePoolAttrAffinityGroupIDs = "affinity_group_ids"
	dsInstancePoolAttrDeployTargetID   = "deploy_target_id"
	dsInstancePoolAttrDescription      = "description"
	dsInstancePoolAttrDiskSize         = "disk_size"
	dsInstancePoolAttrElasticIPIDs     = "elastic_ip_ids"
	dsInstancePoolAttrInstancePrefix   = "instance_prefix"
	dsInstancePoolAttrInstanceType     = "instance_type"
	dsInstancePoolAttrIPv6             = "ipv6"
	dsInstancePoolAttrKeyPair          = "key_pair"
	dsInstancePoolAttrLabels           = "labels"
	dsInstancePoolAttrID               = "id"
	dsInstancePoolAttrName             = "name"
	dsInstancePoolAttrNetworkIDs       = "network_ids"
	dsInstancePoolAttrSecurityGroupIDs = "security_group_ids"
	dsInstancePoolAttrSize             = "size"
	dsInstancePoolAttrState            = "state"
	dsInstancePoolAttrTemplateID       = "template_id"
	dsInstancePoolAttrUserData         = "user_data"
	dsInstancePoolAttrVirtualMachines  = "virtual_machines"
	dsInstancePoolAttrZone             = "zone"
)

func dataSourceInstancePool() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
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
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsInstancePoolAttrID},
			},
			dsInstancePoolAttrID: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsInstancePoolAttrName},
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
			dsInstancePoolAttrVirtualMachines: {
				Type:     schema.TypeSet,
				Computed: true,
				Set:      schema.HashString,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			dsInstancePoolAttrZone: {
				Type:     schema.TypeString,
				Required: true,
			},
		},

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

	if instancePool.AntiAffinityGroupIDs != nil {
		antiAffinityGroupIDs := make([]string, len(*instancePool.AntiAffinityGroupIDs))
		for i, id := range *instancePool.AntiAffinityGroupIDs {
			antiAffinityGroupIDs[i] = id
		}
		if err := d.Set(dsInstancePoolAttrAffinityGroupIDs, antiAffinityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(
		dsInstancePoolAttrDeployTargetID,
		defaultString(instancePool.DeployTargetID, ""),
	); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsInstancePoolAttrDescription, defaultString(instancePool.Description, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsInstancePoolAttrDiskSize, *instancePool.DiskSize); err != nil {
		return diag.FromErr(err)
	}

	if instancePool.ElasticIPIDs != nil {
		elasticIPIDs := make([]string, len(*instancePool.ElasticIPIDs))
		for i, id := range *instancePool.ElasticIPIDs {
			elasticIPIDs[i] = id
		}
		if err := d.Set(dsInstancePoolAttrElasticIPIDs, elasticIPIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(dsInstancePoolAttrInstancePrefix, defaultString(instancePool.InstancePrefix, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsInstancePoolAttrIPv6, defaultBool(instancePool.IPv6Enabled, false)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsInstancePoolAttrKeyPair, instancePool.SSHKey); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsInstancePoolAttrLabels, instancePool.Labels); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsInstancePoolAttrName, instancePool.Name); err != nil {
		return diag.FromErr(err)
	}

	if instancePool.PrivateNetworkIDs != nil {
		privateNetworkIDs := make([]string, len(*instancePool.PrivateNetworkIDs))
		for i, id := range *instancePool.PrivateNetworkIDs {
			privateNetworkIDs[i] = id
		}
		if err := d.Set(dsInstancePoolAttrNetworkIDs, privateNetworkIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if instancePool.SecurityGroupIDs != nil {
		securityGroupIDs := make([]string, len(*instancePool.SecurityGroupIDs))
		for i, id := range *instancePool.SecurityGroupIDs {
			securityGroupIDs[i] = id
		}
		if err := d.Set(dsInstancePoolAttrSecurityGroupIDs, securityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		d.Get(dsInstancePoolAttrZone).(string),
		*instancePool.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("error retrieving instance type: %s", err)
	}
	if err := d.Set(dsInstancePoolAttrInstanceType, fmt.Sprintf(
		"%s.%s",
		strings.ToLower(*instanceType.Family),
		strings.ToLower(*instanceType.Size),
	)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsInstancePoolAttrSize, instancePool.Size); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsInstancePoolAttrState, instancePool.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsInstancePoolAttrTemplateID, instancePool.TemplateID); err != nil {
		return diag.FromErr(err)
	}

	if instancePool.UserData != nil {
		userData, err := decodeUserData(*instancePool.UserData)
		if err != nil {
			return diag.Errorf("error decoding user data: %s", err)
		}
		if err := d.Set(dsInstancePoolAttrUserData, userData); err != nil {
			return diag.FromErr(err)
		}
	}

	if instancePool.InstanceIDs != nil {
		instanceIDs := make([]string, len(*instancePool.InstanceIDs))
		for i, id := range *instancePool.InstanceIDs {
			instanceIDs[i] = id
		}
		if err := d.Set(dsInstancePoolAttrVirtualMachines, instanceIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceInstancePoolIDString(d))

	return nil
}
