package exoscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	resDatabaseAttrCreatedAt             = "created_at"
	resDatabaseAttrDiskSize              = "disk_size"
	resDatabaseAttrFeatures              = "features"
	resDatabaseAttrMaintenanceDOW        = "maintenance_dow"
	resDatabaseAttrMaintenanceTime       = "maintenance_time"
	resDatabaseAttrMetadata              = "metadata"
	resDatabaseAttrName                  = "name"
	resDatabaseAttrNodeCPUs              = "node_cpus"
	resDatabaseAttrNodeMemory            = "node_memory"
	resDatabaseAttrNodes                 = "nodes"
	resDatabaseAttrPlan                  = "plan"
	resDatabaseAttrState                 = "state"
	resDatabaseAttrTerminationProtection = "termination_protection"
	resDatabaseAttrType                  = "type"
	resDatabaseAttrUpdatedAt             = "updated_at"
	resDatabaseAttrURI                   = "uri"
	resDatabaseAttrUserConfig            = "user_config"
	resDatabaseAttrZone                  = "zone"
)

func resourceDatabaseIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_database")
}

func resourceDatabase() *schema.Resource {
	s := map[string]*schema.Schema{
		resDatabaseAttrCreatedAt: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resDatabaseAttrDiskSize: {
			Type:     schema.TypeInt,
			Computed: true,
		},
		resDatabaseAttrFeatures: {
			Type:     schema.TypeMap,
			Computed: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resDatabaseAttrMaintenanceDOW: {
			Type:     schema.TypeString,
			Optional: true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(
				[]string{
					"never",
					"monday",
					"tuesday",
					"wednesday",
					"thursday",
					"friday",
					"saturday",
					"sunday",
				},
				false)),
		},
		resDatabaseAttrMaintenanceTime: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resDatabaseAttrMetadata: {
			Type:     schema.TypeMap,
			Computed: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resDatabaseAttrName: {
			Type:     schema.TypeString,
			Required: true,
		},
		resDatabaseAttrNodeCPUs: {
			Type:     schema.TypeInt,
			Computed: true,
		},
		resDatabaseAttrNodeMemory: {
			Type:     schema.TypeInt,
			Computed: true,
		},
		resDatabaseAttrNodes: {
			Type:     schema.TypeInt,
			Computed: true,
		},
		resDatabaseAttrPlan: {
			Type:     schema.TypeString,
			Required: true,
		},
		resDatabaseAttrState: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resDatabaseAttrTerminationProtection: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
		resDatabaseAttrType: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		resDatabaseAttrUpdatedAt: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resDatabaseAttrURI: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		resDatabaseAttrUserConfig: {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		resDatabaseAttrZone: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}

	return &schema.Resource{
		Schema: s,

		CreateContext: resourceDatabaseCreate,
		ReadContext:   resourceDatabaseRead,
		UpdateContext: resourceDatabaseUpdate,
		DeleteContext: resourceDatabaseDelete,

		Importer: &schema.ResourceImporter{
			StateContext: zonedStateContextFunc,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceDatabaseIDString(d))

	zone := d.Get(resDatabaseAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancel()
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))

	client := GetComputeClient(meta)

	database := new(egoscale.DatabaseService)

	maintenanceDOW := d.Get(resDatabaseAttrMaintenanceDOW).(string)
	maintenanceTime := d.Get(resDatabaseAttrMaintenanceTime).(string)
	if maintenanceDOW != "" && maintenanceTime != "" {
		database.Maintenance = &egoscale.DatabaseServiceMaintenance{
			DOW:  maintenanceDOW,
			Time: maintenanceTime,
		}
	}

	if v, ok := d.GetOk(resDatabaseAttrName); ok {
		s := v.(string)
		database.Name = &s
	}

	if v, ok := d.GetOk(resDatabaseAttrPlan); ok {
		s := v.(string)
		database.Plan = &s
	}

	if v, ok := d.GetOk(resDatabaseAttrTerminationProtection); ok {
		b := v.(bool)
		database.TerminationProtection = &b
	}

	if v, ok := d.GetOk(resDatabaseAttrType); ok {
		s := v.(string)
		database.Type = &s
	}

	if v, ok := d.GetOk(resDatabaseAttrUserConfig); ok {
		var userConfig map[string]interface{}
		if err := json.Unmarshal([]byte(v.(string)), &userConfig); err != nil {
			return diag.FromErr(err)
		}
		database.UserConfig = &userConfig
	}

	database, err := client.CreateDatabaseService(ctx, zone, database)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*database.Name)

	log.Printf("[DEBUG] %s: create finished successfully", resourceDatabaseIDString(d))

	return resourceDatabaseRead(ctx, d, meta)
}

func resourceDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceDatabaseIDString(d))

	zone := d.Get(resDatabaseAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	database, err := client.GetDatabaseService(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	// Terraform's TypeMap doesn't support untyped map elements, so we flatten everything
	// to strings as these are only for read-only purposes.
	for k, v := range database.Features {
		database.Features[k] = fmt.Sprint(v)
	}
	for k, v := range database.Metadata {
		database.Metadata[k] = fmt.Sprint(v)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceDatabaseIDString(d))

	return resourceDatabaseApply(ctx, d, database)
}

func resourceDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning update", resourceDatabaseIDString(d))

	zone := d.Get(resDatabaseAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	database, err := client.GetDatabaseService(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var updated bool

	if d.HasChange(resDatabaseAttrMaintenanceDOW) || d.HasChange(resDatabaseAttrMaintenanceTime) {
		database.Maintenance = &egoscale.DatabaseServiceMaintenance{
			DOW:  d.Get(resDatabaseAttrMaintenanceDOW).(string),
			Time: d.Get(resDatabaseAttrMaintenanceTime).(string),
		}
		updated = true
	}

	if d.HasChange(resDatabaseAttrPlan) {
		v := d.Get(resDatabaseAttrPlan).(string)
		database.Plan = &v
		updated = true
	}

	if d.HasChange(resDatabaseAttrTerminationProtection) {
		v := d.Get(resDatabaseAttrTerminationProtection).(bool)
		database.TerminationProtection = &v
		updated = true
	}

	if d.HasChange(resDatabaseAttrUserConfig) {
		var userConfig map[string]interface{}
		if err := json.Unmarshal([]byte(d.Get(resDatabaseAttrUserConfig).(string)), &userConfig); err != nil {
			return diag.FromErr(err)
		}
		database.UserConfig = &userConfig
		updated = true
	}

	if updated {
		if err = client.UpdateDatabaseService(ctx, zone, database); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceDatabaseIDString(d))

	return resourceDatabaseRead(ctx, d, meta)
}

func resourceDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceDatabaseIDString(d))

	zone := d.Get(resDatabaseAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	databaseName := d.Id()
	err := client.DeleteDatabaseService(ctx, zone, &egoscale.DatabaseService{Name: &databaseName})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceDatabaseIDString(d))

	return nil
}

func resourceDatabaseApply(
	_ context.Context,
	d *schema.ResourceData,
	database *egoscale.DatabaseService,
) diag.Diagnostics {
	if err := d.Set(resDatabaseAttrCreatedAt, database.CreatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrDiskSize, *database.DiskSize); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrFeatures, database.Features); err != nil {
		return diag.FromErr(err)
	}

	if database.Maintenance != nil {
		if err := d.Set(resDatabaseAttrMaintenanceDOW, database.Maintenance.DOW); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(resDatabaseAttrMaintenanceTime, database.Maintenance.Time); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resDatabaseAttrMetadata, database.Metadata); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrName, *database.Name); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrNodeCPUs, *database.NodeCPUs); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrNodeMemory, *database.NodeMemory); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrNodes, *database.Nodes); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrPlan, *database.Plan); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrState, *database.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(
		resDatabaseAttrTerminationProtection,
		defaultBool(database.TerminationProtection, false),
	); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrType, *database.Type); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrUpdatedAt, database.UpdatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resDatabaseAttrURI, database.URI.String()); err != nil {
		return diag.FromErr(err)
	}

	if database.UserConfig != nil {
		userConfig, err := json.Marshal(*database.UserConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set(resDatabaseAttrUserConfig, string(userConfig)); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
