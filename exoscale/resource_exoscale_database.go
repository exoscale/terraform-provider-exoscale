package exoscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/xeipuuv/gojsonschema"
)

const (
	resDatabaseAttrCreatedAt             = "created_at"
	resDatabaseAttrDiskSize              = "disk_size"
	resDatabaseAttrMaintenanceDOW        = "maintenance_dow"
	resDatabaseAttrMaintenanceTime       = "maintenance_time"
	resDatabaseAttrName                  = "name"
	resDatabaseAttrNodeCPUs              = "node_cpus"
	resDatabaseAttrNodeMemory            = "node_memory"
	resDatabaseAttrNodes                 = "nodes"
	resDatabaseAttrPlan                  = "plan"
	resDatabaseAttrState                 = "state"
	resDatabaseAttrCA                    = "ca_certificate"
	resDatabaseAttrTerminationProtection = "termination_protection"
	resDatabaseAttrType                  = "type"
	resDatabaseAttrURI                   = "uri"
	resDatabaseAttrUpdatedAt             = "updated_at"
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
		resDatabaseAttrMaintenanceDOW: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			RequiredWith: []string{resDatabaseAttrMaintenanceTime},
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
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			RequiredWith: []string{resDatabaseAttrMaintenanceDOW},
		},
		resDatabaseAttrName: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
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
		resDatabaseAttrCA: {
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
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
				"kafka",
				"mysql",
				"pg",
				"redis",
			}, false)),
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
		resDatabaseAttrZone: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},

		"kafka": resDatabaseKafkaSchema,
		"mysql": resDatabaseMysqlSchema,
		"pg":    resDatabasePgSchema,
		"redis": resDatabaseRedisSchema,
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

	var diags diag.Diagnostics
	switch d.Get(resDatabaseAttrType).(string) {
	case "kafka":
		diags = resourceDatabaseCreateKafka(ctx, d, client.Client)
	case "mysql":
		diags = resourceDatabaseCreateMysql(ctx, d, client.Client)
	case "pg":
		diags = resourceDatabaseCreatePg(ctx, d, client.Client)
	case "redis":
		diags = resourceDatabaseCreateRedis(ctx, d, client.Client)
	}

	if diags.HasError() {
		return diags
	}

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

	// This special case corresponds to a resource import, where the type
	// attribute is not provided. We have to list all existing Database
	// Services in the specified zone, identify the one matching the
	// specified ID to figure out its actual type.
	if d.Get(resDatabaseAttrType).(string) == "" {
		databaseServices, err := client.ListDatabaseServices(ctx, zone)
		if err != nil {
			return diag.Errorf("unable to list Database Services: %v", err)
		}

		for _, s := range databaseServices {
			if *s.Name == d.Id() {
				if err := d.Set(resDatabaseAttrType, *s.Type); err != nil {
					return diag.FromErr(err)
				}
				break
			}
		}

		if d.Get(resDatabaseAttrType).(string) == "" {
			return diag.Errorf("Database Service %q not found in zone %s", d.Id(), zone)
		}
	}

	CACertificate, err := client.GetDatabaseCACertificate(ctx, zone)
	if err != nil {
		return diag.Errorf("unable to get CA Certificate: %v", err)
	}

	if err := d.Set(resDatabaseAttrCA, CACertificate); err != nil {
		return diag.FromErr(err)
	}

	databaseServiceType := d.Get(resDatabaseAttrType).(string)

	switch databaseServiceType {
	case "kafka":
		err = resourceDatabaseApplyKafka(ctx, d, client.Client)
	case "mysql":
		err = resourceDatabaseApplyMysql(ctx, d, client.Client)
	case "pg":
		err = resourceDatabaseApplyPg(ctx, d, client.Client)
	case "redis":
		err = resourceDatabaseApplyRedis(ctx, d, client.Client)
	default:
		return diag.Errorf("unsupported Database Service type %q", databaseServiceType)
	}
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceDatabaseIDString(d))

	return nil
}

func resourceDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning update", resourceDatabaseIDString(d))

	zone := d.Get(resDatabaseAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	var diags diag.Diagnostics
	switch d.Get(resDatabaseAttrType).(string) {
	case "kafka":
		diags = resourceDatabaseUpdateKafka(ctx, d, client.Client)
	case "mysql":
		diags = resourceDatabaseUpdateMysql(ctx, d, client.Client)
	case "pg":
		diags = resourceDatabaseUpdatePg(ctx, d, client.Client)
	case "redis":
		diags = resourceDatabaseUpdateRedis(ctx, d, client.Client)
	}

	if diags.HasError() {
		return diags
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

// validateDatabaseServiceSettings validates user-provided JSON-formatted
// Database Service settings against a reference JSON Schema.
func validateDatabaseServiceSettings(in string, schema interface{}) (map[string]interface{}, error) {
	var userSettings map[string]interface{}

	if err := json.Unmarshal([]byte(in), &userSettings); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON: %w", err)
	}

	res, err := gojsonschema.Validate(
		gojsonschema.NewGoLoader(schema),
		gojsonschema.NewStringLoader(in),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to validate JSON Schema: %w", err)
	}

	if !res.Valid() {
		return nil, errors.New(strings.Join(
			func() []string {
				errs := make([]string, len(res.Errors()))
				for i, err := range res.Errors() {
					errs[i] = err.String()
				}
				return errs
			}(),
			"\n",
		))
	}

	return userSettings, nil
}

// parseDtabaseServiceBackupSchedule parses a Database Service backup
// schedule value expressed in HH:MM format and returns the discrete values
// for hour and minute, or an error if the parsing failed.
func parseDatabaseServiceBackupSchedule(v string) (int64, int64, error) {
	parts := strings.Split(v, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid value %q for backup schedule, expecting HH:MM", v)
	}

	backupHour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid value %q for backup schedule hour, must be between 0 and 23", v)
	}

	backupMinute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid value %q for backup schedule minute, must be between 0 and 59", v)
	}

	return int64(backupHour), int64(backupMinute), nil
}
