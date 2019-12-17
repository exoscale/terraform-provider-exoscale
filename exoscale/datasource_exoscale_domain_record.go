package exoscale

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func datasourceDomainRecord() *schema.Resource {
	return &schema.Resource{

		Read: datasourceDomainRecordRead,

		Schema: map[string]*schema.Schema{
			"domain": {
				Type:        schema.TypeString,
				Description: "Domain of the Record",
				Required:    true,
			},
			"filter": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:          schema.TypeInt,
							Optional:      true,
							ConflictsWith: []string{"filter.0.name", "filter.0.record_type", "filter.0.content"},
						},
						"name": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"filter.0.id", "filter.0.record_type", "filter.0.content"},
						},
						"record_type": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"filter.0.id", "filter.0.content"},
						},
						"content": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"filter.0.id", "filter.0.name", "filter.0.record_type"},
						},
					},
				},
			},
			"records": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeInt,
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
						"create_at": {
							Type:        schema.TypeString,
							Description: "Creation of the Record",
							Optional:    true,
						},
						"update_at": {
							Type:        schema.TypeString,
							Description: "Last update of the Record",
							Optional:    true,
						},
						"record_type": {
							Type:        schema.TypeString,
							Description: "Type of the Record",
							Optional:    true,
						},
						"prio": {
							Type:        schema.TypeInt,
							Description: "Prio of the Record",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func datasourceDomainRecordRead(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetDNSClient(meta)

	dm := d.Get("domain").(string)
	domain, err := client.GetDomain(ctx, dm)
	if err != nil {
		return err
	}

	cfg := d.Get("filter").([]interface{})
	if cfg[0] == nil {
		return errors.New("either name or id must be specified")
	}
	m := cfg[0].(map[string]interface{})

	var records []egoscale.DNSRecord
	switch {
	case m["id"].(int) != 0:
		record, err := client.GetRecord(ctx, domain.Name, int64(m["id"].(int)))
		if err != nil {
			return err
		}
		records = []egoscale.DNSRecord{*record}
	case m["name"].(string) != "" || m["record_type"].(string) != "":
		records, err = client.GetRecordsWithFilters(ctx, domain.Name, m["name"].(string), m["record_type"].(string))
		if err != nil {
			return err
		}
	case m["content"].(string) != "":
		records, err = client.GetRecords(ctx, domain.Name)
		if err != nil {
			return err
		}
		records = datasourceDomainRecordFilter(records, m["content"].(string))
	}

	d.SetId(time.Now().UTC().String())

	return datasourceDomainRecordApply(d, records)
}

func datasourceDomainRecordApply(d *schema.ResourceData, records []egoscale.DNSRecord) error {
	if len(records) == 0 {
		return errors.New("no records found")
	}

	recordsDetails := make([]interface{}, 0)
	for _, r := range records {
		recordsDetails = append(recordsDetails, map[string]interface{}{
			"id":          r.ID,
			"domain":      d.Get("domain").(string),
			"name":        r.Name,
			"content":     r.Content,
			"create_at":   r.CreatedAt,
			"update_at":   r.UpdatedAt,
			"record_type": r.RecordType,
			"prio":        r.Prio,
		})
	}

	err := d.Set("records", recordsDetails)
	if err != nil {
		return fmt.Errorf("Error setting records: %s", err)
	}

	return nil
}

func datasourceDomainRecordFilter(records []egoscale.DNSRecord, content string) []egoscale.DNSRecord {
	if content == "" {
		return records
	}

	return records
}
