package exoscale

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	resSecurityGroupRuleAttrNetwork               = "cidr"
	resSecurityGroupRuleAttrDescription           = "description"
	resSecurityGroupRuleAttrEndPort               = "end_port"
	resSecurityGroupRuleAttrFlowDirection         = "type"
	resSecurityGroupRuleAttrICMPCode              = "icmp_code"
	resSecurityGroupRuleAttrICMPType              = "icmp_type"
	resSecurityGroupRuleAttrProtocol              = "protocol"
	resSecurityGroupRuleAttrPublicSecurityGroup   = "public_security_group"
	resSecurityGroupRuleAttrSecurityGroupID       = "security_group_id"
	resSecurityGroupRuleAttrSecurityGroupName     = "security_group"
	resSecurityGroupRuleAttrStartPort             = "start_port"
	resSecurityGroupRuleAttrUserSecurityGroupID   = "user_security_group_id"
	resSecurityGroupRuleAttrUserSecurityGroupName = "user_security_group"
)

var securityGroupRuleProtocols = []string{
	"AH",
	"ALL",
	"ESP",
	"GRE",
	"ICMP",
	"ICMPv6",
	"IPIP",
	"TCP",
	"UDP",
}

func resourceSecurityGroupRuleIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_security_group_rule")
}

func resourceSecurityGroupRule() *schema.Resource {
	return &schema.Resource{
		Description: "Manage Exoscale [Security Group](https://community.exoscale.com/documentation/compute/security-groups/) Rules.",
		Schema: map[string]*schema.Schema{
			resSecurityGroupRuleAttrDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "A free-form text describing the security group rule.",
			},
			resSecurityGroupRuleAttrEndPort: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(1, 65535),
				ConflictsWith: []string{
					resSecurityGroupRuleAttrICMPCode,
					resSecurityGroupRuleAttrICMPType,
				},
				Description: "A `TCP`/`UDP` port range to match.",
			},
			resSecurityGroupRuleAttrFlowDirection: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"INGRESS", "EGRESS"}, false),
				Description:  "The traffic direction to match (`INGRESS` or `EGRESS`).",
			},
			resSecurityGroupRuleAttrICMPCode: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(0, 255),
				ConflictsWith: []string{
					resSecurityGroupRuleAttrEndPort,
					resSecurityGroupRuleAttrStartPort,
				},
				Description: "An ICMP/ICMPv6 [type/code](https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol#Control_messages) to match.",
			},
			resSecurityGroupRuleAttrICMPType: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(0, 255),
				ConflictsWith: []string{
					resSecurityGroupRuleAttrEndPort,
					resSecurityGroupRuleAttrStartPort,
				},
				Description: "An ICMP/ICMPv6 [type/code](https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol#Control_messages) to match.",
			},
			resSecurityGroupRuleAttrNetwork: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsCIDRNetwork(0, 128),
				ConflictsWith: []string{
					resSecurityGroupRuleAttrPublicSecurityGroup,
					resSecurityGroupRuleAttrUserSecurityGroupID,
					resSecurityGroupRuleAttrUserSecurityGroupName,
				},
				Description: "An (`INGRESS`) source / (`EGRESS`) destination IP subnet (in [CIDR notation](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notation)) to match (conflicts with `public_security_group`/`user_security_group`/`user_security_group_id`).",
			},
			resSecurityGroupRuleAttrProtocol: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "TCP",
				ValidateFunc: validation.StringInSlice(securityGroupRuleProtocols, true),
				// Ignore case differences
				DiffSuppressFunc: suppressCaseDiff,
				Description:      "The network protocol to match (`TCP`, `UDP`, `ICMP`, `ICMPv6`, `AH`, `ESP`, `GRE`, `IPIP` or `ALL`)",
			},
			resSecurityGroupRuleAttrPublicSecurityGroup: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Description: "An (`INGRESS`) source / (`EGRESS`) destination public security group name to match (conflicts with `cidr`/`user_security_group`/`user_security_group_id`). Please use the `user_security_group_id` argument along the [exoscale_security_group](../data-sources/security_group.md) data source instead.",
			},
			resSecurityGroupRuleAttrSecurityGroupID: {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{resSecurityGroupRuleAttrSecurityGroupName},
				Description:   "The parent [exoscale_security_group](./security_group.md) ID.",
			},
			resSecurityGroupRuleAttrSecurityGroupName: {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{resSecurityGroupRuleAttrSecurityGroupID},
				Deprecated:    "Deprecated in favor of `security_group_id`",
				Description:   "The parent security group name. Please use the `security_group_id` argument along the [exoscale_security_group](../data-sources/security_group.md) data source instead.",
			},
			resSecurityGroupRuleAttrStartPort: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(1, 65535),
				ConflictsWith: []string{
					resSecurityGroupRuleAttrICMPCode,
					resSecurityGroupRuleAttrICMPType,
				},
				Description: "A `TCP`/`UDP` port range to match.",
			},
			resSecurityGroupRuleAttrUserSecurityGroupID: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ConflictsWith: []string{
					resSecurityGroupRuleAttrNetwork,
					resSecurityGroupRuleAttrPublicSecurityGroup,
					resSecurityGroupRuleAttrUserSecurityGroupName,
				},
				Description: "An (`INGRESS`) source / (`EGRESS`) destination security group ID to match (conflicts with `cidr`/`public_security_group`/`user_security_group)`).",
			},
			resSecurityGroupRuleAttrUserSecurityGroupName: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ConflictsWith: []string{
					resSecurityGroupRuleAttrNetwork,
					resSecurityGroupRuleAttrPublicSecurityGroup,
					resSecurityGroupRuleAttrUserSecurityGroupID,
				},
				Deprecated:  "Deprecated in favor of `user_security_group_id`",
				Description: "An (`INGRESS`) source / (`EGRESS`) destination security group name to match (conflicts with `cidr`/`public_security_group`/`user_security_group_id`). Please use the `user_security_group_id` argument along the [exoscale_security_group](../data-sources/security_group.md) data source instead.",
			},
		},

		CreateContext: resourceSecurityGroupRuleCreate,
		ReadContext:   resourceSecurityGroupRuleRead,
		DeleteContext: resourceSecurityGroupRuleDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
				parts := strings.SplitN(d.Id(), "/", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf(`invalid ID %q, expected format "<SECURITY-GROUP-ID>/<SECURITY-GROUP-RULE-ID>"`, d.Id())
				}

				d.SetId(parts[1])
				if err := d.Set(resSecurityGroupRuleAttrSecurityGroupID, parts[0]); err != nil {
					return nil, err
				}

				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSecurityGroupRuleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceSecurityGroupRuleIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroupID, bySecurityGroupID := d.GetOk(resSecurityGroupRuleAttrSecurityGroupID)
	securityGroupName, bySecurityGroupName := d.GetOk(resSecurityGroupRuleAttrSecurityGroupName)
	if !bySecurityGroupID && !bySecurityGroupName {
		return diag.Errorf(
			"either %s or %s must be specified",
			resSecurityGroupRuleAttrSecurityGroupName,
			resSecurityGroupRuleAttrSecurityGroupID,
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

	securityGroupRule := &egoscale.SecurityGroupRule{
		Description:   nonEmptyStringPtr(d.Get(resSecurityGroupRuleAttrDescription).(string)),
		FlowDirection: nonEmptyStringPtr(strings.ToLower(d.Get(resSecurityGroupRuleAttrFlowDirection).(string))),
		Protocol:      nonEmptyStringPtr(strings.ToLower(d.Get(resSecurityGroupRuleAttrProtocol).(string))),
	}

	if v, ok := d.GetOk(resSecurityGroupRuleAttrEndPort); ok && v.(int) > 0 {
		port := uint16(v.(int))
		securityGroupRule.EndPort = &port
	}

	if strings.HasPrefix(*securityGroupRule.Protocol, "icmp") {
		icmpCode := int64(d.Get(resSecurityGroupRuleAttrICMPCode).(int))
		icmpType := int64(d.Get(resSecurityGroupRuleAttrICMPType).(int))
		securityGroupRule.ICMPCode = &icmpCode
		securityGroupRule.ICMPType = &icmpType
	}

	network, byNetwork := d.GetOk(resSecurityGroupRuleAttrNetwork)
	userSecurityGroupID, byUserSecurityGroupID := d.GetOk(resSecurityGroupRuleAttrUserSecurityGroupID)
	userSecurityGroupName, byUserSecurityGroupName := d.GetOk(resSecurityGroupRuleAttrUserSecurityGroupName)
	publicSecurityGroupName, byPublicSecurityGroupName := d.GetOk(resSecurityGroupRuleAttrPublicSecurityGroup)

	if !byNetwork && !byUserSecurityGroupID && !byUserSecurityGroupName && !byPublicSecurityGroupName {
		return diag.Errorf(
			"either %s or %s or %s or %s must be specified",
			resSecurityGroupRuleAttrNetwork,
			resSecurityGroupRuleAttrUserSecurityGroupID,
			resSecurityGroupRuleAttrUserSecurityGroupName,
			resSecurityGroupRuleAttrPublicSecurityGroup,
		)
	}

	if byNetwork {
		_, cidr, err := net.ParseCIDR(network.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		securityGroupRule.Network = cidr
	} else if byPublicSecurityGroupName {
		securityGroupName := publicSecurityGroupName.(string)
		securityGroupRule.SecurityGroupName = &securityGroupName
	} else {
		userSecurityGroup, err := client.FindSecurityGroup(
			ctx,
			zone, func() string {
				if byUserSecurityGroupID {
					return userSecurityGroupID.(string)
				}
				return userSecurityGroupName.(string)
			}(),
		)
		if err != nil {
			return diag.Errorf("unable to retrieve Security Group: %v", err)
		}
		securityGroupRule.SecurityGroupID = userSecurityGroup.ID
	}

	if v, ok := d.GetOk(resSecurityGroupRuleAttrStartPort); ok && v.(int) > 0 {
		port := uint16(v.(int))
		securityGroupRule.StartPort = &port
	}

	securityGroupRule, err = client.CreateSecurityGroupRule(ctx, zone, securityGroup, securityGroupRule)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*securityGroupRule.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceSecurityGroupRuleIDString(d),
	})

	return resourceSecurityGroupRuleRead(ctx, d, meta)
}

func resourceSecurityGroupRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceSecurityGroupRuleIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup, err := client.FindSecurityGroup(
		ctx,
		zone, func() string {
			if v, ok := d.GetOk(resSecurityGroupRuleAttrSecurityGroupID); ok {
				return v.(string)
			} else {
				return d.Get(resSecurityGroupRuleAttrSecurityGroupName).(string)
			}
		}(),
	)
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Parent Security Group doesn't exist anymore, so does the Security Group rule.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var securityGroupRule *egoscale.SecurityGroupRule
	for _, r := range securityGroup.Rules {
		if *r.ID == d.Id() {
			securityGroupRule = r
			break
		}
	}
	if securityGroupRule == nil {
		// Resource doesn't exist anymore, signaling the core to remove it from the state.
		d.SetId("")
		return nil
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceSecurityGroupRuleIDString(d),
	})

	return diag.FromErr(resourceSecurityGroupRuleApply(ctx, d, meta, securityGroup, securityGroupRule))
}

func resourceSecurityGroupRuleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceSecurityGroupRuleIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroup, err := client.FindSecurityGroup(
		ctx,
		zone, func() string {
			if v, ok := d.GetOk(resSecurityGroupRuleAttrSecurityGroupID); ok {
				return v.(string)
			} else {
				return d.Get(resSecurityGroupRuleAttrSecurityGroupName).(string)
			}
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	securityGroupRuleID := d.Id()
	if err := client.DeleteSecurityGroupRule(
		ctx,
		zone,
		securityGroup,
		&egoscale.SecurityGroupRule{ID: &securityGroupRuleID},
	); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceSecurityGroupRuleIDString(d),
	})

	return nil
}

