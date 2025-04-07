package exoscale

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
)

const (
	defaultSKSClusterCNI          = "calico"
	defaultSKSClusterServiceLevel = "pro"

	sksClusterAddonExoscaleCCM = "exoscale-cloud-controller"
	sksClusterAddonExoscaleCSI = "exoscale-container-storage-interface"
	sksClusterAddonMS          = "metrics-server"

	resSKSClusterAttrAddons             = "addons"
	resSKSClusterAttrAggregationLayerCA = "aggregation_ca"
	resSKSClusterAttrAutoUpgrade        = "auto_upgrade"
	resSKSClusterAttrCNI                = "cni"
	resSKSClusterAttrControlPlaneCA     = "control_plane_ca"
	resSKSClusterAttrCreatedAt          = "created_at"
	resSKSClusterAttrDescription        = "description"
	resSKSClusterAttrEnableKubeProxy    = "enable_kube_proxy"
	resSKSClusterAttrEndpoint           = "endpoint"
	resSKSClusterAttrExoscaleCCM        = "exoscale_ccm"
	resSKSClusterAttrExoscaleCSI        = "exoscale_csi"
	resSKSClusterAttrKubeletCA          = "kubelet_ca"
	resSKSClusterAttrLabels             = "labels"
	resSKSClusterAttrMetricsServer      = "metrics_server"
	resSKSClusterAttrID                 = "id"
	resSKSClusterAttrName               = "name"
	resSKSClusterAttrNodepools          = "nodepools"
	resSKSClusterAttrOIDCClientID       = "client_id"
	resSKSClusterAttrOIDCGroupsClaim    = "groups_claim"
	resSKSClusterAttrOIDCGroupsPrefix   = "groups_prefix"
	resSKSClusterAttrOIDCIssuerURL      = "issuer_url"
	resSKSClusterAttrOIDCRequiredClaim  = "required_claim"
	resSKSClusterAttrOIDCUsernameClaim  = "username_claim"
	resSKSClusterAttrOIDCUsernamePrefix = "username_prefix"
	resSKSClusterAttrServiceLevel       = "service_level"
	resSKSClusterAttrState              = "state"
	resSKSClusterAttrVersion            = "version"
	resSKSClusterAttrZone               = "zone"
)

func resourceSKSClusterIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_sks_cluster")
}

