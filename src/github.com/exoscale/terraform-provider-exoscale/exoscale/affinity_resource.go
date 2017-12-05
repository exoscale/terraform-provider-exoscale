package exoscale

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"errors"
	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func affinityResource() *schema.Resource {
	return &schema.Resource{
		Create: affinityCreate,
		Read:   affinityRead,
		Update: nil,
		Delete: affinityDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func affinityCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	jobid, err := client.CreateAffinityGroup(d.Get("name").(string))
	if err != nil {
		return err
	}

	var timeoutSeconds = meta.(BaseConfig).timeout
	var retries = timeoutSeconds / DelayBeforeRetry

	/* Poll and save results */
	var resp *egoscale.QueryAsyncJobResultResponse
	var succeeded = false
	for i := 0; i < retries; i++ {
		resp, err = client.PollAsyncJob(jobid)
		if err != nil {
			return err
		}

		if resp.Jobstatus == 1 {
			succeeded = true
			break
		}

		time.Sleep(DelayBeforeRetry * time.Second)
	}

	if !succeeded {
		return errors.New(fmt.Sprintf("Virtual machine creation did not succeed within %d seconds. You may increase "+
			"the timeout in the provider configuration.", timeoutSeconds))
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
	client := GetComputeClient(meta)
	groups, err := client.GetAffinityGroups()
	if err != nil {
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
	client := GetComputeClient(meta)

	log.Printf("## name: %s\n", d.Get("name").(string))
	_, err := client.DeleteAffinityGroup(d.Get("name").(string))

	return err
}
