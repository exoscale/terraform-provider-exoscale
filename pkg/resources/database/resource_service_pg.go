package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	exoscale "github.com/exoscale/egoscale/v2"
	apiv2 "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"

	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourcePgModel struct {
	AdminPassword     types.String `tfsdk:"admin_password"`
	AdminUsername     types.String `tfsdk:"admin_username"`
	BackupSchedule    types.String `tfsdk:"backup_schedule"`
	IpFilter          types.Set    `tfsdk:"ip_filter"`
	Settings          types.String `tfsdk:"pg_settings"`
	Version           types.String `tfsdk:"version"`
	PgbouncerSettings types.String `tfsdk:"pgbouncer_settings"`
	PglookoutSettings types.String `tfsdk:"pglookout_settings"`
	Integrations      types.Set    `tfsdk:"integrations"`
}

// ResourceDbaasIntegrationModel describes a DBaaS service integration enabled
// at creation time (e.g. a `read_replica` integration pointing at a source
// service).
type ResourceDbaasIntegrationModel struct {
	Type          types.String `tfsdk:"type"`
	SourceService types.String `tfsdk:"source_service"`
}

// resourceDbaasIntegrationAttrTypes is the attr.Type map for a single
// element in the `integrations` set.
var resourceDbaasIntegrationAttrTypes = map[string]attr.Type{
	"type":           types.StringType,
	"source_service": types.StringType,
}

// resourceDbaasIntegrationObjectType is the attr.Type of a single element in
// the `integrations` set.
var resourceDbaasIntegrationObjectType = types.ObjectType{AttrTypes: resourceDbaasIntegrationAttrTypes}

// supportedIntegrationTypes lists the integration type values accepted by
// this provider when declared inline on a pg or mysql service. It is
// intentionally narrow: the Exoscale DBaaS API supports additional
// integration types (`logs`, `metrics`, `datasource`) via the standalone
// integration endpoint, but those do not apply to the create-time
// destination model exposed here.
var supportedIntegrationTypes = []string{"read_replica"}

var ResourcePgSchema = schema.SingleNestedAttribute{
	Optional:            true,
	MarkdownDescription: "*pg* database service type specific arguments. Structure is documented below.",
	Attributes: map[string]schema.Attribute{
		"admin_password": schema.StringAttribute{
			MarkdownDescription: "A custom administrator account password (may only be set at creation time).",
			Optional:            true,
			Sensitive:           true,
		},
		"admin_username": schema.StringAttribute{
			MarkdownDescription: "A custom administrator account username (may only be set at creation time).",
			Optional:            true,
		},
		"backup_schedule": schema.StringAttribute{
			MarkdownDescription: "The automated backup schedule (`HH:MM`).",
			Optional:            true,
			Computed:            true,
		},
		"ip_filter": schema.SetAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "A list of CIDR blocks to allow incoming connections from.",
			Optional:            true,
			Computed:            true,
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(validators.IsCIDRNetworkValidator{Min: 0, Max: 128}),
			},
		},
		"pg_settings": schema.StringAttribute{
			MarkdownDescription: "PostgreSQL configuration settings in JSON format (`exo dbaas type show pg --settings=pg` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "PostgreSQL major version (`exo dbaas type show pg` for reference; may only be set at creation time).",
			Optional:            true,
			Computed:            true,
		},
		"pgbouncer_settings": schema.StringAttribute{
			MarkdownDescription: "PgBouncer configuration settings in JSON format (`exo dbaas type show pg --settings=pgbouncer` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"pglookout_settings": schema.StringAttribute{
			MarkdownDescription: "pglookout configuration settings in JSON format (`exo dbaas type show pg --settings=pglookout` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"integrations": ResourceDbaasIntegrationsSchema,
	},
}

