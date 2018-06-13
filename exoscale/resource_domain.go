package exoscale

import (
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
	}
}

func createDomain(d *schema.ResourceData, meta interface{}) error {
	client := GetDNSClient(meta)

	domain, err := client.CreateDomain(d.Get("name").(string))
	if err != nil {
		return err
	}

	return applyDomain(d, *domain)
}

func existsDomain(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetDNSClient(meta)

	_, err := client.GetDomain(d.Id())

	return err == nil, err
}

func readDomain(d *schema.ResourceData, meta interface{}) error {
	client := GetDNSClient(meta)

	domain, err := client.GetDomain(d.Id())
	if err != nil {
		d.SetId("")
		return err
	}

	return applyDomain(d, *domain)
}

func deleteDomain(d *schema.ResourceData, meta interface{}) error {
	client := GetDNSClient(meta)

	err := client.DeleteDomain(d.Id())
	if err != nil {
		d.SetId("")
	}

	return err
}

func importDomain(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := GetDNSClient(meta)
	domain, err := client.GetDomain(d.Id())
	if err != nil {
		return nil, err
	}

	if err := applyDomain(d, *domain); err != nil {
		return nil, err
	}

	records, err := client.GetRecords(d.Id())
	if err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 0, 1)
	resources = append(resources, d)

	for _, record := range records {
		// Ignore the default NS and SOA entries
		if record.RecordType == "NS" || record.RecordType == "SOA" {
			continue
		}
		resource := domainRecordResource()
		d := resource.Data(nil)
		d.SetType("exoscale_domain_record")
		d.Set("domain", domain.Name)
		if err := applyRecord(d, record); err != nil {
			continue
		}

		resources = append(resources, d)
	}

	return resources, nil
}

func applyDomain(d *schema.ResourceData, domain egoscale.DNSDomain) error {
	d.SetId(domain.Name)
	d.Set("name", domain.Name)
	d.Set("state", domain.State)
	d.Set("token", domain.Token)
	d.Set("auto_renew", domain.AutoRenew)
	d.Set("expires_on", domain.ExpiresOn)

	return nil
}
