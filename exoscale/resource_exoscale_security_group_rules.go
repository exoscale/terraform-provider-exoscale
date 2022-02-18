package exoscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"regexp"
	"strconv"
	"strings"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type fetchRuleFunc func(identifier string) (*egoscale.SecurityGroupRule, bool)

const (
	resSecurityGroupRulesAttrCIDRList              = "cidr_list"
	resSecurityGroupRulesAttrDescription           = "description"
	resSecurityGroupRulesAttrICMPCode              = "icmp_code"
	resSecurityGroupRulesAttrICMPType              = "icmp_type"
	resSecurityGroupRulesAttrPorts                 = "ports"
	resSecurityGroupRulesAttrProtocol              = "protocol"
	resSecurityGroupRulesAttrSecurityGroupID       = "security_group_id"
	resSecurityGroupRulesAttrSecurityGroupName     = "security_group"
	resSecurityGroupRulesAttrUserSecurityGroupList = "user_security_group_list"
)

func resourceSecurityGroupRulesIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_security_group_rules")
}

func resourceSecurityGroupRulesSchema() map[string]*schema.Schema {
	ruleSchema := &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				resSecurityGroupRulesAttrCIDRList: {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validation.IsCIDRNetwork(0, 128),
					},
				},
				resSecurityGroupRulesAttrDescription: {
					Type:     schema.TypeString,
					Optional: true,
				},
				resSecurityGroupRulesAttrICMPCode: {
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntBetween(0, 255),
				},
				resSecurityGroupRulesAttrICMPType: {
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntBetween(0, 255),
				},
				resSecurityGroupRulesAttrPorts: {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validatePortRange,
					},
				},
				resSecurityGroupRulesAttrProtocol: {
					Type:         schema.TypeString,
					Optional:     true,
					Default:      "TCP",
					ValidateFunc: validation.StringInSlice(securityGroupRuleProtocols, true),
				},
				resSecurityGroupRulesAttrUserSecurityGroupList: {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},

				// This attribute is intended for internal bookkeeping, not for to public usage.
				"ids": {
					Type:     schema.TypeSet,
					Computed: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}

	return map[string]*schema.Schema{
		resSecurityGroupRulesAttrSecurityGroupID: {
			Type:          schema.TypeString,
			Optional:      true,
			Computed:      true,
			ForceNew:      true,
			ConflictsWith: []string{resSecurityGroupRulesAttrSecurityGroupName},
		},
		resSecurityGroupRulesAttrSecurityGroupName: {
			Type:          schema.TypeString,
			Optional:      true,
			Computed:      true,
			ForceNew:      true,
			ConflictsWith: []string{resSecurityGroupRulesAttrSecurityGroupID},
		},
		"ingress": ruleSchema,
		"egress":  ruleSchema,
	}
}

