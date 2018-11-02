package exoscale

import (
	"fmt"
	"strconv"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func domainRecordResource() *schema.Resource {
	return &schema.Resource{
		Create: createRecord,
		Read:   readRecord,
		Exists: existsRecord,
		Update: updateRecord,
		Delete: deleteRecord,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"domain": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"record_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"A", "AAAA", "ALIAS", "CNAME", "HINFO", "MX", "NAPTR",
					"NS", "POOL", "SPF", "SRV", "SSHFP", "TXT", "URL",
				}, true),
			},
			"content": {
				Type:     schema.TypeString,
				Required: true,
			},
			"ttl": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"prio": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createRecord(d *schema.ResourceData, meta interface{}) error {
	client := GetDNSClient(meta)

	record, err := client.CreateRecord(d.Get("domain").(string), egoscale.DNSRecord{
		Name:       d.Get("name").(string),
		Content:    d.Get("content").(string),
		RecordType: d.Get("record_type").(string),
		TTL:        d.Get("ttl").(int),
		Prio:       d.Get("prio").(int),
	})

	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(record.ID, 10))
	return readRecord(d, meta)
}

func existsRecord(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetDNSClient(meta)

	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	domain := d.Get("domain").(string)

	if domain != "" {
		record, err := client.GetRecord(domain, id)
		if err != nil {
			if _, ok := err.(*egoscale.DNSErrorResponse); !ok {
				return false, err
			}
		}
		return record != nil, nil
	}

	domains, err := client.GetDomains()
	if err != nil {
		return false, err
	}

	for _, domain := range domains {
		record, err := client.GetRecord(domain.Name, id)
		if err != nil {
			if _, ok := err.(*egoscale.DNSErrorResponse); !ok {
				return false, err
			}
		}

		if record != nil {
			return true, nil
		}
	}

	return false, nil
}

func readRecord(d *schema.ResourceData, meta interface{}) error {
	client := GetDNSClient(meta)

	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	domain := d.Get("domain").(string)

	if domain != "" {
		record, err := client.GetRecord(domain, id)
		if err != nil {
			return err
		}

		return applyRecord(d, *record)
	}

	domains, err := client.GetDomains()
	if err != nil {
		return err
	}

	for _, domain := range domains {
		record, err := client.GetRecord(domain.Name, id)
		if err != nil {
			return err
		}

		if record != nil {
			if err := d.Set("domain", domain.Name); err != nil {
				return err
			}
			return applyRecord(d, *record)
		}
	}

	return fmt.Errorf("domain record %s not found", d.Id())
}

func updateRecord(d *schema.ResourceData, meta interface{}) error {
	client := GetDNSClient(meta)

	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	record, err := client.UpdateRecord(d.Get("domain").(string), egoscale.UpdateDNSRecord{
		ID:         id,
		Name:       d.Get("name").(string),
		Content:    d.Get("content").(string),
		RecordType: d.Get("record_type").(string),
		TTL:        d.Get("ttl").(int),
		Prio:       d.Get("prio").(int),
	})

	if err != nil {
		return err
	}

	return applyRecord(d, *record)
}

func deleteRecord(d *schema.ResourceData, meta interface{}) error {
	client := GetDNSClient(meta)

	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	err := client.DeleteRecord(d.Get("domain").(string), id)
	if err != nil {
		d.SetId("")
	}

	return err
}

func applyRecord(d *schema.ResourceData, record egoscale.DNSRecord) error {
	d.SetId(strconv.FormatInt(record.ID, 10))
	if err := d.Set("name", record.Name); err != nil {
		return err
	}
	if err := d.Set("content", record.Content); err != nil {
		return err
	}
	if err := d.Set("record_type", record.RecordType); err != nil {
		return err
	}
	if err := d.Set("ttl", record.TTL); err != nil {
		return err
	}
	if err := d.Set("prio", record.Prio); err != nil {
		return err
	}

	domain := d.Get("domain").(string)
	if record.Name != "" {
		domain = fmt.Sprintf("%s.%s", record.Name, domain)
	}

	if err := d.Set("hostname", domain); err != nil {
		return nil
	}

	return nil
}