// ResourceDbaasIntegrationsSchema describes the `integrations` nested
// attribute exposed on database service resources that support it. An
// integration is configured at service creation time and cannot be updated
// in place, therefore changing it forces resource replacement. It is
// modelled as a set so reordering by the API (on Read) doesn't produce
// spurious plans.
//
// The attribute is Optional+Computed with a `UseStateForUnknown` plan
// modifier so that an operator who omits the attribute from config (most
// commonly after importing a pre-existing replica service) does not see
// Terraform propose a destructive replace simply because the API reports
// an integration that the HCL doesn't mention. The plan value falls back
// to the state value in that case, producing a no-op plan. The
// `RequiresReplace` modifier still fires when the operator actively
// changes a configured integration.
//
// NOTE on the limits of omission: when the operator omits the attribute,
// the provider can only keep refresh and plan stable. It CANNOT preserve
// Terraform's dependency graph — dependency edges come from configuration
// references, not from state — so `terraform destroy` and source
// replacement are NOT fully handled by the provider in that case. The
// only pattern that handles all lifecycle events correctly is declaring
// the `integrations` block explicitly in configuration, with a reference
// to the source's `.name` attribute (which is Required and resolved from
// configuration at plan time):
//
//	source_service = exoscale_dbaas.<primary>.name
//
// References to Computed attributes (.id, .created_at, etc.) must NOT
// be used for source_service. Those attributes carry a
// UseStateForUnknown plan modifier that preserves the old state value
// during a source replacement, suppresses the setRequiresReplace diff
// on the replica, and causes destroy to fail with "Cannot delete ...
// while read replica exists". The schema description spells this out
// in detail for operators.
var ResourceDbaasIntegrationsSchema = schema.SetNestedAttribute{
	MarkdownDescription: "❗ Service integrations declared when the service is created. Only integrations where **this** resource is the destination are supported: for example, to create a PostgreSQL read replica, declare the `integrations` block on the replica (destination) and set `source_service` to the primary's name. Integrations cannot be updated in place — any change to this set destroys and recreates the service (including all data).\n\n" +
		"**Declaring the block explicitly is the recommended pattern** when the source service is also managed by Terraform. Use a reference to the source's `.name` attribute:\n\n" +
		"```hcl\n" +
		"integrations = [{\n" +
		"  type           = \"read_replica\"\n" +
		"  source_service = exoscale_dbaas.<primary>.name\n" +
		"}]\n" +
		"```\n\n" +
		"`.name` is **required** in the source's configuration, so its value is resolved from configuration at plan time and changes propagate through Terraform's dependency graph. When the primary's name changes (or the primary is replaced for any reason), the replica's `source_service` value changes with it, the `setRequiresReplace` plan modifier fires, and the replica is replaced in the correct order. This is the only pattern that handles refresh, plan, destroy, AND source replacement correctly.\n\n" +
		"Computed-name workflows (e.g. `name = \"${random_id.suffix.hex}-primary\"`) are supported: Terraform will normally resolve the primary's name before the replica's create call runs, and the reference is passed through unchanged. In the rare case that the value is still unknown at create time, the provider rejects the create with a clear error naming the `source_service` element, rather than silently submitting an empty value to the API.\n\n" +
		"**Do not reference `Computed` attributes of the source service — notably `.id`, but also `.created_at`, `.state`, and similar — for `source_service`.** These attributes carry a `UseStateForUnknown` plan modifier that copies the prior state value into the plan during a source replacement. That suppresses the `setRequiresReplace` diff on the replica, leaves the replica attached to the destroyed source, and causes `terraform apply` to fail with `Cannot delete ... while read replica exists`. Always use a reference whose value is determined by configuration, not by post-apply computation — in practice, that means `.name`.\n\n" +
		"**Omitting the attribute is safe for refresh and plan only.** On an imported or pre-existing replica, leaving `integrations` out of configuration avoids a spurious forced-replace on refresh (the value is read from the API and preserved in state via `UseStateForUnknown`). However, it removes the Terraform dependency edge, so:\n\n" +
		"- `terraform destroy` may attempt to delete the source before the replica and fail with `Cannot delete ... while read replica exists`. Adding `depends_on = [exoscale_dbaas.<source>]` on the replica restores destroy ordering for **whole-stack destroys only**.\n" +
		"- Replacing the source (rename, zone change, plan change, etc.) does **not** trigger a corresponding replacement of the replica, because Terraform sees no configuration change on the replica. `depends_on` does not fix this — it only affects ordering, not replacement propagation. There is no provider-level workaround for this case; declaring `integrations` explicitly in configuration with `source_service = exoscale_dbaas.<source>.name` is the only complete fix.\n\n" +
		"Omitting the attribute is fully safe when the source service is **not** managed by Terraform in the same state (e.g. an external primary created out-of-band), because there is no dependency graph to preserve.\n\n" +
		"Removing an integration out-of-band (e.g. via the Exoscale dashboard) on a resource that explicitly declares the attribute in configuration still triggers a forced replace on the next plan.",
	Optional: true,
	Computed: true,
	Validators: []validator.Set{
		setvalidator.SizeAtLeast(1),
		integrationsSelfSource(),
	},
	PlanModifiers: []planmodifier.Set{
		setUseStateForUnknown(),
		setRequiresReplace(),
	},
	NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				MarkdownDescription: "❗ Integration type. Currently only `read_replica` is supported.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(supportedIntegrationTypes...),
				},
			},
			"source_service": schema.StringAttribute{
				MarkdownDescription: "❗ Name of the source service to integrate with. For a `read_replica` integration, this is the name of the primary service from which data is replicated.",
				Required:            true,
			},
		},
	},
}

