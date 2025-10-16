package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const (
	resSecurityGroupAttrDescription     = "description"
	resSecurityGroupAttrExternalSources = "external_sources"
	resSecurityGroupAttrName            = "name"
)

func resourceSecurityGroupIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_security_group")
}

func resourceSecurityGroupSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		resSecurityGroupAttrDescription: {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "A free-form text describing the group.",
		},
		resSecurityGroupAttrExternalSources: {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.IsCIDRNetwork(0, 128),
			},
			Description: "A list of external network sources, in [CIDR](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notation) notation.",
		},
		resSecurityGroupAttrName: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			// Migration to OpenAPI-v2: name is normalized to lowercase even if it was defined
			// with uppercase letters with provider < v0.31.
			// Let's ignore case of the name, assuming that anyway, it will be converted to lowercase.
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return strings.EqualFold(old, new)
			},
			Description: "The security group name.",
		},
	}
}

func resourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Schema:        resourceSecurityGroupSchema(),
		Description:   "Manage Exoscale Security Groups.",
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceSecurityGroupResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceSecurityGroupStateUpgradeV0,
				Version: 0,
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
			Create: schema.DefaultTimeout(config.DefaultTimeout),
			Read:   schema.DefaultTimeout(config.DefaultTimeout),
			Delete: schema.DefaultTimeout(config.DefaultTimeout),
		},
	}
}

func resourceSecurityGroupResourceV0() *schema.Resource {
	return &schema.Resource{
		Schema: resourceSecurityGroupSchema(),
	}
}

func resourceSecurityGroupStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
	tflog.Debug(ctx, "beginning migration")

	// OpenAPI-v2 backend returns lowercase names, let's fix the state content
	if name, ok := rawState["name"].(string); ok {
		rawState["name"] = strings.ToLower(name)
		tflog.Debug(ctx, fmt.Sprintf("enforce lowercase on name: %+v", rawState["name"]))
	} else {
		return nil, fmt.Errorf("unable to get resource name during migration")
	}

	tflog.Debug(ctx, "done migration")
	return rawState, nil
}

func resourceSecurityGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceSecurityGroupIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

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

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceSecurityGroupIDString(d),
	})

	return resourceSecurityGroupRead(ctx, d, meta)
}

func resourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceSecurityGroupIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	securityGroup, err := client.GetSecurityGroup(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceSecurityGroupIDString(d),
	})

	return diag.FromErr(resourceSecurityGroupApply(ctx, d, securityGroup))
}

func resourceSecurityGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourceSecurityGroupIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	securityGroup, err := client.GetSecurityGroup(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(resSecurityGroupAttrExternalSources) {
		o, n := d.GetChange(resSecurityGroupAttrExternalSources)
		old := o.(*schema.Set)
		cur := n.(*schema.Set)

		if added := cur.Difference(old); added.Len() > 0 {
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

		if removed := old.Difference(cur); removed.Len() > 0 {
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

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceSecurityGroupIDString(d),
	})

	return resourceSecurityGroupRead(ctx, d, meta)
}

func matchSecurityGroup(inst *v3.Instance, id v3.UUID) bool {
	for _, sg := range inst.SecurityGroups {
		if sg.ID == id {
			return true
		}
	}
	return false
}

func detachSecurityGroup(ctx context.Context, client *v3.Client, id v3.UUID, inst *v3.Instance) (*v3.Operation, error) {
	return client.DetachInstanceFromSecurityGroup(
		ctx,
		id,
		v3.DetachInstanceFromSecurityGroupRequest{
			Instance: &v3.Instance{ID: inst.ID},
		},
	)
}

func resourceSecurityGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	zone := defaultZone
	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	defaultClientV3, err := config.GetClientV3(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	client, err := utils.SwitchClientZone(ctx, defaultClientV3, v3.ZoneName(zone))
	if err != nil {
		return diag.FromErr(err)
	}

	if diags := detachMatchingResource(ctx, client, "SecurityGroup", v3.UUID(d.Id()), matchSecurityGroup, detachSecurityGroup); diags != nil {
		return diags
	}

	op, err := client.DeleteSecurityGroup(ctx, v3.UUID(d.Id()))
	if err != nil && !errors.Is(err, v3.ErrNotFound) {
		return diag.FromErr(err)
	}
	if op != nil {
		if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
			return diag.FromErr(err)
		}
	}

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

	client := getClient(meta)

	securityGroup, err := client.FindSecurityGroup(ctx, zone, d.Id())
	if err != nil {
		return nil, err
	}

	if err := resourceSecurityGroupApply(ctx, d, securityGroup); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
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