func resourceSecurityGroupRules() *schema.Resource {
	return &schema.Resource{
		Schema:        resourceSecurityGroupRulesSchema(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceSecurityGroupRulesResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceSecurityGroupRulesStateUpgradeV0,
				Version: 0,
			},
		},

		CreateContext: resourceSecurityGroupRulesCreate,
		ReadContext:   resourceSecurityGroupRulesRead,
		UpdateContext: resourceSecurityGroupRulesUpdate,
		DeleteContext: resourceSecurityGroupRulesDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSecurityGroupRulesResourceV0() *schema.Resource {
	return &schema.Resource{
		Schema: resourceSecurityGroupRulesSchema(),
	}
}

// Helper structure and functions to ease the migration process
type stateSecurityGroupRule struct {
	CIDRList              []string `json:"cidr_list,omitempty"`
	Description           string   `json:"description"`
	ICMPCode              *uint8   `json:"icmp_code,omitempty"`
	ICMPType              *uint8   `json:"icmp_type,omitempty"`
	IDs                   []string `json:"ids,omitempty"`
	Ports                 []string `json:"ports,omitempty"`
	Protocol              string   `json:"protocol,omitempty"`
	UserSecurityGroupList []string `json:"user_security_group_list,omitempty"`
}

func newStateSecurityGroupRuleFromInterface(rawStatePart interface{}) (*stateSecurityGroupRule, error) {
	serializedRule, err := json.Marshal(rawStatePart)
	if err != nil {
		return nil, err
	}

	securityGroupRule := stateSecurityGroupRule{}
	if err := json.Unmarshal(serializedRule, &securityGroupRule); err != nil {
		log.Printf("[WARNING] %s", err)
		return nil, err
	}

	return &securityGroupRule, nil
}

func (r stateSecurityGroupRule) toInterface() (map[string]interface{}, error) {
	if len(r.UserSecurityGroupList) == 0 {
		r.UserSecurityGroupList = nil
	}

	serializedRule, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	var securityGroupRule map[string]interface{}
	if err := json.Unmarshal(serializedRule, &securityGroupRule); err != nil {
		return nil, err
	}

	return securityGroupRule, nil
}

func resourceSecurityGroupRulesStateUpgradeV0(_ context.Context, rawState map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
	log.Printf("[DEBUG] beginning migration")

	// If we defined start_port to 0 with a previous version of the provider (< 0.31.x),
	// the API backend will return start_port = 1.
	// As a rule ID depends on its properties, in such a case, a rule ID doesn't match its definition anymore.
	// Here we fix this by updating the rule ID from the current state.
	var ruleIDRegex = regexp.MustCompile(`^([0-9a-z-]{36}_(?:tcp|udp)_.*)_0(-[0-9]+)?$`)
	var rulePortsRegex = regexp.MustCompile(`^0-([0-9]+)$`)

	for _, direction := range []string{"ingress", "egress"} {
		if _, ok := rawState[direction]; !ok {
			log.Printf("[DEBUG] flow direction not defined: '%s', skipping", direction)
			continue
		}

		if rules, ok := rawState[direction].([]interface{}); ok {
			patchRules := false
			for idx, rule := range rules {
				rule, err := newStateSecurityGroupRuleFromInterface(rule)
				if err != nil {
					return nil, err
				}

				// Fix rule IDs (start_port = 0 changed to 1)
				for idx, ruleID := range rule.IDs {
					rule.IDs[idx] = ruleIDRegex.ReplaceAllString(ruleID, "${1}_1${2}")
					if ruleID != rule.IDs[idx] {
						patchRules = true
						log.Printf("[DEBUG] updated rule id from '%s' to '%s'\n", ruleID, rule.IDs[idx])
					}
				}

				// Fix port range for the same reasons
				for idx, ports := range rule.Ports {
					rule.Ports[idx] = rulePortsRegex.ReplaceAllString(ports, "1-${1}")
					if ports != rule.Ports[idx] {
						patchRules = true
						log.Printf("[DEBUG] updated rule ports from '%s' to '%s'\n", ports, rule.Ports[idx])
					}
				}

				if patchRules {
					rule, err := rule.toInterface()
					if err != nil {
						return nil, err
					}

					rules[idx] = rule
					rawState[direction] = rules
					patchRules = false
				}
			}
		} else {
			return nil, fmt.Errorf("unable to deserialize schema during migration (direction = '%s'), state: %+v", direction, rawState)
		}
	}

	log.Printf("[DEBUG] done migration")
	return rawState, nil
}

func resourceSecurityGroupRulesCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceSecurityGroupRulesIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroupID, bySecurityGroupID := d.GetOk(resSecurityGroupRulesAttrSecurityGroupID)
	securityGroupName, bySecurityGroupName := d.GetOk(resSecurityGroupRulesAttrSecurityGroupName)
	if !bySecurityGroupID && !bySecurityGroupName {
		return diag.Errorf(
			"either %s or %s must be specified",
			resSecurityGroupRulesAttrSecurityGroupName,
			resSecurityGroupRulesAttrSecurityGroupID,
		)
	}

	securityGroup, err := client.FindSecurityGroup(
		ctx,
		zone, func() string {
			if bySecurityGroupID {
				return securityGroupID.(string)
			}
			return securityGroupName.(string)
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSecurityGroupRulesAttrSecurityGroupName, *securityGroup.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(resSecurityGroupRulesAttrSecurityGroupID, *securityGroup.ID); err != nil {
		return diag.FromErr(err)
	}

	for _, flowDirection := range []string{"ingress", "egress"} {
		rules := d.Get(flowDirection).(*schema.Set)

		if rules.Len() > 0 {
			for _, r := range rules.List() {
				rule := r.(map[string]interface{})
				ids := rule["ids"].(*schema.Set)

				rulesToAdd, err := securityGroupRulesToAdd(ctx, zone, client.Client, rule)
				if err != nil {
					return diag.FromErr(err)
				}

				for _, ruleToAdd := range rulesToAdd {
					ruleToAdd.FlowDirection = nonEmptyStringPtr(flowDirection)
					securityGroupRule, err := client.CreateSecurityGroupRule(
						ctx,
						zone,
						securityGroup,
						&ruleToAdd,
					)
					if err != nil {
						return diag.FromErr(err)
					}

					id, err := ruleToID(ctx, zone, client.Client, securityGroupRule)
					if err != nil {
						diag.FromErr(err)
					}
					ids.Add(id)
				}
			}
		}
	}

	d.SetId(fmt.Sprintf("%d", rand.Uint64()))

	log.Printf("[DEBUG] %s: create finished successfully", resourceSecurityGroupRulesIDString(d))

	return resourceSecurityGroupRulesRead(ctx, d, meta)
}

func resourceSecurityGroupRulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceSecurityGroupRulesIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroupID, bySecurityGroupID := d.GetOk(resSecurityGroupRulesAttrSecurityGroupID)
	securityGroupName, bySecurityGroupName := d.GetOk(resSecurityGroupRulesAttrSecurityGroupName)
	if !bySecurityGroupID && !bySecurityGroupName {
		return diag.Errorf(
			"either %s or %s must be specified",
			resSecurityGroupRulesAttrSecurityGroupName,
			resSecurityGroupRulesAttrSecurityGroupID,
		)
	}

	securityGroup, err := client.FindSecurityGroup(
		ctx,
		zone, func() string {
			if bySecurityGroupID {
				return securityGroupID.(string)
			}
			return securityGroupName.(string)
		}(),
	)
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Parent Security Group doesn't exist anymore, so do the Security Group rules.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	ruleIDs := make(map[string]int, len(securityGroup.Rules))
	for i, rule := range securityGroup.Rules {
		id, err := ruleToID(ctx, zone, client.Client, rule)
		if err != nil {
			return diag.FromErr(err)
		}
		ruleIDs[id] = i
	}

	if rules := d.Get("ingress").(*schema.Set); rules.Len() > 0 {
		err := readRules(ctx, zone, client.Client, rules, func(id string) (*egoscale.SecurityGroupRule, bool) {
			idx, ok := ruleIDs[id]
			if !ok {
				return nil, false
			}
			return securityGroup.Rules[idx], true
		})
		if err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set("ingress", rules); err != nil {
			return diag.FromErr(err)
		}
	}

	if rules := d.Get("egress").(*schema.Set); rules.Len() > 0 {
		err := readRules(ctx, zone, client.Client, rules, func(id string) (*egoscale.SecurityGroupRule, bool) {
			idx, ok := ruleIDs[id]
			if !ok {
				return nil, false
			}
			return securityGroup.Rules[idx], true
		})
		if err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set("egress", rules); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSecurityGroupRulesIDString(d))

	return nil
}

func resourceSecurityGroupRulesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning update", resourceSecurityGroupRulesIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup, err := client.GetSecurityGroup(ctx, zone, d.Get(resSecurityGroupRulesAttrSecurityGroupID).(string))
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Parent Security Group doesn't exist anymore, so do the Security Group rules.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	for _, flowDirection := range []string{"ingress", "egress"} {
		if d.HasChange(flowDirection) {
			o, n := d.GetChange(flowDirection)
			old := o.(*schema.Set)
			cur := n.(*schema.Set)

			toRemove := old.Difference(cur)
			toAdd := cur.Difference(old)

			for _, r := range toRemove.List() {
				rule := r.(map[string]interface{})
				ids := rule["ids"].(*schema.Set)
				rulesToRemove, err := securityGroupRulesToRemove(rule)
				if err != nil {
					return diag.FromErr(err)
				}

				for identifier, securityGroupRule := range rulesToRemove {
					if err := client.DeleteSecurityGroupRule(ctx, zone, securityGroup, &securityGroupRule); err != nil {
						return diag.FromErr(err)
					}
					ids.Remove(identifier)
				}
			}

			for _, r := range toAdd.List() {
				rule := r.(map[string]interface{})
				ids := rule["ids"].(*schema.Set)
				rulesToAdd, err := securityGroupRulesToAdd(ctx, zone, client.Client, rule)
				if err != nil {
					return diag.FromErr(err)
				}

				for _, ruleToAdd := range rulesToAdd {
					ruleToAdd.FlowDirection = nonEmptyStringPtr(flowDirection)
					securityGroupRule, err := client.CreateSecurityGroupRule(ctx, zone, securityGroup, &ruleToAdd)
					if err != nil {
						return diag.FromErr(err)
					}
					id, err := ruleToID(ctx, zone, client.Client, securityGroupRule)
					if err != nil {
						return diag.FromErr(err)
					}
					ids.Add(id)
				}
			}
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceSecurityGroupRulesIDString(d))

	return resourceSecurityGroupRulesRead(ctx, d, meta)
}

func resourceSecurityGroupRulesDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceSecurityGroupRulesIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup, err := client.GetSecurityGroup(ctx, zone, d.Get(resSecurityGroupRulesAttrSecurityGroupID).(string))
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Parent Security Group doesn't exist anymore, so do the Security Group rules.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	rules := d.
		Get("ingress").(*schema.Set).
		Union(d.Get("egress").(*schema.Set))

	if rules.Len() > 0 {
		for _, r := range rules.List() {
			rule := r.(map[string]interface{})
			ids := rule["ids"].(*schema.Set)

			securityGroupRules, err := securityGroupRulesToRemove(rule)
			if err != nil {
				return diag.FromErr(err)
			}

			for identifier, securityGroupRule := range securityGroupRules {
				if err := client.DeleteSecurityGroupRule(ctx, zone, securityGroup, &securityGroupRule); err != nil {
					return diag.FromErr(err)
				}

				ids.Remove(identifier)
			}
		}
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSecurityGroupRulesIDString(d))

	return nil
}

// readRules performs the reconciliation of the rules using the ruleFunc
func readRules(
	ctx context.Context,
	zone string,
	client *egoscale.Client,
	rules *schema.Set,
	ruleFunc fetchRuleFunc,
) error {
	for _, r := range rules.List() {
		rule := r.(map[string]interface{})
		rules.Remove(r)

		// In case any of those length changes, a rule has been
		// removed and things are missing.
		//
		// Rules should contain all the items formed by
		// (cidr + userSG) × ports
		//
		// For the time being, there is no needs to keep track of that
		// (big) matrix, if anything goes wrong, we have to make
		// sure, the set of rules has to be recreated.
		cidrLen := rule[resSecurityGroupRulesAttrCIDRList].(*schema.Set).Len()
		userSecurityGroupLen := rule[resSecurityGroupRulesAttrUserSecurityGroupList].(*schema.Set).Len()
		portsLen := rule[resSecurityGroupRulesAttrPorts].(*schema.Set).Len()

		expectedLen := (cidrLen + userSecurityGroupLen) * portsLen
		actualLen := 0

		cidrList := schema.NewSet(schema.HashString, nil)
		userSecurityGroupList := schema.NewSet(schema.HashString, nil)
		ports := schema.NewSet(schema.HashString, nil)

		ids := rule["ids"].(*schema.Set)

		for _, id := range ids.List() {
			r, ok := ruleFunc(id.(string))
			if !ok {
				ids.Remove(id)
				continue
			}
			actualLen++

			protocol := strings.ToUpper(*r.Protocol)
			rule[resSecurityGroupRulesAttrProtocol] = protocol
			rule[resSecurityGroupRulesAttrDescription] = defaultString(r.Description, "")
			if r.Network != nil {
				cidrList.Add(r.Network.String())
			}

			if r.SecurityGroupID != nil {
				userSecurityGroup, err := client.GetSecurityGroup(ctx, zone, *r.SecurityGroupID)
				if err != nil {
					return fmt.Errorf("unable to retrieve Security Group: %w", err)
				}
				userSecurityGroupList.Add(*userSecurityGroup.Name)
			}

			if strings.HasPrefix(protocol, "ICMP") {
				rule[resSecurityGroupRulesAttrProtocol] = strings.ReplaceAll(protocol, "V6", "v6")
				rule[resSecurityGroupRulesAttrICMPCode] = int(*r.ICMPCode)
				rule[resSecurityGroupRulesAttrICMPType] = int(*r.ICMPType)
			} else if protocol == "TCP" || protocol == "UDP" {
				var startPort, endPort uint16
				if r.StartPort != nil {
					startPort = *r.StartPort
				}
				if r.EndPort != nil {
					endPort = *r.EndPort
				}
				if startPort == endPort {
					ports.Add(fmt.Sprintf("%d", startPort))
				} else {
					ports.Add(fmt.Sprintf("%d-%d", startPort, endPort))
				}
			}
		}

		if cidrList.Len() == cidrLen &&
			ports.Len() == portsLen &&
			userSecurityGroupList.Len() == userSecurityGroupLen &&
			expectedLen != actualLen {
			// As any changes will trigger an update
			// emptying the ports is the simplest action
			// yet not the most readable one.
			ports = schema.NewSet(schema.HashString, nil)
		}

		rule["ids"] = ids
		rule[resSecurityGroupRulesAttrPorts] = ports
		rule[resSecurityGroupRulesAttrCIDRList] = cidrList
		rule[resSecurityGroupRulesAttrUserSecurityGroupList] = userSecurityGroupList
		rules.Add(rule)
	}

	return nil
}