// createPg function handles PostgreSQL specific part of database resource creation logic.
// configData carries the raw configuration so we can distinguish "operator
// omitted integrations" (config null) from "operator set integrations to an
// unknown expression" (config non-null but plan unknown).
func (r *ServiceResource) createPg(ctx context.Context, data *ServiceResourceModel, configData *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	service := oapi.CreateDbaasServicePgJSONRequestBody{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
	}

	if !data.MaintenanceDOW.IsUnknown() && !data.MaintenanceTime.IsUnknown() {
		service.Maintenance = &struct {
			Dow  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	if data.Pg != nil {
		if !data.Pg.Version.IsUnknown() {
			service.Version = data.Pg.Version.ValueStringPointer()
		}

		if !data.Pg.AdminPassword.IsNull() {
			service.AdminPassword = data.Pg.AdminPassword.ValueStringPointer()
		}

		if !data.Pg.AdminUsername.IsNull() {
			service.AdminUsername = data.Pg.AdminUsername.ValueStringPointer()
		}

		if !data.Pg.IpFilter.IsUnknown() {
			obj := []string{}
			if len(data.Pg.IpFilter.Elements()) > 0 {
				dg := data.Pg.IpFilter.ElementsAs(ctx, &obj, true)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}

			service.IpFilter = &obj
		}
		if !data.Pg.BackupSchedule.IsNull() && !data.Pg.BackupSchedule.IsUnknown() {
			bh, bm, err := parseBackupSchedule(data.Pg.BackupSchedule.ValueString())
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("Unable to parse backup schedule, got error: %s", err))
				return
			}

			service.BackupSchedule = &struct {
				BackupHour   *int64 `json:"backup-hour,omitempty"`
				BackupMinute *int64 `json:"backup-minute,omitempty"`
			}{
				BackupHour:   &bh,
				BackupMinute: &bm,
			}
		}

		settingsSchema, err := r.client.GetDbaasSettingsPgWithResponse(ctx)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
			return
		}
		if settingsSchema.StatusCode() != http.StatusOK {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
			return
		}

		if !data.Pg.Settings.IsUnknown() {
			obj, err := validateSettings(data.Pg.Settings.ValueString(), settingsSchema.JSON200.Settings.Pg)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			service.PgSettings = &obj
		}

		if !data.Pg.PgbouncerSettings.IsUnknown() {
			obj, err := validateSettings(data.Pg.PgbouncerSettings.ValueString(), settingsSchema.JSON200.Settings.Pgbouncer)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			service.PgbouncerSettings = &obj
		}

		if !data.Pg.PglookoutSettings.IsUnknown() {
			obj, err := validateSettings(data.Pg.PglookoutSettings.ValueString(), settingsSchema.JSON200.Settings.Pglookout)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			service.PglookoutSettings = &obj
		}

		// P2: If the operator explicitly set integrations in config but
		// the entire set is still unknown at apply time, reject cleanly.
		// When the config value is null the operator omitted the
		// attribute, which is correct for a standalone service.
		//
		// Note: this is likely unreachable for the current schema shape
		// (SetNestedAttribute with Required string children cannot be
		// assigned from a scalar unknown), but it guards against future
		// schema changes or unexpected framework behavior.
		if data.Pg.Integrations.IsUnknown() && configData.Pg != nil && !configData.Pg.Integrations.IsNull() {
			diagnostics.AddError(
				"pg.integrations: entire integrations set is unknown at create time",
				"The `integrations` attribute was set in configuration but its value "+
					"is still unknown at apply time. This usually means the entire set is "+
					"derived from a resource output that Terraform could not resolve before "+
					"creating this service. Set `integrations` to a list of objects with "+
					"known `type` and `source_service` values, for example:\n\n"+
					"  integrations = [{ type = \"read_replica\", source_service = exoscale_dbaas.<primary>.name }]",
			)
			return
		}

		if !data.Pg.Integrations.IsNull() && !data.Pg.Integrations.IsUnknown() {
			var integrationModels []ResourceDbaasIntegrationModel
			// allowUnhandled=true so unknown per-field values in the
			// set (e.g. source_service pointing at a computed
			// attribute of another resource that is unknown at
			// plan time) do not produce a hard decoding error.
			// types.String already represents unknown natively so
			// this is mostly defense-in-depth, but it also
			// matches the validator's behavior for consistency.
			if dg := data.Pg.Integrations.ElementsAs(ctx, &integrationModels, true); dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
			// Validate every element in a single pass:
			//  1. Reject null/unknown fields — the plan-time validator
			//     is lenient about unknowns, but by apply time
			//     ValueString() would return "" for unknowns and we'd
			//     silently submit source-service="" to the API.
			//  2. Re-run the semantic checks (type ∈ supported, source
			//     ≠ self) that plan-time validators skipped for unknown
			//     nested values.
			selfName := data.Name.ValueString()
			for i, integration := range integrationModels {
				if integration.Type.IsNull() || integration.Type.IsUnknown() {
					diagnostics.AddError(
						"pg.integrations: element type is unknown or null at create time",
						fmt.Sprintf(
							"Element %d of the `pg.integrations` set has no concrete "+
								"`type` value at the time the service is being created. "+
								"This usually means `type` depends on a computed value "+
								"that Terraform could not resolve before the service create "+
								"call. Set `type` to a literal value (currently only "+
								"\"read_replica\" is supported).",
							i,
						),
					)
					return
				}
				if integration.SourceService.IsNull() || integration.SourceService.IsUnknown() {
					diagnostics.AddError(
						"pg.integrations: source_service is unknown or null at create time",
						fmt.Sprintf(
							"Element %d of the `pg.integrations` set has no concrete "+
								"`source_service` value at the time the service is being "+
								"created. This usually means `source_service` references a "+
								"value that Terraform could not resolve before the create "+
								"call — for example, a primary service whose name is itself "+
								"computed from another resource that has not been applied "+
								"yet. Ensure `source_service` references a plan-time-known "+
								"value, typically `exoscale_dbaas.<primary>.name` where the "+
								"primary's name is a literal or a fully-resolved expression.",
							i,
						),
					)
					return
				}
				typeVal := integration.Type.ValueString()
				supported := false
				for _, t := range supportedIntegrationTypes {
					if typeVal == t {
						supported = true
						break
					}
				}
				if !supported {
					diagnostics.AddError(
						"pg.integrations: unsupported integration type",
						fmt.Sprintf(
							"Element %d of the `pg.integrations` set has type %q, "+
								"which is not a supported integration type. "+
								"Supported types: %s.",
							i, typeVal, strings.Join(supportedIntegrationTypes, ", "),
						),
					)
					return
				}
				if integration.SourceService.ValueString() == selfName {
					diagnostics.AddError(
						"pg.integrations: self-referencing source_service",
						fmt.Sprintf(
							"Element %d of the `pg.integrations` set has "+
								"source_service=%q, which is the same service this "+
								"resource declares. Integrations must be declared on "+
								"the destination side, with source_service pointing "+
								"at a different service.",
							i, selfName,
						),
					)
					return
				}
			}
			if len(integrationModels) > 0 {
				integrations := make([]struct {
					DestService   *oapi.DbaasServiceName                            `json:"dest-service,omitempty"`
					Settings      *map[string]interface{}                           `json:"settings,omitempty"`
					SourceService *oapi.DbaasServiceName                            `json:"source-service,omitempty"`
					Type          oapi.CreateDbaasServicePgJSONBodyIntegrationsType `json:"type"`
				}, len(integrationModels))
				for i, integration := range integrationModels {
					source := oapi.DbaasServiceName(integration.SourceService.ValueString())
					integrations[i].SourceService = &source
					integrations[i].Type = oapi.CreateDbaasServicePgJSONBodyIntegrationsType(integration.Type.ValueString())
				}
				service.Integrations = &integrations
			}
		}
	}

	resp, err := r.client.CreateDbaasServicePgWithResponse(
		ctx,
		oapi.DbaasServiceName(data.Name.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service pg, got error: %s", err))
		return
	}
	if resp.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable create database service pg, unexpected status: %s", resp.Status()))
		return
	}

	// The service now exists on Exoscale. Any error return after this
	// point must delete it first — otherwise we orphan a billable
	// database service that Terraform state never knew about, which
	// cannot be cleaned up by CheckDestroy, taint, or any normal
	// workflow. Cleanup uses a fresh, bounded context detached from
	// the request ctx (which may already be cancelled if we got here
	// via ctx.Done()).
	createSucceeded := false
	defer func() {
		if createSucceeded {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		cleanupCtx = apiv2.WithEndpoint(cleanupCtx, apiv2.NewReqEndpoint(r.env, data.Zone.ValueString()))
		name := data.Id.ValueString()
		if delErr := r.client.DeleteDatabaseService(
			cleanupCtx,
			data.Zone.ValueString(),
			&exoscale.DatabaseService{Name: &name},
		); delErr != nil {
			tflog.Warn(ctx, fmt.Sprintf(
				"orphan cleanup failed after createPg error for service %q: %v",
				name, delErr,
			))
		} else {
			tflog.Info(ctx, fmt.Sprintf(
				"orphan cleanup: deleted partially-created pg service %q",
				name,
			))
		}
	}()

	tflog.Info(ctx, "DB Service created, waiting for the service to be in 'running' state")
	apiService := &oapi.DbaasServicePg{}
pooling:
	for {
		select {
		case <-ctx.Done():
			diagnostics.AddError("Error", ctx.Err().Error())
			return
		default:
			res, err := r.client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, got error: %s", err))
				return
			}
			if res.StatusCode() != http.StatusOK {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, unexpected status: %s", res.Status()))
				return
			}
			if res.JSON200.State != nil {
				if *res.JSON200.State == oapi.EnumServiceStatePoweroff {
					diagnostics.AddError("Client Error", fmt.Sprintf("Unexpected service state: %s", *res.JSON200.State))
					return
				}
				if *res.JSON200.State == oapi.EnumServiceStateRunning {
					apiService = res.JSON200
					break pooling
				}
			}
			time.Sleep(time.Second * 2)
		}
	}

	// Fill in unknown values.
	caCert, err := r.client.GetDatabaseCACertificate(ctx, data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}
	data.CA = types.StringValue(caCert)

	data.CreatedAt = types.StringValue(apiService.CreatedAt.String())
	data.DiskSize = types.Int64PointerValue(apiService.DiskSize)
	data.NodeCPUs = types.Int64PointerValue(apiService.NodeCpuCount)
	data.NodeMemory = types.Int64PointerValue(apiService.NodeMemory)
	data.Nodes = types.Int64PointerValue(apiService.NodeCount)
	data.State = types.StringPointerValue((*string)(apiService.State))
	data.UpdatedAt = types.StringValue(apiService.UpdatedAt.String())

	if data.TerminationProtection.IsUnknown() {
		data.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	}

	if data.MaintenanceDOW.IsUnknown() && data.MaintenanceTime.IsUnknown() {
		data.MaintenanceDOW = types.StringNull()
		data.MaintenanceTime = types.StringNull()
		if apiService.Maintenance != nil {
			data.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
			data.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
		}
	}

	if data.Pg.BackupSchedule.IsUnknown() {
		data.Pg.BackupSchedule = types.StringNull()
		if apiService.BackupSchedule != nil {
			backupHour := types.Int64PointerValue(apiService.BackupSchedule.BackupHour)
			backupMinute := types.Int64PointerValue(apiService.BackupSchedule.BackupMinute)
			data.Pg.BackupSchedule = types.StringValue(fmt.Sprintf(
				"%02d:%02d",
				backupHour.ValueInt64(),
				backupMinute.ValueInt64(),
			))
		}
	}

	if data.Pg.IpFilter.IsUnknown() {
		data.Pg.IpFilter = types.SetNull(types.StringType)
		if apiService.IpFilter != nil {
			v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}

			data.Pg.IpFilter = v
		}
	}

	if data.Pg.Version.IsUnknown() {
		data.Pg.Version = types.StringNull()
		if apiService.Version != nil {
			data.Pg.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
		}
	}

	if data.Pg.Settings.IsUnknown() {
		data.Pg.Settings = types.StringNull()
		if apiService.PgSettings != nil {
			settings, err := json.Marshal(*apiService.PgSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Pg.Settings = types.StringValue(string(settings))
		}
	}

	if data.Pg.PgbouncerSettings.IsUnknown() {
		data.Pg.PgbouncerSettings = types.StringNull()
		if apiService.PgbouncerSettings != nil {
			settings, err := json.Marshal(*apiService.PgbouncerSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Pg.PgbouncerSettings = types.StringValue(string(settings))
		}
	}

	if data.Pg.PglookoutSettings.IsUnknown() {
		data.Pg.PglookoutSettings = types.StringNull()
		if apiService.PglookoutSettings != nil {
			settings, err := json.Marshal(*apiService.PglookoutSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Pg.PglookoutSettings = types.StringValue(string(settings))
		}
	}

	// Integrations is Optional+Computed: if the operator did not set
	// the attribute in config, the framework left the plan value
	// unknown at the Create phase. Populate it from the API response
	// so the post-create state has a concrete value. Only surface
	// integrations where the current service is the destination,
	// matching readPg's filter.
	if data.Pg.Integrations.IsUnknown() {
		data.Pg.Integrations = types.SetNull(resourceDbaasIntegrationObjectType)
		if apiService.Integrations != nil {
			var models []ResourceDbaasIntegrationModel
			for _, integration := range *apiService.Integrations {
				if integration.Dest == nil || *integration.Dest != data.Id.ValueString() {
					continue
				}
				models = append(models, ResourceDbaasIntegrationModel{
					Type:          types.StringPointerValue(integration.Type),
					SourceService: types.StringPointerValue(integration.Source),
				})
			}
			if len(models) > 0 {
				v, dg := types.SetValueFrom(ctx, resourceDbaasIntegrationObjectType, models)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
				data.Pg.Integrations = v
			}
		}
	}

	// All post-create population succeeded — cancel the deferred
	// orphan cleanup. Any panic between here and the return still
	// allows the defer to run.
	createSucceeded = true
}

// readPg function handles PostgreSQL specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *ServiceResource) readPg(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) (clearState bool) {

	res, err := r.client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
	if err != nil {
		if errors.Is(err, apiv2.ErrNotFound) {
			return true
		}
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, unexpected status: %s", res.Status()))
		return false
	}

	caCert, err := r.client.GetDatabaseCACertificate(ctx, data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return false
	}
	data.CA = types.StringValue(caCert)

	apiService := res.JSON200

	data.CreatedAt = types.StringValue(apiService.CreatedAt.String())
	data.DiskSize = types.Int64PointerValue(apiService.DiskSize)
	data.NodeCPUs = types.Int64PointerValue(apiService.NodeCpuCount)
	data.NodeMemory = types.Int64PointerValue(apiService.NodeMemory)
	data.Nodes = types.Int64PointerValue(apiService.NodeCount)
	data.Plan = types.StringValue(apiService.Plan)
	data.State = types.StringPointerValue((*string)(apiService.State))
	data.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	data.UpdatedAt = types.StringValue(apiService.UpdatedAt.String())

	data.MaintenanceDOW = types.StringNull()
	data.MaintenanceTime = types.StringNull()
	if apiService.Maintenance != nil {
		data.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
		data.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
	}

	// Database block is required but it may be nil during import.
	if data.Pg == nil {
		data.Pg = &ResourcePgModel{}
	}

	data.Pg.BackupSchedule = types.StringNull()
	if apiService.BackupSchedule != nil {
		backupHour := types.Int64PointerValue(apiService.BackupSchedule.BackupHour)
		backupMinute := types.Int64PointerValue(apiService.BackupSchedule.BackupMinute)
		data.Pg.BackupSchedule = types.StringValue(fmt.Sprintf(
			"%02d:%02d",
			backupHour.ValueInt64(),
			backupMinute.ValueInt64(),
		))
	}

	data.Pg.IpFilter = types.SetNull(types.StringType)
	if apiService.IpFilter != nil {
		v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return false
		}

		data.Pg.IpFilter = v
	}

	data.Pg.Version = types.StringNull()
	if apiService.Version != nil {
		data.Pg.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
	}

	// For database settings, we have a special behaviour:
	// - If not set in plan we handle it as normal computed field;
	// - If set in plan we only read (manage) key(s) that were set!
	//   This prevents showing drift due to Aiven setting default values for keys not specified in plan.
	if data.Pg.Settings.IsUnknown() || apiService.PgSettings == nil {
		data.Pg.Settings = types.StringNull()
		if apiService.PgSettings != nil {
			settings, err := json.Marshal(*apiService.PgSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return false
			}

			data.Pg.Settings = types.StringValue(string(settings))
		}
	} else if data.Pg.Settings.ValueString() != "" {
		var userSettings map[string]any

		if err := json.Unmarshal([]byte(data.Pg.Settings.ValueString()), &userSettings); err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("unable to unmarshal JSON: %s", err))
			return false
		}

		PartialSettingsPatch(userSettings, *apiService.PgSettings)
		settings, err := json.Marshal(userSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Pg.Settings = types.StringValue(string(settings))
	}

	if data.Pg.PgbouncerSettings.IsUnknown() || apiService.PgbouncerSettings == nil {
		data.Pg.PgbouncerSettings = types.StringNull()
		if apiService.PgbouncerSettings != nil {
			settings, err := json.Marshal(*apiService.PgbouncerSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return false
			}

			data.Pg.PgbouncerSettings = types.StringValue(string(settings))
		}
	} else if data.Pg.PgbouncerSettings.ValueString() != "" {
		var userSettings map[string]any

		if err := json.Unmarshal([]byte(data.Pg.PgbouncerSettings.ValueString()), &userSettings); err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("unable to unmarshal JSON: %s", err))
			return false
		}

		PartialSettingsPatch(userSettings, *apiService.PgbouncerSettings)
		settings, err := json.Marshal(userSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Pg.PgbouncerSettings = types.StringValue(string(settings))
	}

	if data.Pg.PglookoutSettings.IsUnknown() || apiService.PglookoutSettings == nil {
		data.Pg.PglookoutSettings = types.StringNull()
		if apiService.PglookoutSettings != nil {
			settings, err := json.Marshal(*apiService.PglookoutSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return false
			}

			data.Pg.PglookoutSettings = types.StringValue(string(settings))
		}
	} else if data.Pg.PglookoutSettings.ValueString() != "" {
		var userSettings map[string]any

		if err := json.Unmarshal([]byte(data.Pg.PglookoutSettings.ValueString()), &userSettings); err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("unable to unmarshal JSON: %s", err))
			return false
		}

		PartialSettingsPatch(userSettings, *apiService.PglookoutSettings)
		settings, err := json.Marshal(userSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Pg.PglookoutSettings = types.StringValue(string(settings))
	}

	// Only surface integrations where the current service is the destination,
	// so that Terraform state reflects exactly what the resource's config
	// declares (i.e. integrations the user asked to create for this service).
	data.Pg.Integrations = types.SetNull(resourceDbaasIntegrationObjectType)
	if apiService.Integrations != nil {
		var integrationModels []ResourceDbaasIntegrationModel
		for _, integration := range *apiService.Integrations {
			if integration.Dest == nil || *integration.Dest != data.Id.ValueString() {
				continue
			}
			integrationModels = append(integrationModels, ResourceDbaasIntegrationModel{
				Type:          types.StringPointerValue(integration.Type),
				SourceService: types.StringPointerValue(integration.Source),
			})
		}
		if len(integrationModels) > 0 {
			v, dg := types.SetValueFrom(ctx, resourceDbaasIntegrationObjectType, integrationModels)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return false
			}
			data.Pg.Integrations = v
		}
	}

	return false
}

