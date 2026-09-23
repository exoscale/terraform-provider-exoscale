package sks

import (
	"context"
	"errors"
	"fmt"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const (
	defaultSKSClusterCNI              = "calico"
	defaultSKSClusterServiceLevel     = "pro"
	defaultSKSClusterAuditInitBackoff = "10s"

	sksClusterAddonExoscaleCCM = "exoscale-cloud-controller"
	sksClusterAddonExoscaleCSI = "exoscale-container-storage-interface"
	sksClusterAddonMS          = "metrics-server"
	sksClusterAddonKarpenter   = "karpenter"
)

const markdownDescriptionResourceCluster = `Manage Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) Clusters.

Corresponding data source: [exoscale_sks_cluster](../data-sources/sks_cluster.md).`

var _ resource.Resource = &ResourceCluster{}
var _ resource.ResourceWithImportState = &ResourceCluster{}

type ResourceCluster struct {
	client *exoscale.Client
}

func NewResourceCluster() resource.Resource {
	return &ResourceCluster{}
}

// oidcModel maps the "oidc {}" nested block.
type oidcModel struct {
	ClientID       types.String `tfsdk:"client_id"`
	GroupsClaim    types.String `tfsdk:"groups_claim"`
	GroupsPrefix   types.String `tfsdk:"groups_prefix"`
	IssuerURL      types.String `tfsdk:"issuer_url"`
	RequiredClaim  types.Map    `tfsdk:"required_claim"`
	UsernameClaim  types.String `tfsdk:"username_claim"`
	UsernamePrefix types.String `tfsdk:"username_prefix"`
}

func oidcAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"client_id":       types.StringType,
		"groups_claim":    types.StringType,
		"groups_prefix":   types.StringType,
		"issuer_url":      types.StringType,
		"required_claim":  types.MapType{ElemType: types.StringType},
		"username_claim":  types.StringType,
		"username_prefix": types.StringType,
	}
}

type auditModel struct {
	Enabled        types.Bool   `tfsdk:"enabled"`
	Endpoint       types.String `tfsdk:"endpoint"`
	InitialBackoff types.String `tfsdk:"initial_backoff"`
	BearerToken    types.String `tfsdk:"bearer_token"`
}

func auditAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":         types.BoolType,
		"endpoint":        types.StringType,
		"initial_backoff": types.StringType,
		"bearer_token":    types.StringType,
	}
}

