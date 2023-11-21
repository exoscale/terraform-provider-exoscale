package instance

import (
	"context"
	"errors"
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

// DataSourceSchema returns a schema for a single Compute instance data source.
func DataSourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		AttrAntiAffinityGroupIDs: {
			Description: "The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrCreatedAt: {
			Description: "The compute instance creation date.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrDeployTargetID: {
			Description: "A deploy target ID.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrDiskSize: {
			Description: "The instance disk size (GiB).",
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
		AttrID: {
			Description: "The compute instance ID to match (conflicts with `name`).",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrIPv6: {
			Description: "Whether IPv6 is enabled on the instance.",
			Type:        schema.TypeBool,
			Computed:    true,
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
			Computed:    true,
		},
		AttrManagerID: {
			Description: "The instance manager ID, if any.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrManagerType: {
			Description: "The instance manager type (instance pool, SKS node pool, etc.), if any.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrName: {
			Description: "The instance name to match (conflicts with `id`).",
			Type:        schema.TypeString,
			Optional:    true,
		},
		AttrPrivateNetworkIDs: {
			Description: "The list of attached [exoscale_private_network](../resources/private_network.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrPublicIPAddress: {
			Description: "The instance (main network interface) IPv4 address.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrReverseDNS: {
			Description: "Domain name for reverse DNS record.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrSSHKey: {
			Description: "The [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the instance.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrSecurityGroupIDs: {
			Description: "The list of attached [exoscale_security_group](../resources/security_group.md) (IDs).",
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		AttrState: {
			Description: "The instance state.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrTemplateID: {
			Description: "The instance [exoscale_template](./template.md) ID.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrType: {
			Description: "The instance type.",
			Type:        schema.TypeString,
			Computed:    true,
		},
		AttrUserData: {
			Description: "The instance [cloud-init](http://cloudinit.readthedocs.io/en/latest/) configuration.",
			Type:        schema.TypeString,
			Computed:    true,
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
		Description: `Fetch Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/) data.

Corresponding resource: [exoscale_compute_instance](../resources/compute_instance.md).`,
		Schema: func() map[string]*schema.Schema {
			schema := DataSourceSchema()

			// adding context-aware schema settings here so getDataSourceComputeInstanceSchema can be used elsewhere
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

	instance, err := client.FindInstance(
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

	d.SetId(*instance.ID)

	data, err := dsBuildData(instance)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		zone,
		*instance.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("unable to retrieve instance type: %s", err)
	}

	data[AttrType] = fmt.Sprintf(
		"%s.%s",
		strings.ToLower(*instanceType.Family),
		strings.ToLower(*instanceType.Size),
	)

	rdns, err := client.GetInstanceReverseDNS(ctx, zone, *instance.ID)
	if err != nil && !errors.Is(err, exoapi.ErrNotFound) {
		return diag.Errorf("unable to retrieve instance reverse-dns: %s", err)
	}
	data[AttrReverseDNS] = strings.TrimSuffix(rdns, ".")

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
func dsBuildData(instance *exo.Instance) (map[string]interface{}, error) {
	data := map[string]interface{}{}

	data[AttrDeployTargetID] = instance.DeployTargetID
	data[AttrDiskSize] = instance.DiskSize
	data[AttrID] = instance.ID
	data[AttrName] = instance.Name
	data[AttrSSHKey] = instance.SSHKey
	data[AttrState] = instance.State
	data[AttrTemplateID] = instance.TemplateID
	data[AttrZone] = instance.Zone

	data[AttrIPv6] = utils.DefaultBool(instance.IPv6Enabled, false)

	if instance.ElasticIPIDs != nil {
		data[AttrElasticIPIDs] = *instance.ElasticIPIDs
	}
	if instance.AntiAffinityGroupIDs != nil {
		data[AttrAntiAffinityGroupIDs] = *instance.AntiAffinityGroupIDs
	}
	if instance.Labels != nil {
		data[AttrLabels] = *instance.Labels
	}
	if instance.PrivateNetworkIDs != nil {
		data[AttrPrivateNetworkIDs] = *instance.PrivateNetworkIDs
	}
	if instance.SecurityGroupIDs != nil {
		data[AttrSecurityGroupIDs] = *instance.SecurityGroupIDs
	}

	if instance.Manager != nil {
		data[AttrManagerID] = instance.Manager.ID
		data[AttrManagerType] = instance.Manager.Type
	}

	if instance.CreatedAt != nil {
		data[AttrCreatedAt] = instance.CreatedAt.String()
	}

	if instance.IPv6Address != nil {
		data[AttrIPv6Address] = instance.IPv6Address.String()
	}

	if instance.PublicIPAddress != nil {
		data[AttrPublicIPAddress] = instance.PublicIPAddress.String()
	}

	if instance.UserData != nil {
		userData, err := utils.DecodeUserData(*instance.UserData)
		if err != nil {
			return nil, fmt.Errorf("unable to decode user data: %w", err)
		}
		data[AttrUserData] = userData
	}

	return data, nil
}
