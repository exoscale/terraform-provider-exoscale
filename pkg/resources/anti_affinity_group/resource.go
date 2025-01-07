package anti_affinity_group

import (
	"context"
	"errors"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/utils"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
)

func Resource() *schema.Resource {
	return &schema.Resource{
		Description: `Manage Exoscale [Anti-Affinity Groups](https://community.exoscale.com/documentation/compute/anti-affinity-groups/).

Corresponding data source: [exoscale_anti_affinity_group](../data-sources/anti_affinity_group.md).`,
		Schema: map[string]*schema.Schema{
			AttrDescription: {
				Description: "A free-form text describing the group.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			AttrName: {
				Description: "The anti-affinity group name.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
		},

		CreateContext: rCreate,
		ReadContext:   rRead,
		DeleteContext: rDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(config.DefaultTimeout),
			Read:   schema.DefaultTimeout(config.DefaultTimeout),
			Delete: schema.DefaultTimeout(config.DefaultTimeout),
		},
	}
}

func rCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	zone := config.DefaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(config.GetEnvironment(meta), zone))
	defer cancel()

	client, err := config.GetClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := client.CreateAntiAffinityGroup(ctx, zone, &egoscale.AntiAffinityGroup{
		Name:        utils.NonEmptyStringPtr(d.Get(AttrName).(string)),
		Description: utils.NonEmptyStringPtr(d.Get(AttrDescription).(string)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*res.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return rRead(ctx, d, meta)
}

func rRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	zone := config.DefaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(config.GetEnvironment(meta), zone))
	defer cancel()

	client, err := config.GetClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := client.GetAntiAffinityGroup(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return diag.FromErr(rApply(ctx, d, res))
}

func rDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	zone := config.DefaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(config.GetEnvironment(meta), zone))
	defer cancel()

	client, err := config.GetClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	if err := client.DeleteAntiAffinityGroup(ctx, zone, &egoscale.AntiAffinityGroup{ID: &id}); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return nil
}

func rApply(
	_ context.Context,
	d *schema.ResourceData,
	res *egoscale.AntiAffinityGroup,
) error {
	if err := d.Set(AttrName, *res.Name); err != nil {
		return err
	}

	if err := d.Set(AttrDescription, utils.DefaultString(res.Description, "")); err != nil {
		return err
	}

	return nil
}
