package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func datasourceDomainRecord() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"domain": {
				Type:        schema.TypeString,
				Description: "Domain of the record",
				Required:    true,
			},
			"name": {
				Type:          schema.TypeString,
				Description:   "Name of the record",
				Optional:      true,
				ConflictsWith: []string{"id"},
			},
			"id": {
				Type:          schema.TypeInt,
				Description:   "ID of the record",
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
		},

		Read: datasourceDomainRecordRead,
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

	var recordID int64
	switch {
	case d.Get("id").(int) != 0:
		recordID = int64(d.Get("id").(int))
	case d.Get("name").(string) != "":
		r, err := client.GetRecordsWithFilters(ctx, domain.Name, d.Get("name").(string), "")
		if err != nil {
			return err
		}
		if len(r) == 0 {
			return fmt.Errorf("record %s: not found", d.Get("name").(string))
		}
		if len(r) > 1 {
			return fmt.Errorf("record %s: more than one record found", d.Get("name").(string))
		}
		recordID = r[0].ID
	default:
		return errors.New("either name or id must be specified")
	}

	record, err := client.GetRecord(ctx, domain.Name, recordID)
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(record.ID, 10))

	if err := d.Set("name", record.Name); err != nil {
		return err
	}

	if err := d.Set("domain", domain.Name); err != nil {
		return err
	}

	return nil
}
