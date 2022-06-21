package exoscale

import (
	"context"
	"log"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	resIAMAccessKeyAttrKey         = "key"
	resIAMAccessKeyAttrName        = "name"
	resIAMAccessKeyAttrOperations  = "operations"
	resIAMAccessKeyAttrResources   = "resources"
	resIAMAccessKeyAttrSecret      = "secret"
	resIAMAccessKeyAttrTags        = "tags"
)

func resourceIAMAccessKeyIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_iam_access_key")
}

func resourceIAMAccessKey() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			resIAMAccessKeyAttrKey: {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			resIAMAccessKeyAttrName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			resIAMAccessKeyAttrOperations: {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			resIAMAccessKeyAttrResources: {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			resIAMAccessKeyAttrSecret: {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			resIAMAccessKeyAttrTags: {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},

		CreateContext: resourceIAMAccessKeyCreate,
		ReadContext:   resourceIAMAccessKeyRead,
		DeleteContext: resourceIAMAccessKeyDelete,

		Importer: &schema.ResourceImporter{
			StateContext: zonedStateContextFunc,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceIAMAccessKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceIAMAccessKeyIDString(d))

	name := d.Get(resIAMAccessKeyAttrName).(string)
	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	opts := make([]egoscale.CreateIAMAccessKeyOpt, 0)

	if v, ok := d.Get(resIAMAccessKeyAttrOperations).(*schema.Set); ok {
		operations := schemaSetToStringArray(v)
		if len(operations) > 0 {
			opts = append(opts, egoscale.CreateIAMAccessKeyWithOperations(operations))
		}
	}

	if v, ok := d.Get(resIAMAccessKeyAttrResources).(*schema.Set); ok {
		resources := schemaSetToStringArray(v)
		if len(resources) > 0 {
			parsedResources := make([]egoscale.IAMAccessKeyResource, len(resources))
			for i, resourceDescription := range resources {
				parsedResource, err := parseIAMAccessKeyResource(resourceDescription)
				if err != nil {
					return diag.FromErr(err)
				}
				parsedResources[i] = *parsedResource
			}

			opts = append(opts, egoscale.CreateIAMAccessKeyWithResources(parsedResources))
		}
	}

	if v, ok := d.Get(resIAMAccessKeyAttrTags).(*schema.Set); ok {
		tags := schemaSetToStringArray(v)
		if len(tags) > 0 {
			opts = append(opts, egoscale.CreateIAMAccessKeyWithTags(tags))
		}
	}

	iamAccessKey, err := client.CreateIAMAccessKey(ctx, zone, name, opts...)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*iamAccessKey.Key)

	if err := d.Set(resIAMAccessKeyAttrKey, *iamAccessKey.Key); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resIAMAccessKeyAttrSecret, iamAccessKey.Secret); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: create finished successfully", resourceIAMAccessKeyIDString(d))

	return resourceIAMAccessKeyRead(ctx, d, meta)
}

func resourceIAMAccessKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceIAMAccessKeyIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	iamAccessKey, err := client.GetIAMAccessKey(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceIAMAccessKeyIDString(d))

	return diag.FromErr(resourceIAMAccessKeyApply(ctx, d, iamAccessKey))
}

func resourceIAMAccessKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceIAMAccessKeyIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	key := d.Id()
	if err := client.RevokeIAMAccessKey(ctx, zone, &egoscale.IAMAccessKey{Key: &key}); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceIAMAccessKeyIDString(d))
	return nil
}

func resourceIAMAccessKeyApply(
	_ context.Context,
	d *schema.ResourceData,
	iamAccessKey *egoscale.IAMAccessKey,
) error {
	if err := d.Set(resIAMAccessKeyAttrName, iamAccessKey.Name); err != nil {
		return err
	}

	if iamAccessKey.Operations != nil {
		if err := d.Set(resIAMAccessKeyAttrOperations, *iamAccessKey.Operations); err != nil {
			return err
		}
	}

	if iamAccessKey.Resources != nil {
		if err := d.Set(resIAMAccessKeyAttrResources, *iamAccessKey.Resources); err != nil {
			return err
		}
	}

	if iamAccessKey.Tags != nil {
		if err := d.Set(resIAMAccessKeyAttrTags, *iamAccessKey.Tags); err != nil {
			return err
		}
	}

	return nil
}
