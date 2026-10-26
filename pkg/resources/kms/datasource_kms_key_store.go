package kms

import (
	"context"
	"errors"
	"fmt"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const dataSourceKMSKeyStoreDescription = `[BETA] Fetch an Exoscale [KMS](https://community.exoscale.com/product/security/kms/) External Key Store.

Corresponding resource: [exoscale_kms_key_store](../resources/kms_key_store.md).`

var _ datasource.DataSourceWithConfigure = &DataSourceKMSKeyStore{}

// DataSourceKMSKeyStoreModel holds the data of a KMS key store.
type DataSourceKMSKeyStoreModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Zone        types.String `tfsdk:"zone"`
	Description types.String `tfsdk:"description"`
	Proxy       types.Object `tfsdk:"proxy"`
	Type        types.String `tfsdk:"type"`
	Status      types.String `tfsdk:"status"`
	StatusSince types.String `tfsdk:"status_since"`
	CreatedAt   types.String `tfsdk:"created_at"`
	Health      types.Object `tfsdk:"health"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// DataSourceKMSKeyStoreProxyModel holds the XKS proxy settings of a KMS key store. The secret key is never returned.
type DataSourceKMSKeyStoreProxyModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	AccessKey types.String `tfsdk:"access_key"`
}

// Types returns nested data model types to be used for conversion.
func (m DataSourceKMSKeyStoreProxyModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"endpoint":   types.StringType,
		"access_key": types.StringType,
	}
}

type DataSourceKMSKeyStore struct {
	client *exoscale.Client
}

func NewDataSourceKMSKeyStore() datasource.DataSource {
	return &DataSourceKMSKeyStore{}
}

func (d *DataSourceKMSKeyStore) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_key_store"
}

func (d *DataSourceKMSKeyStore) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: dataSourceKMSKeyStoreDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The key store ID to match (conflicts with `name`).",
				Description:         "The key store ID to match (conflicts with 'name').",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The key store name to match (conflicts with `id`).",
				Description:         "The key store name to match (conflicts with 'id').",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("id")),
				},
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Description:         "The Exoscale Zone name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the key store.",
				Description:         "The description of the key store.",
				Computed:            true,
			},
			"proxy": schema.SingleNestedAttribute{
				MarkdownDescription: "The customer-managed XKS proxy. The secret key is never returned.",
				Description:         "The customer-managed XKS proxy. The secret key is never returned.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"endpoint": schema.StringAttribute{
						MarkdownDescription: "The HTTPS base URL of the XKS API on the proxy.",
						Description:         "The HTTPS base URL of the XKS API on the proxy.",
						Computed:            true,
					},
					"access_key": schema.StringAttribute{
						MarkdownDescription: "The access key used to sign requests to the proxy.",
						Description:         "The access key used to sign requests to the proxy.",
						Computed:            true,
					},
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The key store type (`external-key-store`).",
				Description:         "The key store type (external-key-store).",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The connection status of the key store (`connected` or `disconnected`). KMS keys can only be created in, and used from, a connected key store.",
				Description:         "The connection status of the key store (connected or disconnected). KMS keys can only be created in, and used from, a connected key store.",
				Computed:            true,
			},
			"status_since": schema.StringAttribute{
				MarkdownDescription: "When the status last changed.",
				Description:         "When the status last changed.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The creation date of the key store.",
				Description:         "The creation date of the key store.",
				Computed:            true,
			},
			"health": schema.SingleNestedAttribute{
				MarkdownDescription: "The latest periodic health check of the proxy. Only connected key stores are checked, every 10 minutes, and a change to the key store resets it.",
				Description:         "The latest periodic health check of the proxy. Only connected key stores are checked, every 10 minutes, and a change to the key store resets it.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						MarkdownDescription: "`healthy`, `unhealthy` or `unknown`.",
						Description:         "healthy, unhealthy or unknown.",
						Computed:            true,
					},
					"status_reason": schema.StringAttribute{
						MarkdownDescription: "The reason for the status (e.g. `active`, `timeout`, `credential_error`).",
						Description:         "The reason for the status (e.g. active, timeout, credential_error).",
						Computed:            true,
					},
					"checked_at": schema.StringAttribute{
						MarkdownDescription: "When the check completed.",
						Description:         "When the check completed.",
						Computed:            true,
					},
					"error_detail": schema.StringAttribute{
						MarkdownDescription: "A short description of the failure, when the check failed.",
						Description:         "A short description of the failure, when the check failed.",
						Computed:            true,
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (d *DataSourceKMSKeyStore) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceKMSKeyStore) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceKMSKeyStoreModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	var id exoscale.UUID
	switch {
	case !data.ID.IsNull():
		id, err = exoscale.ParseUUID(data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse key store ID", err.Error())
			return
		}

	case !data.Name.IsNull():
		// The list does not include the health of the key stores: only use it to find the ID.
		keyStores, err := client.ListKeyStores(ctx)
		if err != nil {
			resp.Diagnostics.AddError("API error listing KMS key stores", apiErrorMessage(err))
			return
		}
		entry, err := keyStores.FindListKeyStoresResponseEntry(data.Name.ValueString())
		if err != nil {
			if errors.Is(err, exoscale.ErrNotFound) {
				resp.Diagnostics.AddError(
					fmt.Sprintf("KMS key store %q not found", data.Name.ValueString()),
					fmt.Sprintf("No key store named %q in zone %s.", data.Name.ValueString(), data.Zone.ValueString()),
				)
				return
			}
			resp.Diagnostics.AddError("unable to find KMS key store", err.Error())
			return
		}
		id = entry.ID

	default: // ExactlyOneOf prevents this, exit as a safe guard
		resp.Diagnostics.AddError("missing values", "name and id are missing")
		return
	}

	keyStore, err := client.GetKeyStore(ctx, id)
	if err != nil {
		if isNotFound(err, kmsErrKeyStoreNotFound) {
			resp.Diagnostics.AddError(fmt.Sprintf("KMS key store %s not found", id), apiErrorMessage(err))
			return
		}
		resp.Diagnostics.AddError("API error reading KMS key store", apiErrorMessage(err))
		return
	}

	data.ID = types.StringValue(keyStore.ID.String())
	data.Name = types.StringValue(keyStore.Name)
	data.Description = types.StringValue(keyStore.Description)
	data.Type = types.StringValue(string(keyStore.Type))
	data.Status = types.StringValue(string(keyStore.Status))
	data.StatusSince = types.StringValue(keyStore.StatusSince.Format(time.RFC3339))
	data.CreatedAt = types.StringValue(keyStore.CreatedAT.Format(time.RFC3339))

	proxy := DataSourceKMSKeyStoreProxyModel{Endpoint: types.StringNull(), AccessKey: types.StringNull()}
	if keyStore.Proxy != nil {
		proxy.Endpoint = types.StringValue(keyStore.Proxy.Endpoint)
		if keyStore.Proxy.Auth != nil {
			proxy.AccessKey = types.StringValue(keyStore.Proxy.Auth.Key)
		}
	}
	proxyObj, diags := types.ObjectValueFrom(ctx, DataSourceKMSKeyStoreProxyModel{}.Types(), proxy)
	resp.Diagnostics.Append(diags...)
	data.Proxy = proxyObj

	health, diags := keyStoreHealthValue(ctx, keyStore.Health)
	resp.Diagnostics.Append(diags...)
	data.Health = health
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
