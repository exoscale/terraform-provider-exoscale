package exoscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/exoscale/egoscale/v2/oapi"
)

const (
	resDatabaseAttrOpensearchForkFromService                              = "fork_from_service"
	resDatabaseAttrOpensearchRecoveryBackupName                           = "recovery_backup_name"
	resDatabaseAttrOpensearchIndexPatterns                                = "index_pattern"
	resDatabaseAttrOpensearchIndexPatternsPattern                         = "pattern"
	resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm                = "sorting_algorithm"
	resDatabaseAttrOpensearchIndexTemplate                                = "index_template"
	resDatabaseAttrOpensearchIndexTemplateMappingNestedObjectsLimit       = "mapping_nested_objects_limit"
	resDatabaseAttrOpensearchIndexTemplateNumberOfReplicas                = "number_of_replicas"
	resDatabaseAttrOpensearchIndexTemplateNumberOfShards                  = "number_of_shards"
	resDatabaseAttrOpensearchIPFilter                                     = "ip_filter"
	resDatabaseAttrOpensearchKeepIndexRefreshInterval                     = "keep_index_refresh_interval"
	resDatabaseAttrOpensearchMaxIndexCount                                = "max_index_count"
	resDatabaseAttrOpensearchOpensearchDashboards                         = "dashboards"
	resDatabaseAttrOpensearchOpensearchDashboardsEnabled                  = "enabled"
	resDatabaseAttrOpensearchOpensearchDashboardsMaxOldSpaceSize          = "max_old_space_size"
	resDatabaseAttrOpensearchOpensearchDashboardsOpensearchRequestTimeout = "request_timeout"
	resDatabaseAttrOpensearchOpensearchSettings                           = "settings"
	resDatabaseAttrOpensearchVersion                                      = "version"
)

var resDatabaseOpensearchSchema = &schema.Schema{
	Description: "*opensearch* database service type specific arguments.",
	Type:        schema.TypeList,
	MaxItems:    1,
	Optional:    true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			resDatabaseAttrOpensearchForkFromService: {
				Description: "Service name",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
			},
			resDatabaseAttrOpensearchRecoveryBackupName: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			resDatabaseAttrOpensearchIndexPatterns: {
				Description: "(can be used multiple times) Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like 'logs.?' and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note 'logs.?' does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.",
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resDatabaseAttrOpensearchMaxIndexCount: {
							Description: "Maximum number of indexes to keep before deleting the oldest one (Minimum value is `0`)",
							Type:        schema.TypeInt,
							Optional:    true,
						},
						resDatabaseAttrOpensearchIndexPatternsPattern: {
							Description: "fnmatch pattern",
							Type:        schema.TypeString,
							Optional:    true,
						},
						resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm: {
							Description: "`alphabetical` or `creation_date`.",
							Type:        schema.TypeString,
							Optional:    true,
						},
					},
				},
				Optional: true,
			},
			resDatabaseAttrOpensearchIndexTemplate: {
				Description: "Template settings for all new indexes",
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resDatabaseAttrOpensearchIndexTemplateMappingNestedObjectsLimit: {
							Description: "The maximum number of nested JSON objects that a single document can contain across all nested types. This limit helps to prevent out of memory errors when a document contains too many nested objects. (Default is 10000. Minimum value is `0`, maximum value is `100000`.)",
							Type:        schema.TypeInt,
							Optional:    true,
						},
						resDatabaseAttrOpensearchIndexTemplateNumberOfReplicas: {
							Description: "The number of replicas each primary shard has. (Minimum value is `0`, maximum value is `29`)",
							Type:        schema.TypeInt,
							Optional:    true,
						},
						resDatabaseAttrOpensearchIndexTemplateNumberOfShards: {
							Description: "The number of primary shards that an index should have. (Minimum value is `1`, maximum value is `1024`.)",
							Type:        schema.TypeInt,
							Optional:    true,
						},
					},
				},
				Optional: true,
			},
			resDatabaseAttrOpensearchIPFilter: {
				Description: "Allow incoming connections from this list of CIDR address block, e.g. `[\"10.20.0.0/16\"]`",
				Type:        schema.TypeSet,
				Set:         schema.HashString,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsCIDRNetwork(0, 128),
				},
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrOpensearchKeepIndexRefreshInterval: {
				Description: "Aiven automation resets index.refresh_interval to default value for every index to be sure that indices are always visible to search. If it doesn't fit your case, you can disable this by setting up this flag to true.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			resDatabaseAttrOpensearchMaxIndexCount: {
				Description: "Maximum number of indexes to keep (Minimum value is `0`)",
				Type:        schema.TypeInt,
				Optional:    true,
			},
			resDatabaseAttrOpensearchOpensearchDashboards: {
				Type:     schema.TypeList,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resDatabaseAttrOpensearchOpensearchDashboardsEnabled: {
							Description: "{Type -  schema.TypeBool, Optional -  true, Default -  true}",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     true,
						},
						resDatabaseAttrOpensearchOpensearchDashboardsMaxOldSpaceSize: {
							Description: "{Type -  schema.TypeInt, Optional -  true, Default -  128}",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     128,
						},
						resDatabaseAttrOpensearchOpensearchDashboardsOpensearchRequestTimeout: {
							Description: "{Type -  schema.TypeInt, Optional -  true, Default -  30000}",
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     30000,
						},
					},
				},
				Optional: true,
			},
			resDatabaseAttrOpensearchOpensearchSettings: {
				Description: "OpenSearch-specific settings, in json. e.g.`jsonencode({thread_pool_search_size: 64})`. Use `exo x get-dbaas-settings-opensearch` to get a list of available settings.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			resDatabaseAttrOpensearchVersion: {
				Description: "OpenSearch major version.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
		},
	},
}

