package exoscale

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"sort"
	"strings"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceComputeInstances() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			dsComputeInstanceAttrZone: {
				Type:     schema.TypeString,
				Required: true,
			},
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						dsComputeInstanceAttrCreatedAt: {
							Type:     schema.TypeString,
							Computed: true,
						},
						dsComputeInstanceAttrID: {
							Type:     schema.TypeString,
							Computed: true,
						},
						dsComputeInstanceAttrIPv6Address: {
							Type:     schema.TypeString,
							Computed: true,
						},
						dsComputeInstanceAttrLabels: {
							Type:     schema.TypeMap,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						dsComputeInstanceAttrName: {
							Type:     schema.TypeString,
							Computed: true,
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
					},
				},
			},
		},

		ReadContext: dataSourceComputeInstancesRead,
	}
}

func dataSourceComputeInstancesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceIDString(d, "exoscale_compute_instances"))

	zone := d.Get(dsComputeInstanceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	instances, err := client.ListInstances(
		ctx,
		zone,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	data := make([]interface{}, 0, len(instances))
	ids := make([]string, 0, len(instances))
	instanceTypes := map[string]string{}

	for _, computeInstance := range instances {
		// we use ID to generate a resource ID, we cannot list instances without ID.
		if computeInstance.ID == nil {
			continue
		}

		ids = append(ids, *computeInstance.ID)
		instanceData := map[string]interface{}{}

		instanceData[dsComputeInstanceAttrID] = computeInstance.ID
		instanceData[dsComputeInstanceAttrState] = computeInstance.State
		instanceData[dsComputeInstanceAttrSSHKey] = computeInstance.SSHKey
		instanceData[dsComputeInstanceAttrTemplateID] = computeInstance.TemplateID
		instanceData[dsComputeInstanceAttrName] = computeInstance.Name

		if computeInstance.CreatedAt != nil {
			instanceData[dsComputeInstanceAttrCreatedAt] = computeInstance.CreatedAt.String()
		}
		if computeInstance.IPv6Address != nil {
			instanceData[dsComputeInstanceAttrIPv6Address] = computeInstance.IPv6Address.String()
		}
		if computeInstance.PublicIPAddress != nil {
			instanceData[dsComputeInstanceAttrPublicIPAddress] = computeInstance.PublicIPAddress.String()
		}
		if computeInstance.PrivateNetworkIDs != nil {
			instanceData[dsComputeInstanceAttrPrivateNetworkIDs] = *computeInstance.PrivateNetworkIDs
		}
		if computeInstance.SecurityGroupIDs != nil {
			instanceData[dsComputeInstanceAttrSecurityGroupIDs] = *computeInstance.SecurityGroupIDs
		}
		if computeInstance.Labels != nil {
			instanceData[dsComputeInstanceAttrLabels] = *computeInstance.Labels
		}

		if computeInstance.InstanceTypeID != nil {
			tid := *computeInstance.InstanceTypeID
			if _, ok := instanceTypes[tid]; !ok {
				instanceType, err := client.GetInstanceType(
					ctx,
					zone,
					tid,
				)
				if err != nil {
					return diag.Errorf("unable to retrieve instance type: %s", err)
				}
				instanceTypes[tid] = fmt.Sprintf(
					"%s.%s",
					strings.ToLower(*instanceType.Family),
					strings.ToLower(*instanceType.Size),
				)
			}

			instanceData[dsComputeInstanceAttrType] = instanceTypes[tid]
		}

		data = append(data, instanceData)
	}

	err = d.Set("instances", data)
	if err != nil {
		return diag.FromErr(err)
	}

	// by sorting instance IDs we can generate the same resource ID regardless of the order in which
	// API returns instances in thelist.
	sort.Strings(ids)

	d.SetId(fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, "")))))

	log.Printf("[DEBUG] %s: read finished successfully", resourceIDString(d, "exoscale_compute_instances"))

	return nil
}
