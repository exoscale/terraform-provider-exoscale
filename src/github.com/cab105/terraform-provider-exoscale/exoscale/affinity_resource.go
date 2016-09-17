package exoscale

import (
	"encoding/json"
	"log"
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pyr/egoscale/src/egoscale"
)

func affinityResource() *schema.Resource {
	return &schema.Resource{
		Create: affinityCreate,
		Read:   affinityRead,
		Update: nil,
		Delete: affinityDelete,

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
		},
	}
}

func affinityCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)

	jobid, err := client.CreateAffinityGroup(d.Get("name").(string)); if err != nil {
		return err
	}

	/* Poll and save results */
	var resp *egoscale.QueryAsyncJobResultResponse
	for i := 0; i < 6; i++ {
		resp, err = client.PollAsyncJob(jobid); if err != nil {
			return err
		}

		if resp.Jobstatus == 1 {
			break
		}
		time.Sleep(5 * time.Second)
	}

	log.Printf("## response: %s\n", resp.Jobresult)

	var affinity egoscale.CreateAffinityGroupResponseWrapper
	if err = json.Unmarshal(resp.Jobresult, &affinity); err != nil {
		return err
	}

	d.SetId(affinity.Wrapped.Id)
	d.Set("name", affinity.Wrapped.Name)

	return affinityRead(d, meta)
}

func affinityRead(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	groups, err := client.GetAffinityGroups(); if err != nil {
		return err
	}

	for k, v := range groups {
		if v == d.Id() {
			d.Set("name", k)
			return nil
		}
	}

	return fmt.Errorf("Affinity Group %s not found", d.Id())
}

func affinityDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)

	log.Printf("## name: %s\n", d.Get("name").(string))
	_, err := client.DeleteAffinityGroup(d.Get("name").(string))

	return err
}