func resourceSecurityGroupRuleApply(
	ctx context.Context,
	d *schema.ResourceData,
	meta interface{},
	securityGroup *egoscale.SecurityGroup,
	securityGroupRule *egoscale.SecurityGroupRule,
) error {
	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	if err := d.Set(
		resSecurityGroupRuleAttrDescription,
		defaultString(securityGroupRule.Description, ""),
	); err != nil {
		return err
	}

	if securityGroupRule.EndPort != nil {
		if err := d.Set(resSecurityGroupRuleAttrEndPort, *securityGroupRule.EndPort); err != nil {
			return err
		}
	}

	if err := d.Set(
		resSecurityGroupRuleAttrFlowDirection,
		strings.ToUpper(*securityGroupRule.FlowDirection),
	); err != nil {
		return err
	}

	if securityGroupRule.ICMPCode != nil {
		if err := d.Set(resSecurityGroupRuleAttrICMPCode, *securityGroupRule.ICMPCode); err != nil {
			return err
		}
	}

	if securityGroupRule.ICMPType != nil {
		if err := d.Set(resSecurityGroupRuleAttrICMPType, *securityGroupRule.ICMPType); err != nil {
			return err
		}
	}

	if securityGroupRule.Network != nil {
		if err := d.Set(resSecurityGroupRuleAttrNetwork, securityGroupRule.Network.String()); err != nil {
			return err
		}
	}

	protocol := strings.ReplaceAll(
		strings.ToUpper(*securityGroupRule.Protocol),
		"V6",
		"v6",
	)
	if err := d.Set(resSecurityGroupRuleAttrProtocol, protocol); err != nil {
		return err
	}

	if err := d.Set(resSecurityGroupRuleAttrSecurityGroupID, *securityGroup.ID); err != nil {
		return err
	}

	if err := d.Set(resSecurityGroupRuleAttrSecurityGroupName, *securityGroup.Name); err != nil {
		return err
	}

	if securityGroupRule.StartPort != nil {
		if err := d.Set(resSecurityGroupRuleAttrStartPort, *securityGroupRule.StartPort); err != nil {
			return err
		}
	}

	if securityGroupRule.SecurityGroupID != nil {
		userSecurityGroup, err := client.GetSecurityGroup(ctx, zone, *securityGroupRule.SecurityGroupID)
		if err != nil {
			return fmt.Errorf(
				"unable to retrieve Security Group %s: %w",
				*securityGroupRule.SecurityGroupID,
				err,
			)
		}

		if err := d.Set(resSecurityGroupRuleAttrUserSecurityGroupID, *userSecurityGroup.ID); err != nil {
			return err
		}
		if err := d.Set(resSecurityGroupRuleAttrUserSecurityGroupName, *userSecurityGroup.Name); err != nil {
			return err
		}
	}
	if securityGroupRule.SecurityGroupName != nil {
		if err := d.Set(resSecurityGroupRuleAttrPublicSecurityGroup, *securityGroupRule.SecurityGroupName); err != nil {
			return err
		}
	}

	return nil
}
