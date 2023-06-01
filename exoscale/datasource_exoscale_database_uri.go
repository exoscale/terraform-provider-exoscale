package exoscale

import (
	"context"
	"net/http"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	dsDatabaseAttrName = "name"
	dsDatabaseAttrType = "type"
	dsDatabaseAttrURI  = "uri"
	dsDatabaseAttrZone = "zone"
)

func dataSourceDatabaseURI() *schema.Resource {
	return &schema.Resource{
		Description: `Fetch Exoscale [Database](https://community.exoscale.com/documentation/dbaas/) URI data.

Corresponding resource: [exoscale_database](../resources/database.md).`,
		Schema: map[string]*schema.Schema{
			dsDatabaseAttrName: {
				Description: "The database name to match.",
				Type:        schema.TypeString,
				Required:    true,
			},
			dsDatabaseAttrType: {
				Type:     schema.TypeString,
				Required: true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					"kafka",
					"mysql",
					"pg",
					"redis",
					"opensearch",
				}, false)),
				Description: "The type of the database service (`kafka`, `mysql`, `opensearch`, `pg`, `redis`).",
			},
			dsDatabaseAttrURI: {
				Description: "The database service connection URI.",
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
			},
			dsDatabaseAttrZone: {
				Description: "(Required) The Exoscale Zone name.",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
		ReadContext: dataSourceDatabaseURIRead,
	}
}

func dataSourceDatabaseURIRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": general.ResourceIDString(d, "exoscale_database_uri"),
	})

	zone := d.Get(dsDatabaseAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancel()
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))

	client := GetComputeClient(meta)

	databaseServiceName := d.Get(dsDatabaseAttrName).(string)
	databaseServiceType := d.Get(dsDatabaseAttrType).(string)

	var databaseServiceUri string
	switch databaseServiceType {
	case "kafka":
		res, err := client.GetDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(databaseServiceName))
		if err != nil {
			return diag.FromErr(err)
		}
		if res.StatusCode() != http.StatusOK {
			return diag.Errorf("API request error: unexpected status %s", res.Status())
		}
		databaseServiceUri = defaultString(res.JSON200.Uri, "")
	case "mysql":
		res, err := client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(databaseServiceName))
		if err != nil {
			return diag.FromErr(err)
		}
		if res.StatusCode() != http.StatusOK {
			return diag.Errorf("API request error: unexpected status %s", res.Status())
		}
		databaseServiceUri = defaultString(res.JSON200.Uri, "")
	case "pg":
		res, err := client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(databaseServiceName))
		if err != nil {
			return diag.FromErr(err)
		}
		if res.StatusCode() != http.StatusOK {
			return diag.Errorf("API request error: unexpected status %s", res.Status())
		}
		databaseServiceUri = defaultString(res.JSON200.Uri, "")
	case "redis":
		res, err := client.GetDbaasServiceRedisWithResponse(ctx, oapi.DbaasServiceName(databaseServiceName))
		if err != nil {
			return diag.FromErr(err)
		}
		if res.StatusCode() != http.StatusOK {
			return diag.Errorf("API request error: unexpected status %s", res.Status())
		}
		databaseServiceUri = defaultString(res.JSON200.Uri, "")
	case "opensearch":
		res, err := client.GetDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(databaseServiceName))
		if err != nil {
			return diag.FromErr(err)
		}
		if res.StatusCode() != http.StatusOK {
			return diag.Errorf("API request error: unexpected status %s", res.Status())
		}
		databaseServiceUri = defaultString(res.JSON200.Uri, "")
	default:
		return diag.Errorf("unsupported Database Service type %q", databaseServiceType)
	}

	d.SetId(databaseServiceName)

	if err := d.Set(dsDatabaseAttrURI, databaseServiceUri); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": general.ResourceIDString(d, "exoscale_database_uri"),
	})

	return nil
}
