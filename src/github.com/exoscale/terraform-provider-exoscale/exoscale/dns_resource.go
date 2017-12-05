package exoscale

import (
	"fmt"
	"strconv"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func dnsResource() *schema.Resource {
	return &schema.Resource{
		Create: dnsCreate,
		Read:   dnsRead,
		Update: dnsUpdate,
		Delete: dnsDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"recordcount": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"record": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
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
						},
						"prio": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"provided": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dnsCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetDnsClient(meta)

	domain, err := client.CreateDomain(d.Get("name").(string))
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(domain.Id, 10))

	if d.Get("record.#").(int) > 0 {
		for i := 0; i < d.Get("record.#").(int); i++ {
			key := fmt.Sprintf("record.%d.", i)
			var rec egoscale.DNSRecord
			rec.Name = d.Get(key + "name").(string)
			rec.Content = d.Get(key + "content").(string)
			rec.RecordType = d.Get(key + "type").(string)
			rec.Ttl = d.Get(key + "ttl").(int)
			rec.Prio = d.Get(key + "prio").(int)

			resp, err := client.CreateRecord(d.Get("name").(string), rec)
			if err != nil {
				return err
			}

			d.Set(key+"id", resp.Record.Id)
		}
	}

	return dnsRead(d, meta)
}

func dnsRead(d *schema.ResourceData, meta interface{}) error {
	client := GetDnsClient(meta)

	domain, err := client.GetDomain(d.Get("name").(string))
	if err != nil {
		return err
	}

	d.Set("state", domain.State)
	d.Set("recordcount", domain.RecordCount)

	recs, err := client.GetRecords(d.Get("name").(string))
	if err != nil {
		return err
	}
	records := make([]map[string]interface{}, domain.RecordCount)

	for k, w := range recs {
		v := w.Record
		m := make(map[string]interface{})
		m["id"] = v.Id
		m["name"] = v.Name
		m["type"] = v.RecordType
		m["ttl"] = v.Ttl
		m["prio"] = v.Prio
		m["content"] = v.Content
		records[k] = m
	}

	d.Set("record", records)

	return nil
}

func dnsUpdate(d *schema.ResourceData, meta interface{}) error {
	client := GetDnsClient(meta)
	err := client.DeleteDomain(d.Get("name").(string))
	if err != nil {
		return err
	}

	return dnsCreate(d, meta)
}

func dnsDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetDnsClient(meta)

	err := client.DeleteDomain(d.Get("name").(string))
	return err
}
