package sks_cluster

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	yaml "gopkg.in/yaml.v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const defaultSKSKubeconfigTTLSeconds = 30 * 24 * 3600

const markdownDescriptionResourceKubeconfig = `Manage Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) Credentials (Kubeconfig).`

var _ resource.Resource = &ResourceKubeconfig{}
var _ resource.ResourceWithModifyPlan = &ResourceKubeconfig{}

type ResourceKubeconfig struct {
	client *exoscale.Client
}

func NewResourceKubeconfig() resource.Resource {
	return &ResourceKubeconfig{}
}

type ResourceKubeconfigModel struct {
	ID                  types.String `tfsdk:"id"`
	ClusterID           types.String `tfsdk:"cluster_id"`
	EarlyRenewalSeconds types.Int64  `tfsdk:"early_renewal_seconds"`
	Groups              types.Set    `tfsdk:"groups"`
	Kubeconfig          types.String `tfsdk:"kubeconfig"`
	ReadyForRenewal     types.Bool   `tfsdk:"ready_for_renewal"`
	TTLSeconds          types.Int64  `tfsdk:"ttl_seconds"`
	User                types.String `tfsdk:"user"`
	Zone                types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceKubeconfig) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sks_kubeconfig"
}

func (r *ResourceKubeconfig) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale Scalable Kubernetes Service (SKS) Credentials (Kubeconfig).",
		MarkdownDescription: markdownDescriptionResourceKubeconfig,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The ID of this resource.",
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Description:         "❗ The parent exoscale_sks_cluster ID.",
				MarkdownDescription: "❗ The parent [exoscale_sks_cluster](./sks_cluster.md) ID.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"early_renewal_seconds": schema.Int64Attribute{
				Description:         "If set, the resource will consider the Kubeconfig to have expired the given number of seconds before its actual CA certificate or client certificate expiry time. This can be useful to deploy an updated Kubeconfig in advance of the expiration of its internal current certificate. Note however that the old certificate remains valid until its true expiration time since this resource does not (and cannot) support revocation. Also note this advance update can only take place if the Terraform configuration is applied during the early renewal period (seconds; default: 0).",
				MarkdownDescription: "If set, the resource will consider the Kubeconfig to have expired the given number of seconds before its actual CA certificate or client certificate expiry time. This can be useful to deploy an updated Kubeconfig in advance of the expiration of its internal current certificate. Note however that the old certificate remains valid until its true expiration time since this resource does not (and cannot) support revocation. Also note this advance update can only take place if the Terraform configuration is applied during the early renewal period (seconds; default: 0).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"groups": schema.SetAttribute{
				Description:         "❗ Group names in the generated Kubeconfig. The certificate present in the Kubeconfig will have these roles set in the Organization field.",
				MarkdownDescription: "❗ Group names in the generated Kubeconfig. The certificate present in the Kubeconfig will have these roles set in the Organization field.",
				ElementType:         types.StringType,
				Required:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"kubeconfig": schema.StringAttribute{
				Description:         "The generated Kubeconfig (YAML content).",
				MarkdownDescription: "The generated Kubeconfig (YAML content).",
				Computed:            true,
				Sensitive:           true,
			},
			"ready_for_renewal": schema.BoolAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"ttl_seconds": schema.Int64Attribute{
				Description:         fmt.Sprintf("❗ The Time-to-Live of the Kubeconfig, after which it will expire / become invalid (seconds; default: %d = 30 days).", defaultSKSKubeconfigTTLSeconds),
				MarkdownDescription: fmt.Sprintf("❗ The Time-to-Live of the Kubeconfig, after which it will expire / become invalid (seconds; default: %d = 30 days).", defaultSKSKubeconfigTTLSeconds),
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(defaultSKSKubeconfigTTLSeconds),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"user": schema.StringAttribute{
				Description:         "❗ User name in the generated Kubeconfig. The certificate present in the Kubeconfig will also have this name set for the CN field.",
				MarkdownDescription: "❗ User name in the generated Kubeconfig. The certificate present in the Kubeconfig will also have this name set for the CN field.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"zone": schema.StringAttribute{
				Description:         "❗ The Exoscale zone name.",
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
		},

		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceKubeconfig) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceKubeconfig) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceKubeconfigModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(plan.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	groups := []string{}
	resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groups, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	generateResp, err := client.GenerateSKSClusterKubeconfig(ctx, exoscale.UUID(plan.ClusterID.ValueString()), exoscale.SKSKubeconfigRequest{
		User:   plan.User.ValueString(),
		Groups: groups,
		Ttl:    plan.TTLSeconds.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when generating SKS cluster kubeconfig", err.Error())
		return
	}

	kubeconfig, err := base64.StdEncoding.DecodeString(generateResp.Kubeconfig)
	if err != nil {
		resp.Diagnostics.AddError("unable to decode kubeconfig content", err.Error())
		return
	}

	id, err := kubeconfigToID(string(kubeconfig))
	if err != nil {
		resp.Diagnostics.AddError("unable to generate kubeconfig ID", err.Error())
		return
	}

	plan.ID = types.StringValue(id)
	plan.Kubeconfig = types.StringValue(string(kubeconfig))
	plan.ReadyForRenewal = types.BoolValue(false)

	tflog.Debug(ctx, "create finished successfully", map[string]any{"id": plan.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is a no-op: there is no API to fetch an existing Kubeconfig back (only
// to generate a new one), so the resource state is left untouched between
// applies. Renewal is instead driven by ModifyPlan/RequiresReplace.
func (r *ResourceKubeconfig) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceKubeconfigModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only ever applies to attributes that don't require replacement
// (currently just early_renewal_seconds): the Kubeconfig itself is never
// regenerated in place.
func (r *ResourceKubeconfig) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceKubeconfigModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "update finished successfully", map[string]any{"id": plan.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op API-wise: there is no revocation support, so we rely on
// the client certificate's own expiration and simply drop the resource from
// state.
func (r *ResourceKubeconfig) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceKubeconfigModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]any{"id": state.ID.ValueString()})
}

// ModifyPlan replicates the former SDKv2 CustomizeDiff behaviour: it flags
// ready_for_renewal and forces replacement once the Kubeconfig's embedded
// certificates are within their early renewal period of expiring.
func (r *ResourceKubeconfig) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		// Create or destroy: nothing to reconcile against.
		return
	}

	var state, plan ResourceKubeconfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kubeconfig := state.Kubeconfig.ValueString()

	clusterCerts, clientCerts, err := KubeconfigExtractCertificates(kubeconfig)
	if err != nil {
		resp.Diagnostics.AddError("unable to parse kubeconfig certificates", err.Error())
		return
	}

	readyForRenewal := len(kubeconfig) == 0
	if !readyForRenewal {
		now := time.Now()
		earlyRenewalPeriod := time.Duration(-plan.EarlyRenewalSeconds.ValueInt64()) * time.Second

		for _, certificate := range append(clusterCerts, clientCerts...) {
			if certificate.NotAfter.Add(earlyRenewalPeriod).Sub(now) <= 0 {
				readyForRenewal = true
				break
			}
		}
	}

	if readyForRenewal {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("ready_for_renewal"), types.BoolValue(true))...)
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("ready_for_renewal"))
	}
}

