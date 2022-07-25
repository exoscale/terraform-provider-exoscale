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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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

		CreateContext: resourceDomainRecordCreate,
		ReadContext:   resourceDomainRecordRead,
		UpdateContext: resourceDomainRecordUpdate,
		DeleteContext: resourceDomainRecordDelete,
		Exists:        resourceDomainRecordExists,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},

		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceDomainRecordV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceDomainRecordStateUpgradeV0,
				Version: 0,
			},
		},
	}
}

func resourceDomainRecordV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type: schema.TypeString,
			},
			"domain": {
				Type: schema.TypeString,
			},
			"record_type": {
				Type: schema.TypeString,
			},
			"name": {
				Type: schema.TypeString,
			},
			"content": {
				Type: schema.TypeString,
			},
		},
	}
}

func resourceDomainRecordStateUpgradeV0(
	ctx context.Context,
	rawState map[string]interface{},
	meta interface{},
) (map[string]interface{}, error) {
	client := GetComputeClient(meta)

	domainName := rawState["domain"].(string)
	domains, err := client.ListDNSDomains(ctx, defaultZone)
	if err != nil {
		return nil, fmt.Errorf("error retrieving domain list: %s", err)
	}

	for _, domain := range domains {
		if *domain.UnicodeName == domainName {
			rawState["domain"] = *domain.ID
			break
		}
	}

	records, err := client.ListDNSDomainRecords(ctx, defaultZone, rawState["domain"].(string))
	if err != nil {
		return nil, fmt.Errorf("error retrieving domain records: %q", err)
	}

	for _, record := range records {
		if *record.Type == rawState["record_type"].(string) &&
			*record.Name == rawState["name"].(string) &&
			*record.Content == rawState["content"] {
			rawState["id"] = *record.ID
			break
		}
	}

	return rawState, nil
}

func resourceDomainRecordCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceDomainRecordIDString(d))

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), defaultZone))
	defer cancel()

	client := GetComputeClient(meta)

	name := d.Get("name").(string)
	content := d.Get("content").(string)
	rtype := d.Get("record_type").(string)
	var ttl *int64
	if t := int64(d.Get("ttl").(int)); t > 0 {
		ttl = &t
	}
	var prio *int64
	if t := int64(d.Get("prio").(int)); t > 0 {
		prio = &t
	}
	record, err := client.CreateDNSDomainRecord(ctx, defaultZone, d.Get("domain").(string), &exo.DNSDomainRecord{
		Name:     &name,
		Content:  &content,
		Type:     &rtype,
		TTL:      ttl,
		Priority: prio,
	})
	if err != nil {
		return diag.Errorf("error creating domain record: %q", err)
	}

	d.SetId(*record.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceDomainRecordIDString(d))

	return resourceDomainRecordRead(ctx, d, meta)
}

func resourceDomainRecordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), defaultZone))
	defer cancel()

	client := GetComputeClient(meta)

	domainID := d.Get("domain").(string)

	if domainID != "" {
		_, err := client.GetDNSDomainRecord(ctx, defaultZone, domainID, d.Id())
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return false, nil
			}
			return false, err
		}

		return true, nil
	}

	// If we reach this stage it means that we're in "import" mode, so we don't have the domain information yet.
	// We have to scroll each existing domain's records and try to find one matching the resource ID.
	log.Printf("[DEBUG] %s: import mode detected, trying to locate the record domain", resourceDomainRecordIDString(d))

	domains, err := client.ListDNSDomains(ctx, defaultZone)
	if err != nil {
		return false, err
	}

	for _, domain := range domains {
		records, err := client.ListDNSDomainRecords(ctx, defaultZone, *domain.ID)
		if err != nil {
			return false, err
		}

		for _, record := range records {
			if *record.ID == d.Id() {
				log.Printf("[DEBUG] %s: found record domain: %s", resourceDomainRecordIDString(d), *domain.UnicodeName)
				return true, nil
			}
		}
	}

	return false, nil
}

func resourceDomainRecordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceDomainRecordIDString(d))

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), defaultZone))
	defer cancel()

	client := GetComputeClient(meta)

	domainID := d.Get("domain").(string)

	if domainID != "" {
		domain, err := client.GetDNSDomain(ctx, defaultZone, domainID)
		if err != nil {
			return diag.Errorf("error retrieving domain: %s", err)
		}

		record, err := client.GetDNSDomainRecord(ctx, defaultZone, domainID, d.Id())
		if err != nil {
			return diag.Errorf("error retrieving domain record: %s", err)
		}

		log.Printf("[DEBUG] %s: read finished successfully", resourceDomainRecordIDString(d))

		err = resourceDomainRecordApply(d, *domain.UnicodeName, record)
		if err != nil {
			return diag.Errorf("%s", err)
		}

		return nil
	}

	// If we reach this stage it means that we're in "import" mode, so we don't have the domain information yet.
	// We have to scroll each existing domain's records and try to find one matching the resource ID.
	log.Printf("[DEBUG] %s: import mode detected, trying to locate the record domain", resourceDomainRecordIDString(d))

	domains, err := client.ListDNSDomains(ctx, defaultZone)
	if err != nil {
		return diag.Errorf("error retrieving domains: %s", err)
	}

	for _, domain := range domains {
		records, err := client.ListDNSDomainRecords(ctx, defaultZone, *domain.ID)
		if err != nil {
			return diag.Errorf("error retrieving domain records: %s", err)
		}

		for _, record := range records {
			if *record.ID == d.Id() {
				if err := d.Set("domain", domain.ID); err != nil {
					return diag.Errorf("%s", err)
				}

				log.Printf("[DEBUG] %s: read finished successfully", resourceDomainRecordIDString(d))

				err = resourceDomainRecordApply(d, *domain.UnicodeName, &record)
				if err != nil {
					return diag.Errorf("%s", err)
				}

				return nil
			}
		}
	}

	return diag.Errorf("domain record %s not found", d.Id())
}

func resourceDomainRecordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning update", resourceDomainRecordIDString(d))

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), defaultZone))
	defer cancel()

	client := GetComputeClient(meta)

	name := d.Get("name").(string)
	content := d.Get("content").(string)
	rtype := d.Get("record_type").(string)
	var ttl *int64
	if t := int64(d.Get("ttl").(int)); t > 0 {
		ttl = &t
	}
	var prio *int64
	if t := int64(d.Get("prio").(int)); t > 0 {
		prio = &t
	}
	id := d.Id()
	err := client.UpdateDNSDomainRecord(ctx, defaultZone, d.Get("domain").(string), &exo.DNSDomainRecord{
		ID:       &id,
		Name:     &name,
		Content:  &content,
		Type:     &rtype,
		TTL:      ttl,
		Priority: prio,
	})
	if err != nil {
		return diag.Errorf("error updating domain record: %s", err)
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceDomainRecordIDString(d))

	domainID := d.Get("domain").(string)

	domain, err := client.GetDNSDomain(ctx, defaultZone, domainID)
	if err != nil {
		return diag.Errorf("error retrieving domain: %s", err)
	}

	record, err := client.GetDNSDomainRecord(ctx, defaultZone, domainID, d.Id())
	if err != nil {
		return diag.Errorf("error retrieving domain record: %s", err)
	}

	err = resourceDomainRecordApply(d, *domain.UnicodeName, record) // FIXME: use resourceDomainRecordRead()
	if err != nil {
		return diag.Errorf("%s", err)
	}

	return nil
}

func resourceDomainRecordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceDomainRecordIDString(d))

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), defaultZone))
	defer cancel()

	client := GetComputeClient(meta)

	record, err := client.GetDNSDomainRecord(ctx, defaultZone, d.Get("domain").(string), d.Id())
	if err != nil {
		return diag.Errorf("error retrieving domain record: %s", err)
	}

	err = client.DeleteDNSDomainRecord(ctx, defaultZone, d.Get("domain").(string), record)
	if err != nil {
		return diag.Errorf("error deleting domain record: %s", err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceDomainRecordIDString(d))

	return nil
}

func resourceDomainRecordApply(d *schema.ResourceData, domainName string, record *exo.DNSDomainRecord) error {
	d.SetId(*record.ID)
	if err := d.Set("name", record.Name); err != nil {
		return err
	}
	if err := d.Set("content", record.Content); err != nil {
		return err
	}
	if err := d.Set("record_type", record.Type); err != nil {
		return err
	}
	if err := d.Set("ttl", record.TTL); err != nil {
		return err
	}
	if err := d.Set("prio", record.Priority); err != nil {
		return err
	}

	hostname := domainName
	if record.Name != nil && *record.Name != "" {
		hostname = fmt.Sprintf("%s.%s", *record.Name, domainName)
	}

	if err := d.Set("hostname", hostname); err != nil {
		return nil
	}

	return nil
}