func ruleToID(
	ctx context.Context,
	zone string,
	client *egoscale.Client,
	securityGroupRule *egoscale.SecurityGroupRule,
) (string, error) {
	var id string

	protocol := strings.ToLower(*securityGroupRule.Protocol)
	if strings.HasPrefix(protocol, "icmp") {
		id = fmt.Sprintf(
			"%s_%s_%d:%d",
			*securityGroupRule.ID,
			protocol,
			*securityGroupRule.ICMPType,
			*securityGroupRule.ICMPCode,
		)
	} else {
		var name string
		if securityGroupRule.Network != nil {
			name = securityGroupRule.Network.String()
		} else {
			userSecurityGroup, err := client.GetSecurityGroup(ctx, zone, *securityGroupRule.SecurityGroupID)
			if err != nil {
				return "", fmt.Errorf("unable to retrieve Security Group: %w", err)
			}
			name = *userSecurityGroup.Name
		}

		if protocol == "tcp" || protocol == "udp" {
			id = fmt.Sprintf(
				"%s_%s_%s_%d-%d",
				*securityGroupRule.ID,
				*securityGroupRule.Protocol,
				name,
				*securityGroupRule.StartPort,
				*securityGroupRule.EndPort,
			)
		} else {
			id = fmt.Sprintf(
				"%s_%s_%s",
				*securityGroupRule.ID,
				*securityGroupRule.Protocol,
				name,
			)
		}
	}

	return id, nil
}