// KubeconfigExtractCertificates parses a Kubeconfig's YAML content and
// returns the cluster (CA) and client certificates it embeds.
func KubeconfigExtractCertificates(kubeconfig string) ([]*x509.Certificate, []*x509.Certificate, error) {
	if len(kubeconfig) == 0 {
		return []*x509.Certificate{}, []*x509.Certificate{}, nil
	}

	var kubeconfigData struct {
		Clusters []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data"`
			} `yaml:"cluster"`
		} `yaml:"clusters"`
		Users []struct {
			User struct {
				ClientCertificateData string `yaml:"client-certificate-data"`
			} `yaml:"user"`
		} `yaml:"users"`
	}

	if err := yaml.Unmarshal([]byte(kubeconfig), &kubeconfigData); err != nil {
		return nil, nil, fmt.Errorf("error decoding kubeconfig certificates: %w", err)
	}

	clusterCertificates := make([]*x509.Certificate, 0, len(kubeconfigData.Clusters))
	for _, cluster := range kubeconfigData.Clusters {
		parsedCertificate, err := kubeconfigRawPEMDataToCertificate(cluster.Cluster.CertificateAuthorityData)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to read cluster CA certificate: %w", err)
		}

		clusterCertificates = append(clusterCertificates, parsedCertificate)
	}

	clientCertificates := make([]*x509.Certificate, 0, len(kubeconfigData.Users))
	for _, user := range kubeconfigData.Users {
		parsedCertificate, err := kubeconfigRawPEMDataToCertificate(user.User.ClientCertificateData)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to read client certificate: %w", err)
		}

		clientCertificates = append(clientCertificates, parsedCertificate)
	}

	return clusterCertificates, clientCertificates, nil
}

// kubeconfigRawPEMDataToCertificate decodes a base64-encoded PEM certificate,
// as embedded in a Kubeconfig file.
func kubeconfigRawPEMDataToCertificate(b64PEMData string) (*x509.Certificate, error) {
	rawPEMData, err := base64.StdEncoding.DecodeString(b64PEMData)
	if err != nil {
		return nil, fmt.Errorf("error decoding base64 kubeconfig certificate: %w", err)
	}

	parsedPEMData, _ := pem.Decode(rawPEMData)
	parsedCertificate, err := x509.ParseCertificate(parsedPEMData.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse kubeconfig x509 certificate: %w", err)
	}

	return parsedCertificate, nil
}

// kubeconfigToID derives a stable resource ID from a Kubeconfig's embedded
// certificate serial numbers.
func kubeconfigToID(kubeconfig string) (string, error) {
	clusterCertificates, clientCertificates, err := KubeconfigExtractCertificates(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("unable to extract certificates from kubeconfig: %w", err)
	}

	certificateIDs := make([]string, 0, len(clusterCertificates)+len(clientCertificates))
	for _, cert := range append(clusterCertificates, clientCertificates...) {
		certificateIDs = append(certificateIDs, cert.SerialNumber.String())
	}

	return strings.Join(certificateIDs, ":"), nil
}