func resourceDatabaseBuildMaintenanceCreate(dg resourceDataGetter) *struct {
	Dow  oapi.CreateDbaasServiceOpensearchJSONBodyMaintenanceDow "json:\"dow\""
	Time string                                                  "json:\"time\""
} {
	// "RequiredWith" in schema makes sure that dow and time are either both set or both nil
	dow := dg.GetStringPtr(resDatabaseAttrMaintenanceDOW)
	time := dg.GetStringPtr(resDatabaseAttrMaintenanceTime)

	if dow == nil && time == nil {
		return nil
	}

	maintenance := &struct {
		Dow  oapi.CreateDbaasServiceOpensearchJSONBodyMaintenanceDow "json:\"dow\""
		Time string                                                  "json:\"time\""
	}{
		Dow:  oapi.CreateDbaasServiceOpensearchJSONBodyMaintenanceDow(*dow),
		Time: *time,
	}

	return maintenance
}

func resourceDatabaseBuildMaintenanceUpdate(dg resourceDataGetter) *struct {
	Dow  oapi.UpdateDbaasServiceOpensearchJSONBodyMaintenanceDow "json:\"dow\""
	Time string                                                  "json:\"time\""
} {
	// "RequiredWith" in schema makes sure that dow and time are either both set or both nil
	dow := dg.GetStringPtr(resDatabaseAttrMaintenanceDOW)
	time := dg.GetStringPtr(resDatabaseAttrMaintenanceTime)

	if dow == nil && time == nil {
		return nil
	}

	maintenance := &struct {
		Dow  oapi.UpdateDbaasServiceOpensearchJSONBodyMaintenanceDow "json:\"dow\""
		Time string                                                  "json:\"time\""
	}{
		Dow:  oapi.UpdateDbaasServiceOpensearchJSONBodyMaintenanceDow(*dow),
		Time: *time,
	}

	return maintenance
}

