package exoscale

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceDomain() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the Domain",
				Required:    true,
			},
		},
		Read: dataSourceDomainRead,
	}
}

func dataSourceDomainRead(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetDNSClient(meta)

	domainName := d.Get("name")

	domain, err := client.GetDomain(ctx, domainName.(string))
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(domain.ID, 10))

	return d.Set("name", domain.Name)
}