type ResourceClusterModel struct {
	ID                         types.String `tfsdk:"id"`
	Zone                       types.String `tfsdk:"zone"`
	Name                       types.String `tfsdk:"name"`
	Description                types.String `tfsdk:"description"`
	Labels                     types.Map    `tfsdk:"labels"`
	AggregationLayerCA         types.String `tfsdk:"aggregation_ca"`
	ControlPlaneCA             types.String `tfsdk:"control_plane_ca"`
	KubeletCA                  types.String `tfsdk:"kubelet_ca"`
	AutoUpgrade                types.Bool   `tfsdk:"auto_upgrade"`
	CNI                        types.String `tfsdk:"cni"`
	CreatedAt                  types.String `tfsdk:"created_at"`
	CreateDefaultSecurityGroup types.Bool   `tfsdk:"create_default_security_group"`
	DefaultSecurityGroupID     types.String `tfsdk:"default_security_group_id"`
	EnableKubeProxy            types.Bool   `tfsdk:"enable_kube_proxy"`
	EnableKarpenter            types.Bool   `tfsdk:"enable_karpenter"`
	Endpoint                   types.String `tfsdk:"endpoint"`
	ExoscaleCCM                types.Bool   `tfsdk:"exoscale_ccm"`
	ExoscaleCSI                types.Bool   `tfsdk:"exoscale_csi"`
	MetricsServer              types.Bool   `tfsdk:"metrics_server"`
	FeatureGates               types.Set    `tfsdk:"feature_gates"`
	Nodepools                  types.Set    `tfsdk:"nodepools"`
	ServiceLevel               types.String `tfsdk:"service_level"`
	State                      types.String `tfsdk:"state"`
	Version                    types.String `tfsdk:"version"`

	Oidc  types.List `tfsdk:"oidc"`
	Audit types.List `tfsdk:"audit"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceCluster) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sks_cluster"
}

func (r *ResourceCluster) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale Scalable Kubernetes Service (SKS) Clusters.",
		MarkdownDescription: markdownDescriptionResourceCluster,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The SKS cluster ID.",
				MarkdownDescription: "The SKS cluster ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"name": schema.StringAttribute{
				Description:         "The SKS cluster name.",
				MarkdownDescription: "The SKS cluster name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A free-form text describing the cluster.",
				MarkdownDescription: "A free-form text describing the cluster.",
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"aggregation_ca": schema.StringAttribute{
				Description:         "The CA certificate (in PEM format) for TLS communications between the control plane and the aggregation layer (e.g. 'metrics-server').",
				MarkdownDescription: "The CA certificate (in PEM format) for TLS communications between the control plane and the aggregation layer (e.g. `metrics-server`).",
				Computed:            true,
				Sensitive:           true,
			},
			"auto_upgrade": schema.BoolAttribute{
				Description:         "Enable automatic upgrading of the control plane version.",
				MarkdownDescription: "Enable automatic upgrading of the control plane version.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"cni": schema.StringAttribute{
				Description:         fmt.Sprintf(`The CNI plugin that is to be used. Available options are "calico" or "cilium". Defaults to %q. Setting empty string will result in a cluster with no CNI.`, defaultSKSClusterCNI),
				MarkdownDescription: fmt.Sprintf(`The CNI plugin that is to be used. Available options are "calico" or "cilium". Defaults to %q. Setting empty string will result in a cluster with no CNI.`, defaultSKSClusterCNI),
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaultSKSClusterCNI),
			},
			"control_plane_ca": schema.StringAttribute{
				Description:         "The CA certificate (in PEM format) for TLS communications between control plane components.",
				MarkdownDescription: "The CA certificate (in PEM format) for TLS communications between control plane components.",
				Computed:            true,
				Sensitive:           true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The cluster creation date.",
				MarkdownDescription: "The cluster creation date.",
				Computed:            true,
			},
			"create_default_security_group": schema.BoolAttribute{
				Description:         "❗ Creates an ad-hoc security group based on the choice of the selected CNI (may only be set at creation time).",
				MarkdownDescription: "❗ Creates an ad-hoc security group based on the choice of the selected CNI (may only be set at creation time).",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"default_security_group_id": schema.StringAttribute{
				Description:         "The ID of the cluster's ad-hoc default security group (when `create_default_security_group` was set at creation time).",
				MarkdownDescription: "The ID of the cluster's ad-hoc default security group (when `create_default_security_group` was set at creation time).",
				Computed:            true,
			},
			"enable_kube_proxy": schema.BoolAttribute{
				Description:         "❗ Indicates whether to deploy the Kubernetes network proxy. (may only be set at creation time)",
				MarkdownDescription: "❗ Indicates whether to deploy the Kubernetes network proxy. (may only be set at creation time)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"enable_karpenter": schema.BoolAttribute{
				Description:         "Indicates whether to deploy Karpenter for cluster autoscaling.",
				MarkdownDescription: "Indicates whether to deploy Karpenter for cluster autoscaling.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"endpoint": schema.StringAttribute{
				Description:         "The cluster API endpoint.",
				MarkdownDescription: "The cluster API endpoint.",
				Computed:            true,
			},
			"exoscale_ccm": schema.BoolAttribute{
				Description:         "Deploy the Exoscale Cloud Controller Manager in the control plane (boolean; default: 'true'; may only be set at creation time).",
				MarkdownDescription: "Deploy the Exoscale [Cloud Controller Manager](https://github.com/exoscale/exoscale-cloud-controller-manager/) in the control plane (boolean; default: `true`; may only be set at creation time).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"exoscale_csi": schema.BoolAttribute{
				Description:         "Deploy the Exoscale Container Storage Interface on worker nodes (boolean; default: 'false'; requires the CCM to be enabled).",
				MarkdownDescription: "Deploy the Exoscale [Container Storage Interface](https://github.com/exoscale/exoscale-csi-driver/) on worker nodes (boolean; default: `false`; requires the CCM to be enabled).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"feature_gates": schema.SetAttribute{
				Description:         "Feature gates options for the cluster.",
				MarkdownDescription: "Feature gates options for the cluster.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
			"kubelet_ca": schema.StringAttribute{
				Description:         "The CA certificate (in PEM format) for TLS communications between kubelets and the control plane.",
				MarkdownDescription: "The CA certificate (in PEM format) for TLS communications between kubelets and the control plane.",
				Computed:            true,
				Sensitive:           true,
			},
			"metrics_server": schema.BoolAttribute{
				Description:         "Deploy the Kubernetes Metrics Server in the control plane (boolean; default: 'true'; may only be set at creation time).",
				MarkdownDescription: "Deploy the [Kubernetes Metrics Server](https://github.com/kubernetes-sigs/metrics-server/) in the control plane (boolean; default: `true`; may only be set at creation time).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"nodepools": schema.SetAttribute{
				Description:         "The list of exoscale_sks_nodepool (IDs) attached to the cluster.",
				MarkdownDescription: "The list of [exoscale_sks_nodepool](./sks_nodepool.md) (IDs) attached to the cluster.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"service_level": schema.StringAttribute{
				Description:         "The service level of the control plane ('pro' or 'starter'; default: 'pro'; may only be set at creation time).",
				MarkdownDescription: "The service level of the control plane (`pro` or `starter`; default: `pro`; may only be set at creation time).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaultSKSClusterServiceLevel),
			},
			"state": schema.StringAttribute{
				Description:         "The cluster state.",
				MarkdownDescription: "The cluster state.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				Description:         "The version of the control plane (default: latest version available from the API; see 'exo compute sks versions' for reference; may only be set at creation time).",
				MarkdownDescription: "The version of the control plane (default: latest version available from the API; see `exo compute sks versions` for reference; may only be set at creation time).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},

		Blocks: map[string]schema.Block{
			"audit": schema.ListNestedBlock{
				Description:         "Parameters for Kubernetes Audit configuration (may only be enabled at creation time)",
				MarkdownDescription: "Parameters for Kubernetes Audit configuration (may only be enabled at creation time)",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"enabled": schema.BoolAttribute{
							Description:         "Whether to run the APIServer with the configured Kubernetes Audit",
							MarkdownDescription: "Whether to run the APIServer with the configured Kubernetes Audit",
							Optional:            true,
						},
						"endpoint": schema.StringAttribute{
							Description:         "The Endpoint URL for the Webserver responsible of processing Audit events",
							MarkdownDescription: "The Endpoint URL for the Webserver responsible of processing Audit events",
							Optional:            true,
						},
						"initial_backoff": schema.StringAttribute{
							Description:         "The Initial Backoff to wait before sending data to the remote server (default '10s')",
							MarkdownDescription: "The Initial Backoff to wait before sending data to the remote server (default `10s`)",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString(defaultSKSClusterAuditInitBackoff),
						},
						"bearer_token": schema.StringAttribute{
							Description:         "The optional bearer token to include in the request header",
							MarkdownDescription: "The optional bearer token to include in the request header",
							Optional:            true,
							Sensitive:           true,
						},
					},
				},
			},
			"oidc": schema.ListNestedBlock{
				Description:         "An OpenID Connect configuration to provide to the Kubernetes API server (may only be set at creation time). Structure is documented below.",
				MarkdownDescription: "An OpenID Connect configuration to provide to the Kubernetes API server (may only be set at creation time). Structure is documented below.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"client_id": schema.StringAttribute{
							Description:         "The OpenID client ID.",
							MarkdownDescription: "The OpenID client ID.",
							Required:            true,
						},
						"groups_claim": schema.StringAttribute{
							Description:         "An OpenID JWT claim to use as the user's group.",
							MarkdownDescription: "An OpenID JWT claim to use as the user's group.",
							Optional:            true,
						},
						"groups_prefix": schema.StringAttribute{
							Description:         "An OpenID prefix prepended to group claims.",
							MarkdownDescription: "An OpenID prefix prepended to group claims.",
							Optional:            true,
						},
						"issuer_url": schema.StringAttribute{
							Description:         "The OpenID provider URL.",
							MarkdownDescription: "The OpenID provider URL.",
							Required:            true,
						},
						"required_claim": schema.MapAttribute{
							Description:         "A map of key/value pairs that describes a required claim in the OpenID Token.",
							MarkdownDescription: "A map of key/value pairs that describes a required claim in the OpenID Token.",
							ElementType:         types.StringType,
							Optional:            true,
						},
						"username_claim": schema.StringAttribute{
							Description:         "An OpenID JWT claim to use as the user name.",
							MarkdownDescription: "An OpenID JWT claim to use as the user name.",
							Optional:            true,
						},
						"username_prefix": schema.StringAttribute{
							Description:         "An OpenID prefix prepended to username claims.",
							MarkdownDescription: "An OpenID prefix prepended to username claims.",
							Optional:            true,
						},
					},
				},
			},
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceCluster) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceCluster) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.SplitN(req.ID, "@", 2)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"invalid import ID",
			fmt.Sprintf(`invalid ID %q, expected format "<ID>@<ZONE>"`, req.ID),
		)
		return
	}

	zone := idParts[1]
	if !in(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
		return
	}

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var t timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &t)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &ResourceClusterModel{
		ID:           types.StringValue(idParts[0]),
		Zone:         types.StringValue(zone),
		Labels:       types.MapNull(types.StringType),
		FeatureGates: types.SetNull(types.StringType),
		Nodepools:    types.SetNull(types.StringType),
		Oidc:         types.ListNull(types.ObjectType{AttrTypes: oidcAttrTypes()}),
		Audit:        types.ListNull(types.ObjectType{AttrTypes: auditAttrTypes()}),
		Timeouts:     t,
	})...)
}

func (r *ResourceCluster) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceClusterModel

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

	createReq := exoscale.CreateSKSClusterRequest{}

	addOns := []string{}
	if plan.ExoscaleCCM.ValueBool() {
		addOns = append(addOns, sksClusterAddonExoscaleCCM)
	}
	if plan.MetricsServer.ValueBool() {
		addOns = append(addOns, sksClusterAddonMS)
	}
	if plan.ExoscaleCSI.ValueBool() {
		addOns = append(addOns, sksClusterAddonExoscaleCSI)
	}
	if plan.EnableKarpenter.ValueBool() {
		addOns = append(addOns, sksClusterAddonKarpenter)
	}
	if len(addOns) > 0 {
		createReq.Addons = addOns
	}

	if plan.AutoUpgrade.ValueBool() {
		v := true
		createReq.AutoUpgrade = &v
	}

	if !plan.EnableKubeProxy.IsNull() && !plan.EnableKubeProxy.IsUnknown() {
		v := plan.EnableKubeProxy.ValueBool()
		createReq.EnableKubeProxy = &v
	}

	if !plan.CreateDefaultSecurityGroup.IsNull() && !plan.CreateDefaultSecurityGroup.IsUnknown() {
		v := plan.CreateDefaultSecurityGroup.ValueBool()
		createReq.CreateDefaultSecurityGroup = &v
	}

	featureGates := []string{}
	if !plan.FeatureGates.IsNull() && !plan.FeatureGates.IsUnknown() {
		resp.Diagnostics.Append(plan.FeatureGates.ElementsAs(ctx, &featureGates, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	createReq.FeatureGates = featureGates

	if cni := plan.CNI.ValueString(); cni != "" {
		createReq.Cni = exoscale.CreateSKSClusterRequestCni(cni)
	}

	if description := plan.Description.ValueString(); description != "" {
		createReq.Description = &description
	}

	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() && len(plan.Labels.Elements()) > 0 {
		labels := exoscale.SKSClusterLabels{}
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Labels = labels
	}

	createReq.Name = plan.Name.ValueString()

	if level := plan.ServiceLevel.ValueString(); level != "" {
		createReq.Level = exoscale.CreateSKSClusterRequestLevel(level)
	}

	resolvedVersion, err := resolveSKSClusterVersion(ctx, client, plan.Version.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to resolve SKS cluster version", err.Error())
		return
	}
	createReq.Version = resolvedVersion

	// Audit
	auditBlock, ok := r.auditBlock(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if ok && auditBlock.Enabled.ValueBool() && auditBlock.Endpoint.ValueString() != "" {
		createReq.Audit = &exoscale.SKSAuditCreate{
			Endpoint: exoscale.SKSAuditEndpoint(auditBlock.Endpoint.ValueString()),
		}
		if v := auditBlock.BearerToken.ValueString(); v != "" {
			createReq.Audit.BearerToken = exoscale.SKSAuditBearerToken(v)
		}
		if v := auditBlock.InitialBackoff.ValueString(); v != "" {
			createReq.Audit.InitialBackoff = exoscale.SKSAuditInitialBackoff(v)
		}
	}

	oidcBlock, ok := r.oidcBlock(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if ok && oidcBlock.ClientID.ValueString() != "" {
		createReq.Oidc = &exoscale.SKSOidc{
			ClientID: oidcBlock.ClientID.ValueString(),
		}
		if v := oidcBlock.GroupsClaim.ValueString(); v != "" {
			createReq.Oidc.GroupsClaim = v
		}
		if v := oidcBlock.IssuerURL.ValueString(); v != "" {
			createReq.Oidc.IssuerURL = v
		}
		if v := oidcBlock.GroupsPrefix.ValueString(); v != "" {
			createReq.Oidc.GroupsPrefix = v
		}
		if !oidcBlock.RequiredClaim.IsNull() && !oidcBlock.RequiredClaim.IsUnknown() {
			claims := map[string]string{}
			resp.Diagnostics.Append(oidcBlock.RequiredClaim.ElementsAs(ctx, &claims, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			createReq.Oidc.RequiredClaim = claims
		}
		if v := oidcBlock.UsernameClaim.ValueString(); v != "" {
			createReq.Oidc.UsernameClaim = v
		}
		if v := oidcBlock.UsernamePrefix.ValueString(); v != "" {
			createReq.Oidc.UsernamePrefix = v
		}
	}

	op, err := client.CreateSKSCluster(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when creating SKS cluster", err.Error())
		return
	}

	op, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create SKS cluster operation failed", err.Error())
		return
	}

	plan.ID = types.StringValue(op.Reference.ID.String())

	tflog.Debug(ctx, "create finished successfully", map[string]any{"id": plan.ID.ValueString()})

	if diags := r.applyCluster(ctx, client, exoscale.UUID(plan.ID.ValueString()), &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceCluster) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceClusterModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(state.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	_, err = client.GetSKSCluster(ctx, exoscale.UUID(state.ID.ValueString()))
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching SKS cluster", err.Error())
		return
	}

	if diags := r.applyCluster(ctx, client, exoscale.UUID(state.ID.ValueString()), &state); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "read finished successfully", map[string]any{"id": state.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceCluster) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResourceClusterModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, config.DefaultTimeout)
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

	clusterID := exoscale.UUID(state.ID.ValueString())

	// First check if we need to upgrade the cluster.
	oldVersion := state.Version.ValueString()
	resolvedNewVersion, err := resolveSKSClusterVersion(ctx, client, plan.Version.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to resolve SKS cluster version", err.Error())
		return
	}
	if resolvedNewVersion != oldVersion {
		if err := await(ctx, client)(client.UpgradeSKSCluster(ctx, clusterID, exoscale.UpgradeSKSClusterRequest{
			Version: resolvedNewVersion,
		})); err != nil {
			resp.Diagnostics.AddError("unable to upgrade SKS cluster", err.Error())
			return
		}
	}

	var updated bool
	updateReq := exoscale.UpdateSKSClusterRequest{}

	if !plan.FeatureGates.Equal(state.FeatureGates) {
		featureGates := []string{}
		if !plan.FeatureGates.IsNull() && !plan.FeatureGates.IsUnknown() {
			resp.Diagnostics.Append(plan.FeatureGates.ElementsAs(ctx, &featureGates, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		updateReq.FeatureGates = featureGates
		updated = true
	}

	if !plan.AutoUpgrade.Equal(state.AutoUpgrade) {
		autoUpgrade := plan.AutoUpgrade.ValueBool()
		updateReq.AutoUpgrade = &autoUpgrade
		updated = true
	}

	if !plan.Labels.Equal(state.Labels) {
		labels := exoscale.SKSClusterLabels{}
		if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
			resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		updateReq.Labels = labels
		updated = true
	}

	if !plan.Name.Equal(state.Name) {
		updateReq.Name = plan.Name.ValueString()
		updated = true
	}

	if !plan.Description.Equal(state.Description) {
		description := plan.Description.ValueString()
		updateReq.Description = &description
		updated = true
	}

	csiChanged := !plan.ExoscaleCSI.Equal(state.ExoscaleCSI)
	if csiChanged && !plan.ExoscaleCSI.ValueBool() {
		resp.Diagnostics.AddError("unable to update SKS cluster", "disabling the CSI addon is not supported")
		return
	}

	if csiChanged || !plan.EnableKarpenter.Equal(state.EnableKarpenter) {
		// The API expects the full list of addons. CCM and metrics-server can
		// only be set at creation time, so keep them as they currently are.
		addOns := []string{}
		if state.ExoscaleCCM.ValueBool() {
			addOns = append(addOns, sksClusterAddonExoscaleCCM)
		}
		if state.MetricsServer.ValueBool() {
			addOns = append(addOns, sksClusterAddonMS)
		}
		if plan.ExoscaleCSI.ValueBool() {
			addOns = append(addOns, sksClusterAddonExoscaleCSI)
		}
		if plan.EnableKarpenter.ValueBool() {
			addOns = append(addOns, sksClusterAddonKarpenter)
		}
		updateReq.Addons = addOns
		updated = true
	}

	if !plan.Audit.Equal(state.Audit) {
		var auditBlock auditModel
		if b, ok := r.auditBlock(ctx, plan, &resp.Diagnostics); ok {
			auditBlock = b
		}
		if resp.Diagnostics.HasError() {
			return
		}

		enableAudit := auditBlock.Enabled.ValueBool()
		updateReq.Audit = &exoscale.SKSAuditUpdate{
			Enabled:  &enableAudit,
			Endpoint: exoscale.SKSAuditEndpoint(auditBlock.Endpoint.ValueString()),
		}
		if enableAudit && updateReq.Audit.Endpoint == "" {
			resp.Diagnostics.AddError("unable to update SKS cluster", "cannot enable audit without setting an endpoint")
			return
		}
		if v := auditBlock.BearerToken.ValueString(); v != "" {
			updateReq.Audit.BearerToken = exoscale.SKSAuditBearerToken(v)
		}
		if v := auditBlock.InitialBackoff.ValueString(); v != "" {
			updateReq.Audit.InitialBackoff = exoscale.SKSAuditInitialBackoff(v)
		}
		updated = true
	}

	if !plan.Oidc.Equal(state.Oidc) {
		var oidcBlock oidcModel
		if b, ok := r.oidcBlock(ctx, plan, &resp.Diagnostics); ok {
			oidcBlock = b
		}
		if resp.Diagnostics.HasError() {
			return
		}

		updateReq.Oidc = &exoscale.SKSOidc{}
		if v := oidcBlock.ClientID.ValueString(); v != "" {
			updateReq.Oidc.ClientID = v
		}
		if v := oidcBlock.GroupsClaim.ValueString(); v != "" {
			updateReq.Oidc.GroupsClaim = v
		}
		if v := oidcBlock.GroupsPrefix.ValueString(); v != "" {
			updateReq.Oidc.GroupsPrefix = v
		}
		if v := oidcBlock.IssuerURL.ValueString(); v != "" {
			updateReq.Oidc.IssuerURL = v
		}
		if !oidcBlock.RequiredClaim.IsNull() && !oidcBlock.RequiredClaim.IsUnknown() {
			claims := map[string]string{}
			resp.Diagnostics.Append(oidcBlock.RequiredClaim.ElementsAs(ctx, &claims, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			updateReq.Oidc.RequiredClaim = claims
		}
		if v := oidcBlock.UsernameClaim.ValueString(); v != "" {
			updateReq.Oidc.UsernameClaim = v
		}
		if v := oidcBlock.UsernamePrefix.ValueString(); v != "" {
			updateReq.Oidc.UsernamePrefix = v
		}
		updated = true
	}

	if updated {
		// due to a bug it's possible for the update operation to remain in
		// pending state forever; we work around this by checking the cluster
		// state.
		updateErrChan := make(chan error)
		getErrChan := make(chan error)

		go func() {
			updateErrChan <- await(ctx, client)(client.UpdateSKSCluster(ctx, clusterID, updateReq))
		}()

		go func() {
			getErrChan <- waitForClusterUpdateToSucceed(ctx, client, clusterID)
		}()

		var err error
		select {
		case err = <-updateErrChan:
		case err = <-getErrChan:
		}

		if err != nil {
			resp.Diagnostics.AddError("unable to update SKS cluster", err.Error())
			return
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]any{"id": clusterID.String()})

	if diags := r.applyCluster(ctx, client, clusterID, &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceCluster) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceClusterModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(state.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	clusterID := exoscale.UUID(state.ID.ValueString())
	if err := await(ctx, client)(client.DeleteSKSCluster(ctx, clusterID)); err != nil {
		resp.Diagnostics.AddError("API returned an error when deleting SKS cluster", err.Error())
		return
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]any{"id": clusterID.String()})
}

// auditBlock decodes the (at most one) "audit {}" block from a model.
func (r *ResourceCluster) auditBlock(ctx context.Context, m ResourceClusterModel, diags *diag.Diagnostics) (auditModel, bool) {
	if m.Audit.IsNull() || m.Audit.IsUnknown() {
		return auditModel{}, false
	}
	var blocks []auditModel
	diags.Append(m.Audit.ElementsAs(ctx, &blocks, false)...)
	if diags.HasError() || len(blocks) == 0 {
		return auditModel{}, false
	}

	return blocks[0], true
}

// oidcBlock decodes the (at most one) "oidc {}" block from a model.
func (r *ResourceCluster) oidcBlock(ctx context.Context, m ResourceClusterModel, diags *diag.Diagnostics) (oidcModel, bool) {
	if m.Oidc.IsNull() || m.Oidc.IsUnknown() {
		return oidcModel{}, false
	}
	var blocks []oidcModel
	diags.Append(m.Oidc.ElementsAs(ctx, &blocks, false)...)
	if diags.HasError() || len(blocks) == 0 {
		return oidcModel{}, false
	}

	return blocks[0], true
}

func (r *ResourceCluster) applyCluster(ctx context.Context, client *exoscale.Client, id exoscale.UUID, model *ResourceClusterModel) diag.Diagnostics {
	var diags diag.Diagnostics

	sksCluster, err := client.GetSKSCluster(ctx, id)
	if err != nil {
		diags.AddError("API returned an error while fetching SKS cluster", err.Error())
		return diags
	}

	certificates, err := readClusterCertificates(ctx, client, sksCluster.ID)
	if err != nil {
		diags.AddError("API returned an error while fetching SKS cluster certificates", err.Error())
		return diags
	}

	model.ID = types.StringValue(sksCluster.ID.String())
	model.Name = types.StringValue(sksCluster.Name)

	addons := sksCluster.Addons
	model.ExoscaleCCM = types.BoolValue(in(addons, sksClusterAddonExoscaleCCM))
	model.MetricsServer = types.BoolValue(in(addons, sksClusterAddonMS))
	model.ExoscaleCSI = types.BoolValue(in(addons, sksClusterAddonExoscaleCSI))
	model.EnableKarpenter = types.BoolValue(in(addons, sksClusterAddonKarpenter))

	model.AggregationLayerCA = types.StringValue(certificates.AggregationCA)
	model.ControlPlaneCA = types.StringValue(certificates.ControlPlaneCA)
	model.KubeletCA = types.StringValue(certificates.KubeletCA)

	model.AutoUpgrade = types.BoolValue(defaultBool(sksCluster.AutoUpgrade, false))
	model.CNI = types.StringValue(string(sksCluster.Cni))
	model.CreatedAt = types.StringValue(sksCluster.CreatedAT.String())

	defaultSGID := ""
	if sksCluster.DefaultSecurityGroupID != nil {
		defaultSGID = sksCluster.DefaultSecurityGroupID.String()
		model.CreateDefaultSecurityGroup = types.BoolValue(true)
	}
	model.DefaultSecurityGroupID = types.StringValue(defaultSGID)

	model.Description = optionalString(sksCluster.Description)
	model.Endpoint = types.StringValue(sksCluster.Endpoint)
	model.ServiceLevel = types.StringValue(string(sksCluster.Level))
	model.State = types.StringValue(string(sksCluster.State))
	model.EnableKubeProxy = types.BoolValue(defaultBool(sksCluster.EnableKubeProxy, true))

	labels := types.MapNull(types.StringType)
	if len(sksCluster.Labels) > 0 {
		l, d := types.MapValueFrom(ctx, types.StringType, sksCluster.Labels)
		diags.Append(d...)
		labels = l
	}
	model.Labels = labels

	nodepools := make([]string, len(sksCluster.Nodepools))
	for i, np := range sksCluster.Nodepools {
		nodepools[i] = np.ID.String()
	}
	nps, d := types.SetValueFrom(ctx, types.StringType, nodepools)
	diags.Append(d...)
	model.Nodepools = nps

	featureGates, d := types.SetValueFrom(ctx, types.StringType, sliceOrEmpty(sksCluster.FeatureGates))
	diags.Append(d...)
	model.FeatureGates = featureGates

	// Preserve a major.minor input version, otherwise store the resolved one.
	if len(strings.Split(model.Version.ValueString(), ".")) == 2 {
		model.Version = types.StringValue(strings.Join(strings.Split(sksCluster.Version, ".")[:2], "."))
	} else {
		model.Version = types.StringValue(sksCluster.Version)
	}

	return diags
}