func resourceDatabaseBuildIndexPatternsCreate(dgos resourceDataGetter) *[]struct {
	MaxIndexCount    *int64                                                                  `json:"max-index-count,omitempty"`
	Pattern          *string                                                                 `json:"pattern,omitempty"`
	SortingAlgorithm *oapi.CreateDbaasServiceOpensearchJSONBodyIndexPatternsSortingAlgorithm `json:"sorting-algorithm,omitempty"`
} {
	ips := &[]struct {
		MaxIndexCount    *int64                                                                  `json:"max-index-count,omitempty"`
		Pattern          *string                                                                 `json:"pattern,omitempty"`
		SortingAlgorithm *oapi.CreateDbaasServiceOpensearchJSONBodyIndexPatternsSortingAlgorithm `json:"sorting-algorithm,omitempty"`
	}{}

	for _, ip := range dgos.GetList(resDatabaseAttrOpensearchIndexPatterns) {
		*ips = append(*ips, struct {
			MaxIndexCount    *int64                                                                  `json:"max-index-count,omitempty"`
			Pattern          *string                                                                 `json:"pattern,omitempty"`
			SortingAlgorithm *oapi.CreateDbaasServiceOpensearchJSONBodyIndexPatternsSortingAlgorithm `json:"sorting-algorithm,omitempty"`
		}{
			MaxIndexCount:    ip.GetInt64Ptr(resDatabaseAttrOpensearchMaxIndexCount),
			Pattern:          ip.GetStringPtr(resDatabaseAttrOpensearchIndexPatternsPattern),
			SortingAlgorithm: (*oapi.CreateDbaasServiceOpensearchJSONBodyIndexPatternsSortingAlgorithm)(ip.GetStringPtr(resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm)),
		})
	}

	return ips
}

func resourceDatabaseBuildIndexPatternsUpdate(dgos resourceDataGetter) *[]struct {
	MaxIndexCount    *int64                                                                  `json:"max-index-count,omitempty"`
	Pattern          *string                                                                 `json:"pattern,omitempty"`
	SortingAlgorithm *oapi.UpdateDbaasServiceOpensearchJSONBodyIndexPatternsSortingAlgorithm `json:"sorting-algorithm,omitempty"`
} {
	ips := &[]struct {
		MaxIndexCount    *int64                                                                  `json:"max-index-count,omitempty"`
		Pattern          *string                                                                 `json:"pattern,omitempty"`
		SortingAlgorithm *oapi.UpdateDbaasServiceOpensearchJSONBodyIndexPatternsSortingAlgorithm `json:"sorting-algorithm,omitempty"`
	}{}

	for _, ip := range dgos.GetList(resDatabaseAttrOpensearchIndexPatterns) {
		*ips = append(*ips, struct {
			MaxIndexCount    *int64                                                                  `json:"max-index-count,omitempty"`
			Pattern          *string                                                                 `json:"pattern,omitempty"`
			SortingAlgorithm *oapi.UpdateDbaasServiceOpensearchJSONBodyIndexPatternsSortingAlgorithm `json:"sorting-algorithm,omitempty"`
		}{
			MaxIndexCount:    ip.GetInt64Ptr(resDatabaseAttrOpensearchMaxIndexCount),
			Pattern:          ip.GetStringPtr(resDatabaseAttrOpensearchIndexPatternsPattern),
			SortingAlgorithm: (*oapi.UpdateDbaasServiceOpensearchJSONBodyIndexPatternsSortingAlgorithm)(ip.GetStringPtr(resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm)),
		})
	}

	return ips
}

func resourceDatabaseBuildIndexTemplates(dgos resourceDataGetter) *struct {
	MappingNestedObjectsLimit *int64 "json:\"mapping-nested-objects-limit,omitempty\""
	NumberOfReplicas          *int64 "json:\"number-of-replicas,omitempty\""
	NumberOfShards            *int64 "json:\"number-of-shards,omitempty\""
} {
	dgIndexTeamplate := dgos.Under(resDatabaseAttrOpensearchIndexTemplate).Under("0")
	return &struct {
		MappingNestedObjectsLimit *int64 "json:\"mapping-nested-objects-limit,omitempty\""
		NumberOfReplicas          *int64 "json:\"number-of-replicas,omitempty\""
		NumberOfShards            *int64 "json:\"number-of-shards,omitempty\""
	}{
		MappingNestedObjectsLimit: dgIndexTeamplate.GetInt64Ptr(resDatabaseAttrOpensearchIndexTemplateMappingNestedObjectsLimit),
		NumberOfReplicas:          dgIndexTeamplate.GetInt64Ptr(resDatabaseAttrOpensearchIndexTemplateNumberOfReplicas),
		NumberOfShards:            dgIndexTeamplate.GetInt64Ptr(resDatabaseAttrOpensearchIndexTemplateNumberOfShards),
	}
}

