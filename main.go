package main

/* Bootstrap the plugin for Terraform */

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/terraform-providers/terraform-provider-exoscale/exoscale"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: exoscale.Provider,
	})
}
