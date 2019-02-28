package exoscale

import (
	"context"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func domainResource() *schema.Resource {
	return &schema.Resource{
		Create: createDomain,
		Exists: existsDomain,
		Read:   readDomain,
		Delete: deleteDomain,

		Importer: &schema.ResourceImporter{
			State: importDomain,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"token": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"auto_renew": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"expires_on": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func createDomain(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetDNSClient(meta)

	domain, err := client.CreateDomain(ctx, d.Get("name").(string))
	if err != nil {
		return err
	}

	d.SetId(domain.Name)
	return readDomain(d, meta)
}

func existsDomain(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetDNSClient(meta)

	_, err := client.GetDomain(ctx, d.Id())
	if err != nil {
		if _, ok := err.(*egoscale.DNSErrorResponse); ok { // nolint: gosimple
			return false, nil
		}
	}

	return err == nil, err
}

func readDomain(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetDNSClient(meta)

	domain, err := client.GetDomain(ctx, d.Id())
	if err != nil {
		return err
	}

	return applyDomain(d, *domain)
}

func deleteDomain(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetDNSClient(meta)

	err := client.DeleteDomain(ctx, d.Id())
	if err != nil {
		d.SetId("")
	}

	return err
}

func importDomain(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetDNSClient(meta)
	domain, err := client.GetDomain(ctx, d.Id())
	if err != nil {
		return nil, err
	}

	if err := applyDomain(d, *domain); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 0, 1)
	resources = append(resources, d)

	records, err := client.GetRecords(ctx, d.Id())
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		// Ignore the default NS and SOA entries
		if record.RecordType == "NS" || record.RecordType == "SOA" {
			continue
		}
		resource := domainRecordResource()
		d := resource.Data(nil)
		d.SetType("exoscale_domain_record")
		if err := d.Set("domain", domain.Name); err != nil {
			return nil, err
		}

		if err := applyRecord(d, record); err != nil {
			continue
		}

		resources = append(resources, d)
	}

	return resources, nil
}

func applyDomain(d *schema.ResourceData, domain egoscale.DNSDomain) error {
	d.SetId(domain.Name)
	if err := d.Set("name", domain.Name); err != nil {
		return err
	}
	if err := d.Set("state", domain.State); err != nil {
		return err
	}
	if err := d.Set("token", domain.Token); err != nil {
		return err
	}
	if err := d.Set("auto_renew", domain.AutoRenew); err != nil {
		return err
	}
	if err := d.Set("expires_on", domain.ExpiresOn); err != nil {
		return err
	}

	return nil
}
