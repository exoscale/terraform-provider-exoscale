package exoscale

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"

	"github.com/exoscale/egoscale"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceIPAddressIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_ipaddress")
}

func resourceIPAddress() *schema.Resource {
	s := map[string]*schema.Schema{
		"zone": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The Exoscale Zone name",
		},
		"healthcheck_mode": {
			Type:         schema.TypeString,
			Description:  "The healthcheck probing mode (must be `tcp`, `http` or `https`).",
			Optional:     true,
			ValidateFunc: validation.StringMatch(regexp.MustCompile("(?:tcp|https?)"), `must be either "tcp", "http", or "https"`),
			ForceNew:     true,
		},
		"healthcheck_port": {
			Type:         schema.TypeInt,
			Description:  "The healthcheck service port to probe (must be between `1` and `65535`).",
			Optional:     true,
			ValidateFunc: validation.IntBetween(1, 65535),
		},
		"healthcheck_path": {
			Type:        schema.TypeString,
			Description: "The healthcheck probe HTTP request path (must be specified in `http`/`https` modes).",
			Optional:    true,
		},
		"healthcheck_tls_skip_verify": {
			Type:        schema.TypeBool,
			Description: "Disable TLS certificate validation in `https` mode (boolean; default: `false`). Note: this parameter can only be changed to `true`, it cannot be reset to `false` later on (requires a resource re-creation).",
			Optional:    true,
		},
		"healthcheck_tls_sni": {
			Type:        schema.TypeString,
			Description: "The healthcheck TLS server name to specify in `https` mode. Note: this parameter can only be changed to a non-empty value, it cannot be reset to its default empty value later on (requires a resource re-creation).",
			Optional:    true,
		},
		"healthcheck_interval": {
			Type:         schema.TypeInt,
			Description:  "The healthcheck probing interval (seconds; must be between `5` and `300`).",
			Optional:     true,
			ValidateFunc: validation.IntBetween(5, 300),
		},
		"healthcheck_timeout": {
			Type:         schema.TypeInt,
			Description:  "The time in seconds before considering a healthcheck probing failed (must be between `2` and `60`).",
			Optional:     true,
			ValidateFunc: validation.IntBetween(2, 60),
		},
		"healthcheck_strikes_ok": {
			Type:         schema.TypeInt,
			Description:  "The number of successful healthcheck probes before considering the target healthy (must be between `1` and `20`).",
			Optional:     true,
			ValidateFunc: validation.IntBetween(1, 20),
		},
		"healthcheck_strikes_fail": {
			Type:         schema.TypeInt,
			Description:  "The number of unsuccessful healthcheck probes before considering the target unhealthy (must be between `1` and `20`).",
			Optional:     true,
			ValidateFunc: validation.IntBetween(1, 20),
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A free-form text describing the Elastic IP (EIP).",
		},
		"reverse_dns": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringMatch(regexp.MustCompile(`^.*\.$`), ""),
			Description:  "The EIP reverse DNS record (must end with a `.`; e.g: `my-eip.example.net.`).",
		},
		"ip_address": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The Elastic IP (EIP) IPv4 address.",
		},
	}

	addTags(s, "tags")

	return &schema.Resource{
		Schema: s,

		Description: "Manage Exoscale Elastic IPs (EIP).",

		Create: resourceIPAddressCreate,
		Read:   resourceIPAddressRead,
		Update: resourceIPAddressUpdate,
		Delete: resourceIPAddressDelete,
		Exists: resourceIPAddressExists,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceIPAddressCreate(d *schema.ResourceData, meta interface{}) error {
	tflog.Debug(context.Background(), "beginning create", map[string]interface{}{
		"id": resourceIPAddressIDString(d),
	})

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	zoneName := d.Get("zone").(string)

	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	req := &egoscale.AssociateIPAddress{
		ZoneID:      zone.ID,
		Description: d.Get("description").(string),
	}

	if req.HealthcheckMode = d.Get("healthcheck_mode").(string); req.HealthcheckMode != "" {
		if req.HealthcheckPort = int64(d.Get("healthcheck_port").(int)); req.HealthcheckPort == 0 {
			return errors.New("healthcheck_port must be specified")
		}

		req.HealthcheckPath = d.Get("healthcheck_path").(string)
		if (req.HealthcheckMode == "http" || req.HealthcheckMode == "https") && req.HealthcheckPath == "" {
			return errors.New("healthcheck_path must be specified in \"http\" or \"https\" mode")
		} else if req.HealthcheckMode == "tcp" && req.HealthcheckPath != "" {
			return errors.New("healthcheck_path must not be specified in \"tcp\" mode")
		}

		if req.HealthcheckInterval = int64(d.Get("healthcheck_interval").(int)); req.HealthcheckInterval == 0 {
			return errors.New("healthcheck_interval must be specified")
		}

		if req.HealthcheckTimeout = int64(d.Get("healthcheck_timeout").(int)); req.HealthcheckTimeout == 0 {
			return errors.New("healthcheck_timeout must be specified")
		} else if req.HealthcheckTimeout >= req.HealthcheckInterval {
			return errors.New("healthcheck_timeout must be lower than healthcheck_interval")
		}

		if req.HealthcheckStrikesOk = int64(d.Get("healthcheck_strikes_ok").(int)); req.HealthcheckStrikesOk == 0 {
			return errors.New("healthcheck_strikes_ok must be specified")
		}

		if req.HealthcheckStrikesFail = int64(d.Get("healthcheck_strikes_fail").(int)); req.HealthcheckStrikesFail == 0 {
			return errors.New("healthcheck_strikes_fail must be specified")
		}

		req.HealthcheckTLSSNI = d.Get("healthcheck_tls_sni").(string)

		req.HealthcheckTLSSkipVerify = d.Get("healthcheck_tls_skip_verify").(bool)
	} else {
		for _, k := range []string{
			"healthcheck_port",
			"healthcheck_path",
			"healthcheck_interval",
			"healthcheck_timeout",
			"healthcheck_strikes_ok",
			"healthcheck_strikes_fail",
			"healthcheck_tls_skip_verify",
			"healthcheck_tls_sni",
		} {
			if _, ok := d.GetOk(k); ok {
				return fmt.Errorf("%q can only be specified with healthcheck_mode", k)
			}
		}
	}

	resp, err := client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	elasticIP := resp.(*egoscale.IPAddress)

	d.SetId(elasticIP.ID.String())

	if err := d.Set("ip_address", elasticIP.IPAddress.String()); err != nil {
		return err
	}

	if reverseDNS := d.Get("reverse_dns").(string); reverseDNS != "" {
		_, err := client.RequestWithContext(ctx, &egoscale.UpdateReverseDNSForPublicIPAddress{
			ID:         elasticIP.ID,
			DomainName: reverseDNS,
		})
		if err != nil {
			return fmt.Errorf("failed to set reverse DNS: %s", err)
		}
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
				tflog.Warn(ctx, "failure to create the tags, but the ip address was created", map[string]interface{}{
					"api_error": e.Error(),
				})
			}

			return err
		}
	}

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceIPAddressIDString(d),
	})

	return resourceIPAddressRead(d, meta)
}

func resourceIPAddressExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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
			return false, fmt.Errorf("%q is neither a valid ID or IP address", d.Id())
		}
		ipAddress.IPAddress = ip
	} else {
		ipAddress.ID = id
	}

	if _, err = client.GetWithContext(ctx, ipAddress); err != nil {
		return d.Id() != "", handleNotFound(d, err)
	}

	return true, nil
}

func resourceIPAddressRead(d *schema.ResourceData, meta interface{}) error {
	tflog.Debug(context.Background(), "beginning read", map[string]interface{}{
		"id": resourceIPAddressIDString(d),
	})

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

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceIPAddressIDString(d),
	})

	return resourceIPAddressApply(d, resp.(*egoscale.IPAddress), client)
}

func resourceIPAddressUpdate(d *schema.ResourceData, meta interface{}) error { //nolint:gocyclo
	tflog.Debug(context.Background(), "beginning update", map[string]interface{}{
		"id": resourceIPAddressIDString(d),
	})

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

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

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	eipPartials := make([]string, 0)
	updateEIP := egoscale.UpdateIPAddress{}

	if d.HasChange("healthcheck_port") {
		eipPartials = append(eipPartials, "healthcheck_port")
		updateEIP.HealthcheckPort = int64(d.Get("healthcheck_port").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckPort == 0 {
			return errors.New("healthcheck_port must be specified")
		}
	}

	if d.HasChange("healthcheck_path") {
		eipPartials = append(eipPartials, "healthcheck_path")
		updateEIP.HealthcheckPath = d.Get("healthcheck_path").(string)
		if healthcheckMode, ok := d.GetOk("healthcheck_mode"); ok {
			if (healthcheckMode == "http" || healthcheckMode == "https") && updateEIP.HealthcheckPath == "" {
				return errors.New("healthcheck_path must be specified in \"http\" or \"https\" mode")
			} else if healthcheckMode == "tcp" && updateEIP.HealthcheckPath != "" {
				return errors.New("healthcheck_path must not be specified in \"tcp\" mode")
			}
		}
	}

	if d.HasChange("healthcheck_interval") {
		eipPartials = append(eipPartials, "healthcheck_interval")
		updateEIP.HealthcheckInterval = int64(d.Get("healthcheck_interval").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckInterval == 0 {
			return errors.New("healthcheck_interval must be specified")
		}
	}

	if d.HasChange("healthcheck_timeout") {
		eipPartials = append(eipPartials, "healthcheck_timeout")
		updateEIP.HealthcheckTimeout = int64(d.Get("healthcheck_timeout").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckTimeout == 0 {
			return errors.New("healthcheck_timeout must be specified")
		}
	}

	if d.HasChange("healthcheck_strikes_ok") {
		eipPartials = append(eipPartials, "healthcheck_strikes_ok")
		updateEIP.HealthcheckStrikesOk = int64(d.Get("healthcheck_strikes_ok").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckStrikesOk == 0 {
			return errors.New("healthcheck_strikes_ok must be specified")
		}
	}

	if d.HasChange("healthcheck_strikes_fail") {
		eipPartials = append(eipPartials, "healthcheck_strikes_fail")
		updateEIP.HealthcheckStrikesFail = int64(d.Get("healthcheck_strikes_fail").(int))
		if _, ok := d.GetOk("healthcheck_mode"); ok && updateEIP.HealthcheckStrikesFail == 0 {
			return errors.New("healthcheck_strikes_fail must be specified")
		}
	}

	if d.HasChange("healthcheck_tls_sni") {
		healthcheckTLSSNI := d.Get("healthcheck_tls_sni").(string)
		if healthcheckTLSSNI == "" {
			return errors.New("healthcheck_tls_sni cannot be reset to an empty value")
		}
		eipPartials = append(eipPartials, "healthcheck_tls_sni")
		updateEIP.HealthcheckTLSSNI = healthcheckTLSSNI
		if healthcheckMode, ok := d.GetOk("healthcheck_mode"); ok && healthcheckMode != "https" {
			return errors.New("healthcheck_tls_sni is only valid in https mode")
		}
	}

	if d.HasChange("healthcheck_tls_skip_verify") {
		healthcheckTLSSkipVerify := d.Get("healthcheck_tls_skip_verify").(bool)
		if !healthcheckTLSSkipVerify {
			return errors.New("healthcheck_tls_skip_verify cannot be disabled")
		}
		eipPartials = append(eipPartials, "healthcheck_tls_skip_verify")
		updateEIP.HealthcheckTLSSkipVerify = healthcheckTLSSkipVerify
		if healthcheckMode, ok := d.GetOk("healthcheck_mode"); ok && healthcheckMode != "https" {
			return errors.New("healthcheck_tls_skip_verify is only valid in https mode")
		}
	}

	if d.HasChange("description") {
		eipPartials = append(eipPartials, "description")
		updateEIP.Description = d.Get("description").(string)
	}

	if d.HasChange("reverse_dns") {
		eipPartials = append(eipPartials, "reverse_dns")
		if reverseDNS := d.Get("reverse_dns").(string); reverseDNS == "" {
			commands = append(commands, partialCommand{
				request: &egoscale.DeleteReverseDNSFromPublicIPAddress{ID: id},
			})
		} else {
			commands = append(commands, partialCommand{
				request: &egoscale.UpdateReverseDNSForPublicIPAddress{
					ID:         id,
					DomainName: reverseDNS,
				},
			})
		}
	}

	if len(eipPartials) > 0 {
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
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceIPAddressIDString(d),
	})

	return resourceIPAddressRead(d, meta)
}

func resourceIPAddressDelete(d *schema.ResourceData, meta interface{}) error {
	tflog.Debug(context.Background(), "beginning delete", map[string]interface{}{
		"id": resourceIPAddressIDString(d),
	})

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	eip := &egoscale.IPAddress{ID: id}

	if err := client.DeleteWithContext(ctx, eip); err != nil {
		return err
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceIPAddressIDString(d),
	})

	return nil
}

func resourceIPAddressApply(d *schema.ResourceData, ip *egoscale.IPAddress, client *egoscale.Client) error {
	d.SetId(ip.ID.String())
	if err := d.Set("ip_address", ip.IPAddress.String()); err != nil {
		return err
	}
	if err := d.Set("zone", ip.ZoneName); err != nil {
		return err
	}
	if err := d.Set("description", ip.Description); err != nil {
		return err
	}

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
		if err := d.Set("healthcheck_tls_sni", ip.Healthcheck.TLSSNI); err != nil {
			return err
		}
		if err := d.Set("healthcheck_tls_skip_verify", ip.Healthcheck.TLSSkipVerify); err != nil {
			return err
		}
	}

	resp, err := client.Request(&egoscale.QueryReverseDNSForPublicIPAddress{ID: egoscale.MustParseUUID(d.Id())})
	if err != nil {
		return fmt.Errorf("failed to retrieve reverse DNS: %s", err)
	}
	if ip := resp.(*egoscale.IPAddress); len(ip.ReverseDNS) > 0 {
		if err := d.Set("reverse_dns", ip.ReverseDNS[0].DomainName); err != nil {
			return err
		}
	}

	tags := make(map[string]interface{})
	for _, tag := range ip.Tags {
		tags[tag.Key] = tag.Value
	}
	if err := d.Set("tags", tags); err != nil {
		return err
	}

	return nil
}
