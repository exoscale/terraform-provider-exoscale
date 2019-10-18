package exoscale

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

var supportedRecordTypes = []string{
	"A", "AAAA", "ALIAS", "CAA", "CNAME",
	"HINFO", "MX", "NAPTR", "NS", "POOL",
	"SPF", "SRV", "SSHFP", "TXT", "URL",
}

func resourceDomainRecordIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_domain_record")
}

func resourceDomainRecord() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"domain": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"record_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(supportedRecordTypes, true),
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
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

		Create: resourceDomainRecordCreate,
		Read:   resourceDomainRecordRead,
		Update: resourceDomainRecordUpdate,
		Delete: resourceDomainRecordDelete,
		Exists: resourceDomainRecordExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceDomainRecordCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceDomainRecordIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetDNSClient(meta)

	record, err := client.CreateRecord(ctx, d.Get("domain").(string), egoscale.DNSRecord{
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

	log.Printf("[DEBUG] %s: create finished successfully", resourceDomainRecordIDString(d))

	return resourceDomainRecordRead(d, meta)
}

func resourceDomainRecordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetDNSClient(meta)

	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	domain := d.Get("domain").(string)

	if domain != "" {
		record, err := client.GetRecord(ctx, domain, id)
		if err != nil {
			if _, ok := err.(*egoscale.DNSErrorResponse); !ok {
				return false, err
			}

			return true, err
		}

		return record != nil, nil
	}

	// If we reach this stage it means that we're in "import" mode, so we don't have the domain information yet.
	// We have to scroll each existing domain's records and try to find one matching the resource ID.
	log.Printf("[DEBUG] %s: import mode detected, trying to locate the record domain", resourceDomainRecordIDString(d))

	domains, err := client.GetDomains(ctx)
	if err != nil {
		return false, err
	}

	for _, domain := range domains {
		records, err := client.GetRecords(ctx, domain.Name)
		if err != nil {
			return false, err
		}

		for _, record := range records {
			if record.ID == id {
				log.Printf("[DEBUG] %s: found record domain: %s", resourceDomainRecordIDString(d), domain.Name)
				return true, nil
			}
		}
	}

	return false, nil
}

func resourceDomainRecordRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceDomainRecordIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetDNSClient(meta)

	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	domain := d.Get("domain").(string)

	if domain != "" {
		record, err := client.GetRecord(ctx, domain, id)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] %s: read finished successfully", resourceDomainRecordIDString(d))

		return resourceDomainRecordApply(d, *record)
	}

	// If we reach this stage it means that we're in "import" mode, so we don't have the domain information yet.
	// We have to scroll each existing domain's records and try to find one matching the resource ID.
	log.Printf("[DEBUG] %s: import mode detected, trying to locate the record domain", resourceDomainRecordIDString(d))

	domains, err := client.GetDomains(ctx)
	if err != nil {
		return err
	}

	for _, domain := range domains {
		records, err := client.GetRecords(ctx, domain.Name)
		if err != nil {
			return err
		}

		for _, record := range records {
			if record.ID == id {
				if err := d.Set("domain", domain.Name); err != nil {
					return err
				}

				log.Printf("[DEBUG] %s: read finished successfully", resourceDomainRecordIDString(d))

				return resourceDomainRecordApply(d, record)
			}
		}
	}

	return fmt.Errorf("domain record %s not found", d.Id())
}

func resourceDomainRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceDomainRecordIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetDNSClient(meta)

	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	record, err := client.UpdateRecord(ctx, d.Get("domain").(string), egoscale.UpdateDNSRecord{
		ID:      id,
		Name:    d.Get("name").(string),
		Content: d.Get("content").(string),
		TTL:     d.Get("ttl").(int),
		Prio:    d.Get("prio").(int),
	})

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceDomainRecordIDString(d))

	return resourceDomainRecordApply(d, *record) // FIXME: use resourceDomainRecordRead()
}

func resourceDomainRecordDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceDomainRecordIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetDNSClient(meta)

	id, _ := strconv.ParseInt(d.Id(), 10, 64)
	if err := client.DeleteRecord(ctx, d.Get("domain").(string), id); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceDomainRecordIDString(d))

	return nil
}

func resourceDomainRecordApply(d *schema.ResourceData, record egoscale.DNSRecord) error {
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
