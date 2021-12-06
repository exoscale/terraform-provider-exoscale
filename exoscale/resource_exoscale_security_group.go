package exoscale

import (
	"context"
	"errors"
	"log"
	"strings"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	resSecurityGroupAttrDescription     = "description"
	resSecurityGroupAttrExternalSources = "external_sources"
	resSecurityGroupAttrName            = "name"
)

func resourceSecurityGroupIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_security_group")
}

func resourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			resSecurityGroupAttrDescription: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			resSecurityGroupAttrExternalSources: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsCIDRNetwork(0, 128),
				},
			},
			resSecurityGroupAttrName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},

		CreateContext: resourceSecurityGroupCreate,
		ReadContext:   resourceSecurityGroupRead,
		UpdateContext: resourceSecurityGroupUpdate,
		DeleteContext: resourceSecurityGroupDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceSecurityGroupImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSecurityGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceSecurityGroupIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup, err := client.CreateSecurityGroup(ctx, zone, &egoscale.SecurityGroup{
		Name:        nonEmptyStringPtr(d.Get(resSecurityGroupAttrName).(string)),
		Description: nonEmptyStringPtr(d.Get(resSecurityGroupAttrDescription).(string)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if externalSourcesSet, ok := d.GetOk(resSecurityGroupAttrExternalSources); ok {
		for _, cidr := range externalSourcesSet.(*schema.Set).List() {
			if err := client.AddExternalSourceToSecurityGroup(ctx, zone, securityGroup, cidr.(string)); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	d.SetId(*securityGroup.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceSecurityGroupIDString(d))

	return resourceSecurityGroupRead(ctx, d, meta)
}

func resourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceSecurityGroupIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup, err := client.GetSecurityGroup(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSecurityGroupIDString(d))

	return diag.FromErr(resourceSecurityGroupApply(ctx, d, securityGroup))
}

func resourceSecurityGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning update", resourceSecurityGroupIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup, err := client.GetSecurityGroup(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(resSecurityGroupAttrExternalSources) {
		o, n := d.GetChange(resSecurityGroupAttrExternalSources)
		old := o.(*schema.Set)
		cur := n.(*schema.Set)

		added := cur.Difference(old)
		if added.Len() > 0 {
			for _, cidr := range added.List() {
				if err := client.AddExternalSourceToSecurityGroup(
					ctx,
					zone,
					securityGroup,
					cidr.(string),
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}

		removed := old.Difference(cur)
		if removed.Len() > 0 {
			for _, cidr := range removed.List() {
				if err := client.RemoveExternalSourceFromSecurityGroup(
					ctx,
					zone,
					securityGroup,
					cidr.(string),
				); err != nil {
					return diag.FromErr(err)
				}
			}
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceSecurityGroupIDString(d))

	return resourceSecurityGroupRead(ctx, d, meta)
}

func resourceSecurityGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceSecurityGroupIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroupID := d.Id()
	if err := client.DeleteSecurityGroup(ctx, zone, &egoscale.SecurityGroup{ID: &securityGroupID}); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSecurityGroupIDString(d))

	return nil
}

func resourceSecurityGroupImport(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
) ([]*schema.ResourceData, error) {
	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup, err := client.FindSecurityGroup(ctx, zone, d.Id())
	if err != nil {
		return nil, err
	}

	if err := resourceSecurityGroupApply(ctx, d, securityGroup); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 0)
	resources = append(resources, d)

	for _, securityGroupRule := range securityGroup.Rules {
		resource := resourceSecurityGroupRule()
		rd := resource.Data(nil)
		rd.SetType("exoscale_security_group_rule")

		if err := rd.Set("type", strings.ToUpper(*securityGroupRule.FlowDirection)); err != nil {
			return nil, err
		}

		if err := resourceSecurityGroupRuleApply(ctx, rd, meta, securityGroup, securityGroupRule); err != nil {
			return nil, err
		}

		resources = append(resources, rd)
	}

	return resources, nil
}

func resourceSecurityGroupApply(
	_ context.Context,
	d *schema.ResourceData,
	securityGroup *egoscale.SecurityGroup,
) error {
	if err := d.Set(resSecurityGroupAttrName, *securityGroup.Name); err != nil {
		return err
	}

	if securityGroup.ExternalSources != nil {
		if err := d.Set(resSecurityGroupAttrExternalSources, *securityGroup.ExternalSources); err != nil {
			return err
		}
	}

	if err := d.Set(resSecurityGroupAttrDescription, defaultString(securityGroup.Description, "")); err != nil {
		return err
	}

	return nil
}
