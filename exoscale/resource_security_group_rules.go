package exoscale

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func securityGroupRulesResource() *schema.Resource {
	ruleSchema := &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"ids": {
					Type:     schema.TypeSet,
					Computed: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"description": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"cidr_list": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validation.CIDRNetwork(0, 128),
					},
				},
				"protocol": {
					Type:         schema.TypeString,
					Optional:     true,
					Default:      "TCP",
					ValidateFunc: validation.StringInSlice(supportedProtocols, true),
				},
				"ports": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: ValidatePortRange,
					},
				},
				"icmp_type": {
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntBetween(0, 255),
				},
				"icmp_code": {
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntBetween(0, 255),
				},
				"user_security_group_list": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
			},
		},
	}

	return &schema.Resource{
		Create: createSecurityGroupRules,
		Read:   readSecurityGroupRules,
		Update: updateSecurityGroupRules,
		Delete: deleteSecurityGroupRules,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},

		Schema: map[string]*schema.Schema{
			"security_group_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"security_group"},
			},
			"security_group": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"security_group_id"},
			},
			"ingress": ruleSchema,
			"egress":  ruleSchema,
		},
	}
}

func updateSecurityGroupRules(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	sgID, err := egoscale.ParseUUID(d.Get("security_group_id").(string))
	if err != nil {
		return err
	}

	if d.HasChange("ingress") {
		o, n := d.GetChange("ingress")
		old := o.(*schema.Set)
		new := n.(*schema.Set)

		toRemove := old.Difference(new)
		toAdd := new.Difference(old)

		for _, r := range toRemove.List() {
			rule := r.(map[string]interface{})
			ids := rule["ids"].(*schema.Set)
			reqs, err := ruleToRevoke(rule)
			if err != nil {
				return err
			}

			for identifier, req := range reqs {
				if err := client.BooleanRequestWithContext(ctx, req); err != nil {
					return err
				}

				ids.Remove(identifier)
			}
		}

		for _, r := range toAdd.List() {
			rule := r.(map[string]interface{})
			ids := rule["ids"].(*schema.Set)
			reqs, err := ruleToAuthorize(ctx, client, rule)
			if err != nil {
				return err
			}

			for _, req := range reqs {
				req.SecurityGroupID = sgID
				resp, err := client.RequestWithContext(ctx, req)
				if err != nil {
					return err
				}

				sg := resp.(*egoscale.SecurityGroup)
				if len(sg.IngressRule) != 1 {
					return fmt.Errorf("one ingress was supposed to be updated. Does %#v already exist?", req)
				}
				rule := sg.IngressRule[0]
				id := ingressRuleToID(rule)
				ids.Add(id)
			}
		}
	}

	if d.HasChange("egress") {
		o, n := d.GetChange("egress")
		old := o.(*schema.Set)
		new := n.(*schema.Set)

		toRemove := old.Difference(new)
		toAdd := new.Difference(old)

		for _, r := range toRemove.List() {
			rule := r.(map[string]interface{})
			ids := rule["ids"].(*schema.Set)
			reqs, err := ruleToRevoke(rule)
			if err != nil {
				return err
			}

			for identifier, req := range reqs {
				if err := client.BooleanRequestWithContext(ctx, (egoscale.RevokeSecurityGroupEgress)(req)); err != nil {
					return err
				}

				ids.Remove(identifier)
			}
		}

		for _, r := range toAdd.List() {
			rule := r.(map[string]interface{})
			ids := rule["ids"].(*schema.Set)
			reqs, err := ruleToAuthorize(ctx, client, rule)
			if err != nil {
				return err
			}

			for _, req := range reqs {
				req.SecurityGroupID = sgID
				ereq := (egoscale.AuthorizeSecurityGroupEgress)(req)
				resp, err := client.RequestWithContext(ctx, ereq)
				if err != nil {
					return err
				}

				sg := resp.(*egoscale.SecurityGroup)
				if len(sg.EgressRule) != 1 {
					return fmt.Errorf("one egress was supposed to be updated. Does %#v already exist?", ereq)
				}
				rule := sg.EgressRule[0]
				id := egressRuleToID(rule)
				ids.Add(id)
			}
		}
	}

	return readSecurityGroupRules(d, meta)
}

