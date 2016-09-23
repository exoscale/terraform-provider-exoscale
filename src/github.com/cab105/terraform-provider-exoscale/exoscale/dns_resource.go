package exoscale

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pyr/egoscale/src/egoscale"
)

func dnsResource() *schema.Resource {
	return &schema.Resource{
		Create: dnsCreate,
		Read:   dnsRead,
		Update: dnsUpdate,
		Delete: dnsDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:		schema.TypeString,
				Computed:	true,
			},
			"name": &schema.Schema{
				Type:		schema.TypeString,
				Required:	true,
				ForceNew:	true,
			},
			"state": &schema.Schema{
				Type:		schema.TypeString,
				Computed:	true,
			},
			"recordcount": &schema.Schema{
				Type:		schema.TypeInt,
				Computed:	true,
			},
			"record": &schema.Schema{
				Type:		schema.TypeList,
				Optional:	true,
				Elem:		&schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:		schema.TypeInt,
							Computed:	true,
						},
						"name":	&schema.Schema{
							Type:		schema.TypeString,
							Required:	true,
						},
						"type": &schema.Schema{
							Type:		schema.TypeString,
							Required:	true,
						},
						"content": &schema.Schema{
							Type:		schema.TypeString,
							Required:	true,
						},
						"ttl": &schema.Schema{
							Type:		schema.TypeInt,
							Optional:	true,
						},
						"prio": &schema.Schema{
							Type:		schema.TypeInt,
							Optional:	true,
						},
						"provided": &schema.Schema{
							Type:		schema.TypeBool,
							Computed:	true,
						},
					},
				},
			},
		},
	}
}

func dnsCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(DNSEndpoint, meta)

	domain, err := client.CreateDomain(d.Get("name").(string)); if err != nil {
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

			resp, err := client.CreateRecord(d.Get("name").(string), rec); if err != nil {
				return err
			}

			d.Set(key + "id", resp.Record.Id)
		}
	}

	return dnsRead(d, meta)
}

func dnsRead(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(DNSEndpoint, meta)

	domain, err := client.GetDomain(d.Get("name").(string)); if err != nil {
		return err
	}

	d.Set("state", domain.State)
	d.Set("recordcount", domain.RecordCount)

	recs, err := client.GetRecords(d.Get("name").(string)); if err != nil {
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
	client := GetClient(DNSEndpoint, meta)
	err := client.DeleteDomain(d.Get("name").(string))
	if err != nil {
		return err
	}

	return dnsCreate(d, meta)
}

func dnsDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(DNSEndpoint, meta)

	err := client.DeleteDomain(d.Get("name").(string))
	return err
}
