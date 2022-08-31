package exoscale

import (
	"context"
	"errors"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	resAntiAffinityGroupAttrDescription = "description"
	resAntiAffinityGroupAttrName        = "name"
)

func resourceAntiAffinityGroupIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_anti_affinity_group")
}

func resourceAntiAffinityGroup() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			resAntiAffinityGroupAttrDescription: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			resAntiAffinityGroupAttrName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},

		CreateContext: resourceAntiAffinityGroupCreate,
		ReadContext:   resourceAntiAffinityGroupRead,
		DeleteContext: resourceAntiAffinityGroupDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceAntiAffinityGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceAntiAffinityGroupIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	antiAffinityGroup, err := client.CreateAntiAffinityGroup(ctx, zone, &egoscale.AntiAffinityGroup{
		Name:        nonEmptyStringPtr(d.Get(resAntiAffinityGroupAttrName).(string)),
		Description: nonEmptyStringPtr(d.Get(resAntiAffinityGroupAttrDescription).(string)),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*antiAffinityGroup.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceAntiAffinityGroupIDString(d),
	})

	return resourceAntiAffinityGroupRead(ctx, d, meta)
}

func resourceAntiAffinityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceAntiAffinityGroupIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	antiAffinityGroup, err := client.GetAntiAffinityGroup(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceAntiAffinityGroupIDString(d),
	})

	return diag.FromErr(resourceAntiAffinityGroupApply(ctx, d, antiAffinityGroup))
}

func resourceAntiAffinityGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceAntiAffinityGroupIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	antiAffinityGroupID := d.Id()
	if err := client.DeleteAntiAffinityGroup(ctx, zone, &egoscale.AntiAffinityGroup{ID: &antiAffinityGroupID}); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceAntiAffinityGroupIDString(d),
	})

	return nil
}

func resourceAntiAffinityGroupApply(
	_ context.Context,
	d *schema.ResourceData,
	antiAffinityGroup *egoscale.AntiAffinityGroup,
) error {
	if err := d.Set(resAntiAffinityGroupAttrName, *antiAffinityGroup.Name); err != nil {
		return err
	}

	if err := d.Set(resAntiAffinityGroupAttrDescription, defaultString(antiAffinityGroup.Description, "")); err != nil {
		return err
	}

	return nil
}