func createSecurityGroupRules(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	sg, err := inferSecurityGroup(d)
	if err != nil {
		return err
	}

	resp, err := client.GetWithContext(ctx, sg)
	if err != nil {
		if e := handleNotFound(d, err); e != nil {
			return e
		}
	}

	sg = resp.(*egoscale.SecurityGroup)

	d.SetId(fmt.Sprintf("%d", rand.Uint64()))
	if err := d.Set("security_group", sg.Name); err != nil {
		return err
	}
	if err := d.Set("security_group_id", sg.ID.String()); err != nil {
		return err
	}

	if rules := d.Get("ingress").(*schema.Set); rules.Len() > 0 {
		for _, r := range rules.List() {
			rule := r.(map[string]interface{})
			ids := rule["ids"].(*schema.Set)
			reqs, err := ruleToAuthorize(ctx, client, rule)
			if err != nil {
				return err
			}

			for _, req := range reqs {
				req.SecurityGroupID = sg.ID
				resp, err := client.RequestWithContext(ctx, req)
				if err != nil {
					return err
				}

				sg := resp.(*egoscale.SecurityGroup)
				if len(sg.IngressRule) != 1 {
					return fmt.Errorf("one ingress was supposed to be created. Does %#v already exist?", req)
				}
				rule := sg.IngressRule[0]
				id := ingressRuleToID(rule)
				ids.Add(id)
			}
		}
	}

	if rules := d.Get("egress").(*schema.Set); rules.Len() > 0 {
		for _, r := range rules.List() {
			rule := r.(map[string]interface{})
			ids := rule["ids"].(*schema.Set)
			reqs, err := ruleToAuthorize(ctx, client, rule)
			if err != nil {
				return err
			}

			for _, req := range reqs {
				req.SecurityGroupID = sg.ID
				ereq := (*egoscale.AuthorizeSecurityGroupEgress)(&req)
				resp, err := client.RequestWithContext(ctx, ereq)
				if err != nil {
					return err
				}

				sg := resp.(*egoscale.SecurityGroup)
				if len(sg.EgressRule) != 1 {
					return fmt.Errorf("one egress was supposed to be created. Does %#v already exist?", ereq)
				}
				rule := sg.EgressRule[0]
				id := egressRuleToID(rule)
				ids.Add(id)

				log.Printf("[DEBUG] rule %s was built!\n", id)
			}

			log.Printf("[DEBUG] Ingress RuleID %+v\n", ids)
		}
	}

	return readSecurityGroupRules(d, meta)
}

func readSecurityGroupRules(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	sg, err := inferSecurityGroup(d)
	if err != nil {
		return err
	}

	resp, err := client.GetWithContext(ctx, sg)
	if err != nil {
		return handleNotFound(d, err)
	}

	sg = resp.(*egoscale.SecurityGroup)

	ingressRules := make(map[string]int, len(sg.IngressRule))
	for i, rule := range sg.IngressRule {
		id := ingressRuleToID(rule)
		ingressRules[id] = i
	}

	if rules := d.Get("ingress").(*schema.Set); rules.Len() > 0 {
		readRules(rules, func(identifier string) (*egoscale.IngressRule, bool) {
			idx, ok := ingressRules[identifier]
			if !ok {
				return nil, false
			}
			return &sg.IngressRule[idx], true
		})
		if err := d.Set("ingress", rules); err != nil {
			return err
		}
	}

	egressRules := make(map[string]int, len(sg.EgressRule))
	for i, rule := range sg.EgressRule {
		id := egressRuleToID(rule)
		egressRules[id] = i
	}

	if rules := d.Get("egress").(*schema.Set); rules.Len() > 0 {
		readRules(rules, func(identifier string) (*egoscale.IngressRule, bool) {
			idx, ok := egressRules[identifier]
			if !ok {
				return nil, false
			}
			return (*egoscale.IngressRule)(&sg.EgressRule[idx]), true
		})
		if err := d.Set("egress", rules); err != nil {
			return err
		}
	}

	return nil
}

type fetchRuleFunc func(identifier string) (*egoscale.IngressRule, bool)