// updatePg function handles PostgreSQL specific part of database resource Update logic.
func (r *ServiceResource) updatePg(ctx context.Context, stateData *ServiceResourceModel, planData *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	service := oapi.UpdateDbaasServicePgJSONRequestBody{}

	if (!planData.MaintenanceDOW.Equal(stateData.MaintenanceDOW) && !planData.MaintenanceDOW.IsUnknown()) ||
		(!planData.MaintenanceTime.Equal(stateData.MaintenanceTime) && !planData.MaintenanceTime.IsUnknown()) {
		service.Maintenance = &struct {
			Dow  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow(planData.MaintenanceDOW.ValueString()),
			Time: planData.MaintenanceTime.ValueString(),
		}
		stateData.MaintenanceDOW = planData.MaintenanceDOW
		stateData.MaintenanceTime = planData.MaintenanceTime
		updated = true
	}

	if !planData.Plan.Equal(stateData.Plan) {
		service.Plan = planData.Plan.ValueStringPointer()
		stateData.Plan = planData.Plan
		updated = true
	}

	if !planData.TerminationProtection.Equal(stateData.TerminationProtection) {
		service.TerminationProtection = planData.TerminationProtection.ValueBoolPointer()
		stateData.TerminationProtection = planData.TerminationProtection
		updated = true
	}

	if planData.Pg != nil {
		if stateData.Pg == nil {
			stateData.Pg = &ResourcePgModel{}
		}

		if !planData.Pg.BackupSchedule.IsUnknown() && !planData.Pg.BackupSchedule.Equal(stateData.Pg.BackupSchedule) {
			bh, bm, err := parseBackupSchedule(planData.Pg.BackupSchedule.ValueString())
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("Unable to parse backup schedule, got error: %s", err))
				return
			}

			service.BackupSchedule = &struct {
				BackupHour   *int64 `json:"backup-hour,omitempty"`
				BackupMinute *int64 `json:"backup-minute,omitempty"`
			}{
				BackupHour:   &bh,
				BackupMinute: &bm,
			}
			stateData.Pg.BackupSchedule = planData.Pg.BackupSchedule
			updated = true
		}

		if !planData.Pg.IpFilter.Equal(stateData.Pg.IpFilter) {
			obj := []string{}
			if len(planData.Pg.IpFilter.Elements()) > 0 {
				dg := planData.Pg.IpFilter.ElementsAs(ctx, &obj, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}
			service.IpFilter = &obj
			stateData.Pg.IpFilter = planData.Pg.IpFilter
			updated = true
		}

		settingsSchema, err := r.client.GetDbaasSettingsPgWithResponse(ctx)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
			return
		}
		if settingsSchema.StatusCode() != http.StatusOK {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
			return
		}

		if !planData.Pg.Settings.Equal(stateData.Pg.Settings) {
			if planData.Pg.Settings.ValueString() != "" {
				obj, err := validateSettings(planData.Pg.Settings.ValueString(), settingsSchema.JSON200.Settings.Pg)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Pg settings: %s", err))
					return
				}
				service.PgSettings = &obj
			}
			stateData.Pg.Settings = planData.Pg.Settings
			updated = true
		}

		if !planData.Pg.PgbouncerSettings.Equal(stateData.Pg.PgbouncerSettings) {
			if planData.Pg.PgbouncerSettings.ValueString() != "" {
				obj, err := validateSettings(planData.Pg.PgbouncerSettings.ValueString(), settingsSchema.JSON200.Settings.Pgbouncer)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Pgbouncer settings: %s", err))
					return
				}
				service.PgbouncerSettings = &obj
			}
			stateData.Pg.PgbouncerSettings = planData.Pg.PgbouncerSettings
			updated = true
		}

		if !planData.Pg.PglookoutSettings.Equal(stateData.Pg.PglookoutSettings) {
			if planData.Pg.PglookoutSettings.ValueString() != "" {
				obj, err := validateSettings(planData.Pg.PglookoutSettings.ValueString(), settingsSchema.JSON200.Settings.Pglookout)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Pglookout settings: %s", err))
					return
				}
				service.PglookoutSettings = &obj
			}
			stateData.Pg.PglookoutSettings = planData.Pg.PglookoutSettings
			updated = true
		}

		// Defensive no-op: integrations is Optional with a RequiresReplace
		// plan modifier, so Terraform Core never calls Update when the
		// set differs from state. Keep stateData in sync with plan here
		// as a safety net in case that contract is ever weakened.
		stateData.Pg.Integrations = planData.Pg.Integrations
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]any{})
		return
	}

	// Aiven would overwrite the backup schedule with random value if we don't specify it explicitly every time.
	if service.BackupSchedule == nil && !stateData.Pg.BackupSchedule.IsUnknown() {
		bh, bm, err := parseBackupSchedule(stateData.Pg.BackupSchedule.ValueString())
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("Unable to parse backup schedule, got error: %s", err))
			return
		}

		service.BackupSchedule = &struct {
			BackupHour   *int64 `json:"backup-hour,omitempty"`
			BackupMinute *int64 `json:"backup-minute,omitempty"`
		}{
			BackupHour:   &bh,
			BackupMinute: &bm,
		}
	}

	if res, err := r.client.UpdateDbaasServicePgWithResponse(
		ctx,
		oapi.DbaasServiceName(planData.Id.ValueString()),
		service,
	); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service pg, got error: %s", err))
		return
	} else if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service pg, unexpected status: %s", res.Status()))
		return
	}

	apiService := &oapi.DbaasServicePg{}
	if res, err := r.client.GetDbaasServicePgWithResponse(
		ctx,
		oapi.DbaasServiceName(stateData.Id.ValueString()),
	); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, got error: %s", err))
		return
	} else if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, unexpected status: %s", res.Status()))
		return
	} else {
		apiService = res.JSON200
	}

	// Fill in unknown values.
	stateData.NodeCPUs = types.Int64PointerValue(apiService.NodeCpuCount)
	stateData.NodeMemory = types.Int64PointerValue(apiService.NodeMemory)
	stateData.Nodes = types.Int64PointerValue(apiService.NodeCount)
	stateData.State = types.StringPointerValue((*string)(apiService.State))
	if apiService.UpdatedAt != nil {
		stateData.UpdatedAt = types.StringValue(apiService.UpdatedAt.String())
	}
	if stateData.TerminationProtection.IsUnknown() {
		stateData.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	}

	if stateData.Pg == nil {
		return
	}

	if stateData.Pg.IpFilter.IsUnknown() {
		stateData.Pg.IpFilter = types.SetNull(types.StringType)
		if apiService.IpFilter != nil {
			v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
			stateData.Pg.IpFilter = v
		}
	}

	if stateData.Pg.Version.IsUnknown() {
		stateData.Pg.Version = types.StringNull()
		if apiService.Version != nil {
			stateData.Pg.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
		}
	}

	if stateData.Pg.Settings.IsUnknown() {
		stateData.Pg.Settings = types.StringNull()
		if apiService.PgSettings != nil {
			settings, err := json.Marshal(*apiService.PgSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			stateData.Pg.Settings = types.StringValue(string(settings))
		}
	}

	if stateData.Pg.PgbouncerSettings.IsUnknown() {
		stateData.Pg.PgbouncerSettings = types.StringNull()
		if apiService.PgbouncerSettings != nil {
			settings, err := json.Marshal(*apiService.PgbouncerSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			stateData.Pg.PgbouncerSettings = types.StringValue(string(settings))
		}
	}

	if stateData.Pg.PglookoutSettings.IsUnknown() {
		stateData.Pg.PglookoutSettings = types.StringNull()
		if apiService.PglookoutSettings != nil {
			settings, err := json.Marshal(*apiService.PglookoutSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			stateData.Pg.PglookoutSettings = types.StringValue(string(settings))
		}
	}
}
