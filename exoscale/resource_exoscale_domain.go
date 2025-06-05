package exoscale

import (
	"context"
	"errors"
	"fmt"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDomainIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_domain")
}

func resourceDomain() *schema.Resource {
	return &schema.Resource{
		Description: "Manage Exoscale DNS Domains.",
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The DNS domain name.",
			},
			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Deprecated:  "Not used, will be removed in the future",
				Description: "A security token that can be used as an alternative way to manage DNS domains via the Exoscale API.",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Deprecated:  "Not used, will be removed in the future",
				Description: "The domain state.",
			},
			"auto_renew": {
				Type:        schema.TypeBool,
				Computed:    true,
				Deprecated:  "Not used, will be removed in the future",
				Description: "Whether the DNS domain has automatic renewal enabled (boolean).",
			},
			"expires_on": {
				Type:        schema.TypeString,
				Computed:    true,
				Deprecated:  "Not used, will be removed in the future",
				Description: "The domain expiration date, if known.",
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
			Create: schema.DefaultTimeout(config.DefaultTimeout),
			Read:   schema.DefaultTimeout(config.DefaultTimeout),
			Delete: schema.DefaultTimeout(config.DefaultTimeout),
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

	client := getClient(meta)

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
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return diag.FromErr(err)
	}

	domainName := d.Get("name").(string)
	o, err := client.CreateDNSDomain(ctx, v3.CreateDNSDomainRequest{UnicodeName: domainName})
	if err != nil {
		return diag.Errorf("unable to create domain: %s", err)
	}

	// bug in the api: the spec advertises we return a domain when we return an operation
	_, err = client.Wait(ctx, &v3.Operation{
		ID: o.ID,
	}, v3.OperationStateSuccess)
	if err != nil {
		return diag.Errorf("unable to create domain: %s", err)
	}

	domains, err := client.ListDNSDomains(ctx)
	if err != nil {
		return diag.Errorf("unable to retrieve domain after creation: %s", err)
	}
	domain, err := domains.FindDNSDomain(domainName)
	if err != nil {
		return diag.Errorf("unable to retrieve domain after creation: %s", err)
	}

	d.SetId(domain.ID.String())

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	err = resourceDomainApply(d, &domain)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	return nil
}

func resourceDomainExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return false, err
	}

	_, err = client.GetDNSDomain(ctx, v3.UUID(d.Id()))
	if err != nil {
		if errors.Is(err, v3.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceDomainRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return diag.FromErr(err)
	}

	domain, err := client.GetDNSDomain(ctx, v3.UUID(d.Id()))
	if err != nil {
		return diag.Errorf("error retrieving domain: %s", err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	err = resourceDomainApply(d, domain)
	if err != nil {
		return diag.Errorf("%s", err)
	}

	return nil
}

func resourceDomainDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return diag.FromErr(err)
	}

	domain, err := client.GetDNSDomain(ctx, v3.UUID(d.Id()))
	if err != nil {
		return diag.Errorf("error retrieving domain: %s", err)
	}

	op, err := client.DeleteDNSDomain(ctx, domain.ID)
	if err != nil {
		return diag.Errorf("error deleting domain: %s", err)
	}
	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		return diag.Errorf("error deleting domain: %s", err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	return nil
}

func resourceDomainImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return nil, err
	}

	domain, err := client.GetDNSDomain(ctx, v3.UUID(d.Id()))
	if err != nil {
		return nil, err
	}

	if err := resourceDomainApply(d, domain); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func resourceDomainApply(d *schema.ResourceData, domain *v3.DNSDomain) error {
	d.SetId(domain.ID.String())
	if err := d.Set("name", domain.UnicodeName); err != nil {
		return err
	}

	return nil
}
