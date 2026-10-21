package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const ResourceRecordDescription = `Manage Exoscale [DNS](https://community.exoscale.com/product/networking/dns/) Domain Records.

Corresponding data source: [exoscale_domain_record](../data-sources/domain_record.md).`

// DNS record types managed through this resource.
var supportedRecordTypes = []string{
	"A", "AAAA", "ALIAS", "CAA", "CNAME",
	"HINFO", "MX", "NAPTR", "NS", "POOL",
	"SPF", "SRV", "SSHFP", "TXT", "URL",
}

var _ resource.Resource = &ResourceRecord{}
var _ resource.ResourceWithImportState = &ResourceRecord{}

type ResourceRecordModel struct {
	ID                types.String `tfsdk:"id"`
	Domain            types.String `tfsdk:"domain"`
	RecordType        types.String `tfsdk:"record_type"`
	Name              types.String `tfsdk:"name"`
	Content           types.String `tfsdk:"content"`
	ContentNormalized types.String `tfsdk:"content_normalized"`
	Ttl               types.Int64  `tfsdk:"ttl"`
	Prio              types.Int64  `tfsdk:"prio"`
	Hostname          types.String `tfsdk:"hostname"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type ResourceRecord struct {
	client *exoscale.Client
}

func NewResourceRecord() resource.Resource {
	return &ResourceRecord{}
}

func (r *ResourceRecord) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_domain_record"
}

func (r *ResourceRecord) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale DNS Domain Records.",
		MarkdownDescription: ResourceRecordDescription,
		Version:             1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Description:         "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Description:         "❗ The parent exoscale_domain to attach the record to.",
				MarkdownDescription: "❗ The parent [exoscale_domain](./domain.md) to attach the record to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"record_type": schema.StringAttribute{
				Description:         "❗ The record type (A, AAAA, ALIAS, CAA, CNAME, HINFO, MX, NAPTR, NS, POOL, SPF, SRV, SSHFP, TXT, URL).",
				MarkdownDescription: "❗ The record type (`A`, `AAAA`, `ALIAS`, `CAA`, `CNAME`, `HINFO`, `MX`, `NAPTR`, `NS`, `POOL`, `SPF`, `SRV`, `SSHFP`, `TXT`, `URL`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(supportedRecordTypes...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The record name, Leave blank to create a root record (similar to using '@' in a DNS zone file).",
				MarkdownDescription: "The record name, Leave blank (`\"\"`) to create a root record (similar to using `@` in a DNS zone file).",
				Required:            true,
			},
			"content": schema.StringAttribute{
				Description:         "The record value. Format follows specific record type. For example SRV record format would be '<weight> <port> <target>'",
				MarkdownDescription: "The record value. Format follows specific record type. For example SRV record format would be `<weight> <port> <target>`",
				Required:            true,
			},
			"content_normalized": schema.StringAttribute{
				Description:         "The normalized value of the record",
				MarkdownDescription: "The normalized value of the record",
				Computed:            true,
			},
			"ttl": schema.Int64Attribute{
				Description:         "The record TTL (seconds; minimum '0'; default: '3600').",
				MarkdownDescription: "The record TTL (seconds; minimum `0`; default: `3600`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"prio": schema.Int64Attribute{
				Description:         "The record priority (for types that support it; minimum '0').",
				MarkdownDescription: "The record priority (for types that support it; minimum `0`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Description:         "The record *Fully Qualified Domain Name* (FQDN). Useful for aliasing 'A'/'AAAA' records with 'CNAME'.",
				MarkdownDescription: "The record *Fully Qualified Domain Name* (FQDN). Useful for aliasing `A`/`AAAA` records with `CNAME`.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceRecord) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceRecord) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ResourceRecordModel

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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(config.DefaultZone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	domainID, err := exoscale.ParseUUID(plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent domain ID", err.Error())
		return
	}

	createReq := exoscale.CreateDNSDomainRecordRequest{
		Name:    plan.Name.ValueString(),
		Content: plan.Content.ValueString(),
		Type:    exoscale.CreateDNSDomainRecordRequestType(plan.RecordType.ValueString()),
	}
	if !plan.Ttl.IsUnknown() {
		createReq.Ttl = plan.Ttl.ValueInt64()
	}
	if !plan.Prio.IsUnknown() {
		createReq.Priority = plan.Prio.ValueInt64()
	}

	op, err := client.CreateDNSDomainRecord(ctx, domainID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when creating domain record", err.Error())
		return
	}

	op, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create domain record operation failed", err.Error())
		return
	}

	record, err := client.GetDNSDomainRecord(ctx, domainID, op.Reference.ID)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when reading domain record after creation", err.Error())
		return
	}

	domain, err := client.GetDNSDomain(ctx, domainID)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when reading domain", err.Error())
		return
	}

	state := recordToModel(*record, domain.UnicodeName, plan.Timeouts)
	// "domain", "name", "record_type" and "content" are Required
	// (non-Computed) attributes: the applied state must match the planned
	// value exactly.
	state.Domain = plan.Domain
	state.Name = plan.Name
	state.RecordType = plan.RecordType
	state.Content = plan.Content
	if !plan.Ttl.IsUnknown() {
		state.Ttl = plan.Ttl
	}
	if !plan.Prio.IsUnknown() {
		state.Prio = plan.Prio
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceRecord) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state ResourceRecordModel

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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(config.DefaultZone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	domainID, err := exoscale.ParseUUID(state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent domain ID", err.Error())
		return
	}

	record, err := client.GetDNSDomainRecord(ctx, domainID, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned error when reading domain record", err.Error())
		return
	}

	domain, err := client.GetDNSDomain(ctx, domainID)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when reading domain", err.Error())
		return
	}

	newState := recordToModel(*record, domain.UnicodeName, state.Timeouts)
	newState.Domain = state.Domain

	// Default to the remote content (this also covers a fresh import, where
	// state.Content/state.ContentNormalized are still null). If the content
	// we last stored as "normalized" still matches the remote value, the
	// remote hasn't changed out-of-band: keep the raw content we already
	// have in state instead, since the API may return a normalized form
	// (e.g. quoted TXT) that would otherwise look like a perpetual diff
	// against the user's config.
	content := record.Content
	if !state.ContentNormalized.IsNull() && state.ContentNormalized.ValueString() == record.Content {
		content = state.Content.ValueString()
	}
	newState.Content = types.StringValue(content)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ResourceRecord) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceRecordModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(config.DefaultZone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	domainID, err := exoscale.ParseUUID(plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent domain ID", err.Error())
		return
	}
	id, err := exoscale.ParseUUID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	updateReq := exoscale.UpdateDNSDomainRecordRequest{
		Name:    plan.Name.ValueString(),
		Content: plan.Content.ValueString(),
	}
	if !plan.Ttl.IsUnknown() {
		updateReq.Ttl = plan.Ttl.ValueInt64()
	}
	if !plan.Prio.IsUnknown() {
		updateReq.Priority = plan.Prio.ValueInt64()
	}

	op, err := client.UpdateDNSDomainRecord(ctx, domainID, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when updating domain record", err.Error())
		return
	}
	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("update domain record operation failed", err.Error())
		return
	}

	record, err := client.GetDNSDomainRecord(ctx, domainID, id)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when reading domain record after update", err.Error())
		return
	}

	domain, err := client.GetDNSDomain(ctx, domainID)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when reading domain", err.Error())
		return
	}

	state := recordToModel(*record, domain.UnicodeName, plan.Timeouts)
	state.Domain = plan.Domain
	state.Name = plan.Name
	state.RecordType = plan.RecordType
	state.Content = plan.Content
	if !plan.Ttl.IsUnknown() {
		state.Ttl = plan.Ttl
	}
	if !plan.Prio.IsUnknown() {
		state.Prio = plan.Prio
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceRecord) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ResourceRecordModel

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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(config.DefaultZone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	domainID, err := exoscale.ParseUUID(state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent domain ID", err.Error())
		return
	}
	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	op, err := client.DeleteDNSDomainRecord(ctx, domainID, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned error when deleting domain record", err.Error())
		return
	}

	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete domain record operation failed", err.Error())
		return
	}
}

func (r *ResourceRecord) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, "@")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf("Expected import identifier with format: domain_id@record_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
}

// recordToModel converts a DNSDomainRecord (and its parent domain's unicode
// name, for hostname derivation) into a ResourceRecordModel. The Domain field
// is intentionally left unset: callers know the domain identity through
// different means (plan value, or the domain scanned during import) and
// should assign it themselves.
func recordToModel(record exoscale.DNSDomainRecord, domainUnicodeName string, t timeouts.Value) ResourceRecordModel {
	hostname := domainUnicodeName
	if record.Name != "" {
		hostname = fmt.Sprintf("%s.%s", record.Name, domainUnicodeName)
	}

	return ResourceRecordModel{
		ID:                types.StringValue(record.ID.String()),
		RecordType:        types.StringValue(string(record.Type)),
		Name:              types.StringValue(record.Name),
		Content:           types.StringValue(record.Content),
		ContentNormalized: types.StringValue(record.Content),
		Ttl:               types.Int64Value(record.Ttl),
		Prio:              types.Int64Value(record.Priority),
		Hostname:          types.StringValue(hostname),
		Timeouts:          t,
	}
}
