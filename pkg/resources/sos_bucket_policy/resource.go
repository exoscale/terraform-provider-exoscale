package sos_bucket_policy

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/sos"
)

const ResourceSOSBucketPolicyDescription = "Manage Exoscale [SOS Bucket Policies](https://community.exoscale.com/documentation/storage/bucketpolicy/).\n"

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceSOSBucketPolicy{}
var _ resource.ResourceWithImportState = &ResourceSOSBucketPolicy{}

// ResourceSOSBucketPolicy defines the resource implementation.
type ResourceSOSBucketPolicy struct {
	baseConfig *providerConfig.BaseConfig
}

// NewResourceSOSBucketPolicy creates instance of ResourceSOSBucketPolicy.
func NewResourceSOSBucketPolicy() resource.Resource {
	return &ResourceSOSBucketPolicy{}
}

// ResourceSOSBucketPolicyModel defines the resource data model.
type ResourceSOSBucketPolicyModel struct {
	Bucket types.String         `tfsdk:"bucket"`
	Policy jsontypes.Normalized `tfsdk:"policy"`
	Zone   types.String         `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies resource name.
func (r *ResourceSOSBucketPolicy) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sos_bucket_policy"
}

// Schema defines resource attributes.
func (r *ResourceSOSBucketPolicy) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: ResourceSOSBucketPolicyDescription,
		Attributes: map[string]schema.Attribute{
			AttrBucket: schema.StringAttribute{
				Description: attrBucketDescription,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			AttrPolicy: schema.StringAttribute{
				Description: attrPolicyDescription,
				CustomType:  jsontypes.NormalizedType{},
				Required:    true,
			},
			AttrZone: schema.StringAttribute{
				MarkdownDescription: attrZoneDescription,
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

// Configure sets up resource dependencies.
func (r *ResourceSOSBucketPolicy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.baseConfig = &req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Config
}

func (r *ResourceSOSBucketPolicy) NewSOSClient(ctx context.Context, zone string) (*s3.Client, error) {
	return sos.NewSOSClient(ctx, zone, r.baseConfig.Key, r.baseConfig.Secret)
}

func (r *ResourceSOSBucketPolicy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceSOSBucketPolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	sosClient, err := r.NewSOSClient(ctx, plan.Zone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to create SOS client",
			err.Error(),
		)
		return
	}

	_, err = sosClient.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: plan.Bucket.ValueStringPointer(),
		Policy: plan.Policy.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to put bucket policy",
			err.Error(),
		)
		return
	}

	// Save plan into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource created", map[string]interface{}{
		AttrBucket: plan.Bucket,
	})
}

// Read (refresh) resources by receiving Terraform prior state data, performing read logic, and saving refreshed Terraform state data.
func (r *ResourceSOSBucketPolicy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceSOSBucketPolicyModel

	// Load Terraform prior data into the model.
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout.
	timeout, diags := state.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sosClient, err := r.NewSOSClient(ctx, state.Zone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to create SOS client",
			err.Error(),
		)
		return
	}

	policy, err := sosClient.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: state.Bucket.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to get bucket policy",
			err.Error(),
		)
		return
	}

	if policy == nil {
		resp.Diagnostics.AddError(
			"bucket policy is nil",
			"",
		)
		return
	}

	// Save updated state into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource read done", map[string]interface{}{
		AttrBucket: state.Bucket,
	})
}

// Update resources in-place by receiving Terraform prior state, configuration, and plan data, performing update logic, and saving updated Terraform state data.
func (r *ResourceSOSBucketPolicy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan ResourceSOSBucketPolicyModel

	// Read Terraform prior state data (for comparison) into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	sosClient, err := r.NewSOSClient(ctx, state.Zone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to create SOS client",
			err.Error(),
		)
		return
	}

	if !plan.Policy.Equal(state.Policy) {
		_, err = sosClient.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
			Bucket: plan.Bucket.ValueStringPointer(),
			Policy: plan.Policy.ValueStringPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"failed to put bucket policy",
				err.Error(),
			)
			return
		}
	}

	state.Policy = plan.Policy

	// Save updated state into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource update done", map[string]interface{}{
		AttrBucket: state.Bucket,
	})
}

// Delete resources by receiving Terraform prior state data and performing deletion logic.
func (r *ResourceSOSBucketPolicy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceSOSBucketPolicyModel

	// Load Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout.
	timeout, diags := state.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sosClient, err := r.NewSOSClient(ctx, state.Zone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to create SOS client",
			err.Error(),
		)
		return
	}

	_, err = sosClient.DeleteBucketPolicy(ctx, &s3.DeleteBucketPolicyInput{
		Bucket: state.Bucket.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to put bucket policy",
			err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]interface{}{
		AttrBucket: state.Bucket,
	})
}

// ImportState lets Terraform begin managing existing infrastructure resources.
func (r *ResourceSOSBucketPolicy) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf("Expected import identifier with format: id@zone. Got: %q", req.ID),
		)
		return
	}

	var state ResourceSOSBucketPolicyModel

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = timeouts

	state.Bucket = types.StringValue(idParts[0])
	state.Zone = types.StringValue(idParts[1])
	// TODO policy?

	// Save state into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource imported", map[string]interface{}{
		AttrBucket: state.Bucket,
	})
}