func resourceDatabaseBuildOpensearchSettings(ctx context.Context, dgos resourceDataGetter, client *egoscale.Client) (*map[string]interface{}, error) {
	if s := dgos.GetStringPtr(resDatabaseAttrOpensearchOpensearchSettings); s != nil {
		settingsSchema, err := client.GetDbaasSettingsOpensearchWithResponse(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve Database Service settings: %v", err)
		}
		if settingsSchema.StatusCode() != http.StatusOK {
			return nil, fmt.Errorf("API request error: unexpected status %s", settingsSchema.Status())
		}

		settings, err := validateDatabaseServiceSettings(*s, settingsSchema.JSON200.Settings.Opensearch)
		if err != nil {
			return nil, err
		}

		return &settings, nil
	}

	return nil, nil
}

func resourceDatabaseBuildDashboard(dgos resourceDataGetter) *struct {
	Enabled                  *bool  "json:\"enabled,omitempty\""
	MaxOldSpaceSize          *int64 "json:\"max-old-space-size,omitempty\""
	OpensearchRequestTimeout *int64 "json:\"opensearch-request-timeout,omitempty\""
} {
	dgDashboard := dgos.Under(resDatabaseAttrOpensearchOpensearchDashboards).Under("0")

	return &struct {
		Enabled                  *bool  "json:\"enabled,omitempty\""
		MaxOldSpaceSize          *int64 "json:\"max-old-space-size,omitempty\""
		OpensearchRequestTimeout *int64 "json:\"opensearch-request-timeout,omitempty\""
	}{
		Enabled:                  dgDashboard.GetBoolPtr(resDatabaseAttrOpensearchOpensearchDashboardsEnabled),
		MaxOldSpaceSize:          dgDashboard.GetInt64Ptr(resDatabaseAttrOpensearchOpensearchDashboardsMaxOldSpaceSize),
		OpensearchRequestTimeout: dgDashboard.GetInt64Ptr(resDatabaseAttrOpensearchOpensearchDashboardsOpensearchRequestTimeout),
	}
}

func resourceDatabaseCreateOpensearch(ctx context.Context, d *schema.ResourceData, client *egoscale.Client) diag.Diagnostics {
	dg := newResourceDataGetter(d)
	dgos := dg.Under("opensearch").Under("0")

	settings, err := resourceDatabaseBuildOpensearchSettings(ctx, dgos, client)
	if err != nil {
		return diag.FromErr(err)
	}

	req := oapi.CreateDbaasServiceOpensearchJSONRequestBody{
		ForkFromService:          (*oapi.DbaasServiceName)(dgos.GetStringPtr(resDatabaseAttrOpensearchForkFromService)),
		IndexPatterns:            resourceDatabaseBuildIndexPatternsCreate(dgos),
		IndexTemplate:            resourceDatabaseBuildIndexTemplates(dgos),
		IpFilter:                 dgos.GetSet(resDatabaseAttrOpensearchIPFilter),
		KeepIndexRefreshInterval: dgos.GetBoolPtr(resDatabaseAttrOpensearchKeepIndexRefreshInterval),
		Maintenance:              resourceDatabaseBuildMaintenanceCreate(dg),
		MaxIndexCount:            dgos.GetInt64Ptr(resDatabaseAttrOpensearchMaxIndexCount),
		OpensearchDashboards:     resourceDatabaseBuildDashboard(dgos),
		OpensearchSettings:       settings,
		Plan:                     *dg.GetStringPtr(resDatabaseAttrPlan), // required field, never nil
		RecoveryBackupName:       dgos.GetStringPtr(resDatabaseAttrOpensearchRecoveryBackupName),
		TerminationProtection:    dg.GetBoolPtr(resDatabaseAttrTerminationProtection),
		Version:                  dgos.GetStringPtr(resDatabaseAttrOpensearchVersion),
	}

	name := *dg.GetStringPtr(resDatabaseAttrName)
	d.SetId(name)

	res, err := client.CreateDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(name), req)
	if err != nil {
		return diag.FromErr(err)
	}
	if res.StatusCode() != http.StatusOK {
		return diag.Errorf("API request error: unexpected status %s", res.Status())
	}

	return nil
}

