package exoscale

import (
	"context"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"

	exo "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceDomainRecord() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDomainRecordRead,

		Description: `Fetch Exoscale [DNS](https://community.exoscale.com/documentation/dns/) Domain Records data.

Corresponding resource: [exoscale_domain_record](../resources/domain_record.md).`,

		Schema: map[string]*schema.Schema{
			"domain": {
				Description: "The [exoscale_domain](./domain.md) name to match.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"filter": {
				Description: "Filter to apply when looking up domain records.",
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description:   "The record ID to match.",
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"filter.0.name", "filter.0.record_type", "filter.0.content_regex"},
						},
						"name": {
							Description:   "The domain record name to match.",
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"filter.0.id", "filter.0.content_regex"},
						},
						"record_type": {
							Description:   "The record type to match.",
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"filter.0.id", "filter.0.content_regex"},
						},
						"content_regex": {
							Description:   "A regular expression to match the record content.",
							Type:          schema.TypeString,
							Optional:      true,
							ValidateFunc:  validation.StringIsValidRegExp,
							ConflictsWith: []string{"filter.0.id", "filter.0.name", "filter.0.record_type"},
						},
					},
				},
			},
			"records": {
				Description: "The list of matching records. Structure is documented below.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Description: "ID of the Record",
							Optional:    true,
						},
						"domain": {
							Type:        schema.TypeString,
							Description: "Domain of the Record",
							Optional:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the Record",
							Optional:    true,
						},
						"content": {
							Type:        schema.TypeString,
							Description: "Content of the Record",
							Optional:    true,
						},
						"record_type": {
							Type:        schema.TypeString,
							Description: "Type of the Record",
							Optional:    true,
						},
						"ttl": {
							Type:        schema.TypeInt,
							Description: "TTL of the Record",
							Optional:    true,
						},
						"prio": {
							Type:        schema.TypeInt,
							Description: "Priority of the Record",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceDomainRecordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceIDString(d, "exoscale_domain"),
	})

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), defaultZone))
	defer cancel()

	client := GetComputeClient(meta)

	domainName := d.Get("domain").(string)
	var domain *exo.DNSDomain

	domains, err := client.ListDNSDomains(ctx, defaultZone)
	if err != nil {
		return diag.Errorf("error retrieving domain list: %s", err)
	}

	for _, item := range domains {
		if *item.UnicodeName == domainName {
			t := item
			domain = &t
			break
		}
	}

	if domain == nil {
		return diag.Errorf("domain %q not found", domainName)
	}

	flist := d.Get("filter").([]interface{})
	if len(flist) != 1 || flist[0] == nil {
		return diag.Errorf("filter not valid")
	}

	filter := flist[0].(map[string]interface{})
	id := filter["id"].(string)
	name := filter["name"].(string)
	rtype := filter["record_type"].(string)
	cregex := filter["content_regex"].(string)

	var records []exo.DNSDomainRecord
	var ids []string

	switch {
	case id != "":
		record, err := client.GetDNSDomainRecord(ctx, defaultZone, *domain.ID, id)
		if err != nil {
			return diag.Errorf("error retrieving domain record: %s", err)
		}
		records = append(records, *record)
		ids = append(ids, *record.ID)
	case name != "" || rtype != "":
		r, err := client.ListDNSDomainRecords(ctx, defaultZone, *domain.ID)
		if err != nil {
			return diag.Errorf("error retrieving domain record list: %s", err)
		}
		for _, record := range r {
			if name != "" && *record.Name != name {
				continue
			}
			if rtype != "" && *record.Type != rtype {
				continue
			}
			t := record
			records = append(records, t)
			ids = append(ids, *record.ID)
		}
	case cregex != "":
		regexp, err := regexp.Compile(cregex)
		if err != nil {
			return diag.Errorf("error parsing regex: %s", err)
		}
		r, err := client.ListDNSDomainRecords(ctx, defaultZone, *domain.ID)
		if err != nil {
			return diag.Errorf("error retrieving domain record list: %s", err)
		}
		for _, record := range r {
			if !regexp.MatchString(*record.Content) {
				continue
			}
			t := record
			records = append(records, t)
			ids = append(ids, *record.ID)
		}
	}

	if len(records) == 0 {
		return diag.Errorf("no records found")
	}

	// derive ID from result
	d.SetId(fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, "")))))

	recordsDetails := make([]map[string]interface{}, len(records))
	for i, r := range records {
		recordsDetails[i] = map[string]interface{}{
			"id":          r.ID,
			"domain":      *domain.ID,
			"name":        r.Name,
			"content":     r.Content,
			"record_type": r.Type,
			"ttl":         r.TTL,
			"prio":        r.Priority,
		}
	}

	err = d.Set("records", recordsDetails)
	if err != nil {
		return diag.Errorf("Error setting records: %s", err)
	}

	return nil
}
