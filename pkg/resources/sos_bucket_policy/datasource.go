package sos_bucket_policy

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/sos"
)

const DataSourceSOSBucketPolicyDescription = "Fetch Exoscale [SOS Bucket Policies](https://community.exoscale.com/documentation/storage/bucketpolicy/)."

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSourceWithConfigure = &DataSourceSOSBucketPolicy{}

// DataSourceSOSBucketPolicy defines the resource implementation.
type DataSourceSOSBucketPolicy struct {
	baseConfig *providerConfig.BaseConfig
}

// NewDataSourceSOSBucketPolicy creates instance of ResourceSOSBucketPolicy.
func NewDataSourceSOSBucketPolicy() datasource.DataSource {
	return &DataSourceSOSBucketPolicy{}
}

// DataSourceSOSBucketPolicyModel defines the resource data model.
type DataSourceSOSBucketPolicyModel struct {
	Bucket types.String         `tfsdk:"bucket"`
	Policy jsontypes.Normalized `tfsdk:"policy"`
	Zone   types.String         `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies resource name.
func (d *DataSourceSOSBucketPolicy) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_sos_bucket_policy"
}

// Schema defines resource attributes.
func (d *DataSourceSOSBucketPolicy) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceSOSBucketPolicyDescription,
		Attributes: map[string]schema.Attribute{
			AttrBucket: schema.StringAttribute{
				MarkdownDescription: attrBucketDescription,
				Required:            true,
			},
			AttrPolicy: schema.StringAttribute{
				MarkdownDescription: attrPolicyDescription,
				CustomType:          jsontypes.NormalizedType{},
				Computed:            true,
			},
			AttrZone: schema.StringAttribute{
				MarkdownDescription: attrZoneDescription,
				Required:            true,
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

// Configure sets up datasource dependencies.
func (d *DataSourceSOSBucketPolicy) Configure(
	ctx context.Context,
	r datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if r.ProviderData == nil {
		return
	}

	d.baseConfig = &r.ProviderData.(*providerConfig.ExoscaleProviderConfig).Config
}

func (d *DataSourceSOSBucketPolicy) NewSOSClient(ctx context.Context, zone string) (*s3.Client, error) {
	return sos.NewSOSClient(ctx, zone, d.baseConfig.SOSEndpoint, d.baseConfig.Key, d.baseConfig.Secret)
}

// Read defines how the data source updates Terraform's state to reflect the retrieved data.
func (d *DataSourceSOSBucketPolicy) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var plan DataSourceSOSBucketPolicyModel

	// Load Terraform plan into the model.
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout.
	timeout, diags := plan.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sosClient, err := d.NewSOSClient(ctx, plan.Zone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to create SOS client",
			err.Error(),
		)
		return
	}

	// Read remote state.
	policyOut, err := sosClient.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: plan.Bucket.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to get bucket policy",
			err.Error(),
		)
		return
	}

	plan.Policy = jsontypes.NewNormalizedValue(*policyOut.Policy)

	// Save updated state into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "datasource read done", map[string]interface{}{
		AttrBucket: plan.Bucket,
	})
}