func resourceDatabaseUpdateOpensearch(
	ctx context.Context,
	d *schema.ResourceData,
	client *egoscale.Client,
) diag.Diagnostics {
	dg := newResourceDataGetter(d)
	dgos := dg.Under("opensearch").Under("0")

	settings, err := resourceDatabaseBuildOpensearchSettings(ctx, dgos, client)
	if err != nil {
		return diag.FromErr(err)
	}

	databaseService := oapi.UpdateDbaasServiceOpensearchJSONRequestBody{
		IndexPatterns:            resourceDatabaseBuildIndexPatternsUpdate(dgos),
		IndexTemplate:            resourceDatabaseBuildIndexTemplates(dgos),
		IpFilter:                 dgos.GetSet(resDatabaseAttrOpensearchIPFilter),
		KeepIndexRefreshInterval: dgos.GetBoolPtr(resDatabaseAttrOpensearchKeepIndexRefreshInterval),
		Maintenance:              resourceDatabaseBuildMaintenanceUpdate(dg),
		MaxIndexCount:            dgos.GetInt64Ptr(resDatabaseAttrOpensearchMaxIndexCount),
		OpensearchDashboards:     resourceDatabaseBuildDashboard(dgos),
		OpensearchSettings:       settings,
		Plan:                     dg.GetStringPtr(resDatabaseAttrPlan),
		TerminationProtection:    dg.GetBoolPtr(resDatabaseAttrTerminationProtection),
	}

	res, err := client.UpdateDbaasServiceOpensearchWithResponse(
		ctx,
		oapi.DbaasServiceName(*dg.GetStringPtr(resDatabaseAttrName)),
		databaseService,
	)
	if err != nil {
		return diag.FromErr(err)
	}
	if res.StatusCode() != http.StatusOK {
		return diag.Errorf("API request error: unexpected status %s", res.Status())
	}

	return nil
}

