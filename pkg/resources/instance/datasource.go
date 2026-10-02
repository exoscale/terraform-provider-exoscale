package instance

import (
	"context"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionDatasource = `Fetch Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/) data.

Corresponding resource: [exoscale_compute_instance](../resources/compute_instance.md).`

var _ datasource.DataSource = (*DataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*DataSource)(nil)

type DataSource struct {
	client *exoscale.Client
}

// NewDataSource creates an instance of DataSource.
func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

// InstanceModel defines the attributes of a Compute instance, shared by the
// exoscale_compute_instance data source and the entries of
// exoscale_compute_instance_list.
type InstanceModel struct {
	AntiAffinityGroupIDs types.Set    `tfsdk:"anti_affinity_group_ids"`
	CreatedAt            types.String `tfsdk:"created_at"`
	DeployTargetID       types.String `tfsdk:"deploy_target_id"`
	DiskSize             types.Int64  `tfsdk:"disk_size"`
	ElasticIPIDs         types.Set    `tfsdk:"elastic_ip_ids"`
	EnableSecureBoot     types.Bool   `tfsdk:"enable_secure_boot"`
	EnableTPM            types.Bool   `tfsdk:"enable_tpm"`
	ID                   types.String `tfsdk:"id"`
	IPv6                 types.Bool   `tfsdk:"ipv6"`
	IPv6Address          types.String `tfsdk:"ipv6_address"`
	Labels               types.Map    `tfsdk:"labels"`
	ManagerID            types.String `tfsdk:"manager_id"`
	ManagerType          types.String `tfsdk:"manager_type"`
	Name                 types.String `tfsdk:"name"`
	PrivateNetworkIDs    types.Set    `tfsdk:"private_network_ids"`
	PublicIPAddress      types.String `tfsdk:"public_ip_address"`
	ReverseDNS           types.String `tfsdk:"reverse_dns"`
	SSHKey               types.String `tfsdk:"ssh_key"`
	SSHKeys              types.Set    `tfsdk:"ssh_keys"`
	SecurityGroupIDs     types.Set    `tfsdk:"security_group_ids"`
	State                types.String `tfsdk:"state"`
	TemplateID           types.String `tfsdk:"template_id"`
	Type                 types.String `tfsdk:"type"`
	UserData             types.String `tfsdk:"user_data"`
	Zone                 types.String `tfsdk:"zone"`
}

// DataSourceModel defines the exoscale_compute_instance data source data model.
type DataSourceModel struct {
	InstanceModel

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// instanceAttributes returns the computed attributes describing a Compute
// instance.
func instanceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		AttrAntiAffinityGroupIDs: schema.SetAttribute{
			MarkdownDescription: "The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs).",
			ElementType:         types.StringType,
			Computed:            true,
		},
		AttrCreatedAt: schema.StringAttribute{
			MarkdownDescription: "The compute instance creation date.",
			Computed:            true,
		},
		AttrDeployTargetID: schema.StringAttribute{
			MarkdownDescription: "A deploy target ID.",
			Computed:            true,
		},
		AttrDiskSize: schema.Int64Attribute{
			MarkdownDescription: "The instance disk size (GiB).",
			Computed:            true,
		},
		AttrElasticIPIDs: schema.SetAttribute{
			MarkdownDescription: "The list of attached [exoscale_elastic_ip](../resources/elastic_ip.md) (IDs).",
			ElementType:         types.StringType,
			Computed:            true,
		},
		AttrEnableSecureBoot: schema.BoolAttribute{
			MarkdownDescription: "Indicates if the instance has secure boot enabled.",
			Computed:            true,
		},
		AttrEnableTPM: schema.BoolAttribute{
			MarkdownDescription: "Indicates if the instance has TPM enabled.",
			Computed:            true,
		},
		AttrID: schema.StringAttribute{
			MarkdownDescription: "The compute instance ID.",
			Computed:            true,
		},
		AttrIPv6: schema.BoolAttribute{
			MarkdownDescription: "Whether IPv6 is enabled on the instance.",
			Computed:            true,
		},
		AttrIPv6Address: schema.StringAttribute{
			MarkdownDescription: "The instance (main network interface) IPv6 address (if enabled).",
			Computed:            true,
		},
		AttrLabels: schema.MapAttribute{
			MarkdownDescription: "A map of key/value labels.",
			ElementType:         types.StringType,
			Computed:            true,
		},
		AttrManagerID: schema.StringAttribute{
			MarkdownDescription: "The instance manager ID, if any.",
			Computed:            true,
		},
		AttrManagerType: schema.StringAttribute{
			MarkdownDescription: "The instance manager type (instance pool, SKS node pool, etc.), if any.",
			Computed:            true,
		},
		AttrName: schema.StringAttribute{
			MarkdownDescription: "The instance name.",
			Computed:            true,
		},
		AttrPrivateNetworkIDs: schema.SetAttribute{
			MarkdownDescription: "The list of attached [exoscale_private_network](../resources/private_network.md) (IDs).",
			ElementType:         types.StringType,
			Computed:            true,
		},
		AttrPublicIPAddress: schema.StringAttribute{
			MarkdownDescription: "The instance (main network interface) IPv4 address.",
			Computed:            true,
		},
		AttrReverseDNS: schema.StringAttribute{
			MarkdownDescription: "Domain name for reverse DNS record.",
			Computed:            true,
		},
		AttrSSHKey: schema.StringAttribute{
			MarkdownDescription: "The [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the instance.",
			DeprecationMessage:  "Use ssh_keys instead",
			Computed:            true,
		},
		AttrSSHKeys: schema.SetAttribute{
			MarkdownDescription: "The list of [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the instance.",
			ElementType:         types.StringType,
			Computed:            true,
		},
		AttrSecurityGroupIDs: schema.SetAttribute{
			MarkdownDescription: "The list of attached [exoscale_security_group](../resources/security_group.md) (IDs).",
			ElementType:         types.StringType,
			Computed:            true,
		},
		AttrState: schema.StringAttribute{
			MarkdownDescription: "The instance state.",
			Computed:            true,
		},
		AttrTemplateID: schema.StringAttribute{
			MarkdownDescription: "The instance [exoscale_template](./template.md) ID.",
			Computed:            true,
		},
		AttrType: schema.StringAttribute{
			MarkdownDescription: "The instance type.",
			Computed:            true,
		},
		AttrUserData: schema.StringAttribute{
			MarkdownDescription: "The instance [cloud-init](http://cloudinit.readthedocs.io/en/latest/) configuration.",
			Computed:            true,
		},
		AttrZone: schema.StringAttribute{
			MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
			Computed:            true,
		},
	}
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compute_instance"
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := instanceAttributes()

	attributes[AttrID] = schema.StringAttribute{
		MarkdownDescription: "The compute instance ID to match (conflicts with `name`).",
		Optional:            true,
		Computed:            true,
		Validators: []validator.String{
			stringvalidator.ConflictsWith(path.MatchRoot(AttrName)),
		},
	}
	attributes[AttrName] = schema.StringAttribute{
		MarkdownDescription: "The instance name to match (conflicts with `id`).",
		Optional:            true,
		Computed:            true,
		Validators: []validator.String{
			stringvalidator.ConflictsWith(path.MatchRoot(AttrID)),
		},
	}
	attributes[AttrZone] = schema.StringAttribute{
		MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.OneOf(config.Zones...),
		},
	}

	resp.Schema = schema.Schema{
		Description:         "Fetch Exoscale Compute Instances data.",
		MarkdownDescription: markdownDescriptionDatasource,

		Attributes: attributes,
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (d *DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
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

	zone := state.Zone.ValueString()

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	var nameOrID string

	switch {
	case state.ID.ValueString() != "":
		nameOrID = state.ID.ValueString()
	case state.Name.ValueString() != "":
		nameOrID = state.Name.ValueString()
	default:
		resp.Diagnostics.AddError("missing values", "either name or id must be specified")
		return
	}

	instances, err := client.ListInstances(ctx)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching instances", err.Error())
		return
	}

	found, err := instances.FindListInstancesResponseInstances(nameOrID)
	if err != nil {
		resp.Diagnostics.AddError("unable to retrieve instance", err.Error())
		return
	}

	// The list endpoint does not return every attribute, fetch the instance itself.
	instance, err := client.GetInstance(ctx, found.ID)
	if err != nil {
		resp.Diagnostics.AddError("unable to retrieve instance", err.Error())
		return
	}

	model, diags := instanceModel(ctx, instance, zone)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if instance.InstanceType != nil {
		instanceType, err := client.GetInstanceType(ctx, instance.InstanceType.ID)
		if err != nil {
			resp.Diagnostics.AddError("unable to retrieve instance type", err.Error())
			return
		}
		model.Type = types.StringValue(instanceTypeName(instanceType))
	}

	rdns, err := reverseDNS(ctx, client, instance.ID)
	if err != nil {
		resp.Diagnostics.AddError("unable to retrieve instance reverse-dns", err.Error())
		return
	}
	model.ReverseDNS = utils.OptionalString(strings.TrimSuffix(rdns, "."))

	state.InstanceModel = model

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Trace(ctx, "datasource read done", map[string]any{"id": state.ID})
}