// readRules performs the reconciliation of the rules using the ruleFunc
func readRules(rules *schema.Set, ruleFunc fetchRuleFunc) {
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
		cidrLen := rule["cidr_list"].(*schema.Set).Len()
		userSecurityGroupLen := rule["user_security_group_list"].(*schema.Set).Len()
		portsLen := rule["ports"].(*schema.Set).Len()

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

			prot := strings.ToUpper(r.Protocol)
			rule["protocol"] = prot
			rule["description"] = r.Description
			if r.CIDR != nil {
				cidrList.Add(r.CIDR.String())
			}

			if r.SecurityGroupName != "" {
				userSecurityGroupList.Add(r.SecurityGroupName)
			}

			if strings.HasPrefix(prot, "ICMP") {
				rule["protocol"] = strings.Replace(prot, "V6", "v6", -1)
				rule["icmp_code"] = (int)(r.IcmpCode)
				rule["icmp_type"] = (int)(r.IcmpType)
			} else {
				if r.StartPort == r.EndPort {
					ports.Add(fmt.Sprintf("%d", r.StartPort))
				} else {
					ports.Add(fmt.Sprintf("%d-%d", r.StartPort, r.EndPort))
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
		rule["cidr_list"] = cidrList
		rule["ports"] = ports
		rule["user_security_group_list"] = userSecurityGroupList

		rules.Add(rule)
	}
}

func deleteSecurityGroupRules(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	if rules := d.Get("ingress").(*schema.Set); rules.Len() > 0 {
		for _, r := range rules.List() {
			rule := r.(map[string]interface{})
			ids := rule["ids"].(*schema.Set)
			reqs, err := ruleToRevoke(rule)
			if err != nil {
				return err
			}

			for identifier, req := range reqs {
				if err := client.BooleanRequestWithContext(ctx, req); err != nil {
					return err
				}

				ids.Remove(identifier)
			}
		}
	}

	if rules := d.Get("egress").(*schema.Set); rules.Len() > 0 {
		for _, r := range rules.List() {
			rule := r.(map[string]interface{})
			ids := rule["ids"].(*schema.Set)
			reqs, err := ruleToRevoke(rule)
			if err != nil {
				return err
			}
			for identifier, req := range reqs {
				if err := client.BooleanRequestWithContext(ctx, (*egoscale.RevokeSecurityGroupEgress)(&req)); err != nil {
					return err
				}

				ids.Remove(identifier)
			}
		}
	}

	d.SetId("")
	return nil
}

func ingressRuleToID(rule egoscale.IngressRule) string {
	p := strings.ToLower(rule.Protocol)
	if strings.HasPrefix(p, "icmp") {
		return fmt.Sprintf("%s_%s_%d:%d", rule.RuleID, p, rule.IcmpType, rule.IcmpCode)
	}

	name := rule.SecurityGroupName
	if rule.CIDR != nil {
		name = rule.CIDR.String()
	}

	return fmt.Sprintf("%s_%s_%s_%d-%d", rule.RuleID, rule.Protocol, name, rule.StartPort, rule.EndPort)
}

func egressRuleToID(rule egoscale.EgressRule) string {
	p := strings.ToLower(rule.Protocol)
	if strings.HasPrefix(p, "icmp") {
		return fmt.Sprintf("%s_%s_%d:%d", rule.RuleID, p, rule.IcmpType, rule.IcmpCode)
	}

	name := rule.SecurityGroupName
	if rule.CIDR != nil {
		name = rule.CIDR.String()
	}

	return fmt.Sprintf("%s_%s_%s_%d-%d", rule.RuleID, rule.Protocol, name, rule.StartPort, rule.EndPort)
}

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

// ruleToRevoke converts a rule (or rules) into a list of revoke requests.
func ruleToRevoke(rule map[string]interface{}) (map[string]egoscale.RevokeSecurityGroupIngress, error) {
	ids := rule["ids"].(*schema.Set)
	reqs := make(map[string]egoscale.RevokeSecurityGroupIngress, ids.Len())

	for _, identifier := range ids.List() {
		metas := strings.SplitN(identifier.(string), "_", 2)

		id, err := egoscale.ParseUUID(metas[0])
		if err != nil {
			return nil, err
		}

		reqs[identifier.(string)] = egoscale.RevokeSecurityGroupIngress{
			ID: id,
		}
	}

	return reqs, nil
}

// ruleToAuthorize converts a rule (or rules) into a list of authorize requests.
func ruleToAuthorize(ctx context.Context, client *egoscale.Client, rule map[string]interface{}) ([]egoscale.AuthorizeSecurityGroupIngress, error) {
	description := rule["description"].(string)
	protocol := rule["protocol"].(string)

	rs := []egoscale.AuthorizeSecurityGroupIngress{}

	req := egoscale.AuthorizeSecurityGroupIngress{
		Description: description,
	}

	if strings.HasPrefix(protocol, "ICMP") {
		req.Protocol = protocol
		req.IcmpType = uint8(rule["icmp_type"].(int))
		req.IcmpCode = uint8(rule["icmp_code"].(int))
		rs = append(rs, req)
	} else {
		ports := preparePorts(rule["ports"].(*schema.Set))
		for _, portRange := range ports {
			req.Protocol = strings.ToLower(protocol)
			req.StartPort = portRange[0]
			req.EndPort = portRange[1]

			rs = append(rs, req)
		}
	}

	reqs := []egoscale.AuthorizeSecurityGroupIngress{}

	cidrSet := rule["cidr_list"].(*schema.Set)
	for _, req := range rs {
		for _, c := range cidrSet.List() {
			cidr, err := egoscale.ParseCIDR(c.(string))

			if err != nil {
				return nil, err
			}

			req.CIDRList = []egoscale.CIDR{*cidr}
			reqs = append(reqs, req)
		}
		req.CIDRList = []egoscale.CIDR{}
	}

	userSecurityGroupSet := rule["user_security_group_list"].(*schema.Set)
	for _, req := range rs {
		for _, u := range userSecurityGroupSet.List() {
			_, err := egoscale.ParseUUID(u.(string))
			if err == nil {
				return nil, fmt.Errorf("user_security_group_list must be referenced by name only, got ID %q", u.(string))
			}

			sg := &egoscale.SecurityGroup{
				Name: u.(string),
			}

			resp, err := client.GetWithContext(ctx, sg)
			if err != nil {
				return nil, err
			}

			sg = resp.(*egoscale.SecurityGroup)
			req.UserSecurityGroupList = []egoscale.UserSecurityGroup{sg.UserSecurityGroup()}
			reqs = append(reqs, req)
		}
	}

	return reqs, nil
}
