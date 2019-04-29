package exoscale

import (
	"context"
	"fmt"
	"log"
	"net"
	"regexp"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func elasticIPResource() *schema.Resource {
	s := map[string]*schema.Schema{
		"zone": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "Name of the zone",
		},
		"healthcheck_mode": {
			Type:         schema.TypeString,
			Description:  "Healthcheck probing mode",
			Optional:     true,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("(?:tcp|http)"), `must be either "tcp" or "http"`),
			ForceNew:     true,
		},
		"healthcheck_port": {
			Type:         schema.TypeInt,
			Description:  "Healthcheck service port to probe",
			Optional:     true,
			ValidateFunc: validation.IntBetween(1, 65535),
		},
		"healthcheck_path": {
			Type:        schema.TypeString,
			Description: "Healthcheck probe HTTP request path, must be specified in \"http\" mode",
			Optional:    true,
		},
		"healthcheck_interval": {
			Type:         schema.TypeInt,
			Description:  "Healthcheck probing interval in seconds",
			Optional:     true,
			ValidateFunc: validation.IntBetween(5, 300),
		},
		"healthcheck_timeout": {
			Type:         schema.TypeInt,
			Description:  "Time in seconds before considering a healthcheck probing failed",
			Optional:     true,
			ValidateFunc: validation.IntBetween(2, 60),
		},
		"healthcheck_strikes_ok": {
			Type:         schema.TypeInt,
			Description:  "Number of successful healthcheck probes before considering the target healthy",
			Optional:     true,
			ValidateFunc: validation.IntBetween(1, 20),
		},
		"healthcheck_strikes_fail": {
			Type:         schema.TypeInt,
			Description:  "Number of unsuccessful healthcheck probes before considering the target unhealthy",
			Optional:     true,
			ValidateFunc: validation.IntBetween(1, 20),
		},
		"ip_address": {
			Type:     schema.TypeString,
			Computed: true,
		},
	}

	addTags(s, "tags")

	return &schema.Resource{
		Create: createElasticIP,
		Read:   readElasticIP,
		Update: updateElasticIP,
		Exists: existsElasticIP,
		Delete: deleteElasticIP,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},

		Schema: s,
	}
}

func createElasticIP(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	zoneName := d.Get("zone").(string)

	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	req := &egoscale.AssociateIPAddress{ZoneID: zone.ID}

	if req.HealthcheckMode = d.Get("healthcheck_mode").(string); req.HealthcheckMode != "" {
		if req.HealthcheckPort = int64(d.Get("healthcheck_port").(int)); req.HealthcheckPort == 0 {
			return fmt.Errorf("healthcheck_port must be specified")
		}

		req.HealthcheckPath = d.Get("healthcheck_path").(string)
		if req.HealthcheckMode == "http" && req.HealthcheckPath == "" {
			return fmt.Errorf("healthcheck_path must be specified in \"http\" mode")
		} else if req.HealthcheckMode == "tcp" && req.HealthcheckPath != "" {
			return fmt.Errorf("healthcheck_path must not be specified in \"tcp\" mode")
		}

		if req.HealthcheckInterval = int64(d.Get("healthcheck_interval").(int)); req.HealthcheckInterval == 0 {
			return fmt.Errorf("healthcheck_interval must be specified")
		}

		if req.HealthcheckTimeout = int64(d.Get("healthcheck_timeout").(int)); req.HealthcheckTimeout == 0 {
			return fmt.Errorf("healthcheck_timeout must be specified")
		} else if req.HealthcheckTimeout >= req.HealthcheckInterval {
			return fmt.Errorf("healthcheck_timeout must be lower than healthcheck_interval")
		}

		if req.HealthcheckStrikesOk = int64(d.Get("healthcheck_strikes_ok").(int)); req.HealthcheckStrikesOk == 0 {
			return fmt.Errorf("healthcheck_strikes_ok must be specified")
		}

		if req.HealthcheckStrikesFail = int64(d.Get("healthcheck_strikes_fail").(int)); req.HealthcheckStrikesFail == 0 {
			return fmt.Errorf("healthcheck_strikes_fail must be specified")
		}
	} else {
		for _, k := range []string{
			"healthcheck_port",
			"healthcheck_path",
			"healthcheck_interval",
			"healthcheck_timeout",
			"healthcheck_strikes_ok",
			"healthcheck_strikes_fail",
		} {
			if _, ok := d.GetOkExists(k); ok {
				return fmt.Errorf("%q can only be specified with healthcheck_mode", k)
			}
		}
	}

	resp, err := client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	elasticIP, ok := resp.(*egoscale.IPAddress)
	if !ok {
		return fmt.Errorf("wrong type: an IPAddress was expected, got %T", resp)
	}
	d.SetId(elasticIP.ID.String())
	if err := d.Set("ip_address", elasticIP.IPAddress.String()); err != nil {
		return err
	}

	cmd, err := createTags(d, "tags", elasticIP.ResourceType())
	if err != nil {
		return err
	}

	if cmd != nil {
		if err := client.BooleanRequestWithContext(ctx, cmd); err != nil {
			// Attempting to destroy the freshly created ip address
			e := client.BooleanRequestWithContext(ctx, &egoscale.DisassociateIPAddress{
				ID: elasticIP.ID,
			})

			if e != nil {
				log.Printf("[WARNING] Failure to create the tags, but the ip address was created. %v", e)
			}

			return err
		}
	}

	return readElasticIP(d, meta)
}

func existsElasticIP(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return false, err
	}

	resp, err := client.RequestWithContext(ctx, &egoscale.ListPublicIPAddresses{
		ID: id,
	})

	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	elasticIPes := resp.(*egoscale.ListPublicIPAddressesResponse)
	return elasticIPes.Count == 1, nil
}

