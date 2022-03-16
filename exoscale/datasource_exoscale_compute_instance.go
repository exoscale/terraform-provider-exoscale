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
	dsComputeInstanceAttrSSHKey               = "ssh_key"
	dsComputeInstanceAttrSecurityGroupIDs     = "security_group_ids"
	dsComputeInstanceAttrState                = "state"
	dsComputeInstanceAttrTemplateID           = "template_id"
	dsComputeInstanceAttrType                 = "type"
	dsComputeInstanceAttrUserData             = "user_data"
	dsComputeInstanceAttrZone                 = "zone"
)

func dataSourceComputeInstance() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
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
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsComputeInstanceAttrName},
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
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsComputeInstanceAttrID},
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
		},

		ReadContext: dataSourceComputeInstanceRead,
	}
}

func dataSourceComputeInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceComputeInstanceIDString(d))

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

	if computeInstance.AntiAffinityGroupIDs != nil {
		antiAffinityGroupIDs := make([]string, len(*computeInstance.AntiAffinityGroupIDs))
		for i, id := range *computeInstance.AntiAffinityGroupIDs {
			antiAffinityGroupIDs[i] = id
		}
		if err := d.Set(dsComputeInstanceAttrAntiAffinityGroupIDs, antiAffinityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(dsComputeInstanceAttrCreatedAt, computeInstance.CreatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(
		dsComputeInstanceAttrDeployTargetID,
		defaultString(computeInstance.DeployTargetID, ""),
	); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsComputeInstanceAttrDiskSize, *computeInstance.DiskSize); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.ElasticIPIDs != nil {
		elasticIPIDs := make([]string, len(*computeInstance.ElasticIPIDs))
		for i, id := range *computeInstance.ElasticIPIDs {
			elasticIPIDs[i] = id
		}
		if err := d.Set(dsComputeInstanceAttrElasticIPIDs, elasticIPIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(dsComputeInstanceAttrIPv6, defaultBool(computeInstance.IPv6Enabled, false)); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.IPv6Address != nil {
		if err := d.Set(dsComputeInstanceAttrIPv6Address, computeInstance.IPv6Address.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(dsComputeInstanceAttrLabels, computeInstance.Labels); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.Manager != nil {
		if err := d.Set(dsComputeInstanceAttrManagerID, computeInstance.Manager.ID); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(dsComputeInstanceAttrManagerType, computeInstance.Manager.Type); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(dsComputeInstanceAttrName, *computeInstance.Name); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.PrivateNetworkIDs != nil {
		privateNetworkIDs := make([]string, len(*computeInstance.PrivateNetworkIDs))
		for i, id := range *computeInstance.PrivateNetworkIDs {
			privateNetworkIDs[i] = id
		}
		if err := d.Set(dsComputeInstanceAttrPrivateNetworkIDs, privateNetworkIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if computeInstance.PublicIPAddress != nil {
		if err := d.Set(dsComputeInstanceAttrPublicIPAddress, computeInstance.PublicIPAddress.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(dsComputeInstanceAttrSSHKey, computeInstance.SSHKey); err != nil {
		return diag.FromErr(err)
	}

	if computeInstance.SecurityGroupIDs != nil {
		securityGroupIDs := make([]string, len(*computeInstance.SecurityGroupIDs))
		for i, id := range *computeInstance.SecurityGroupIDs {
			securityGroupIDs[i] = id
		}
		if err := d.Set(dsComputeInstanceAttrSecurityGroupIDs, securityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(dsComputeInstanceAttrState, computeInstance.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsComputeInstanceAttrTemplateID, computeInstance.TemplateID); err != nil {
		return diag.FromErr(err)
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		d.Get(dsComputeInstanceAttrZone).(string),
		*computeInstance.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("unable to retrieve instance type: %s", err)
	}
	if err := d.Set(dsComputeInstanceAttrType, fmt.Sprintf(
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
		if err := d.Set(dsComputeInstanceAttrUserData, userData); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceComputeInstanceIDString(d))

	return nil
}