// instanceModel maps an API instance onto the data source model. The instance
// type name and the reverse DNS record are held by other API resources: they
// are left null for the caller to fill in.
func instanceModel(ctx context.Context, instance *exoscale.Instance, zone string) (InstanceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := InstanceModel{
		CreatedAt:        types.StringValue(instance.CreatedAT.String()),
		DiskSize:         types.Int64Value(instance.DiskSize),
		EnableSecureBoot: types.BoolValue(utils.DefaultBool(instance.SecurebootEnabled, false)),
		EnableTPM:        types.BoolValue(utils.DefaultBool(instance.TpmEnabled, false)),
		ID:               types.StringValue(instance.ID.String()),
		IPv6:             types.BoolValue(instance.PublicIPAssignment == exoscale.PublicIPAssignmentDual),
		IPv6Address:      utils.OptionalString(instance.Ipv6Address),
		Name:             types.StringValue(instance.Name),
		State:            types.StringValue(string(instance.State)),
		Zone:             types.StringValue(zone),

		// An instance without deploy target has always been reported with an
		// empty ID rather than none.
		DeployTargetID: types.StringValue(""),

		ManagerID:       types.StringNull(),
		ManagerType:     types.StringNull(),
		PublicIPAddress: types.StringNull(),
		ReverseDNS:      types.StringNull(),
		SSHKey:          types.StringNull(),
		TemplateID:      types.StringNull(),
		Type:            types.StringNull(),
		UserData:        types.StringNull(),
	}

	if instance.DeployTarget != nil {
		model.DeployTargetID = types.StringValue(instance.DeployTarget.ID.String())
	}

	if instance.Manager != nil {
		model.ManagerID = types.StringValue(instance.Manager.ID.String())
		model.ManagerType = types.StringValue(string(instance.Manager.Type))
	}

	if instance.PublicIP != nil {
		model.PublicIPAddress = types.StringValue(instance.PublicIP.String())
	}

	if instance.SSHKey != nil {
		model.SSHKey = types.StringValue(instance.SSHKey.Name)
	}

	if instance.Template != nil {
		model.TemplateID = types.StringValue(instance.Template.ID.String())
	}

	if instance.UserData != "" {
		userData, err := utils.DecodeUserData(instance.UserData)
		if err != nil {
			diags.AddError("unable to decode user data", err.Error())
			return model, diags
		}
		model.UserData = types.StringValue(userData)
	}

	labels := map[string]string{}
	for k, v := range instance.Labels {
		labels[k] = v
	}

	var dg diag.Diagnostics

	model.Labels, dg = types.MapValueFrom(ctx, types.StringType, labels)
	diags.Append(dg...)

	for _, set := range []struct {
		target *types.Set
		values []string
	}{
		{&model.AntiAffinityGroupIDs, utils.AntiAffiniGroupsToAntiAffinityGroupIDs(instance.AntiAffinityGroups)},
		{&model.ElasticIPIDs, utils.ElasticIPsToElasticIPIDs(instance.ElasticIPS)},
		{&model.PrivateNetworkIDs, privateNetworkIDs(instance)},
		{&model.SSHKeys, sshKeyNames(instance)},
		{&model.SecurityGroupIDs, utils.SecurityGroupsToSecurityGroupIDs(instance.SecurityGroups)},
	} {
		*set.target, dg = types.SetValueFrom(ctx, types.StringType, set.values)
		diags.Append(dg...)
	}

	return model, diags
}