func resourceSKSCluster() *schema.Resource {
	s := map[string]*schema.Schema{
		resSKSClusterAttrAddons: {
			Type:     schema.TypeSet,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
			Computed: true,
			Deprecated: "This attribute has been replaced by `exoscale_ccm`/`metrics_server` " +
				"attributes, it will be removed in a future release.",
		},

		resSKSClusterAttrAggregationLayerCA: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The CA certificate (in PEM format) for TLS communications between the control plane and the aggregation layer (e.g. `metrics-server`).",
		},
		resSKSClusterAttrAutoUpgrade: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Enable automatic upgrading of the control plane version.",
		},
		resSKSClusterAttrCNI: {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     defaultSKSClusterCNI,
			Description: fmt.Sprintf(`The CNI plugin that is to be used. Available options are "calico" or "cilium". Defaults to %q. Setting empty string will result in a cluster with no CNI.`, defaultSKSClusterCNI),
		},
		resSKSClusterAttrControlPlaneCA: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The CA certificate (in PEM format) for TLS communications between control plane components.",
		},
		resSKSClusterAttrCreatedAt: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The cluster creation date.",
		},
		resSKSClusterAttrDescription: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A free-form text describing the cluster.",
		},
		resSKSClusterAttrEnableKubeProxy: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Indicates whether to deploy the Kubernetes network proxy. (may only be set at creation time)",
			ForceNew:    true,
		},
		resSKSClusterAttrEndpoint: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The cluster API endpoint.",
		},
		resSKSClusterAttrExoscaleCCM: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Deploy the Exoscale [Cloud Controller Manager](https://github.com/exoscale/exoscale-cloud-controller-manager/) in the control plane (boolean; default: `true`; may only be set at creation time).",
		},
		resSKSClusterAttrKubeletCA: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The CA certificate (in PEM format) for TLS communications between kubelets and the control plane.",
		},
		resSKSClusterAttrMetricsServer: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Deploy the [Kubernetes Metrics Server](https://github.com/kubernetes-sigs/metrics-server/) in the control plane (boolean; default: `true`; may only be set at creation time).",
		},
		resSKSClusterAttrExoscaleCSI: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Deploy the Exoscale [Container Storage Interface](https://github.com/exoscale/exoscale-csi-driver/) on worker nodes (boolean; default: `false`; requires the CCM to be enabled).",
		},
		resSKSClusterAttrLabels: {
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: "A map of key/value labels.",
		},
		resSKSClusterAttrName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The SKS cluster name.",
		},
		resSKSClusterAttrID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The SKS cluster ID.",
		},
		resSKSClusterAttrNodepools: {
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "The list of [exoscale_sks_nodepool](./sks_nodepool.md) (IDs) attached to the cluster.",
		},
		"oidc": {
			Type:        schema.TypeList,
			MaxItems:    1,
			Optional:    true,
			Computed:    true,
			Description: "An OpenID Connect configuration to provide to the Kubernetes API server (may only be set at creation time). Structure is documented below.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					resSKSClusterAttrOIDCClientID: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The OpenID client ID.",
					},
					resSKSClusterAttrOIDCGroupsClaim: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An OpenID JWT claim to use as the user's group.",
					},
					resSKSClusterAttrOIDCGroupsPrefix: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An OpenID prefix prepended to group claims.",
					},
					resSKSClusterAttrOIDCIssuerURL: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The OpenID provider URL.",
					},
					resSKSClusterAttrOIDCRequiredClaim: {
						Type:        schema.TypeMap,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Optional:    true,
						Description: "A map of key/value pairs that describes a required claim in the OpenID Token.",
					},
					resSKSClusterAttrOIDCUsernameClaim: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An OpenID JWT claim to use as the user name.",
					},
					resSKSClusterAttrOIDCUsernamePrefix: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An OpenID prefix prepended to username claims.",
					},
				},
			},
		},
		resSKSClusterAttrServiceLevel: {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     defaultSKSClusterServiceLevel,
			Description: "The service level of the control plane (`pro` or `starter`; default: `pro`; may only be set at creation time).",
		},
		resSKSClusterAttrState: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The cluster state.",
		},
		resSKSClusterAttrVersion: {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The version of the control plane (default: latest version available from the API; see `exo compute sks versions` for reference; may only be set at creation time).",
		},
		resSKSClusterAttrZone: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
		},
	}

	return &schema.Resource{
		Schema: s,

		Description: `Manage Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/documentation/sks/) Clusters.`,

		CreateContext: resourceSKSClusterCreate,
		ReadContext:   resourceSKSClusterRead,
		UpdateContext: resourceSKSClusterUpdate,
		DeleteContext: resourceSKSClusterDelete,

		Importer: &schema.ResourceImporter{
			StateContext: zonedStateContextFunc,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(config.DefaultTimeout),
			Read:   schema.DefaultTimeout(config.DefaultTimeout),
			Update: schema.DefaultTimeout(config.DefaultTimeout),
			Delete: schema.DefaultTimeout(config.DefaultTimeout),
		},
	}
}

func resourceSKSClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, zone)
	if err != nil {
		return diag.FromErr(err)
	}

	createReq := v3.CreateSKSClusterRequest{}

	var addOns []string
	if addonsSet, ok := d.Get(resSKSClusterAttrAddons).(*schema.Set); ok && addonsSet.Len() > 0 {
		addOns = make([]string, addonsSet.Len())
		for i, a := range addonsSet.List() {
			addOns[i] = a.(string)
		}
	}
	if enableCCM := d.Get(resSKSClusterAttrExoscaleCCM).(bool); enableCCM && !in(addOns, sksClusterAddonExoscaleCCM) {
		addOns = append(addOns, sksClusterAddonExoscaleCCM)
	}
	if enableMS := d.Get(resSKSClusterAttrMetricsServer).(bool); enableMS && !in(addOns, sksClusterAddonMS) {
		addOns = append(addOns, sksClusterAddonMS)
	}
	if enableCSI := d.Get(resSKSClusterAttrExoscaleCSI).(bool); enableCSI && !in(addOns, sksClusterAddonExoscaleCSI) {
		addOns = append(addOns, sksClusterAddonExoscaleCSI)
	}
	if len(addOns) > 0 {
		createReq.Addons = addOns
	}

	if autoUpgrade := d.Get(resSKSClusterAttrAutoUpgrade).(bool); autoUpgrade {
		createReq.AutoUpgrade = &autoUpgrade
	}

	if enableKubeProxy, isSet := d.GetOk(resSKSClusterAttrEnableKubeProxy); isSet {
		v := enableKubeProxy.(bool)
		createReq.EnableKubeProxy = &v
	}

	if v, ok := d.GetOk(resSKSClusterAttrCNI); ok {
		createReq.Cni = v3.CreateSKSClusterRequestCni(v.(string))
	}

	if v, ok := d.GetOk(resSKSClusterAttrDescription); ok {
		description := v.(string)
		createReq.Description = &description
	}

	if l, ok := d.GetOk(resSKSClusterAttrLabels); ok {
		labels := make(map[string]string)
		for k, v := range l.(map[string]interface{}) {
			labels[k] = v.(string)
		}
		createReq.Labels = labels
	}

	if v, ok := d.GetOk(resSKSClusterAttrName); ok {
		createReq.Name = v.(string)
	}

	if v, ok := d.GetOk(resSKSClusterAttrServiceLevel); ok {
		createReq.Level = v3.CreateSKSClusterRequestLevel(v.(string))
	}

	version := d.Get(resSKSClusterAttrVersion).(string)
	if version == "" {
		versions, err := client.ListSKSClusterVersions(ctx)
		if err != nil {
			return diag.Errorf("error retrieving SKS versions: %s", err)
		}
		if len(versions.SKSClusterVersions) == 0 {
			return diag.Errorf("ListSKSClusterVersions: API returned empty list")
		}

		version = versions.SKSClusterVersions[0]
	}
	createReq.Version = version

	if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCClientID)); ok {
		createReq.Oidc = &v3.SKSOidc{
			ClientID: v.(string),
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCGroupsClaim)); ok {
			createReq.Oidc.GroupsClaim = v.(string)
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCIssuerURL)); ok {
			createReq.Oidc.IssuerURL = v.(string)
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCGroupsPrefix)); ok {
			createReq.Oidc.GroupsPrefix = v.(string)
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCIssuerURL)); ok {
			createReq.Oidc.IssuerURL = v.(string)
		}

		if c, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCRequiredClaim)); ok {
			claims := make(map[string]string)
			for k, v := range c.(map[string]interface{}) {
				claims[k] = v.(string)
			}
			createReq.Oidc.RequiredClaim = claims
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCUsernameClaim)); ok {
			createReq.Oidc.UsernameClaim = v.(string)
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCUsernamePrefix)); ok {
			createReq.Oidc.UsernamePrefix = v.(string)
		}
	}

	op, err := client.CreateSKSCluster(ctx, createReq)
	if err != nil {
		return diag.FromErr(err)
	}

	op, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(string(op.Reference.ID))

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	return resourceSKSClusterRead(ctx, d, meta)
}

func resourceSKSClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, zone)
	if err != nil {
		return diag.FromErr(err)
	}

	sksCluster, err := client.GetSKSCluster(ctx, v3.UUID(d.Id()))
	if err != nil {
		if errors.Is(err, v3.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	certificates, err := readClusterCertificates(ctx, client, sksCluster.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(resourceSKSClusterApply(ctx, d, sksCluster, certificates))
}

func waitForClusterUpdateToSucceed(ctx context.Context, client *v3.Client, clusterID v3.UUID) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	hasStartedUpdate := false
	for {
		select {
		case <-ticker.C:
			cluster, err := client.GetSKSCluster(ctx, clusterID)
			if err != nil {
				return err
			}

			if hasStartedUpdate && cluster.State != "updating" {
				return nil
			} else if cluster.State == "updating" {
				hasStartedUpdate = true
			}
		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				return err
			}

			return nil
		}
	}
}

func await(ctx context.Context, client *v3.Client) func(op *v3.Operation, err error) error {
	return func(op *v3.Operation, err error) error {
		if err != nil {
			return err
		}

		_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
		if err != nil {
			return err
		}

		return nil
	}
}

func resourceSKSClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, zone)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := v3.UUID(d.Id())

	// First check if we need to upgrade cluster
	if d.HasChange(resSKSClusterAttrVersion) {
		v := d.Get(resSKSClusterAttrVersion).(string)
		if err := await(ctx, client)(client.UpgradeSKSCluster(ctx, clusterID, v3.UpgradeSKSClusterRequest{
			Version: v,
		})); err != nil {
			return diag.FromErr(err)
		}
	}

	var updated bool
	updateReq := v3.UpdateSKSClusterRequest{}

	if d.HasChange(resSKSClusterAttrAutoUpgrade) {
		autoUpgrade := d.Get(resSKSClusterAttrAutoUpgrade).(bool)
		updateReq.AutoUpgrade = &autoUpgrade
		updated = true
	}

	if d.HasChange(resSKSClusterAttrLabels) {
		labels := make(map[string]string)
		for k, v := range d.Get(resSKSClusterAttrLabels).(map[string]interface{}) {
			labels[k] = v.(string)
		}
		updateReq.Labels = labels
		updated = true
	}

	if d.HasChange(resSKSClusterAttrName) {
		name := d.Get(resSKSClusterAttrName).(string)
		updateReq.Name = name
		updated = true
	}

	if d.HasChange(resSKSClusterAttrDescription) {
		description := d.Get(resSKSClusterAttrDescription).(string)
		updateReq.Description = &description
		updated = true
	}

	if d.HasChange(resSKSClusterAttrExoscaleCSI) {
		enableCSI := d.Get(resSKSClusterAttrExoscaleCSI).(bool)
		if !enableCSI {
			return diag.Errorf("disabling the CSI addon is not supported")
		}

		addons := d.Get(resSKSClusterAttrAddons).(*schema.Set)
		if enableCSI && !addons.Contains(sksClusterAddonExoscaleCSI) {
			addonStrings := make([]string, 0, addons.Len())
			for _, v := range addons.List() {
				addonStrings = append(addonStrings, v.(string))
			}
			//nolint:gocritic
			updateReq.Addons = append(addonStrings, sksClusterAddonExoscaleCSI)
			updated = true
		}
	}

	if updated {
		// due to a bug it's possible for the update operation
		// to remain in pending state forever
		// we work around this by checking the cluster state
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
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	return resourceSKSClusterRead(ctx, d, meta)
}

func resourceSKSClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, zone)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := v3.UUID(d.Id())
	if err := await(ctx, client)(client.DeleteSKSCluster(ctx, clusterID)); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	return nil
}

func resourceSKSClusterApply(_ context.Context, d *schema.ResourceData, sksCluster *v3.SKSCluster, certificates *SKSClusterCertificates) error {
	if len(sksCluster.Addons) > 0 {
		if err := d.Set(resSKSClusterAttrAddons, sksCluster.Addons); err != nil {
			return err
		}

		if err := d.Set(resSKSClusterAttrExoscaleCCM, in(sksCluster.Addons, sksClusterAddonExoscaleCCM)); err != nil {
			return err
		}

		if err := d.Set(resSKSClusterAttrMetricsServer, in(sksCluster.Addons, sksClusterAddonMS)); err != nil {
			return err
		}

		if err := d.Set(resSKSClusterAttrExoscaleCSI, in(sksCluster.Addons, sksClusterAddonExoscaleCSI)); err != nil {
			return err
		}
	}

	if err := d.Set(resSKSClusterAttrAggregationLayerCA, certificates.AggregationCA); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrAutoUpgrade, defaultBool(sksCluster.AutoUpgrade, false)); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrCNI, sksCluster.Cni); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrControlPlaneCA, certificates.ControlPlaneCA); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrCreatedAt, sksCluster.CreatedAT.String()); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrDescription, sksCluster.Description); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrEndpoint, sksCluster.Endpoint); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrKubeletCA, certificates.KubeletCA); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrLabels, sksCluster.Labels); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrName, sksCluster.Name); err != nil {
		return err
	}

	nodepools := make([]string, len(sksCluster.Nodepools))
	for i, nodepool := range sksCluster.Nodepools {
		nodepools[i] = nodepool.ID.String()
	}
	if err := d.Set(resSKSClusterAttrNodepools, nodepools); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrServiceLevel, sksCluster.Level); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrState, sksCluster.State); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrVersion, sksCluster.Version); err != nil {
		return err
	}

	return nil
}

// resSKSClusterAttrOIDC returns a sks_cluster resource attribute key formatted for an "oidc {}" block.
func resSKSClusterAttrOIDC(a string) string {
	return fmt.Sprintf("oidc.0.%s", a)
}

type SKSClusterCertificates struct {
	AggregationCA  string
	ControlPlaneCA string
	KubeletCA      string
}

// readClusterCertificates returns an SKS Cluster related CA certificates
func readClusterCertificates(ctx context.Context, client *v3.Client, clusterID v3.UUID) (*SKSClusterCertificates, error) {
	encodedAggregationCertificate, err := client.GetSKSClusterAuthorityCert(ctx, clusterID, "aggregation")
	if err != nil {
		return nil, err
	}

	encodedControlPlaneCertificate, err := client.GetSKSClusterAuthorityCert(ctx, clusterID, "control-plane")
	if err != nil {
		return nil, err
	}

	encodedKubeletCertificate, err := client.GetSKSClusterAuthorityCert(ctx, clusterID, "kubelet")
	if err != nil {
		return nil, err
	}

	aggregationCertificate, err := base64.StdEncoding.DecodeString(encodedAggregationCertificate.Cacert)
	if err != nil {
		return nil, err
	}

	controlPlaneCertificate, err := base64.StdEncoding.DecodeString(encodedControlPlaneCertificate.Cacert)
	if err != nil {
		return nil, err
	}

	kubeletCertificate, err := base64.StdEncoding.DecodeString(encodedKubeletCertificate.Cacert)
	if err != nil {
		return nil, err
	}

	return &SKSClusterCertificates{
		AggregationCA:  string(aggregationCertificate),
		ControlPlaneCA: string(controlPlaneCertificate),
		KubeletCA:      string(kubeletCertificate),
	}, nil
}