// preparePorts converts a list of network port specification
// strings (format: START[-END]) into a list of start/end uint16 couples.
func preparePorts(values *schema.Set) [][2]uint16 {
	ports := make([][2]uint16, values.Len())
	for i, v := range values.List() {
		ps := strings.Split(v.(string), "-")

		startPort, _ := strconv.ParseUint(ps[0], 10, 16)
		endPort := startPort
		if len(ps) == 2 {
			endPort, _ = strconv.ParseUint(ps[1], 10, 16)
		}

		ports[i] = [2]uint16{
			uint16(startPort),
			uint16(endPort),
		}
	}

	return ports
}

// securityGroupRulesToRemove expands a configuration rule block into a list of
// egoscale.SecurityGroupRule to be removed.
func securityGroupRulesToRemove(rule map[string]interface{}) (map[string]egoscale.SecurityGroupRule, error) {
	ids := rule["ids"].(*schema.Set)
	rules := make(map[string]egoscale.SecurityGroupRule, ids.Len())

	for _, identifier := range ids.List() {
		metas := strings.SplitN(identifier.(string), "_", 2)
		id := metas[0]
		rules[identifier.(string)] = egoscale.SecurityGroupRule{ID: &id}
	}

	return rules, nil
}

// securityGroupRulesToAdd expands an ingress/egress rule configuration block
// into a list of egoscale.SecurityGroupRule to be added.
func securityGroupRulesToAdd(
	ctx context.Context,
	zone string,
	client *egoscale.Client,
	rule map[string]interface{},
) ([]egoscale.SecurityGroupRule, error) {
	protocol := strings.ToLower(rule[resSecurityGroupRulesAttrProtocol].(string))

	baseRules := make([]egoscale.SecurityGroupRule, 0)
	securityGroupRule := egoscale.SecurityGroupRule{
		Description: nonEmptyStringPtr(rule[resSecurityGroupRulesAttrDescription].(string)),
	}

	if strings.HasPrefix(protocol, "icmp") { // nolint:gocritic
		icmpCode := int64(rule[resSecurityGroupRulesAttrICMPCode].(int))
		icmpType := int64(rule[resSecurityGroupRulesAttrICMPType].(int))
		securityGroupRule.Protocol = &protocol
		securityGroupRule.ICMPCode = &icmpCode
		securityGroupRule.ICMPType = &icmpType
		baseRules = append(baseRules, securityGroupRule)
	} else if protocol == "tcp" || protocol == "udp" {
		ports := preparePorts(rule[resSecurityGroupRulesAttrPorts].(*schema.Set))
		for _, portRange := range ports {
			portRange := portRange
			securityGroupRule.Protocol = &protocol
			securityGroupRule.StartPort = &portRange[0]
			securityGroupRule.EndPort = &portRange[1]
			baseRules = append(baseRules, securityGroupRule)
		}
	} else {
		securityGroupRule.Protocol = &protocol
		baseRules = append(baseRules, securityGroupRule)
	}

	expandedRules := make([]egoscale.SecurityGroupRule, 0)

	cidrSet := rule[resSecurityGroupRulesAttrCIDRList].(*schema.Set)
	for _, r := range baseRules {
		er := r
		for _, c := range cidrSet.List() {
			_, network, err := net.ParseCIDR(c.(string))
			if err != nil {
				return nil, err
			}
			er.Network = network
			expandedRules = append(expandedRules, er)
		}
	}

	userSecurityGroupSet := rule[resSecurityGroupRulesAttrUserSecurityGroupList].(*schema.Set)
	for _, r := range baseRules {
		er := r
		for _, x := range userSecurityGroupSet.List() {
			userSecurityGroup, err := client.FindSecurityGroup(ctx, zone, x.(string))
			if err != nil {
				return nil, fmt.Errorf("unable to retrieve Security Group %q: %w", x.(string), err)
			}
			er.SecurityGroupID = userSecurityGroup.ID
			expandedRules = append(expandedRules, er)
		}
	}

	return expandedRules, nil
}
