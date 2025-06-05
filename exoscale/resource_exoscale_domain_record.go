package exoscale

import (
	"context"
	"errors"
	"fmt"

	exoapi "github.com/exoscale/egoscale/v2/api"
	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var supportedRecordTypes = []string{
	"A", "AAAA", "ALIAS", "CAA", "CNAME",
	"HINFO", "MX", "NAPTR", "NS", "POOL",
	"SPF", "SRV", "SSHFP", "TXT", "URL",
}

func resourceDomainRecordIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_domain_record")
}

func resourceDomainRecord() *schema.Resource {
	return &schema.Resource{
		Description: `Manage Exoscale [DNS](https://community.exoscale.com/documentation/dns/) Domain Records.

Corresponding data source: [exoscale_domain_record](../data-sources/domain_record.md).`,
		Schema: map[string]*schema.Schema{
			"domain": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The parent [exoscale_domain](./domain.md) to attach the record to.",
			},
			"record_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(supportedRecordTypes, true),
				Description:  "The record type (`A`, `AAAA`, `ALIAS`, `CAA`, `CNAME`, `HINFO`, `MX`, `NAPTR`, `NS`, `POOL`, `SPF`, `SRV`, `SSHFP`, `TXT`, `URL`).",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The record name, Leave blank (`\"\"`) to create a root record (similar to using `@` in a DNS zone file).",
			},
			"content": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The record value. Format follows specific record type. For example SRV record format would be `<weight> <port> <target>`",
			},
			"content_normalized": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The normalized value of the record",
			},
			"ttl": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "The record TTL (seconds; minimum `0`; default: `3600`).",
			},
			"prio": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "The record priority (for types that support it; minimum `0`).",
			},
			"hostname": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The record *Fully Qualified Domain Name* (FQDN). Useful for aliasing `A`/`AAAA` records with `CNAME`.",
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
			Create: schema.DefaultTimeout(config.DefaultTimeout),
			Read:   schema.DefaultTimeout(config.DefaultTimeout),
			Update: schema.DefaultTimeout(config.DefaultTimeout),
			Delete: schema.DefaultTimeout(config.DefaultTimeout),
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
	client := getClient(meta)

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
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceDomainRecordIDString(d),
	})

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	domainId := v3.UUID(d.Get("domain").(string))
	content := d.Get("content").(string)
	rtype := d.Get("record_type").(string)
	ttl := int64(d.Get("ttl").(int))
	prio := int64(d.Get("prio").(int))
	op, err := client.CreateDNSDomainRecord(ctx,
		domainId, v3.CreateDNSDomainRecordRequest{
			Name:     name,
			Content:  content,
			Type:     v3.CreateDNSDomainRecordRequestType(rtype),
			Ttl:      ttl,
			Priority: prio,
		})
	if err != nil {
		return diag.Errorf("error creating domain record: %q", err)
	}

	op, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		return diag.Errorf("error creating domain record: %q", err)
	}

	d.SetId(op.Reference.ID.String())

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	return resourceDomainRecordRead(ctx, d, meta)
}

func resourceDomainRecordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return false, err
	}

	domainID := v3.UUID(d.Get("domain").(string))

	if domainID != "" {
		_, err := client.GetDNSDomainRecord(ctx, domainID, v3.UUID(d.Id()))
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
	tflog.Debug(ctx, "import mode detected, trying to locate the record domain", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	domains, err := client.ListDNSDomains(ctx)
	if err != nil {
		return false, err
	}

	for _, domain := range domains.DNSDomains {
		records, err := client.ListDNSDomainRecords(ctx, domain.ID)
		if err != nil {
			return false, err
		}
		_, err = records.FindDNSDomainRecord(d.Id())
		if err != nil {
			if errors.Is(err, v3.ErrNotFound) {
				continue
			}
			return false, err
		}
		tflog.Debug(ctx, "found record domain", map[string]interface{}{
			"id":          resourceDomainIDString(d),
			"domain_name": domain.UnicodeName,
		})
		return true, nil
	}

	return false, nil
}

func resourceDomainRecordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return diag.FromErr(err)
	}

	domainID := v3.UUID(d.Get("domain").(string))

	if domainID != "" {
		domain, err := client.GetDNSDomain(ctx, domainID)
		if err != nil {
			return diag.Errorf("error retrieving domain: %s", err)
		}

		record, err := client.GetDNSDomainRecord(ctx, domainID, v3.UUID(d.Id()))
		if err != nil {
			return diag.Errorf("error retrieving domain record: %s", err)
		}

		tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
			"id": resourceDomainIDString(d),
		})

		if contentNormalized := d.Get("content_normalized").(string); record.Content != "" &&
			contentNormalized != "" && // skip create
			contentNormalized != record.Content {
			// If the record content has changed, we need to update the record in the remote
			tflog.Debug(ctx, "DNSimple Zone Record content changed", map[string]interface{}{
				"state":  contentNormalized,
				"remote": record.Content,
			})
			if err := d.Set("content", record.Content); err != nil {
				return diag.Errorf("error setting domain content: %s", err)
			}
		}

		err = resourceDomainRecordApply(d, domain.UnicodeName, *record)
		if err != nil {
			return diag.Errorf("%s", err)
		}

		return nil
	}

	// If we reach this stage it means that we're in "import" mode, so we don't have the domain information yet.
	// We have to scroll each existing domain's records and try to find one matching the resource ID.
	tflog.Debug(ctx, "import mode detected, trying to locate the record domain", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	domains, err := client.ListDNSDomains(ctx)
	if err != nil {
		return diag.Errorf("error retrieving domains: %s", err)
	}

	for _, domain := range domains.DNSDomains {
		records, err := client.ListDNSDomainRecords(ctx, domain.ID)
		if err != nil {
			return diag.Errorf("error retrieving domain records: %s", err)
		}

		record, err := records.FindDNSDomainRecord(d.Id())
		if err != nil {
			if errors.Is(err, v3.ErrNotFound) {
				continue
			}
			return diag.Errorf("error FindDNSDomain domain records: %s", err)

		}

		if err := d.Set("domain", domain.ID); err != nil {
			return diag.Errorf("%s", err)
		}

		tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
			"id": resourceDomainIDString(d),
		})

		// For import we need to set 'content' now
		if err := d.Set("content", record.Content); err != nil {
			return diag.Errorf("error setting domain content: %s", err)
		}

		err = resourceDomainRecordApply(d, domain.UnicodeName, record)
		if err != nil {
			return diag.Errorf("%s", err)
		}

		return nil
	}

	return diag.Errorf("domain record %s not found", d.Id())
}

func resourceDomainRecordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return diag.FromErr(err)
	}

	domainID := v3.UUID(d.Get("domain").(string))
	name := d.Get("name").(string)
	content := d.Get("content").(string)
	ttl := int64(d.Get("ttl").(int))
	prio := int64(d.Get("prio").(int))
	id := d.Id()
	op, err := client.UpdateDNSDomainRecord(ctx, domainID, v3.UUID(id), v3.UpdateDNSDomainRecordRequest{
		Name:     name,
		Content:  content,
		Ttl:      ttl,
		Priority: prio,
	})
	if err != nil {
		return diag.Errorf("error updating domain record: %s", err)
	}

	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		return diag.Errorf("error updating domain record: %s", err)
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	domain, err := client.GetDNSDomain(ctx, domainID)
	if err != nil {
		return diag.Errorf("error retrieving domain: %s", err)
	}

	record, err := client.GetDNSDomainRecord(ctx, domainID, v3.UUID(d.Id()))
	if err != nil {
		return diag.Errorf("error retrieving domain record: %s", err)
	}

	err = resourceDomainRecordApply(d, domain.UnicodeName, *record) // FIXME: use resourceDomainRecordRead()
	if err != nil {
		return diag.Errorf("%s", err)
	}

	return nil
}

func resourceDomainRecordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, defaultZone)
	if err != nil {
		return diag.FromErr(err)
	}

	domainID := v3.UUID(d.Get("domain").(string))

	record, err := client.GetDNSDomainRecord(ctx, domainID, v3.UUID(d.Id()))
	if err != nil {
		return diag.Errorf("error retrieving domain record: %s", err)
	}

	op, err := client.DeleteDNSDomainRecord(ctx, domainID, record.ID)
	if err != nil {
		return diag.Errorf("error deleting domain record: %s", err)
	}
	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		return diag.Errorf("error deleting domain record: %s", err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	return nil
}

func resourceDomainRecordApply(d *schema.ResourceData, domainName string, record v3.DNSDomainRecord) error {
	d.SetId(string(record.ID))
	if err := d.Set("name", record.Name); err != nil {
		return err
	}
	if err := d.Set("content_normalized", record.Content); err != nil {
		return err
	}
	if err := d.Set("record_type", record.Type); err != nil {
		return err
	}
	if err := d.Set("ttl", record.Ttl); err != nil {
		return err
	}
	if err := d.Set("prio", record.Priority); err != nil {
		return err
	}

	hostname := domainName
	if record.Name != "" {
		hostname = fmt.Sprintf("%s.%s", record.Name, domainName)
	}

	if err := d.Set("hostname", hostname); err != nil {
		return nil
	}

	return nil
}