func resourceDatabaseApplyOpensearch(ctx context.Context, d *schema.ResourceData, client *egoscale.Client) error {
	resp, err := client.GetDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(d.Id()))
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("API request error: unexpected status %s", resp.Status())
	}

	opensearch := map[string]interface{}{
		resDatabaseAttrOpensearchKeepIndexRefreshInterval: resp.JSON200.KeepIndexRefreshInterval,
		resDatabaseAttrOpensearchMaxIndexCount:            resp.JSON200.MaxIndexCount,
		resDatabaseAttrOpensearchVersion:                  strings.SplitN(*resp.JSON200.Version, ".", 2)[0],
	}

	if v := resp.JSON200.IpFilter; v != nil {
		opensearch[resDatabaseAttrOpensearchIPFilter] = *v
	}

	if resp.JSON200.OpensearchSettings != nil {
		// if indices_fielddata_cache_size is not set, the API returns it as nil
		// this is a fix to avoid a null -> null diff each time
		if (*resp.JSON200.OpensearchSettings)["indices_fielddata_cache_size"] == nil {
			delete(*resp.JSON200.OpensearchSettings, "indices_fielddata_cache_size")
		}

		s, err := json.Marshal(*resp.JSON200.OpensearchSettings)
		if err != nil {
			return fmt.Errorf("failed to json encode settings: %s", err)
		}
		opensearch[resDatabaseAttrOpensearchOpensearchSettings] = string(s)
	}

	if resp.JSON200.IndexPatterns != nil {
		indexPatterns := []map[string]interface{}{}

		for i, ip := range *resp.JSON200.IndexPatterns {
			// opensearch.0.max_index_count may create an index pattern at the
			// end of the array, we exclude it from opensearch.0.index_pattern
			if resp.JSON200.MaxIndexCount != nil && i == len(*resp.JSON200.IndexPatterns)-1 &&
				*ip.Pattern == "*" && *ip.MaxIndexCount == *resp.JSON200.MaxIndexCount && *ip.SortingAlgorithm == "creation_date" {
				continue
			}

			indexPatterns = append(indexPatterns,
				map[string]interface{}{
					resDatabaseAttrOpensearchMaxIndexCount:                 ip.MaxIndexCount,
					resDatabaseAttrOpensearchIndexPatternsPattern:          ip.Pattern,
					resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm: ip.SortingAlgorithm,
				})
		}

		opensearch[resDatabaseAttrOpensearchIndexPatterns] = indexPatterns
	}

	if resp.JSON200.IndexTemplate != nil {
		opensearch[resDatabaseAttrOpensearchIndexTemplate] = []map[string]interface{}{{
			resDatabaseAttrOpensearchIndexTemplateMappingNestedObjectsLimit: resp.JSON200.IndexTemplate.MappingNestedObjectsLimit,
			resDatabaseAttrOpensearchIndexTemplateNumberOfReplicas:          resp.JSON200.IndexTemplate.NumberOfReplicas,
			resDatabaseAttrOpensearchIndexTemplateNumberOfShards:            resp.JSON200.IndexTemplate.NumberOfShards,
		}}
	}

	if resp.JSON200.OpensearchDashboards != nil {
		opensearch[resDatabaseAttrOpensearchOpensearchDashboards] = []map[string]interface{}{{
			resDatabaseAttrOpensearchOpensearchDashboardsEnabled:                  resp.JSON200.OpensearchDashboards.Enabled,
			resDatabaseAttrOpensearchOpensearchDashboardsMaxOldSpaceSize:          resp.JSON200.OpensearchDashboards.MaxOldSpaceSize,
			resDatabaseAttrOpensearchOpensearchDashboardsOpensearchRequestTimeout: resp.JSON200.OpensearchDashboards.OpensearchRequestTimeout,
		}}
	}

	resource := map[string]interface{}{
		resDatabaseAttrCreatedAt:             resp.JSON200.CreatedAt.String(),
		resDatabaseAttrDiskSize:              resp.JSON200.DiskSize,
		resDatabaseAttrName:                  resp.JSON200.Name,
		resDatabaseAttrNodeCPUs:              resp.JSON200.NodeCpuCount,
		resDatabaseAttrNodeMemory:            resp.JSON200.NodeMemory,
		resDatabaseAttrNodes:                 resp.JSON200.NodeCount,
		resDatabaseAttrPlan:                  resp.JSON200.Plan,
		resDatabaseAttrState:                 resp.JSON200.State,
		resDatabaseAttrTerminationProtection: resp.JSON200.TerminationProtection,
		resDatabaseAttrType:                  resp.JSON200.Type,
		resDatabaseAttrUpdatedAt:             resp.JSON200.UpdatedAt.String(),
		resDatabaseAttrURI:                   defaultString(resp.JSON200.Uri, ""),
		resDatabaseAttrMaintenanceDOW:        resp.JSON200.Maintenance.Dow,
		resDatabaseAttrMaintenanceTime:       resp.JSON200.Maintenance.Time,
		"opensearch":                         []interface{}{opensearch},
	}

	for k, v := range resource {
		if err := d.Set(k, v); err != nil {
			return err
		}
	}

	return nil
}
