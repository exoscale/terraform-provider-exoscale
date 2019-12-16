package exoscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func datasourceDomain() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the domain",
				Required:    true,
			},
		},
		Read: datasourceDomainRead,
	}
}

func datasourceDomainRead(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetDNSClient(meta)

	domainName := d.Get("name")

	domain, err := client.GetDomain(ctx, domainName.(string))
	if err != nil {
		return err
	}

	if err := d.Set("name", domain.Name); err != nil {
		return err
	}

	return nil
}
