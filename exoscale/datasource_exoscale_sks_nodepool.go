package exoscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TODO implement data source

func dataSourceSKSNodepool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSKSNodepoolRead,
	}
}

func dataSourceSKSNodepoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}
