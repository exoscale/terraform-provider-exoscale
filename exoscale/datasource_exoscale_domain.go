package exoscale

import (
	"context"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDomain() *schema.Resource {
	return &schema.Resource{
		Description: `Fetch Exoscale [DNS](https://community.exoscale.com/product/networking/dns/) Domains data.

Corresponding resource: [exoscale_domain](../resources/domain.md).`,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The DNS domain name to match.",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
		ReadContext: dataSourceDomainRead,
	}
}

func dataSourceDomainRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": general.ResourceIDString(d, "exoscale_domain"),
	})

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return diag.FromErr(err)
	}

	domainName := d.Get("name").(string)

	domains, err := client.ListDNSDomains(ctx)
	if err != nil {
		return diag.Errorf("error retrieving domain list: %s", err)
	}

	domain, err := domains.FindDNSDomain(domainName)
	if err != nil {
		return diag.Errorf("error retrieving domain: %s", err)
	}

	d.SetId(domain.ID.String())

	err = d.Set("name", domain.UnicodeName)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": general.ResourceIDString(d, "exoscale_domain"),
	})

	return nil
}
