package exoscale

import (
	"context"
	"errors"
	"fmt"
	"log"

	exo "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDomainIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_domain")
}

func resourceDomain() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"token": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Not used, will be removed in the future",
			},
			"state": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Not used, will be removed in the future",
			},
			"auto_renew": {
				Type:       schema.TypeBool,
				Computed:   true,
				Deprecated: "Not used, will be removed in the future",
			},
			"expires_on": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Not used, will be removed in the future",
			},
		},

		CreateContext: resourceDomainCreate,
		ReadContext:   resourceDomainRead,
		DeleteContext: resourceDomainDelete,
		Exists:        resourceDomainExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceDomainImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},

		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceDomainV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceDomainStateUpgradeV0,
				Version: 0,
			},
		},
	}
}

func resourceDomainV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type: schema.TypeString,
			},
		},
	}
}

func resourceDomainStateUpgradeV0(
	ctx context.Context,
	rawState map[string]interface{},
	meta interface{},
) (map[string]interface{}, error) {
	client := GetComputeClient(meta)

	name := rawState["id"].(string)
	domains, err := client.ListDNSDomains(ctx, defaultZone)
	if err != nil {
		return nil, fmt.Errorf("error retrieving domain list: %s", err)
	}

	for _, domain := range domains {
		if *domain.UnicodeName == name {
			rawState["id"] = *domain.ID
			return rawState, nil
		}
	}

	return nil, fmt.Errorf("domain not found: %q", name)
}

func resourceDomainCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceDomainIDString(d))

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	domainName := d.Get("name").(string)
	domain, err := client.CreateDNSDomain(ctx, defaultZone, &exo.DNSDomain{UnicodeName: &domainName})
	if err != nil {
		return diag.Errorf("unable to retrieve instance type: %s", err)
	}

	d.SetId(*domain.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceDomainIDString(d))

	err = resourceDomainApply(d, domain)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	return nil
}

func resourceDomainExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	_, err := client.GetDNSDomain(ctx, defaultZone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceDomainRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceDomainIDString(d))

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	domain, err := client.GetDNSDomain(ctx, defaultZone, d.Id())
	if err != nil {
		return diag.Errorf("error retrieving domain: %s", err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceDomainIDString(d))

	err = resourceDomainApply(d, domain)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	return nil
}

func resourceDomainDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceDomainIDString(d))

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	domain, err := client.GetDNSDomain(ctx, defaultZone, d.Id())
	if err != nil {
		return diag.Errorf("error retrieving domain: %s", err)
	}

	err = client.DeleteDNSDomain(ctx, defaultZone, domain)
	if err != nil {
		return diag.Errorf("error deleting domain: %s", err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceDomainIDString(d))

	return nil
}

func resourceDomainImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	domain, err := client.GetDNSDomain(ctx, defaultZone, d.Id())
	if err != nil {
		return nil, err
	}

	if err := resourceDomainApply(d, domain); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 0, 1)
	resources = append(resources, d)

	records, err := client.ListDNSDomainRecords(ctx, defaultZone, d.Id())
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		// Ignore the default NS and SOA entries
		if *record.Type == "NS" || *record.Type == "SOA" {
			continue
		}
		resource := resourceDomainRecord()
		data := resource.Data(nil)
		data.SetType("exoscale_domain_record")
		if err := data.Set("domain", d.Id()); err != nil {
			return nil, err
		}

		if err := resourceDomainRecordApply(data, *domain.UnicodeName, &record); err != nil {
			continue
		}

		resources = append(resources, data)
	}

	return resources, nil
}

func resourceDomainApply(d *schema.ResourceData, domain *exo.DNSDomain) error {
	d.SetId(*domain.ID)
	if err := d.Set("name", domain.UnicodeName); err != nil {
		return err
	}

	return nil
}
