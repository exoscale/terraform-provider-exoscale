package exoscale

import (
	"context"
	"fmt"
	"log"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	resIAMAccessKeyAttrKey            = "key"
	resIAMAccessKeyAttrName           = "name"
	resIAMAccessKeyAttrOperations     = "operations"
	resIAMAccessKeyAttrResources      = "resources"
	resIAMAccessKeyAttrSecret         = "secret"
	resIAMAccessKeyAttrTags           = "tags"
	resIAMAccessKeyAttrTagsOperations = "tags_operations"
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
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					o, n := d.GetChange(resIAMAccessKeyAttrOperations)
					if o == nil || n == nil {
						return false
					}

					oldOperations := schemaSetToStringArray(o.(*schema.Set))
					newOperations := schemaSetToStringArray(n.(*schema.Set))
					diff := map[string]bool{}

					// diff = oldOperations - newOperations
					for _, oldOperation := range oldOperations {
						diff[oldOperation] = true
					}

					for _, newOperation := range newOperations {
						if diff[newOperation] {
							diff[newOperation] = false
						} else {
							return false
						}
					}

					// ignore to-be-removed operations if the operation belongs to at least one tag
					if tagsOperations, ok := d.Get(resIAMAccessKeyAttrTagsOperations).(*schema.Set); ok {
						for _, tagOperation := range schemaSetToStringArray(tagsOperations) {
							if diff[tagOperation] {
								diff[tagOperation] = false
							} else {
								return false
							}
						}
					}

					// can't suppress diff if an operation is neither:
					// - matching a user-defined operations
					// - matching a set of operations matching at least a required tag
					for _, element := range diff {
						if element {
							return false
						}
					}

					return true
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
			resIAMAccessKeyAttrTagsOperations: {
				Type:     schema.TypeSet,
				Computed: true,
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

	accessKey, err := client.GetIAMAccessKey(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	operations, err := client.ListIAMAccessKeyOperations(ctx, zone)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceIAMAccessKeyIDString(d))

	return diag.FromErr(resourceIAMAccessKeyApply(ctx, d, *accessKey, operations))
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
	accessKey egoscale.IAMAccessKey,
	operations []*egoscale.IAMAccessKeyOperation,
) error {
	if err := d.Set(resIAMAccessKeyAttrName, accessKey.Name); err != nil {
		return err
	}

	if accessKey.Operations != nil {
		if err := d.Set(resIAMAccessKeyAttrOperations, accessKey.Operations); err != nil {
			return err
		}
	}

	if accessKey.Resources != nil {
		resources := []string{}
		for _, r := range *accessKey.Resources {
			resources = append(resources, fmt.Sprintf("%s/%s:%s", r.Domain, r.ResourceType, r.ResourceName))
		}

		if err := d.Set(resIAMAccessKeyAttrResources, resources); err != nil {
			return err
		}
	}

	tagsOperations := map[string][]string{}
	for _, operation := range operations {
		for _, tag := range operation.Tags {
			tagsOperations[tag] = append(tagsOperations[tag], operation.Name)
		}
	}

	if accessKey.Tags != nil {
		operationsFromTags := []string{}
		for _, requestedTag := range *accessKey.Tags {
			operationsFromTags = append(operationsFromTags, tagsOperations[requestedTag]...)
		}

		operationsFromTags = unique(operationsFromTags)

		if err := d.Set(resIAMAccessKeyAttrTags, accessKey.Tags); err != nil {
			return err
		}

		if err := d.Set(resIAMAccessKeyAttrTagsOperations, operationsFromTags); err != nil {
			return err
		}
	}

	return nil
}
