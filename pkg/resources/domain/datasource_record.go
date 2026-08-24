package domain

import (
	"context"
	"crypto/md5" //nolint:gosec // used only to derive a stable synthetic data source ID, not for security purposes
	"fmt"
	"regexp"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const DataSourceRecordDescription = `Fetch Exoscale [DNS](https://community.exoscale.com/product/networking/dns/) Domain Records data.

Corresponding resource: [exoscale_domain_record](../resources/domain_record.md).`

var _ datasource.DataSourceWithConfigure = &DataSourceRecord{}

type DataSourceRecord struct {
	client *exoscale.Client
}

func NewDataSourceRecord() datasource.DataSource {
	return &DataSourceRecord{}
}

type DataSourceRecordFilterModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	RecordType   types.String `tfsdk:"record_type"`
	ContentRegex types.String `tfsdk:"content_regex"`
}

type DataSourceRecordItemModel struct {
	ID         types.String `tfsdk:"id"`
	Domain     types.String `tfsdk:"domain"`
	Name       types.String `tfsdk:"name"`
	Content    types.String `tfsdk:"content"`
	RecordType types.String `tfsdk:"record_type"`
	Ttl        types.Int64  `tfsdk:"ttl"`
	Prio       types.Int64  `tfsdk:"prio"`
}

type DataSourceRecordModel struct {
	ID      types.String                `tfsdk:"id"`
	Domain  types.String                `tfsdk:"domain"`
	Filter  DataSourceRecordFilterModel `tfsdk:"filter"`
	Records []DataSourceRecordItemModel `tfsdk:"records"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceRecord) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_record"
}

func (d *DataSourceRecord) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceRecordDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Description:         "The ID of this resource.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The [exoscale_domain](./domain.md) name to match.",
				Description:         "The exoscale name to match.",
				Required:            true,
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "The list of matching records. Structure is documented below.",
				Description:         "The list of matching records. Structure is documented below.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "ID of the Record",
							Description:         "ID of the Record",
							Computed:            true,
						},
						"domain": schema.StringAttribute{
							MarkdownDescription: "Domain of the Record",
							Description:         "Domain of the Record",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the Record",
							Description:         "Name of the Record",
							Computed:            true,
						},
						"content": schema.StringAttribute{
							MarkdownDescription: "Content of the Record",
							Description:         "Content of the Record",
							Computed:            true,
						},
						"record_type": schema.StringAttribute{
							MarkdownDescription: "Type of the Record",
							Description:         "Type of the Record",
							Computed:            true,
						},
						"ttl": schema.Int64Attribute{
							MarkdownDescription: "TTL of the Record",
							Description:         "TTL of the Record",
							Computed:            true,
						},
						"prio": schema.Int64Attribute{
							MarkdownDescription: "Priority of the Record",
							Description:         "Priority of the Record",
							Computed:            true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Filter to apply when looking up domain records.",
				Description:         "Filter to apply when looking up domain records.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "The record ID to match (conflicts with `name`, `record_type` and `content_regex`).",
						Description:         "The record ID to match (conflicts with 'name', 'record_type' and 'content_regex').",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.Expressions{
								path.MatchRelative().AtParent().AtName("name"),
								path.MatchRelative().AtParent().AtName("record_type"),
								path.MatchRelative().AtParent().AtName("content_regex"),
							}...),
						},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The domain record name to match (conflicts with `id` and `content_regex`; can be combined with `record_type`).",
						Description:         "The domain record name to match (conflicts with 'id' and 'content_regex'; can be combined with 'record_type').",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.Expressions{
								path.MatchRelative().AtParent().AtName("id"),
								path.MatchRelative().AtParent().AtName("content_regex"),
							}...),
						},
					},
					"record_type": schema.StringAttribute{
						MarkdownDescription: "The record type to match (conflicts with `id` and `content_regex`; can be combined with `name`).",
						Description:         "The record type to match (conflicts with 'id' and 'content_regex'; can be combined with 'name').",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.Expressions{
								path.MatchRelative().AtParent().AtName("id"),
								path.MatchRelative().AtParent().AtName("content_regex"),
							}...),
						},
					},
					"content_regex": schema.StringAttribute{
						MarkdownDescription: "A regular expression to match the record content (conflicts with `id`, `name` and `record_type`).",
						Description:         "A regular expression to match the record content (conflicts with 'id', 'name' and 'record_type').",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.Expressions{
								path.MatchRelative().AtParent().AtName("id"),
								path.MatchRelative().AtParent().AtName("name"),
								path.MatchRelative().AtParent().AtName("record_type"),
							}...),
						},
					},
				},
			},
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (d *DataSourceRecord) Configure(ctx context.Context, r datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceRecord) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataSourceRecordModel

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

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(config.DefaultZone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	domains, err := client.ListDNSDomains(ctx)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when listing domains", err.Error())
		return
	}

	domain, err := domains.FindDNSDomain(state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("domain %q not found", state.Domain.ValueString()), err.Error())
		return
	}

	id := state.Filter.ID.ValueString()
	name := state.Filter.Name.ValueString()
	rtype := state.Filter.RecordType.ValueString()
	cregex := state.Filter.ContentRegex.ValueString()

	var records []exoscale.DNSDomainRecord

	switch {
	case id != "":
		recordID, err := exoscale.ParseUUID(id)
		if err != nil {
			resp.Diagnostics.AddError("unable to parse record ID", err.Error())
			return
		}
		record, err := client.GetDNSDomainRecord(ctx, domain.ID, recordID)
		if err != nil {
			resp.Diagnostics.AddError("API returned error when reading domain record", err.Error())
			return
		}
		records = append(records, *record)

	case cregex != "":
		re, err := regexp.Compile(cregex)
		if err != nil {
			resp.Diagnostics.AddError("unable to compile content_regex", err.Error())
			return
		}
		list, err := client.ListDNSDomainRecords(ctx, domain.ID)
		if err != nil {
			resp.Diagnostics.AddError("API returned error when listing domain records", err.Error())
			return
		}
		for _, record := range list.DNSDomainRecords {
			if re.MatchString(record.Content) {
				records = append(records, record)
			}
		}

	case name != "" || rtype != "":
		list, err := client.ListDNSDomainRecords(ctx, domain.ID)
		if err != nil {
			resp.Diagnostics.AddError("API returned error when listing domain records", err.Error())
			return
		}
		for _, record := range list.DNSDomainRecords {
			if name != "" && record.Name != name {
				continue
			}
			if rtype != "" && record.Type != exoscale.DNSDomainRecordType(rtype) {
				continue
			}
			records = append(records, record)
		}

	default:
		resp.Diagnostics.AddError("filter not valid", "at least one of id, name, record_type or content_regex must be set")
		return
	}

	if len(records) == 0 {
		resp.Diagnostics.AddError("no records found", "")
		return
	}

	ids := make([]string, len(records))
	items := make([]DataSourceRecordItemModel, len(records))
	for i, record := range records {
		ids[i] = record.ID.String()
		items[i] = DataSourceRecordItemModel{
			ID:         types.StringValue(record.ID.String()),
			Domain:     types.StringValue(domain.ID.String()),
			Name:       types.StringValue(record.Name),
			Content:    types.StringValue(record.Content),
			RecordType: types.StringValue(string(record.Type)),
			Ttl:        types.Int64Value(record.Ttl),
			Prio:       types.Int64Value(record.Priority),
		}
	}

	state.ID = types.StringValue(fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))) //nolint:gosec
	state.Records = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