func readElasticIP(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	ipAddress := &egoscale.IPAddress{
		IsElastic: true,
	}

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		ip := net.ParseIP(d.Id())
		if ip == nil {
			return fmt.Errorf("%q is neither a valid ID or IP address", d.Id())
		}
		ipAddress.IPAddress = ip
	} else {
		ipAddress.ID = id
	}

	resp, err := client.GetWithContext(ctx, ipAddress)
	if err != nil {
		return handleNotFound(d, err)
	}

	return applyElasticIP(d, resp.(*egoscale.IPAddress))
}

func updateElasticIP(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	d.Partial(true)

	commands := make([]partialCommand, 0)

	updateTags, err := updateTags(d, "tags", new(egoscale.IPAddress).ResourceType())
	if err != nil {
		return err
	}
	for _, update := range updateTags {
		commands = append(commands, partialCommand{
			partial: "tags",
			request: update,
		})
	}

	eipPartials := make([]string, 0)
	updateEIP := egoscale.UpdateIPAddress{}
	if d.HasChange("healthcheck_port") {
		eipPartials = append(eipPartials, "healthcheck_port")
		updateEIP.HealthcheckPort = int64(d.Get("healthcheck_port").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckPort == 0 {
			return fmt.Errorf("healthcheck_port must be specified")
		}
	}
	if d.HasChange("healthcheck_path") {
		eipPartials = append(eipPartials, "healthcheck_path")
		updateEIP.HealthcheckPath = d.Get("healthcheck_path").(string)
		if healthcheckMode, ok := d.GetOk("healthcheck_mode"); ok {
			if healthcheckMode == "http" && updateEIP.HealthcheckPath == "" {
				return fmt.Errorf("healthcheck_path must be specified in \"http\" mode")
			} else if healthcheckMode == "tcp" && updateEIP.HealthcheckPath != "" {
				return fmt.Errorf("healthcheck_path must not be specified in \"tcp\" mode")
			}
		}
	}
	if d.HasChange("healthcheck_interval") {
		eipPartials = append(eipPartials, "healthcheck_interval")
		updateEIP.HealthcheckInterval = int64(d.Get("healthcheck_interval").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckInterval == 0 {
			return fmt.Errorf("healthcheck_interval must be specified")
		}
	}
	if d.HasChange("healthcheck_timeout") {
		eipPartials = append(eipPartials, "healthcheck_timeout")
		updateEIP.HealthcheckTimeout = int64(d.Get("healthcheck_timeout").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckTimeout == 0 {
			return fmt.Errorf("healthcheck_timeout must be specified")
		}
	}
	if d.HasChange("healthcheck_strikes_ok") {
		eipPartials = append(eipPartials, "healthcheck_strikes_ok")
		updateEIP.HealthcheckStrikesOk = int64(d.Get("healthcheck_strikes_ok").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckStrikesOk == 0 {
			return fmt.Errorf("healthcheck_strikes_ok must be specified")
		}
	}
	if d.HasChange("healthcheck_strikes_fail") {
		eipPartials = append(eipPartials, "healthcheck_strikes_fail")
		updateEIP.HealthcheckStrikesFail = int64(d.Get("healthcheck_strikes_fail").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckStrikesFail == 0 {
			return fmt.Errorf("healthcheck_strikes_fail must be specified")
		}
	}
	if len(eipPartials) > 0 {
		id, err := egoscale.ParseUUID(d.Id())
		if err != nil {
			return err
		}

		updateEIP.ID = id
		commands = append(commands, partialCommand{
			partials: eipPartials,
			request:  updateEIP,
		})
	}

	for _, cmd := range commands {
		_, err := client.RequestWithContext(ctx, cmd.request)
		if err != nil {
			return err
		}

		d.SetPartial(cmd.partial)
		if cmd.partials != nil {
			for _, partial := range cmd.partials {
				d.SetPartial(partial)
			}
		}
	}

	err = readElasticIP(d, meta)
	if err != nil {
		return err
	}

	d.Partial(false)

	return err
}

func deleteElasticIP(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	return client.DeleteWithContext(ctx, &egoscale.IPAddress{
		ID: id,
	})
}

func applyElasticIP(d *schema.ResourceData, ip *egoscale.IPAddress) error {
	d.SetId(ip.ID.String())
	if err := d.Set("ip_address", ip.IPAddress.String()); err != nil {
		return err
	}
	if err := d.Set("zone", ip.ZoneName); err != nil {
		return err
	}

	// healthcheck
	if ip.Healthcheck != nil {
		if err := d.Set("healthcheck_mode", ip.Healthcheck.Mode); err != nil {
			return err
		}
		if err := d.Set("healthcheck_port", ip.Healthcheck.Port); err != nil {
			return err
		}
		if err := d.Set("healthcheck_path", ip.Healthcheck.Path); err != nil {
			return err
		}
		if err := d.Set("healthcheck_interval", ip.Healthcheck.Interval); err != nil {
			return err
		}
		if err := d.Set("healthcheck_timeout", ip.Healthcheck.Timeout); err != nil {
			return err
		}
		if err := d.Set("healthcheck_strikes_ok", ip.Healthcheck.StrikesOk); err != nil {
			return err
		}
		if err := d.Set("healthcheck_strikes_fail", ip.Healthcheck.StrikesFail); err != nil {
			return err
		}
	}

	// tags
	tags := make(map[string]interface{})
	for _, tag := range ip.Tags {
		tags[tag.Key] = tag.Value
	}
	if err := d.Set("tags", tags); err != nil {
		return err
	}

	return nil
}
